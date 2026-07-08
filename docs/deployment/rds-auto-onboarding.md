# RDS/Aurora Auto-Onboarding (Multi-Account) — Getting Started

**Version**: 2.0
**Last Updated**: July 7, 2026
**Audience**: DevOps, DBAs, Platform Engineers

## Overview

This feature automatically onboards new AWS RDS instances and Aurora clusters into a
**single** Deeb instance, across **multiple AWS accounts**. One central scheduled sweep:

1. For each configured AWS account, assumes a per-account IAM role.
2. Discovers RDS instances and Aurora clusters in that account's regions.
3. Creates the `deeb` database user on each new instance (broad schema-management
   grants, including DROP).
4. Registers the instance in Deeb under the account's mapped **environment**.
5. Assigns the account's mapped **project** to every database synced from the instance.

The sweep is **idempotent**: instances already in Deeb are skipped, so re-runs never churn
the `deeb` user or its stored password.

### Why a single central sweep

Deeb's runtime is **account-agnostic** — it connects to any database by `host:port` +
stored credentials over the network. Given a network path (VPC peering / Transit Gateway /
PrivateLink) from Deeb to every account's databases, one Deeb can manage them all. The only
AWS-account-specific work — discovering databases and creating the `deeb` user — is done by
the onboarder, which reaches each account via cross-account `sts:AssumeRole`.

### Account → environment/project mapping

One AWS account maps to exactly one Deeb environment + project. Example:

| AWS account | `environment`  | `project`         |
|-------------|----------------|-------------------|
| Prod        | `Production`   | `Proj Production` |
| Staging     | `Staging`      | `Proj Staging`    |

Because Deeb does not enforce unique instance names, the onboarder disambiguates the
registered name across accounts as **`<db-identifier> (<alias>)`** (e.g. `orders (prod)`).

### Architecture

```
ONE EKS CronJob (near Deeb, base IRSA identity)   every 5 min
   │
   for each account in accounts.yaml:
   │   sts:AssumeRole → arn:aws:iam::<acct>:role/DeebOnboarderRole
   │   for each region:
   │       discover RDS + Aurora (Secrets Manager master creds only)
   │       skip instances already in Deeb (by "<name> (<alias>)")
   │       for each NEW instance:
   │           read master creds (Secrets Manager, in that account)
   │           CREATE the `deeb` DB user   ── needs network to the DB
   │           POST /api/v1/database-instances (env from account, sync_all_databases)
   │           PUT  /api/v1/databases/:id      (assign account's project)
   ▼
 one Deeb instance (one API token)
```

---

## Prerequisites

- A running Deeb instance (backend at migration **000034** or later — see step 1).
- An EKS cluster (near Deeb) with:
  - [external-secrets.io](https://external-secrets.io/) and a `ClusterSecretStore` named
    `aws-secrets-manager` (the chart's existing convention).
  - IRSA enabled.
- **Network reachability** from where Deeb (and this cluster) run to the RDS databases in
  **every** target account (peering/TGW/PrivateLink). This is a hard prerequisite: without
  it, registration succeeds but change-apply and user-creation fail.
- Target RDS/Aurora with **master credentials managed by AWS Secrets Manager** (others are
  skipped and logged).

---

## Step 1 — Apply the backend migration

The API-token table ships in migration `000034`:

```bash
cd backend
deeb migrate up
deeb migrate version     # confirm version >= 34
```

## Step 2 — Create the environments and projects in Deeb

Nothing is seeded. Create every environment/project you will map to, e.g. `Production` /
`Staging` environments and `Proj Production` / `Proj Staging` projects. Names must match the
`environment` / `project` values in the accounts config (matching is case-insensitive).

## Step 3 — Issue one API token and store it in Secrets Manager

One central token authenticates the single onboarder:

```bash
cd backend
deeb token issue --name rds-onboarder
# → prints a deeb_pat_… token ONCE. Copy it now; it is not recoverable.
```

Store it in Secrets Manager (in the account where the onboarder runs), as JSON:

```bash
aws secretsmanager create-secret \
  --name deeb/onboarder \
  --secret-string '{"deeb_api_token":"deeb_pat_XXXXXXXX"}'
```

Manage tokens with `deeb token list` / `deeb token revoke <id>`.

## Step 4 — Provision the IAM roles (Terraform)

The IAM footprint is managed in **platform-iac** (`terraform/aws/modules/deeb-onboarder`
+ `accounts/{production,staging}/deeb-onboarder`, Atlantis projects
`prod-deeb-onboarder` / `stg-deeb-onboarder`). It creates two layers:

- **Per-account role `DeebOnboarderRole`** in each target account — grants `rds:Describe*`,
  `rds:ListTagsForResource`, and read of the RDS-managed master secret (scoped to `rds!*`);
  trusts the base identity. ARNs:
  - Production: `arn:aws:iam::492684252576:role/DeebOnboarderRole`
  - Staging: `arn:aws:iam::555901236465:role/DeebOnboarderRole`
- **Base IRSA role `DeebOnboarderBaseRole`** in the Production (master) account —
  `arn:aws:iam::492684252576:role/DeebOnboarderBaseRole`. Assumed by the onboarder
  ServiceAccount `system:serviceaccount:deeb-tooling:deeb-onboarder` via the
  `EKS-NSX-Tooling` cluster OIDC, and permitted to `sts:AssumeRole` the per-account roles.

Apply via Atlantis (open a PR) or `make apply TF_DIR=terraform/aws/accounts/<acct>/deeb-onboarder`.

> Prereq: the `EKS-NSX-Tooling` cluster's OIDC provider must be registered as an IAM OIDC
> provider in the Production account (standard IRSA). To add another target account later,
> create an `accounts/<acct>/deeb-onboarder` dir from the module and add its role ARN to
> production's `target_role_arns`.

## Step 5 — Build and push the image

```bash
cd scripts/aws-onboarding
docker build -t <ecr>/deeb-onboarder:<tag> .
docker push <ecr>/deeb-onboarder:<tag>
```

## Step 6 — Deploy the single CronJob (Helm)

The onboarder is part of the `helm/deeb` chart, disabled by default. Provide the accounts
list and the base role:

The onboarder runs in the `EKS-NSX-Tooling` cluster, namespace `deeb-tooling`. The
ServiceAccount name **must** be `deeb-onboarder` to match the IRSA trust from Step 4.

```yaml
onboarder:
  enabled: true
  image:
    repository: <ecr>/deeb-onboarder
    tag: "<tag>"
  schedule: "*/5 * * * *"
  serviceAccount:
    name: deeb-onboarder                # must match the IRSA trust (system:serviceaccount:deeb-tooling:deeb-onboarder)
    annotations:
      eks.amazonaws.com/role-arn: arn:aws:iam::492684252576:role/DeebOnboarderBaseRole
  externalSecret:
    enabled: true
    remoteKey: deeb/onboarder          # Secrets Manager path holding deeb_api_token
    remoteProperty: deeb_api_token
  accounts:
    - account_id: "492684252576"
      alias: prod
      role_arn: arn:aws:iam::492684252576:role/DeebOnboarderRole
      regions: ["us-east-1"]
      environment: Production
      project: Proj Production
    - account_id: "555901236465"
      alias: staging
      role_arn: arn:aws:iam::555901236465:role/DeebOnboarderRole
      regions: ["us-east-1"]
      environment: Staging
      project: Proj Staging
```

Deploy into the `deeb-tooling` namespace (e.g. `helm ... -n deeb-tooling`). The accounts list
is rendered into a ConfigMap mounted at `/etc/deeb-onboarder/accounts.yaml` (the pod restarts
automatically when it changes). `DEEB_API_URL` defaults to the chart's API URL; override with
`onboarder.deebApiUrl`.

### Tag filtering (which instances get onboarded)

By default the sweep only onboards instances tagged `Environment` in
`prod, production, staging, homol, tooling`; anything else (including untagged) is skipped
and logged. Override globally via `onboarder.tagFilters`, or per-account via a `tag_filters`
key on an account entry (`{}` = onboard all in that account). The filter is generic —
`{tag_key: [allowed values]}`, matched case-insensitively, all keys required:

```yaml
onboarder:
  tagFilters:
    Environment: ["prod", "production", "staging", "homol", "tooling"]
    # add other tag keys as needed, e.g. Team: ["payments"]
```

### Do a dry run first

```yaml
onboarder:
  args: ["--dry-run"]
```

Trigger an immediate run instead of waiting for the schedule:

```bash
kubectl create job --from=cronjob/deeb-onboarder deeb-onboarder-manual
kubectl logs -l app.kubernetes.io/component=onboarder --tail=200 -f
```

You should see per-account `=== Account … ===` banners, discovery counts, and
`[DRY RUN] Would register …` lines. Remove `args` to go live.

---

## Verifying it works

1. Create a throwaway RDS/Aurora in one of the configured accounts (master creds in Secrets
   Manager).
2. Within one schedule interval, check logs for `SUCCESS: onboarded <name> (<alias>)`.
3. In Deeb: the instance appears under its mapped environment; its databases show the mapped
   project + environment; the registered name carries the account alias.
4. Connect to the DB and confirm a `deeb` user exists.
5. Re-run → the instance is **skipped** (no duplicate, no password change).
6. Open a change plan against a cross-account instance — this validates the network path end
   to end, not just registration.

---

## Troubleshooting

| Symptom | Cause / fix |
|---------|-------------|
| `AccessDenied` on `sts:AssumeRole` | Base role missing `sts:AssumeRole` on the per-account role, or the per-account role's trust policy doesn't allow the base identity. |
| `Cannot resolve hostname …` / connection timeout | No network path or SG ingress from the onboarder to that account's DB. Test `nc -zv <endpoint> 3306|5432` from a pod. |
| `Deeb environment/project not found` | Create it in Deeb (step 2) or fix the account's `environment`/`project` in the config. |
| `401 Unauthorized` | Token missing/revoked/wrong. Re-issue with `deeb token issue`, update the Secrets Manager value. |
| Instance never onboarded | It has no Secrets Manager master creds (skipped, logged). Enable "Manage master credentials in Secrets Manager" on the RDS resource. |
| One account errors, others fine | Per-account failures are logged and isolated; the sweep continues. Fix that account's role/mapping and it onboards next run. |
| PostgreSQL `pg_hba.conf rejects connection … no encryption` | Expected — PostgreSQL is registered with `require_ssl: true`. |

Config reference and script internals: `scripts/aws-onboarding/README.md`.

---

## How authentication works (reference)

The onboarder authenticates to Deeb with an **API token** (`deeb_pat_…`): SHA-256-hashed at
rest, bound to a user (the system OrganizationAdmin) so it flows through the existing RBAC,
long-lived but revocable, and accepted only on the `Authorization` header (never the login
cookie). AWS access is via the base IRSA identity assuming each account's `DeebOnboarderRole`
role; assumed-role credentials are short-lived and rebuilt each sweep.
