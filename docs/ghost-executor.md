# Ghost Executor (gh-ost / pg-online-schema-change)

Deeb supports an alternative execution backend for **DDL change plans**:
the **ghost executor**. Instead of running SQL directly in a transaction,
the change runs through an online-schema-change tool:

- **MySQL** → [gh-ost](https://github.com/github/gh-ost) (shadow table copy + binlog tailing + atomic rename)
- **PostgreSQL** → [pg-online-schema-change](https://github.com/pg-dba/pg-online-schema-change)

The change-plan workflow (create → review → approve → execute) stays
identical; only the execution engine differs. Executor choice is explicit
user preference at plan creation (`execution_backend: direct | ghost`).

## Execution deployment decision: subprocess vs Go library

The ghost executor intentionally shells out to the gh-ost / pg-online-schema-change binaries via `exec.CommandContext` (see `internal/services/ghost_executor.go`).

gh-ost **is** an importable Go module (`github.com/github/gh-ost/go/logic` exposing `Migrator.Migrate()`), so a future migration to an in-process executor is possible. The subprocess choice was deliberate for this first iteration because:

- **Validated runbook path** — the ghost-lab rehearsal (process doc) used the CLI with flag files; the executor matches that exact behavior.
- **Independent lifecycles** — a gh-ost copy legitimately takes hours; an in-process migrator would hold deeb resources for that whole window and couples deeb's lifecycle (deployments, crashes) to in-flight migrations. A subprocess can be killed/restarted without deeb restarting.
- **Binary parity with docs** — gh-ost CLI flags (`--exact-rowcount --concurrent-rowcount --chunk-size --max-load --critical-load --cut-over --allow-on-master --execute`) are precisely what `GhostExecutor.BuildArgs` emits, so the DDT/runbook stays the source of truth for flags.
- **Two tools, one abstraction** — pg-online-schema-change does not have a Go library; subprocess treats both engines uniformly.

A future in-process integration is a documented option; it would replace `GhostExecutor.Run` while keeping every other layer (validation, status transitions, queue semantics, UI) unchanged.

## Execution tracking & cutover control

What deeb persists and what the user can do at each stage:

1. **Execution starts** — `ExecutionService.executeGhost` (services/execution_service.go) creates an execution log entry, a changelog entry, acquires the database execution lock, and marks the plan `InProgress`. The **postpone flag** `deeb-<plan-id>.postpone` is created up front so gh-ost parks awaiting cutover rather than swapping immediately (ghost-lab workflow).
2. **Copy phase streams** — gh-ost output lines are scanned by `ParseGhostStatus`; when it sees `postponing cut-over`, the callback flips the plan status to **`AwaitingCutover`**. All tool output is also emitted into the configured logger for operators.
3. **Cutover approval UX** — the user clicks **Approve Cutover** in the UI (or hits `POST /api/v1/change-plans/:id/cutover-approve`). The backend calls `GhostExecutor.ApproveCutover`, which **removes the postpone flag file**; gh-ost immediately performs the atomic `RENAME TABLE`. Plan transitions to `Completed` and the lock releases, unblocking queued plans for the same DB.
4. **Abort UX** — the **Abort Migration** button (`POST /api/v1/change-plans/:id/abort`) creates the **panic flag** `deeb-<plan-id>.panic`; gh-ost aborts cleanly before cut-over, the original table is untouched, and the plan is marked `Failed`/`Cancelled`.
5. **Socket control (not wired today)** — gh-ost also supports `--serve-socket-file` (default `/tmp/gh-ost.<db>.<table>.sock`) where operators can push `throttle`, `no-throttle`, `chunk-size=N`, `status`. Deeb intentionally uses only the flag-file model (part of the ghost-lab validated setup, engine-uniform). Socket-based throttle/status probing is noted as a possible future enhancement below.

## Constraints

Ghost execution is only allowed when **all** of the following hold:

1. `change_type = DDL`
2. The SQL is a **single `ALTER TABLE` statement** (one table per plan)
3. The target database's environment has `is_production = false`
4. The target engine is **MySQL** (pg-online-schema-change for PostgreSQL is
   not implemented yet — selection is rejected for postgres)
5. The target MySQL server has `binlog_format=ROW` and
   `binlog_row_image=FULL` (verified live at plan creation/update against
   the instance; misconfigured RDS parameter groups fail fast)

DML plans and production environments always use the direct executor.
Validation happens both at the API layer (create/update) and again at
execution time (defense-in-depth).

## Authoring gate: gh-ost dry run

A ghost plan must pass a **gh-ost dry run** (`gh-ost` without `--execute`)
against the live target **at create/update time**. The dry run validates
privileges, binlog access, and ALTER syntax without migrating. If it fails,
the plan cannot be created or saved with `execution_backend=ghost` — the
error is returned immediately to the author. Approval requires no further
ghost validation beyond the standard SQL Review error check.

## Pre-flight checks and disk headroom

`GET /api/v1/change-plans/:id/ghost-preflight` verifies the target before
execution and returns per-check pass/fail with remediation:

- **MySQL**: `binlog_format=ROW`, `binlog_row_image=FULL`, grants
  (`ALL PRIVILEGES ON <db>.*`, `REPLICATION SLAVE`, `REPLICATION CLIENT`)
- **Disk headroom estimate**: reads the table's `DATA_LENGTH + INDEX_LENGTH`
  from `information_schema` and reports the approximate free disk gh-ost
  needs (full table copy + new index + binlog growth during the copy).
  This is an **estimate shown as a warning** — verifying actual free disk
  (CloudWatch `FreeStorageSpace` on RDS) is the operator's responsibility.
- **PostgreSQL** (future): primary key + no triggers

## Workflow

1. Plan is approved and execution starts (same lock/queue semantics as direct).
2. The ghost tool copies the table (shadow table) while streaming progress.
3. When the copy finishes, the tool parks behind the **postpone flag**;
   the plan moves to status `AwaitingCutover`.
4. A user with execution permissions clicks **Approve Cutover**
   (`POST /api/v1/change-plans/:id/cutover-approve`) → deeb removes the
   flag → atomic `RENAME TABLE` completes in under a second → plan `Completed`.
5. **Abort** (`POST /api/v1/change-plans/:id/abort`) creates the panic flag;
   the tool aborts cleanly before cutover and the original table is untouched.

The database execution lock is held for the whole run (copy + cutover wait),
so other plans for the same database stay queued until completion.

## Prerequisites

### MySQL (gh-ost)

Target server must have:

```sql
-- binlog configuration (RDS parameter group / my.cnf)
binlog_format    = ROW
binlog_row_image = FULL

-- grants for the user Deeb connects as
GRANT ALL PRIVILEGES ON <db>.* TO '<user>'@'%';
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '<user>'@'%';
```

The target table must have a PRIMARY KEY and no triggers.

### PostgreSQL (pg-online-schema-change)

Target table must have a PRIMARY KEY and no triggers.

### Pre-flight check

`GET /api/v1/change-plans/:id/ghost-preflight` verifies all of the above
against the live target and returns per-check pass/fail with remediation
guidance. Run it before selecting `execution_backend: ghost`.

## Configuration

| Env variable         | Config path                          | Default | Description                              |
|----------------------|--------------------------------------|---------|------------------------------------------|
| `GHOST_BIN`          | `features.ghost_executor.ghost_bin`  | _(off)_ | Path to the `gh-ost` binary              |
| `PGOSC_BIN`          | `features.ghost_executor.pgosc_bin`  | _(off)_ | Path to `pg-online-schema-change`        |
| `GHOST_FLAG_DIR`     | `features.ghost_executor.flag_dir`   | `.`     | Directory for postpone/panic flag files  |
| `GHOST_MAX_DURATION` | `features.ghost_executor.max_duration` | `6h`  | Max ghost run duration (queue window)    |

Flag files are per-plan: `<flag_dir>/deeb-<plan-id>.postpone` and
`deeb-<plan-id>.panic`.

## Fixed tool flags (MySQL/gh-ost)

```
--exact-rowcount --concurrent-rowcount --chunk-size=1000
--max-load=Threads_running=25 --critical-load=Threads_running=64
--cut-over=default --allow-on-master --execute
```

Throttle-control-replicas and max-lag knobs are deliberately deferred to
future configuration (Decisions Locked #3).

## Rollback

- **Pre-cutover**: Abort endpoint (panic flag) → original table untouched.
- **Post-cutover**: gh-ost keeps the old table as `_<table>_del` (we do not
  pass `--ok-to-drop-table`); deeb exposes two actions on **Completed** ghost
  plans (execution permission required):

  1. **Rollback** (`POST /api/v1/change-plans/:id/ghost-rollback`) — atomic
     swap-back: `RENAME TABLE <table> → _<table>_new, _<table>_del → <table>`.
     ⚠️ **Data-loss warning**: writes made to the live table since cutover are
     NOT in the backup; they are preserved in `_<table>_new` for manual
     reconciliation, but the live table reverts to pre-migration contents.
     The plan is marked `Failed` with a note pointing at the sidelined table.
     For ADD INDEX plans, also consider the gentler `DROP INDEX` alternative.
  2. **Drop Backup Table** (`POST /api/v1/change-plans/:id/ghost-drop-backup`) —
     `DROP TABLE IF EXISTS _<table>_del` after the validation window passes.
     This removes the rollback net permanently.

  Both actions are available in the change plan detail UI (actions dropdown)
  only for Completed ghost plans.

  Note: these paths assume the gh-ost `_del` naming convention;
  pg-online-schema-change artifact naming differs (future parity work).

## Status additions

`AwaitingCutover` is a first-class plan status:

```
Approved → InProgress → AwaitingCutover → Completed
                     ↘ (tool failure / abort) → Failed
```

## Future work (out of scope now)

- Automatic executor selection via `RiskAssessmentService` (DDL + table size)
- Configurable throttle/replica-lag knobs
- pg-online-schema-change cutover flag parity with gh-ost
- Socket-based control (`--serve-socket-file` channel for throttle/status probing) instead of or alongside flag files
- Percent-complete parsing from gh-ost output into the execution log (currently only the `AwaitingCutover` flip is persisted; tool stdout otherwise streams to logs only)
- Optional in-process Go integration (`gh-ost/go/logic`) if subprocess hosting ever becomes a blocker
