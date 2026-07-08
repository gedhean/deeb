# Local tests for the multi-account onboarder

Two levels, from fast to full.

## 1. Logic tests (no infra, seconds)

Stubs AWS/DB/HTTP and checks config parsing/validation, alias defaulting, assume-role vs
ambient session, cross-account name disambiguation + skip, and project assignment.

```bash
cd scripts/aws-onboarding
pip install -r requirements-aws.txt   # (PyYAML is the only import the logic test needs)
make test-unit
# -> ALL MULTI-ACCOUNT LOGIC TESTS PASSED
```

## 2. End-to-end (moto + local DBs + local Deeb)

Exercises the real code paths — assume-role, RDS discovery, reading master creds from
Secrets Manager, creating the `deeb` DB user (real SQL), registering in Deeb, syncing
databases, and assigning the project — with AWS mocked by moto and RDS stood in for by
local MySQL/Postgres containers.

### Prerequisites

- Docker (for the test stack) and `pip install -r requirements-aws.txt`.
- A **local Deeb running** (see the repo Getting Started): apply migrations, start the
  backend, and issue a token:
  ```bash
  cd backend
  export DATABASE_URL=... JWT_SECRET=... ENCRYPTION_KEY=...
  go run . migrate up
  go run . token issue --name e2e-local     # copy the deeb_pat_… value
  make run-air &
  ```

### Run

```bash
cd scripts/aws-onboarding
make test-up                                  # moto + target-mysql + target-postgres (waits for healthy)

export DEEB_API_URL=http://localhost:8080
export DEEB_API_TOKEN=deeb_pat_XXXX
# If Deeb runs ON the host (not a container), also: export DEEB_TARGET_HOST=127.0.0.1
make test-e2e
# -> E2E LOCAL TEST PASSED

make test-down                                # tear down when finished
```

The e2e creates a `Local` environment and `Proj Local` project in Deeb if missing, seeds
moto with master-credential secrets + RDS instances, creates an `app_orders` database in
each container, onboards both, and verifies each is registered under `Local`, its
`app_orders` database is assigned to `Proj Local`, and the `deeb` DB user exists.

### Ports (chosen to avoid common local collisions)

| Service | Host port | Note |
|---------|-----------|------|
| moto | 5001 | 5000 is often taken by macOS AirPlay |
| target-mysql | 3307 | 3306 is often a native/Homebrew MySQL |
| target-postgres | 5433 | 5432 is Deeb's own Postgres |

### Notes / caveats

- moto returns fake `*.rds.amazonaws.com` endpoints, and it isolates resources per account.
  The test seeds moto through the **assumed-role** session (so the secret lives in the same
  account the onboarder reads from), then points each discovered instance at the local
  container and overrides its master-secret ARN to the controlled secret.
- **Two host perspectives**: the test script runs on the host and creates the `deeb` user via
  `127.0.0.1:<port>`; Deeb runs in a container and reaches those DBs via
  `host.docker.internal:<port>` — so instances are registered with that host. Set
  `DEEB_TARGET_HOST=127.0.0.1` when Deeb runs directly on the host.
- The Postgres container has no TLS, so its instance is registered with `require_ssl=false`
  (real RDS PostgreSQL enforces SSL — the onboarder registers PG with `require_ssl=true`).
- Re-runnable: already-registered instances are skipped. For a clean slate, `make test-down`
  and delete the test instances/databases from Deeb.
