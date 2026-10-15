#!/usr/bin/env python3
"""
AWS RDS/Aurora -> Deeb multi-account onboarding sweep.

A single central sweep (deployed once, near Deeb) onboards RDS/Aurora databases across
MANY AWS accounts into one Deeb instance. For each account listed in the config it
assumes a per-account IAM role, then in each configured region it:

  1. Discovers RDS instances and Aurora clusters (only those with Secrets Manager
     master credentials).
  2. Skips instances already registered in Deeb (matched by host) - so a DB already
     onboarded under any name is never re-registered and its deeb user/password never churned.
  3. For each new instance: creates the `deeb` database user, registers the instance in
     Deeb under the account's mapped environment, and assigns the account's mapped
     project to every synced database.

Because instance names can collide across accounts (Deeb does not enforce a unique name),
the registered Deeb instance name is disambiguated per account as `"<identifier> (<alias>)"`.

Configuration:

  Environment variables:
    DEEB_API_URL          Base URL of the Deeb API (required)
    DEEB_API_TOKEN        API token (deeb_pat_...) for a service account (required)
    DEEB_ONBOARDER_CONFIG Path to the accounts YAML (default: /etc/deeb-onboarder/accounts.yaml)

  Accounts YAML (DEEB_ONBOARDER_CONFIG):
    accounts:
      - account_id: "111111111111"
        alias: "prod"                # short label; disambiguates instance names
        role_arn: "arn:aws:iam::111111111111:role/deeb-onboarder"  # empty => ambient creds
        regions: ["us-east-1"]
        environment: "Production"
        project: "Proj Production"

Flags:
  --dry-run              Discover and report intended actions without making changes.
"""

import json
import logging
import os
import secrets
import socket
import string
import sys
from typing import Any, Dict, List, Optional

import boto3
import psycopg2
import pymysql
import requests
import yaml
from psycopg2 import sql
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT

import discover

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

DEFAULT_CONFIG_PATH = "/etc/deeb-onboarder/accounts.yaml"
DEEB_USER = "deeb"
PASSWORD_LENGTH = 32
CONNECT_TIMEOUT_SECONDS = 30
PAGE_SIZE = 100
ASSUME_ROLE_SESSION_NAME = "deeb-onboarder"

# Only onboard instances whose tags match this filter, unless overridden in config.
# A filter is {tag_key: [allowed_value, ...]}; an instance must, for every key, carry
# that tag with a value in the allowed list (values matched case-insensitively).
# Set `tag_filters: {}` in config to onboard regardless of tags.
DEFAULT_TAG_FILTERS = {"Environment": ["prod", "production", "staging", "homol", "tooling", "ai-prod"]}

# Tag whose value drives per-account env/project overrides (see normalize_env_overrides).
# Overridable per account via `env_override_tag`.
ENV_OVERRIDE_TAG = "Environment"

# RDS-protected roles that must not be granted to the deeb user.
RDS_PROTECTED_ROLES = {"rdsadmin", "rds_superuser", "rds_replication", "rds_password"}


class ConfigError(Exception):
    """Raised when required configuration is missing or invalid."""


def load_config() -> Dict[str, Any]:
    """Build config from environment variables + the accounts YAML file.

    Returns a dict with deeb_api_url, deeb_api_token, and a validated `accounts` list.
    Fails fast (ConfigError) on any missing/invalid piece.
    """
    deeb_api_url = os.environ.get("DEEB_API_URL", "").rstrip("/")
    deeb_api_token = os.environ.get("DEEB_API_TOKEN", "")
    config_path = os.environ.get("DEEB_ONBOARDER_CONFIG", DEFAULT_CONFIG_PATH)

    missing = [k for k, v in {"DEEB_API_URL": deeb_api_url, "DEEB_API_TOKEN": deeb_api_token}.items() if not v]
    if missing:
        raise ConfigError("Missing required environment variables: " + ", ".join(missing))

    try:
        with open(config_path, "r") as f:
            raw = yaml.safe_load(f) or {}
    except OSError as e:
        raise ConfigError(f"Could not read accounts config at {config_path}: {e}")

    accounts = raw.get("accounts")
    if not isinstance(accounts, list) or not accounts:
        raise ConfigError(f"{config_path} must contain a non-empty 'accounts' list")

    # Top-level default; may be overridden per account. Absent => DEFAULT_TAG_FILTERS.
    default_tag_filters = (
        normalize_tag_filters("tag_filters", raw["tag_filters"])
        if "tag_filters" in raw
        else DEFAULT_TAG_FILTERS
    )
    normalized = [normalize_account(i, a, default_tag_filters) for i, a in enumerate(accounts)]

    return {
        "deeb_api_url": deeb_api_url,
        "deeb_api_token": deeb_api_token,
        "accounts": normalized,
    }


def normalize_tag_filters(where: str, raw: Any) -> Dict[str, List[str]]:
    """Validate a tag-filter map: {tag_key: [allowed_value, ...]}. {} means no filtering."""
    if raw is None:
        return {}
    if not isinstance(raw, dict):
        raise ConfigError(f"{where} must be a map of tag_key -> [allowed values]")
    filters: Dict[str, List[str]] = {}
    for key, values in raw.items():
        if isinstance(values, (str, int, float)):
            values = [values]
        if not isinstance(values, list) or not values:
            raise ConfigError(f"{where}.{key} must be a non-empty list of allowed values")
        filters[str(key)] = [str(v) for v in values]
    return filters


def normalize_account(index: int, account: Dict[str, Any], default_tag_filters: Dict[str, List[str]]) -> Dict[str, Any]:
    """Validate and normalize a single account entry (alias defaults to account_id)."""
    if not isinstance(account, dict):
        raise ConfigError(f"accounts[{index}] must be a mapping")

    account_id = str(account.get("account_id", "")).strip()
    environment = str(account.get("environment", "")).strip()
    project = str(account.get("project", "")).strip()
    regions = account.get("regions")

    problems = []
    if not account_id:
        problems.append("account_id")
    if not environment:
        problems.append("environment")
    if not project:
        problems.append("project")
    if not isinstance(regions, list) or not regions:
        problems.append("regions (non-empty list)")
    if problems:
        raise ConfigError(f"accounts[{index}] is missing/invalid: {', '.join(problems)}")

    tag_filters = (
        normalize_tag_filters(f"accounts[{index}].tag_filters", account["tag_filters"])
        if "tag_filters" in account
        else default_tag_filters
    )

    env_overrides = normalize_env_overrides(f"accounts[{index}].env_overrides", account.get("env_overrides"))

    return {
        "account_id": account_id,
        "alias": str(account.get("alias") or account_id),
        "role_arn": str(account.get("role_arn") or "").strip(),
        "regions": [str(r) for r in regions],
        "environment": environment,
        "project": project,
        "tag_filters": tag_filters,
        "env_override_tag": str(account.get("env_override_tag") or ENV_OVERRIDE_TAG),
        "env_overrides": env_overrides,
    }


def normalize_env_overrides(context: str, raw: Any) -> Dict[str, Dict[str, str]]:
    """Normalize an account's env_overrides map: {tag_value: {environment, project}}.

    Routes instances to a different Deeb environment/project based on a tag value (the
    `env_override_tag`, default `Environment`) — e.g. so `Environment=tooling` DBs in the
    production account land in the Tooling env/project rather than the account default.
    Tag-value keys are lowercased for case-insensitive matching. Empty/absent -> {}.
    """
    if raw is None:
        return {}
    if not isinstance(raw, dict):
        raise ConfigError(f"{context} must be a map of tag_value -> {{environment, project}}")

    overrides: Dict[str, Dict[str, str]] = {}
    for tag_value, mapping in raw.items():
        env = str((mapping or {}).get("environment", "")).strip() if isinstance(mapping, dict) else ""
        proj = str((mapping or {}).get("project", "")).strip() if isinstance(mapping, dict) else ""
        if not env or not proj:
            raise ConfigError(f"{context}.{tag_value} must set both 'environment' and 'project'")
        overrides[str(tag_value).strip().lower()] = {"environment": env, "project": proj}
    return overrides


def matches_tag_filters(tags: Dict[str, str], filters: Dict[str, List[str]]) -> bool:
    """True if `tags` satisfies every filter: each key present with a value in its
    allowed list (case-insensitive). Empty filters match everything."""
    for key, allowed in filters.items():
        value = tags.get(key)
        if value is None:
            return False
        if value.lower() not in {a.lower() for a in allowed}:
            return False
    return True


def composed_name(identifier: str, alias: str) -> str:
    """Disambiguate a DB identifier across accounts for the Deeb instance name."""
    return f"{identifier} ({alias})"


def _extract_list(data: Any, *keys: str) -> List[Dict[str, Any]]:
    """Extract a list from an API response that may be a bare list or a wrapped object."""
    if isinstance(data, list):
        return data
    if isinstance(data, dict):
        for key in keys:
            if isinstance(data.get(key), list):
                return data[key]
    raise ValueError(f"Unexpected response shape; expected a list under one of {keys}")


class DeebOnboarder:
    """Onboards discovered RDS/Aurora resources across multiple AWS accounts into Deeb."""

    def __init__(self, config: Dict[str, Any], dry_run: bool = False):
        self.deeb_api_url = config["deeb_api_url"]
        self.accounts = config["accounts"]
        self.dry_run = dry_run

        # Base identity (IRSA) used to assume the per-account roles.
        self.sts = boto3.client("sts")

        self.session = requests.Session()
        self.session.headers.update(
            {
                "Content-Type": "application/json",
                "Authorization": f"Bearer {config['deeb_api_token']}",
            }
        )

    # --- AWS sessions -----------------------------------------------------

    def session_for(self, account: Dict[str, Any]) -> boto3.Session:
        """Return a boto3 Session for the account: assume its role, or use ambient creds."""
        role_arn = account["role_arn"]
        if not role_arn:
            return boto3.Session()

        resp = self.sts.assume_role(RoleArn=role_arn, RoleSessionName=ASSUME_ROLE_SESSION_NAME)
        creds = resp["Credentials"]
        return boto3.Session(
            aws_access_key_id=creds["AccessKeyId"],
            aws_secret_access_key=creds["SecretAccessKey"],
            aws_session_token=creds["SessionToken"],
        )

    # --- Deeb API lookups -------------------------------------------------

    def _get_named_id(self, path: str, name: str, *list_keys: str) -> Optional[str]:
        """GET a collection and return the id of the entry whose name matches (case-insensitive)."""
        response = self.session.get(f"{self.deeb_api_url}{path}")
        response.raise_for_status()
        items = _extract_list(response.json(), *list_keys)
        for item in items:
            if item.get("name", "").lower() == name.lower():
                return item["id"]
        return None

    def resolve_environment_id(self, name: str) -> str:
        env_id = self._get_named_id("/api/v1/environments", name, "environments", "data", "items")
        if not env_id:
            raise ValueError(f"Deeb environment not found: {name!r}")
        return env_id

    def resolve_project_id(self, name: str) -> str:
        project_id = self._get_named_id("/api/v1/projects", name, "projects", "data", "items")
        if not project_id:
            raise ValueError(f"Deeb project not found: {name!r}")
        return project_id

    def registered_instances_by_host(self) -> Dict[str, Dict[str, Any]]:
        """Return {host_lower: instance_dict} for everything registered in Deeb.

        Host is the dedup key: it uniquely identifies the RDS endpoint regardless of
        the name an instance was registered under."""
        by_host: Dict[str, Dict[str, Any]] = {}
        offset = 0
        while True:
            response = self.session.get(
                f"{self.deeb_api_url}/api/v1/database-instances",
                params={"limit": PAGE_SIZE, "offset": offset},
            )
            response.raise_for_status()
            instances = _extract_list(response.json(), "database_instances", "data", "items")
            for inst in instances:
                host = (inst.get("host") or "").strip().lower()
                if host:
                    by_host[host] = inst
            if len(instances) < PAGE_SIZE:
                break
            offset += PAGE_SIZE
        return by_host

    def registered_instance_hosts(self) -> set:
        """Return the set of hosts (lowercased) already registered in Deeb.

        Host is the dedup key: it uniquely identifies the RDS endpoint regardless of the
        name an instance was registered under, so a DB already onboarded under any name
        (e.g. a manual/earlier import) is not registered again."""
        hosts = set()
        offset = 0
        while True:
            response = self.session.get(
                f"{self.deeb_api_url}/api/v1/database-instances",
                params={"limit": PAGE_SIZE, "offset": offset},
            )
            response.raise_for_status()
            instances = _extract_list(response.json(), "database_instances", "data", "items")
            for inst in instances:
                host = (inst.get("host") or "").strip().lower()
                if host:
                    hosts.add(host)
            if len(instances) < PAGE_SIZE:
                break
            offset += PAGE_SIZE
        return hosts

    # --- AWS + DB-user provisioning --------------------------------------

    def get_master_credentials(self, secretsmanager_client, secret_arn: str) -> Dict[str, str]:
        response = secretsmanager_client.get_secret_value(SecretId=secret_arn)
        credentials = json.loads(response["SecretString"])
        return {"username": credentials.get("username"), "password": credentials.get("password")}

    def generate_secure_password(self) -> str:
        alphabet = string.ascii_letters + string.digits
        return "".join(secrets.choice(alphabet) for _ in range(PASSWORD_LENGTH))

    def _resolve_host(self, host: str):
        try:
            socket.gethostbyname(host)
        except socket.gaierror as dns_error:
            raise ConnectionError(
                f"Cannot resolve hostname '{host}': {dns_error}. "
                "Ensure the sweep has a network path (peering/TGW/PrivateLink) and "
                "security-group ingress to the database."
            )

    def ensure_mysql_grants(self, cursor) -> None:
        """Apply the full deeb MySQL grant set. Idempotent — safe to run any
        number of times. Shared by create_mysql_deeb_user and the
        reconcile-existing-instances path so privileges stay convergent."""
        cursor.execute(
            """
            GRANT ALTER, ALTER ROUTINE, CREATE, CREATE ROUTINE, CREATE VIEW, DROP,
            EVENT, EXECUTE, INDEX, PROCESS, REFERENCES, SHOW DATABASES, SHOW VIEW, TRIGGER, USAGE,
            RELOAD, LOCK TABLES, REPLICATION CLIENT, REPLICATION SLAVE
            ON *.* TO 'deeb'@'%' WITH GRANT OPTION
            """
        )
        try:
            cursor.execute("GRANT SET_USER_ID ON *.* TO 'deeb'@'%'")
        except Exception as e:
            logger.debug("Could not grant SET_USER_ID (may not be available): %s", e)
        cursor.execute("FLUSH PRIVILEGES")

    def ensure_mysql_ghost_instruments(self, cursor) -> None:
        """Enable the metadata-lock instrument gh-ost requires (PR #1536).
        Skipped with a clear log when performance_schema is OFF — the UPDATE
        would no-op until PS is enabled via parameter group + reboot."""
        cursor.execute("SHOW VARIABLES LIKE 'performance_schema'")
        row = cursor.fetchone()
        ps_on = row is not None and str(row[1]).upper() == "ON"
        if not ps_on:
            logger.warning(
                "performance_schema is OFF on this instance — skipping metadata-lock "
                "instrument enable. gh-ost dry runs will fail with 'metadata lock "
                "instrument not enabled' until performance_schema=ON is set in the "
                "RDS parameter group and the instance is rebooted."
            )
            return
        cursor.execute(
            """
            UPDATE performance_schema.setup_instruments
            SET ENABLED='YES', TIMED='YES'
            WHERE NAME='wait/lock/metadata/sql/mdl'
            """
        )
        logger.info("Enabled metadata-lock instrument (wait/lock/metadata/sql/mdl)")

    def ensure_mysql_grants_on_instance(self, host: str, port: int, master_user: str, master_password: str) -> None:
        """Re-apply deeb grants on an already-onboarded instance without
        touching the user's password (no rotation)."""
        if self.dry_run:
            logger.info("[DRY RUN] Would ensure MySQL deeb grants on %s:%s", host, port)
            return
        self._resolve_host(host)
        connection = pymysql.connect(
            host=host, port=port, user=master_user, password=master_password,
            connect_timeout=CONNECT_TIMEOUT_SECONDS,
        )
        try:
            cursor = connection.cursor()
            self.ensure_mysql_grants(cursor)
            self.ensure_mysql_ghost_instruments(cursor)
            connection.commit()
            logger.info("Ensured MySQL deeb grants on %s:%s", host, port)
        finally:
            connection.close()

    def reset_mysql_deeb_user_password(self, host: str, port: int, master_user: str, master_password: str, deeb_password: str) -> None:
        """Reset the deeb user's password (and pin the plugin) via master
        credentials. Used when the stored credential no longer authenticates —
        e.g. plugin flip on engine upgrade or a recreated user."""
        if self.dry_run:
            logger.info("[DRY RUN] Would reset MySQL deeb user password on %s:%s", host, port)
            return
        self._resolve_host(host)
        connection = pymysql.connect(
            host=host, port=port, user=master_user, password=master_password,
            connect_timeout=CONNECT_TIMEOUT_SECONDS,
        )
        try:
            cursor = connection.cursor()
            safe_password = deeb_password.replace("'", "''")
            cursor.execute(
                f"ALTER USER 'deeb'@'%' IDENTIFIED WITH caching_sha2_password BY '{safe_password}'"
            )
            self.ensure_mysql_grants(cursor)
            self.ensure_mysql_ghost_instruments(cursor)
            connection.commit()
            logger.info("Reset MySQL deeb user password on %s:%s", host, port)
        finally:
            connection.close()

    def create_mysql_deeb_user(self, host: str, port: int, master_user: str, master_password: str, deeb_password: str) -> bool:
        """Provision the deeb MySQL user. Returns True when the password was
        (re)set — user created or plugin migrated — so the caller knows to
        register/update the stored credentials. Password is never rotated on
        subsequent runs."""
        if self.dry_run:
            logger.info("[DRY RUN] Would create MySQL deeb user on %s:%s", host, port)
            return True

        self._resolve_host(host)
        connection = pymysql.connect(
            host=host, port=port, user=master_user, password=master_password,
            connect_timeout=CONNECT_TIMEOUT_SECONDS,
        )
        try:
            cursor = connection.cursor()

            # Check if the user exists; create it only when missing so we never
            # drop-and-recreate (avoids wiping a working user on re-runs).
            cursor.execute("SELECT plugin FROM mysql.user WHERE user = %s AND host = %s", ("deeb", "%"))
            row = cursor.fetchone()
            exists = row is not None
            current_plugin = row[0] if row else None

            safe_password = deeb_password.replace("'", "''")
            password_set = False

            if not exists:
                cursor.execute(
                    f"CREATE USER 'deeb'@'%' IDENTIFIED WITH caching_sha2_password BY '{safe_password}'"
                )
                logger.info("Created MySQL deeb user on %s:%s", host, port)
                password_set = True
            elif current_plugin != "caching_sha2_password":
                # One-time plugin migration (e.g. mysql_native_password →
                # caching_sha2_password) — the only moment an existing user's
                # password is (re)set. Old password never needed (master
                # credentials perform the reset).
                cursor.execute(
                    f"ALTER USER 'deeb'@'%' IDENTIFIED WITH caching_sha2_password BY '{safe_password}'"
                )
                logger.info(
                    "Migrated MySQL deeb user plugin %s → caching_sha2_password on %s:%s",
                    current_plugin, host, port,
                )
                password_set = True
            else:
                logger.info("MySQL deeb user already provisioned on %s:%s; ensuring grants only (password untouched)", host, port)

            # GRANT is idempotent in MySQL — re-granting existing privileges
            # is a no-op, so we can always run it to ensure the grants are right.
            self.ensure_mysql_grants(cursor)
            self.ensure_mysql_ghost_instruments(cursor)

            connection.commit()
            logger.info("Ensured MySQL deeb user grants on %s:%s", host, port)
            return password_set
        finally:
            connection.close()

    def create_postgres_deeb_user(self, host: str, port: int, master_user: str, master_password: str, deeb_password: str):
        if self.dry_run:
            logger.info("[DRY RUN] Would create PostgreSQL deeb user on %s:%s", host, port)
            return

        self._resolve_host(host)
        connection = psycopg2.connect(
            host=host, port=port, user=master_user, password=master_password,
            database="postgres", connect_timeout=CONNECT_TIMEOUT_SECONDS,
        )
        try:
            connection.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
            cursor = connection.cursor()

            # Create only when missing; never drop-and-recreate on re-runs.
            cursor.execute("SELECT 1 FROM pg_roles WHERE rolname = 'deeb'")
            exists = cursor.fetchone() is not None

            if not exists:
                safe_password = deeb_password.replace("'", "''")
                cursor.execute(f"CREATE USER deeb WITH PASSWORD '{safe_password}'")
                logger.info("Created PostgreSQL deeb user on %s:%s", host, port)
            else:
                logger.info("PostgreSQL deeb user already exists on %s:%s; syncing password and ensuring grants", host, port)
                # Sync the password so re-runs align the DB user with the
                # credentials Deeb will store for this instance.
                safe_password = deeb_password.replace("'", "''")
                cursor.execute(f"ALTER USER deeb WITH PASSWORD '{safe_password}'")

            cursor.execute("ALTER USER deeb WITH CREATEDB")
            cursor.execute("ALTER ROLE deeb INHERIT")

            cursor.execute(
                """
                SELECT DISTINCT pg_get_userbyid(datdba) AS owner_role
                FROM pg_database
                WHERE datname NOT IN ('postgres', 'template0', 'template1')
                  AND datistemplate = false
                  AND pg_get_userbyid(datdba) != 'deeb'
                """
            )
            for (owner_role,) in cursor.fetchall():
                if owner_role.lower() in RDS_PROTECTED_ROLES:
                    logger.info("Skipping RDS-protected role: %s", owner_role)
                    continue
                try:
                    cursor.execute(sql.SQL("GRANT {} TO deeb").format(sql.Identifier(owner_role)))
                    logger.info("Granted role %s to deeb", owner_role)
                except Exception as e:
                    logger.warning("Could not grant role %s to deeb: %s", owner_role, e)

            logger.info("Ensured PostgreSQL deeb user grants on %s:%s", host, port)
        finally:
            connection.close()

    def create_deeb_user(self, instance: Dict[str, Any], master_creds: Dict[str, str], deeb_password: str):
        args = (instance["endpoint"], instance["port"], master_creds["username"], master_creds["password"], deeb_password)
        if instance["engine"] == "mysql":
            self.create_mysql_deeb_user(*args)
        elif instance["engine"] == "postgresql":
            self.create_postgres_deeb_user(*args)
        else:
            raise ValueError(f"Unsupported engine: {instance['engine']}")

    # --- Deeb registration + project assignment --------------------------

    def test_instance_connection_in_deeb(self, instance_id: str) -> bool:
        """Ask deeb to test the instance connection with its stored credentials.
        True = stored password works (no rotation needed); False = drift."""
        if self.dry_run:
            return True
        try:
            response = self.session.post(
                f"{self.deeb_api_url}/api/v1/database-instances/{instance_id}/test-connection",
            )
            if response.status_code != 200:
                return False
            body = response.json()
            # Handler returns success info or an error payload; treat explicit
            # failure markers as drift.
            return not ("error" in body or body.get("success") is False)
        except Exception as e:
            logger.warning("Connection test via deeb failed for instance %s: %s", instance_id, e)
            return False

    def update_instance_password_in_deeb(self, instance_id: str, deeb_password: str) -> None:
        """Update the stored connection password for an already-registered instance
        so deeb's encrypted credentials match the rotated DB user password."""
        if self.dry_run:
            logger.info("[DRY RUN] Would update stored password for instance %s", instance_id)
            return
        response = self.session.put(
            f"{self.deeb_api_url}/api/v1/database-instances/{instance_id}",
            json={"password": deeb_password},
        )
        response.raise_for_status()
        logger.info("Updated stored password for instance %s", instance_id)

    def register_in_deeb(self, instance: Dict[str, Any], name: str, deeb_password: str, environment_id: str) -> Dict[str, Any]:
        if self.dry_run:
            logger.info("[DRY RUN] Would register %s in Deeb", name)
            return {"id": "dry-run-id"}

        engine_map = {"mysql": "mysql", "postgresql": "postgres"}
        payload = {
            "name": name,
            "description": "RDS/Aurora instance auto-onboarded from AWS",
            "database_engine": engine_map[instance["engine"]],
            "host": instance["endpoint"],
            "port": instance["port"],
            "username": DEEB_USER,
            "password": deeb_password,
            "use_rds_iam_auth": False,
            "require_ssl": instance["engine"] == "postgresql",
            "environment_id": environment_id,
            "sync_all_databases": True,
        }
        response = self.session.post(f"{self.deeb_api_url}/api/v1/database-instances", json=payload)
        response.raise_for_status()
        result = response.json()
        logger.info("Registered %s in Deeb (id: %s)", name, result["id"])

        # Warn when the instance has no user databases yet: deeb only registers
        # databases that exist at sync time (system DBs excluded), so an empty
        # instance shows up registered-but-empty until its owner creates one and
        # a sync runs. This explains "registered but invisible" reports.
        sync_result = result.get("sync_results") or {}
        if sync_result.get("total_found") == 0:
            logger.warning(
                "%s has no user databases yet (only system databases) — nothing to "
                "assign. Deeb does not create databases; once the owner creates one, "
                "run the instance sync to pick it up.",
                name,
            )
        return result

    def assign_databases_to_project(self, instance_id: str, project_id: str, environment_id: str) -> int:
        """Assign the configured project + environment to every synced database of the instance."""
        if self.dry_run:
            logger.info("[DRY RUN] Would assign project %s to databases of instance %s", project_id, instance_id)
            return 0

        response = self.session.get(
            f"{self.deeb_api_url}/api/v1/databases",
            params={"database_instance_id": instance_id, "limit": PAGE_SIZE},
        )
        response.raise_for_status()
        databases = _extract_list(response.json(), "databases", "data", "items")

        assigned = 0
        for db in databases:
            if db.get("project_id") == project_id and db.get("environment_id") == environment_id:
                continue
            put = self.session.put(
                f"{self.deeb_api_url}/api/v1/databases/{db['id']}",
                json={"project_id": project_id, "environment_id": environment_id},
            )
            put.raise_for_status()
            assigned += 1
        logger.info("Assigned project to %d database(s) of instance %s", assigned, instance_id)
        return assigned

    # --- Orchestration ----------------------------------------------------

    def onboard_instance(
        self,
        instance: Dict[str, Any],
        name: str,
        secretsmanager_client,
        environment_id: str,
        project_id: str,
    ) -> Optional[Dict[str, Any]]:
        logger.info("Onboarding %s (%s)", name, instance["engine"])
        try:
            master_creds = self.get_master_credentials(secretsmanager_client, instance["master_secret_arn"])
            deeb_password = self.generate_secure_password()
            self.create_deeb_user(instance, master_creds, deeb_password)
            result = self.register_in_deeb(instance, name, deeb_password, environment_id)
            if not self.dry_run:
                self.assign_databases_to_project(result["id"], project_id, environment_id)
            logger.info("SUCCESS: onboarded %s", name)
            return result
        except Exception as e:
            logger.error("FAILED: %s -> %s", name, e)
            if hasattr(e, "response") and e.response is not None:
                logger.error("API response: %s", e.response.text)
            return None

    def onboard_account(self, account: Dict[str, Any], registered: Dict[str, Dict[str, Any]]) -> tuple:
        """Onboard all new instances for a single account across its regions. Returns (ok, failed, skipped)."""
        alias = account["alias"]
        tag_filters = account["tag_filters"]
        env_overrides = account["env_overrides"]
        override_tag = account["env_override_tag"]
        logger.info("=== Account %s (%s) -> env=%s project=%s tag_filters=%s env_overrides=%s ===",
                    account["account_id"], alias, account["environment"], account["project"],
                    tag_filters or "(none)", env_overrides or "(none)")

        # Resolve env/project names to IDs lazily and cache them: an account may map
        # instances to several (env, project) pairs via env_overrides, and we don't want
        # to fail the whole account when only one target env/project is missing in Deeb.
        id_cache: Dict[tuple, tuple] = {}

        def resolve_target(env_name: str, proj_name: str) -> tuple:
            key = (env_name, proj_name)
            if key not in id_cache:
                id_cache[key] = (self.resolve_environment_id(env_name), self.resolve_project_id(proj_name))
            return id_cache[key]

        session = self.session_for(account)

        ok = failed = skipped = filtered = 0
        for region in account["regions"]:
            rds_client = session.client("rds", region_name=region)
            secretsmanager_client = session.client("secretsmanager", region_name=region)

            candidates, no_secrets = discover.discover_all(rds_client)
            for entry in no_secrets:
                logger.warning("[%s/%s] Skipping (no Secrets Manager): %s", alias, region, entry)

            for instance in candidates:
                # Dedup by host: when the endpoint is already registered, rotate the
                # deeb user password on the target (plugin-safe ALTER) and update the
                # stored credentials in Deeb so they never drift, then re-apply grants.
                host = instance["endpoint"].strip().lower()
                if host in registered:
                    if instance["engine"] == "mysql":
                        try:
                            deeb_instance_id = registered[host]["id"]
                            # Probe: does deeb's stored password still authenticate?
                            # Healthy runs are a no-op (cron-safe); only drift
                            # (plugin flip, user recreated, password changed
                            # elsewhere) triggers a reset + re-sync.
                            if self.test_instance_connection_in_deeb(deeb_instance_id):
                                logger.info("[%s/%s] %s credentials healthy; ensuring grants only",
                                            alias, region, instance["name"])
                                master_creds = self.get_master_credentials(secretsmanager_client, instance["master_secret_arn"])
                                self.ensure_mysql_grants_on_instance(
                                    instance["endpoint"], instance["port"],
                                    master_creds["username"], master_creds["password"],
                                )
                            else:
                                logger.info("[%s/%s] %s credentials drifted; resetting deeb user password",
                                            alias, region, instance["name"])
                                master_creds = self.get_master_credentials(secretsmanager_client, instance["master_secret_arn"])
                                new_password = self.generate_secure_password()
                                self.reset_mysql_deeb_user_password(
                                    instance["endpoint"], instance["port"],
                                    master_creds["username"], master_creds["password"],
                                    new_password,
                                )
                                self.update_instance_password_in_deeb(deeb_instance_id, new_password)
                        except Exception as e:
                            logger.warning("[%s/%s] Could not reconcile user/grants for %s: %s",
                                           alias, region, instance["name"], e)
                    else:
                        # PostgreSQL reconcile: grants/instruments are ensured at
                        # registration time and drift is rare; still log so the
                        # sweep account for every discovered instance (a silent
                        # skip looks like a discovery bug when reading logs).
                        logger.info("[%s/%s] %s already registered; ensuring reconcile (postgresql)",
                                    alias, region, instance["name"])
                    skipped += 1
                    continue
                tags = instance.get("tags", {})
                if not matches_tag_filters(tags, tag_filters):
                    logger.info("[%s/%s] Filtered out %s by tags (tags=%s)",
                                alias, region, instance["name"], tags)
                    filtered += 1
                    continue

                # Route to an env/project override keyed by the tag value, else account default.
                override = env_overrides.get(str(tags.get(override_tag, "")).strip().lower())
                env_name = override["environment"] if override else account["environment"]
                proj_name = override["project"] if override else account["project"]
                try:
                    environment_id, project_id = resolve_target(env_name, proj_name)
                except ValueError as e:
                    logger.error("[%s/%s] %s -> cannot resolve env=%r project=%r: %s",
                                 alias, region, instance["name"], env_name, proj_name, e)
                    failed += 1
                    continue

                name = composed_name(instance["name"], alias)
                result = self.onboard_instance(instance, name, secretsmanager_client, environment_id, project_id)
                if result is not None:
                    registered[host] = result  # avoid re-onboarding within this run
                    ok += 1
                else:
                    failed += 1

        logger.info("Account %s done: onboarded=%d failed=%d skipped_existing=%d filtered_by_tags=%d",
                    alias, ok, failed, skipped, filtered)
        return ok, failed, skipped

    def run_sweep(self) -> int:
        """Run one multi-account onboarding sweep. Returns process exit code (0 on success)."""
        logger.info("Starting sweep across %d account(s) (dry_run=%s)", len(self.accounts), self.dry_run)

        registered = self.registered_instances_by_host()
        logger.info("%d instance host(s) already registered in Deeb", len(registered))

        total_ok = total_failed = total_skipped = 0
        for account in self.accounts:
            try:
                ok, failed, skipped = self.onboard_account(account, registered)
                total_ok += ok
                total_failed += failed
                total_skipped += skipped
            except Exception as e:
                # A bad account (assume-role denied, env/project missing) must not abort the sweep.
                logger.error("Account %s failed: %s", account.get("alias", account.get("account_id")), e)
                if hasattr(e, "response") and e.response is not None:
                    logger.error("API response: %s", e.response.text)

        logger.info("Sweep complete: onboarded=%d failed=%d skipped_existing=%d",
                    total_ok, total_failed, total_skipped)
        return 0


def main() -> int:
    dry_run = "--dry-run" in sys.argv[1:]

    try:
        config = load_config()
    except ConfigError as e:
        logger.error("%s", e)
        return 2

    try:
        onboarder = DeebOnboarder(config, dry_run=dry_run)
        return onboarder.run_sweep()
    except KeyboardInterrupt:
        logger.info("Interrupted")
        return 1
    except Exception as e:
        logger.error("Sweep failed: %s", e)
        if hasattr(e, "response") and e.response is not None:
            logger.error("API response: %s", e.response.text)
        return 1


if __name__ == "__main__":
    sys.exit(main())
