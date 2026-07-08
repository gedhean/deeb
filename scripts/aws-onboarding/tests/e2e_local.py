#!/usr/bin/env python3
"""
Local end-to-end test for the multi-account onboarder.

Exercises the REAL onboarder code paths against:
  - moto (mock AWS): STS assume-role, Secrets Manager, RDS describe.
  - local MySQL + Postgres containers standing in for RDS databases.
  - a real, locally-running Deeb (its API + DB).

What is real vs adjusted:
  - Real: assume-role, RDS discovery, reading master creds from Secrets Manager,
    creating the `deeb` DB user (actual SQL on the containers), registering the
    instance in Deeb, syncing databases, and assigning the project.
  - Adjusted for local reality: each discovered instance's endpoint is pointed at
    127.0.0.1 (moto returns fake *.rds.amazonaws.com hosts), and its master-secret
    ARN is our controlled secret. The Postgres instance is registered with
    require_ssl=false because the throwaway container has no TLS (real RDS PG has it).

Prerequisites:
  - `docker compose -f docker-compose.test.yaml up -d --wait` (moto + target DBs).
  - A local Deeb running, reachable at DEEB_API_URL with DEEB_API_TOKEN.

Env:
  DEEB_API_URL, DEEB_API_TOKEN   (required)
  AWS_ENDPOINT_URL               (default http://localhost:5001)
"""

import importlib.util
import os
import sys

import psycopg2
import pymysql
import requests

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPTS = os.path.dirname(HERE)
REGION = os.environ.get("AWS_DEFAULT_REGION", "us-east-1")
ACCOUNT_ID = "111111111111"
ALIAS = "local"
ENV_NAME = "Local"
PROJECT_NAME = "Proj Local"

# The host as seen from where each actor runs:
#  - the e2e script runs on the HOST, so it creates the deeb user via 127.0.0.1.
#  - Deeb runs in a container, so it must reach the host via host.docker.internal.
# If you run Deeb ON the host instead, set DEEB_TARGET_HOST=127.0.0.1.
LOCAL_HOST = "127.0.0.1"
DEEB_TARGET_HOST = os.environ.get("DEEB_TARGET_HOST", "host.docker.internal")
MYSQL_HOST_PORT = 3307
PG_HOST_PORT = 5433

# Point boto3 at moto and give it dummy creds if the caller didn't.
os.environ.setdefault("AWS_ENDPOINT_URL", "http://localhost:5001")
os.environ.setdefault("AWS_ACCESS_KEY_ID", "testing")
os.environ.setdefault("AWS_SECRET_ACCESS_KEY", "testing")
os.environ.setdefault("AWS_DEFAULT_REGION", REGION)

TARGETS = [
    {
        "engine": "mysql", "identifier": "orders-mysql", "moto_engine": "mysql",
        "host_port": MYSQL_HOST_PORT, "master_user": "root", "master_password": "master",
        "sample_db": "app_orders", "secret_name": "deeb/test/mysql-master",
        "env_tag": "staging",   # in the default Environment tag filter -> onboarded
    },
    {
        "engine": "postgresql", "identifier": "orders-postgres", "moto_engine": "postgres",
        "host_port": PG_HOST_PORT, "master_user": "postgres", "master_password": "master",
        "sample_db": "app_orders", "secret_name": "deeb/test/pg-master",
        "env_tag": "tooling",   # in the default Environment tag filter -> onboarded
    },
]

# An extra instance whose Environment tag is NOT in the default filter — discovered but
# must be filtered out. No local container: it is never connected to.
EXCLUDED_IDENTIFIER = "orders-excluded"
EXCLUDED_ENV_TAG = "sandbox"


def load_onboarder():
    sys.path.insert(0, SCRIPTS)
    spec = importlib.util.spec_from_file_location("onboarder", os.path.join(SCRIPTS, "aws-onboard-to-deeb.py"))
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def deeb_session(token):
    s = requests.Session()
    s.headers.update({"Content-Type": "application/json", "Authorization": f"Bearer {token}"})
    return s


def _extract_list_deeb(deeb, base_url, path, params, *keys):
    """GET a Deeb collection and return its list (bare list or under one of `keys`)."""
    data = deeb.get(f"{base_url}{path}", params=params).json()
    if isinstance(data, list):
        return data
    for k in keys:
        if isinstance(data.get(k), list):
            return data[k]
    return []


def ensure_named(session, base_url, path, name, list_keys):
    """Return the id of a named resource, creating it (POST {name}) if absent."""
    r = session.get(f"{base_url}{path}")
    r.raise_for_status()
    data = r.json()
    items = data if isinstance(data, list) else next((data[k] for k in list_keys if isinstance(data.get(k), list)), [])
    for item in items:
        if item.get("name", "").lower() == name.lower():
            return item["id"]
    r = session.post(f"{base_url}{path}", json={"name": name})
    r.raise_for_status()
    return r.json()["id"]


def seed_moto(secretsmanager, rds):
    """Create the master-credential secrets and (best-effort) RDS instances in moto.

    Idempotent: moto keeps state across runs, so an existing secret/instance is reused."""
    secret_arns = {}
    for t in TARGETS:
        secret_string = f'{{"username":"{t["master_user"]}","password":"{t["master_password"]}"}}'
        try:
            resp = secretsmanager.create_secret(Name=t["secret_name"], SecretString=secret_string)
            secret_arns[t["identifier"]] = resp["ARN"]
        except secretsmanager.exceptions.ResourceExistsException:
            secretsmanager.put_secret_value(SecretId=t["secret_name"], SecretString=secret_string)
            secret_arns[t["identifier"]] = secretsmanager.describe_secret(SecretId=t["secret_name"])["ARN"]
        # Best-effort RDS instance so real discovery has something to find. Endpoint/secret
        # are overridden later regardless of what moto returns; already-exists is fine.
        try:
            rds.create_db_instance(
                DBInstanceIdentifier=t["identifier"],
                DBInstanceClass="db.t3.micro",
                Engine=t["moto_engine"],
                MasterUsername=t["master_user"],
                AllocatedStorage=20,
                ManageMasterUserPassword=True,
                Tags=[{"Key": "Environment", "Value": t["env_tag"]}],
            )
        except Exception as e:  # noqa: BLE001 - already-exists / version differences are non-fatal
            print(f"  (note) moto create_db_instance({t['identifier']}): {e}")

    # An instance tagged outside the default filter; must be discovered but filtered out.
    try:
        rds.create_db_instance(
            DBInstanceIdentifier=EXCLUDED_IDENTIFIER,
            DBInstanceClass="db.t3.micro",
            Engine="postgres",
            MasterUsername="postgres",
            AllocatedStorage=20,
            ManageMasterUserPassword=True,
            Tags=[{"Key": "Environment", "Value": EXCLUDED_ENV_TAG}],
        )
    except Exception as e:  # noqa: BLE001
        print(f"  (note) moto create_db_instance({EXCLUDED_IDENTIFIER}): {e}")
    return secret_arns


def cleanup_prior_run(deeb, base_url, names):
    """Delete test instances (and their databases) from a previous run so this run is fresh."""
    instances = _extract_list_deeb(deeb, base_url, "/api/v1/database-instances",
                                   {"limit": 100}, "database_instances")
    for inst in instances:
        if inst["name"] not in names:
            continue
        dbs = _extract_list_deeb(deeb, base_url, "/api/v1/databases",
                                 {"database_instance_id": inst["id"], "limit": 100}, "databases")
        for db in dbs:
            deeb.delete(f"{base_url}/api/v1/databases/{db['id']}").raise_for_status()
        deeb.delete(f"{base_url}/api/v1/database-instances/{inst['id']}").raise_for_status()
        print(f"   cleaned up prior {inst['name']}")


def seed_container_sample_dbs():
    """Create a user database in each container so sync has something to assign a project to."""
    my = pymysql.connect(host=LOCAL_HOST, port=MYSQL_HOST_PORT, user="root", password="master", connect_timeout=15)
    try:
        my.cursor().execute("CREATE DATABASE IF NOT EXISTS app_orders")
        my.commit()
    finally:
        my.close()

    pg = psycopg2.connect(host=LOCAL_HOST, port=PG_HOST_PORT, user="postgres", password="master", dbname="postgres", connect_timeout=15)
    try:
        pg.autocommit = True
        cur = pg.cursor()
        cur.execute("SELECT 1 FROM pg_database WHERE datname = 'app_orders'")
        if not cur.fetchone():
            cur.execute("CREATE DATABASE app_orders")
    finally:
        pg.close()


def register_instance(onboarder, base_url, instance, name, password, env_id, require_ssl):
    """Register in Deeb. Mirrors register_in_deeb but lets the caller force require_ssl
    (the local Postgres container has no TLS)."""
    engine_map = {"mysql": "mysql", "postgresql": "postgres"}
    payload = {
        "name": name,
        "description": "e2e local test instance",
        "database_engine": engine_map[instance["engine"]],
        "host": instance["endpoint"],
        "port": instance["port"],
        "username": "deeb",
        "password": password,
        "use_rds_iam_auth": False,
        "require_ssl": require_ssl,
        "environment_id": env_id,
        "sync_all_databases": True,
    }
    r = onboarder.session.post(f"{base_url}/api/v1/database-instances", json=payload)
    r.raise_for_status()
    return r.json()


def verify_deeb_user_exists(target):
    if target["engine"] == "mysql":
        conn = pymysql.connect(host=LOCAL_HOST, port=MYSQL_HOST_PORT, user="root", password="master", connect_timeout=15)
        try:
            cur = conn.cursor()
            cur.execute("SELECT COUNT(*) FROM mysql.user WHERE user = 'deeb'")
            assert cur.fetchone()[0] == 1, "deeb MySQL user not found"
        finally:
            conn.close()
    else:
        conn = psycopg2.connect(host=LOCAL_HOST, port=PG_HOST_PORT, user="postgres", password="master", dbname="postgres", connect_timeout=15)
        try:
            cur = conn.cursor()
            cur.execute("SELECT 1 FROM pg_roles WHERE rolname = 'deeb'")
            assert cur.fetchone() is not None, "deeb Postgres role not found"
        finally:
            conn.close()


def main():
    base_url = os.environ.get("DEEB_API_URL", "").rstrip("/")
    token = os.environ.get("DEEB_API_TOKEN", "")
    if not base_url or not token:
        print("ERROR: DEEB_API_URL and DEEB_API_TOKEN must be set (start Deeb locally first).")
        return 2

    m = load_onboarder()
    deeb = deeb_session(token)

    print("1. Ensuring Deeb environment + project exist and cleaning prior test artifacts...")
    env_id = ensure_named(deeb, base_url, "/api/v1/environments", ENV_NAME, ("environments", "data", "items"))
    proj_id = ensure_named(deeb, base_url, "/api/v1/projects", PROJECT_NAME, ("projects", "data", "items"))
    cleanup_prior_run(deeb, base_url, [m.composed_name(t["identifier"], ALIAS) for t in TARGETS])
    print(f"   env={env_id} project={proj_id}")

    account = {
        "account_id": ACCOUNT_ID, "alias": ALIAS,
        "role_arn": f"arn:aws:iam::{ACCOUNT_ID}:role/deeb-onboarder",  # moto STS accepts it
        "regions": [REGION], "environment": ENV_NAME, "project": PROJECT_NAME,
    }
    config = {"deeb_api_url": base_url, "deeb_api_token": token, "accounts": [account]}
    onboarder = m.DeebOnboarder(config, dry_run=False)

    # moto isolates resources per account. The onboarder reads AWS via an assumed-role
    # session (account 111111111111), so seed with THAT session's clients — not ambient.
    print("2. Assume-role (moto) + seed secrets/RDS in the assumed account...")
    session = onboarder.session_for(account)  # real assume-role against moto
    rds = session.client("rds", region_name=REGION)
    secretsmanager = session.client("secretsmanager", region_name=REGION)
    secret_arns = seed_moto(secretsmanager, rds)
    seed_container_sample_dbs()

    print("3. Discover via moto...")
    candidates, skipped = m.discover.discover_all(rds)
    by_id = {c["name"]: c for c in candidates}
    print(f"   discovered={sorted(by_id)} skipped={skipped}")

    # Tag filtering (against real moto-fetched tags): targets pass the default filter,
    # the sandbox-tagged instance is discovered but rejected.
    if EXCLUDED_IDENTIFIER in by_id:
        for t in TARGETS:
            cand = by_id.get(t["identifier"])
            if cand:
                assert m.matches_tag_filters(cand["tags"], m.DEFAULT_TAG_FILTERS), \
                    f"{t['identifier']} (tags={cand['tags']}) should pass the default filter"
        assert not m.matches_tag_filters(by_id[EXCLUDED_IDENTIFIER]["tags"], m.DEFAULT_TAG_FILTERS), \
            "sandbox-tagged instance should be filtered out"
        print(f"   tag filter OK: {EXCLUDED_IDENTIFIER} (Environment={EXCLUDED_ENV_TAG}) correctly excluded")
    else:
        print(f"   (note) {EXCLUDED_IDENTIFIER} not discovered (moto tag support) — skipping filter assertion")

    print("4. Onboarding each target (real create-user + register + assign)...")
    onboarded = []
    for t in TARGETS:
        cand = by_id.get(t["identifier"]) or {"name": t["identifier"], "engine": t["engine"]}
        # Adjust to local reality: our controlled secret; endpoint is set per actor below.
        cand["engine"] = t["engine"]
        cand["port"] = t["host_port"]
        cand["master_secret_arn"] = secret_arns[t["identifier"]]

        name = m.composed_name(t["identifier"], ALIAS)
        creds = onboarder.get_master_credentials(secretsmanager, cand["master_secret_arn"])  # real, from moto
        password = onboarder.generate_secure_password()

        # Create the deeb user from the HOST (127.0.0.1 -> published container port).
        cand["endpoint"] = LOCAL_HOST
        onboarder.create_deeb_user(cand, creds, password)  # real SQL on the container

        # Register with the host as Deeb (in its container) can reach it.
        cand["endpoint"] = DEEB_TARGET_HOST
        if t["engine"] == "mysql":
            result = onboarder.register_in_deeb(cand, name, password, env_id)  # real production method
        else:
            result = register_instance(onboarder, base_url, cand, name, password, env_id, require_ssl=False)
        onboarder.assign_databases_to_project(result["id"], proj_id, env_id)  # real
        onboarded.append((t, name))

    print("5. Verifying...")
    instances = m._extract_list(
        deeb.get(f"{base_url}/api/v1/database-instances", params={"limit": 100}).json(),
        "database_instances", "data", "items",
    )
    by_name = {i["name"]: i for i in instances}
    for t, name in onboarded:
        assert name in by_name, f"instance {name} not registered in Deeb"
        inst_id = by_name[name]["id"]
        dbs = m._extract_list(
            deeb.get(f"{base_url}/api/v1/databases", params={"database_instance_id": inst_id, "limit": 100}).json(),
            "databases", "data", "items",
        )
        sample = [d for d in dbs if d["database_name"] == t["sample_db"]]
        assert sample, f"{t['sample_db']} not synced for {name}"
        assert sample[0]["project_id"] == proj_id, f"{t['sample_db']} not assigned to project"
        assert sample[0]["environment_id"] == env_id, f"{t['sample_db']} not assigned to environment"
        verify_deeb_user_exists(t)
        print(f"   OK: {name} -> db '{t['sample_db']}' assigned to {PROJECT_NAME}/{ENV_NAME}; deeb user present")

    # Dedup key is host: the onboarded endpoint is now registered, so a re-run skips it.
    reg_after = onboarder.registered_instance_hosts()
    assert DEEB_TARGET_HOST.lower() in reg_after, f"{DEEB_TARGET_HOST} not in registered hosts"

    print("E2E LOCAL TEST PASSED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
