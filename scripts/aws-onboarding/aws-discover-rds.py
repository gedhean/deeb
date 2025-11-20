#!/usr/bin/env python3
"""
AWS RDS Discovery Script

Scans an AWS account for RDS instances and Aurora clusters,
then generates a YAML configuration file grouped by Environment tag.
"""

import boto3
import yaml
import logging
import sys
import argparse
from typing import Dict, List, Any, Optional
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)


def get_rds_client():
    """Get RDS client using default AWS credentials."""
    try:
        return boto3.client("rds")
    except Exception as e:
        logger.error(f"Failed to create RDS client: {e}")
        sys.exit(1)


def get_instance_tags(rds_client, resource_arn: str) -> Dict[str, str]:
    """Retrieve tags for an RDS resource."""
    try:
        response = rds_client.list_tags_for_resource(ResourceName=resource_arn)
        return {tag["Key"]: tag["Value"] for tag in response.get("TagList", [])}
    except Exception as e:
        logger.warning(f"Failed to get tags for {resource_arn}: {e}")
        return {}


def process_rds_instances(
    rds_client, filter_environment: Optional[str] = None
) -> tuple[Dict[str, List[Dict[str, Any]]], List[str]]:
    """
    Process RDS instances and return grouped instances and non-secrets-manager list.

    Args:
        rds_client: Boto3 RDS client
        filter_environment: Optional environment tag value to filter by (case-insensitive)

    Returns:
        tuple: (environments_dict, no_secrets_manager_list)
    """
    environments = {}
    no_secrets_manager = []

    try:
        paginator = rds_client.get_paginator("describe_db_instances")

        for page in paginator.paginate():
            for instance in page["DBInstances"]:
                instance_id = instance["DBInstanceIdentifier"]
                arn = instance["DBInstanceArn"]

                logger.info(f"Processing RDS instance: {instance_id}")

                # Get tags
                tags = get_instance_tags(rds_client, arn)
                environment = tags.get("Environment", "untagged")

                # Filter by environment if specified
                if filter_environment and environment.lower() != filter_environment.lower():
                    logger.debug(
                        f"Skipping {instance_id} - environment '{environment}' doesn't match filter '{filter_environment}'"
                    )
                    continue

                # Check if master secret is managed by Secrets Manager
                master_secret = instance.get("MasterUserSecret")

                if not master_secret:
                    logger.warning(
                        f"Instance {instance_id} does not use AWS Secrets Manager for master credentials"
                    )
                    no_secrets_manager.append(
                        f"{instance_id} (Environment: {environment})"
                    )
                    continue

                # Determine engine type
                engine = instance["Engine"]
                if "mysql" in engine.lower():
                    engine_type = "mysql"
                elif "postgres" in engine.lower():
                    engine_type = "postgresql"
                else:
                    logger.warning(
                        f"Unsupported engine type for {instance_id}: {engine}"
                    )
                    continue

                # Build instance info
                instance_info = {
                    "name": instance_id,
                    "engine": engine_type,
                    "endpoint": instance["Endpoint"]["Address"],
                    "port": instance["Endpoint"]["Port"],
                    "master_secret_arn": master_secret["SecretArn"],
                    "is_cluster": False,
                    "db_instance_identifier": instance_id,
                    "arn": arn,
                }

                # Group by environment
                if environment not in environments:
                    environments[environment] = []
                environments[environment].append(instance_info)

                logger.info(
                    f"Added RDS instance {instance_id} to environment '{environment}'"
                )

    except Exception as e:
        logger.error(f"Failed to process RDS instances: {e}")
        raise

    return environments, no_secrets_manager


def process_aurora_clusters(
    rds_client, filter_environment: Optional[str] = None
) -> tuple[Dict[str, List[Dict[str, Any]]], List[str]]:
    """
    Process Aurora clusters and return grouped clusters and non-secrets-manager list.

    Args:
        rds_client: Boto3 RDS client
        filter_environment: Optional environment tag value to filter by (case-insensitive)

    Returns:
        tuple: (environments_dict, no_secrets_manager_list)
    """
    environments = {}
    no_secrets_manager = []

    try:
        paginator = rds_client.get_paginator("describe_db_clusters")

        for page in paginator.paginate():
            for cluster in page["DBClusters"]:
                cluster_id = cluster["DBClusterIdentifier"]
                arn = cluster["DBClusterArn"]

                logger.info(f"Processing Aurora cluster: {cluster_id}")

                # Get tags
                tags = get_instance_tags(rds_client, arn)
                environment = tags.get("Environment", "untagged")

                # Filter by environment if specified
                if filter_environment and environment.lower() != filter_environment.lower():
                    logger.debug(
                        f"Skipping {cluster_id} - environment '{environment}' doesn't match filter '{filter_environment}'"
                    )
                    continue

                # Check if master secret is managed by Secrets Manager
                master_secret = cluster.get("MasterUserSecret")

                if not master_secret:
                    logger.warning(
                        f"Cluster {cluster_id} does not use AWS Secrets Manager for master credentials"
                    )
                    no_secrets_manager.append(
                        f"{cluster_id} (Environment: {environment}, Type: Aurora Cluster)"
                    )
                    continue

                # Determine engine type
                engine = cluster["Engine"]
                if "mysql" in engine.lower() or "aurora" in engine.lower():
                    engine_type = "mysql"
                elif "postgres" in engine.lower():
                    engine_type = "postgresql"
                else:
                    logger.warning(
                        f"Unsupported engine type for {cluster_id}: {engine}"
                    )
                    continue

                # Build cluster info
                cluster_info = {
                    "name": cluster_id,
                    "engine": engine_type,
                    "endpoint": cluster["Endpoint"],
                    "port": cluster["Port"],
                    "master_secret_arn": master_secret["SecretArn"],
                    "is_cluster": True,
                    "db_cluster_identifier": cluster_id,
                    "arn": arn,
                }

                # Group by environment
                if environment not in environments:
                    environments[environment] = []
                environments[environment].append(cluster_info)

                logger.info(
                    f"Added Aurora cluster {cluster_id} to environment '{environment}'"
                )

    except Exception as e:
        logger.error(f"Failed to process Aurora clusters: {e}")
        raise

    return environments, no_secrets_manager


def merge_environments(env1: Dict, env2: Dict) -> Dict:
    """Merge two environment dictionaries."""
    merged = env1.copy()
    for env, instances in env2.items():
        if env in merged:
            merged[env].extend(instances)
        else:
            merged[env] = instances
    return merged


def write_yaml_config(environments: Dict[str, List[Dict[str, Any]]], output_file: str):
    """Write the discovered RDS resources to a YAML file."""
    config = {"environments": environments}

    try:
        with open(output_file, "w") as f:
            yaml.dump(config, f, default_flow_style=False, sort_keys=True, indent=2)
        logger.info(f"Configuration written to {output_file}")
    except Exception as e:
        logger.error(f"Failed to write YAML config: {e}")
        raise


def write_no_secrets_manager_log(instances: List[str], log_file: str):
    """Write instances without Secrets Manager to a log file."""
    if not instances:
        logger.info("All instances use AWS Secrets Manager for master credentials")
        return

    try:
        with open(log_file, "w") as f:
            f.write("# RDS Instances/Clusters without AWS Secrets Manager\n")
            f.write(f"# Generated at: {datetime.now().isoformat()}\n\n")
            for instance in instances:
                f.write(f"{instance}\n")
        logger.warning(
            f"Found {len(instances)} instances without Secrets Manager. See {log_file}"
        )
    except Exception as e:
        logger.error(f"Failed to write no-secrets-manager log: {e}")
        raise


def parse_arguments():
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description='Discover AWS RDS instances and Aurora clusters',
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  # Discover all RDS instances/clusters
  %(prog)s
  
  # Discover only production databases
  %(prog)s --environment production
  
  # Discover only staging databases
  %(prog)s --environment staging
        """
    )
    
    parser.add_argument(
        '--environment', '-e',
        type=str,
        help='Filter by Environment tag (case-insensitive). Only discovers instances/clusters with matching Environment tag. Default: discover all environments.',
        metavar='ENV'
    )
    
    parser.add_argument(
        '--output', '-o',
        type=str,
        default='rds-discovered.yaml',
        help='Output YAML file path (default: rds-discovered.yaml)',
        metavar='FILE'
    )
    
    parser.add_argument(
        '--verbose', '-v',
        action='store_true',
        help='Enable verbose/debug logging'
    )
    
    return parser.parse_args()


def main():
    """Main execution function."""
    args = parse_arguments()
    
    # Set log level
    if args.verbose:
        logger.setLevel(logging.DEBUG)
    
    logger.info("Starting AWS RDS Discovery")
    
    if args.environment:
        logger.info(f"Filtering by Environment tag: '{args.environment}'")
    else:
        logger.info("Discovering ALL environments (no filter applied)")
    
    # Get RDS client
    rds_client = get_rds_client()
    region = rds_client.meta.region_name
    logger.info(f"Using AWS region: {region}")
    
    # Process RDS instances
    logger.info("Scanning RDS instances...")
    instance_envs, instance_no_sm = process_rds_instances(rds_client, args.environment)
    
    # Process Aurora clusters
    logger.info("Scanning Aurora clusters...")
    cluster_envs, cluster_no_sm = process_aurora_clusters(rds_client, args.environment)

    # Merge results
    all_environments = merge_environments(instance_envs, cluster_envs)
    all_no_secrets_manager = instance_no_sm + cluster_no_sm

    # Summary
    total_instances = sum(len(instances) for instances in all_environments.values())
    logger.info("\nDiscovery Summary:")
    logger.info(f"  Total environments: {len(all_environments)}")
    logger.info(f"  Total instances/clusters: {total_instances}")
    for env, instances in sorted(all_environments.items()):
        logger.info(f"    {env}: {len(instances)} instance(s)")

    # Write outputs
    output_yaml = "rds-discovered.yaml"
    log_file = "rds-no-secrets-manager.log"

    write_yaml_config(all_environments, output_yaml)
    write_no_secrets_manager_log(all_no_secrets_manager, log_file)

    logger.info("\nDiscovery completed successfully!")
    logger.info(
        f"Review the configuration in '{output_yaml}' before running the onboarding script."
    )


if __name__ == "__main__":
    try:
        main()
    except KeyboardInterrupt:
        logger.info("\nDiscovery interrupted by user")
        sys.exit(1)
    except Exception as e:
        logger.error(f"Discovery failed: {e}")
        sys.exit(1)
