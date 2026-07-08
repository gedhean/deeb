# AWS RDS/Aurora → Deeb Multi-Account Auto-Onboarding

Automatically onboards new AWS RDS instances and Aurora clusters into a **single** Deeb
instance, across **many AWS accounts**. A single central sweep discovers databases in each
configured account (via cross-account `sts:AssumeRole`), creates the `deeb` database user,
registers each instance in Deeb, and assigns the account's mapped environment + project.

For the full deployment walkthrough (IAM, Helm, verification), see
[docs/deployment/rds-auto-onboarding.md](../../docs/deployment/rds-auto-onboarding.md).

## Files

- `aws-onboard-to-deeb.py` — the multi-account sweep (config-driven, non-interactive).
- `discover.py` — RDS/Aurora discovery helpers (given an `rds` client).
- `aws-discover-rds.py` — optional standalone inventory script (ad-hoc previewing; not used
  by the sweep).
- `Dockerfile` — image for the sweep (runs as an EKS CronJob).

## How it works

Each run:

1. Loads the accounts config and collects the endpoint hosts already registered in Deeb.
2. For each account: assumes its IAM role (or uses ambient creds if `role_arn` is empty).
3. For each region: discovers RDS/Aurora with Secrets Manager master creds.
4. Skips instances already in Deeb (matched by **endpoint host**, so a DB already registered
   under any name isn't duplicated) — existing instances and their `deeb` password are never
   touched.
5. For each new instance: resolves its target env/project (account default, or an
   `env_overrides` entry matching its tag), creates the `deeb` user, registers it
   (`sync_all_databases`), then assigns that project to every synced database.

Per-instance and per-account failures are logged and isolated; only fatal config/auth errors
exit non-zero.

## Configuration

### Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `DEEB_API_URL` | yes | Base URL of the Deeb API |
| `DEEB_API_TOKEN` | yes | API token (`deeb_pat_…`) for a service account |
| `DEEB_ONBOARDER_CONFIG` | no | Path to the accounts YAML (default `/etc/deeb-onboarder/accounts.yaml`) |

### Accounts YAML (`DEEB_ONBOARDER_CONFIG`)

```yaml
# Optional. Only onboard instances whose tags match. Defaults to the Environment set
# below if omitted; a per-account tag_filters overrides it.
tag_filters:
  Environment: ["prod", "production", "staging", "homol", "tooling"]

accounts:
  - account_id: "111111111111"
    alias: "prod"                 # short label; disambiguates instance names across accounts
    role_arn: "arn:aws:iam::111111111111:role/DeebOnboarderRole"  # empty => use ambient creds
    regions: ["us-east-1"]
    environment: "Production"     # default env for this account; must exist in Deeb
    project: "Proj Production"    # default project; must exist in Deeb
    env_overrides:                # optional: route by tag value (see below)
      tooling:
        environment: "Tooling"    # must exist in Deeb
        project: "Proj Tooling"   # must exist in Deeb
  - account_id: "222222222222"
    alias: "staging"
    role_arn: "arn:aws:iam::222222222222:role/DeebOnboarderRole"
    regions: ["eu-west-1", "us-east-1"]
    environment: "Staging"
    project: "Proj Staging"
    # tag_filters: {}             # optional per-account override ({} = onboard all)
```

Each account has one **default** (environment, project). `alias` defaults to `account_id`
if omitted.

### Env/project overrides by tag

A single AWS account often hosts more than one logical environment — e.g. the production
account also holds tooling databases. `env_overrides` routes those to a different Deeb
env/project without a separate account entry:

```yaml
env_overrides:            # {tag_value: {environment, project}}
  tooling:
    environment: "Tooling"
    project: "Proj Tooling"
```

- Keyed by the value of the **`Environment`** tag (set `env_override_tag` on the account to
  drive it off a different tag). Values are matched case-insensitively.
- An instance whose tag value matches a key is registered under that env/project; anything
  not listed falls back to the account's default `environment`/`project`.
- Every referenced environment and project must already exist in Deeb. A missing target is
  logged and skips only the affected instance(s) — the rest of the account still onboards.
- Overrides decide **mapping**, `tag_filters` decides **inclusion**; they are independent.

### Tag filtering

`tag_filters` restricts which discovered instances are onboarded — `{tag_key: [allowed
values]}`. An instance must, **for every key**, carry that tag with a value in the list
(matched case-insensitively); instances that don't match (including untagged ones) are
skipped and logged. The mechanism is generic — filter by any tag(s), e.g.
`Team: ["payments"]`.

- **Default** (no `tag_filters` in config): `Environment` in
  `prod, production, staging, homol, tooling`.
- **Per-account override**: set `tag_filters` on an account entry; `{}` onboards that
  account's instances regardless of tags.
- Requires the role's `rds:ListTagsForResource` permission (already granted by the
  Terraform module).

### Flags

- `--dry-run` — discover and report intended actions without making changes.

## Local usage

```bash
pip3 install -r requirements-aws.txt

export DEEB_API_URL=https://deeb.example.com
export DEEB_API_TOKEN=deeb_pat_xxxx
export DEEB_ONBOARDER_CONFIG=./accounts.yaml   # your local copy

python3 aws-onboard-to-deeb.py --dry-run       # preview
python3 aws-onboard-to-deeb.py                 # apply
```

Your local AWS credentials (default provider chain) must be able to `sts:AssumeRole` into
each account's `deeb-onboarder` role, and you need a network path to the databases to create
the `deeb` user.

## IAM (summary)

- **Base identity**: `sts:AssumeRole` on each account's `deeb-onboarder` role.
- **Per-account `deeb-onboarder` role**: trusts the base identity; grants
  `rds:DescribeDBInstances`, `rds:DescribeDBClusters`, `rds:ListTagsForResource`,
  `secretsmanager:GetSecretValue`.

Full policies and trust relationships are in the deployment guide.

## Security notes

- `deeb` user passwords are randomly generated (32 chars) and stored encrypted in Deeb.
- Master credentials are read from Secrets Manager and never persisted by the sweep.
- Assumed-role credentials are short-lived and rebuilt each run (session name `deeb-onboarder`).
- The API token is long-lived but revocable; scope its bound user's role appropriately.
