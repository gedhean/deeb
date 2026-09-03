# Ghost Executor (gh-ost / pg-online-schema-change)

Deeb supports an alternative execution backend for **DDL change plans**:
the **ghost executor**. Instead of running SQL directly in a transaction,
the change runs through an online-schema-change tool:

- **MySQL** → [gh-ost](https://github.com/github/gh-ost) (shadow table copy + binlog tailing + atomic rename)
- **PostgreSQL** → [pg-online-schema-change](https://github.com/pg-dba/pg-online-schema-change)

The change-plan workflow (create → review → approve → execute) stays
identical; only the execution engine differs. Executor choice is explicit
user preference at plan creation (`execution_backend: direct | ghost`).

## Constraints

Ghost execution is only allowed when **all** of the following hold:

1. `change_type = DDL`
2. The SQL is a **single `ALTER TABLE` statement** (one table per plan)
3. The target database's environment has `is_production = false`

DML plans and production environments always use the direct executor.
Validation happens both at the API layer (create/update) and again at
execution time (defense-in-depth).

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
  pass `--ok-to-drop-table`); atomic rename swap-back, or `DROP INDEX` for
  ADD INDEX plans. Retained old table is the rollback net — drop it manually
  after the validation window.

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
