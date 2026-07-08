#!/usr/bin/env python3
"""
Shared RDS/Aurora discovery helpers.

Enumerates RDS instances and Aurora clusters in the current AWS account/region and
returns a flat list of onboarding candidates. Only resources whose master credentials
are managed by AWS Secrets Manager are returned; the rest are reported separately so
the caller can surface them.

Each candidate carries its `tags` (a {key: value} dict) so the caller can apply tag
filters (e.g. only onboard instances tagged Environment=production). This module is
account-scoped: the target Deeb environment and project are decided by the AWS account
the sweep runs in (see the CronJob config).
"""

import logging
from typing import Any, Dict, List, Tuple

logger = logging.getLogger(__name__)


def _engine_type(engine: str) -> str:
    """Map an RDS engine string to Deeb's engine value, or '' if unsupported."""
    engine = engine.lower()
    if "mysql" in engine or "aurora" in engine:
        return "mysql"
    if "postgres" in engine:
        return "postgresql"
    return ""


def _tags(rds_client, arn: str) -> Dict[str, str]:
    """Return the resource's tags as a {key: value} dict (empty on failure)."""
    try:
        resp = rds_client.list_tags_for_resource(ResourceName=arn)
        return {t["Key"]: t["Value"] for t in resp.get("TagList", [])}
    except Exception as e:  # noqa: BLE001 - tags are best-effort; never fail discovery
        logger.warning("Could not list tags for %s: %s", arn, e)
        return {}


def discover_instances(rds_client) -> Tuple[List[Dict[str, Any]], List[str]]:
    """Return (candidates, skipped) for standalone RDS DB instances."""
    candidates: List[Dict[str, Any]] = []
    skipped: List[str] = []

    paginator = rds_client.get_paginator("describe_db_instances")
    for page in paginator.paginate():
        for instance in page["DBInstances"]:
            instance_id = instance["DBInstanceIdentifier"]

            master_secret = instance.get("MasterUserSecret")
            if not master_secret:
                skipped.append(f"{instance_id} (no Secrets Manager master credentials)")
                continue

            engine = _engine_type(instance["Engine"])
            if not engine:
                logger.warning("Unsupported engine for %s: %s", instance_id, instance["Engine"])
                continue

            # A standalone instance that is part of a cluster is onboarded via its cluster.
            if instance.get("DBClusterIdentifier"):
                continue

            arn = instance["DBInstanceArn"]
            candidates.append(
                {
                    "name": instance_id,
                    "engine": engine,
                    "endpoint": instance["Endpoint"]["Address"],
                    "port": instance["Endpoint"]["Port"],
                    "master_secret_arn": master_secret["SecretArn"],
                    "is_cluster": False,
                    "arn": arn,
                    "tags": _tags(rds_client, arn),
                }
            )

    return candidates, skipped


def discover_clusters(rds_client) -> Tuple[List[Dict[str, Any]], List[str]]:
    """Return (candidates, skipped) for Aurora DB clusters."""
    candidates: List[Dict[str, Any]] = []
    skipped: List[str] = []

    paginator = rds_client.get_paginator("describe_db_clusters")
    for page in paginator.paginate():
        for cluster in page["DBClusters"]:
            cluster_id = cluster["DBClusterIdentifier"]

            master_secret = cluster.get("MasterUserSecret")
            if not master_secret:
                skipped.append(f"{cluster_id} (Aurora cluster, no Secrets Manager master credentials)")
                continue

            engine = _engine_type(cluster["Engine"])
            if not engine:
                logger.warning("Unsupported engine for %s: %s", cluster_id, cluster["Engine"])
                continue

            arn = cluster["DBClusterArn"]
            candidates.append(
                {
                    "name": cluster_id,
                    "engine": engine,
                    "endpoint": cluster["Endpoint"],
                    "port": cluster["Port"],
                    "master_secret_arn": master_secret["SecretArn"],
                    "is_cluster": True,
                    "arn": arn,
                    "tags": _tags(rds_client, arn),
                }
            )

    return candidates, skipped


def discover_all(rds_client) -> Tuple[List[Dict[str, Any]], List[str]]:
    """Discover all onboarding candidates (instances + clusters) in the account/region."""
    instances, skipped_i = discover_instances(rds_client)
    clusters, skipped_c = discover_clusters(rds_client)
    return instances + clusters, skipped_i + skipped_c
