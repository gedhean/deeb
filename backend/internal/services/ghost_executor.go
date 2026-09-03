// Package services provides business logic services for the Deeb application.
package services

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
)

// AlterPlanParser extracts the target table name and ALTER clause from a
// single ALTER TABLE DDL plan. Ghost executor rejects anything else.
type AlterPlanParser interface {
	Parse(sqlContent string) (table string, alterClause string, err error)
}

// GhostExecutorState identifies user-visible statuses derived from tool output.
const (
	GhostStatusCopying   = "copy"
	GhostStatusAwaiting  = "awaiting-cutover"
	GhostStatusCompleted = "completed"
	GhostStatusFailed    = "failed"
)

// GhostExecutor carries configuration needed to run gh-ost (mysql) or
// pg-online-schema-change (postgres). It performs validation, builds the tool
// invocation, and knows where the control flag files live.
type GhostExecutor struct {
	ghostBin     string
	pgoscBin     string
	flagDir      string
	envRepo      repository.EnvironmentRepository
	databaseRepo repository.DatabaseRepository
	maxDuration  time.Duration
}

// NewGhostExecutor creates a new GhostExecutor; flagDir may be empty to use
// the process working directory. maxDuration of zero defaults to 6 hours.
func NewGhostExecutor(
	ghostBin, pgoscBin, flagDir string,
	envRepo repository.EnvironmentRepository,
	databaseRepo repository.DatabaseRepository,
	maxDuration time.Duration,
) *GhostExecutor {
	if maxDuration == 0 {
		maxDuration = 6 * time.Hour
	}
	return &GhostExecutor{
		ghostBin:     ghostBin,
		pgoscBin:     pgoscBin,
		flagDir:      flagDir,
		envRepo:      envRepo,
		databaseRepo: databaseRepo,
		maxDuration:  maxDuration,
	}
}

// ValidateGhostEligibility enforces ghost constraints (DDL + non-production env).
func (g *GhostExecutor) ValidateGhostEligibility(ctx context.Context, plan *models.ChangePlan) error {
	if plan == nil {
		return fmt.Errorf("plan is nil")
	}
	if plan.ChangeType != models.ChangeTypeDDL {
		return fmt.Errorf("ghost executor requires DDL change type")
	}

	if g.envRepo != nil && g.databaseRepo != nil {
		dbRow, err := g.databaseRepo.GetByID(ctx, plan.TargetDatabaseID)
		if err == nil {
			env, err := g.envRepo.GetByID(ctx, dbRow.EnvironmentID)
			if err == nil && env != nil && env.IsProduction {
				return fmt.Errorf("ghost executor not permitted on production environments")
			}
		}
	}

	return nil
}

// BuildArgs composes tool invocation for the given engine. Returns the
// resolved binary path and the argument list. It does not launch the tool.
func (g *GhostExecutor) BuildArgs(
	engine models.DatabaseEngine,
	host, user, password, databaseName, table, alterClause string,
	port int,
	flagPostpone, flagPanic string,
) (string, []string, error) {
	switch engine {
	case models.DatabaseEngineMySQL:
		if g.ghostBin == "" {
			return "", nil, fmt.Errorf("gh-ost binary not configured (GHOST_BIN)")
		}
		args := []string{
			"--host=" + host,
			fmt.Sprintf("--port=%d", port),
			"--user=" + user,
			"--password=" + password,
			"--database=" + databaseName,
			"--table=" + table,
			"--alter=" + alterClause,
			"--allow-on-master",
			"--exact-rowcount",
			"--concurrent-rowcount",
			"--chunk-size=1000",
			"--max-load=Threads_running=25",
			"--critical-load=Threads_running=64",
			"--cut-over=default",
			"--execute",
		}
		if flagPostpone != "" {
			args = append(args, "--postpone-cut-over-flag-file="+flagPostpone)
		}
		if flagPanic != "" {
			args = append(args, "--panic-flag-file="+flagPanic)
		}
		return g.ghostBin, args, nil

	case models.DatabaseEnginePostgreSQL:
		if g.pgoscBin == "" {
			return "", nil, fmt.Errorf("pg-online-schema-change binary not configured (PGOSC_BIN)")
		}
		// pg-online-schema-change flags; specific knob coverage deferred per decisions.
		args := []string{
			"--host=" + host,
			fmt.Sprintf("--port=%d", port),
			"--user=" + user,
			"--password=" + password,
			"--dbname=" + databaseName,
			"--table=" + table,
			"--alter=" + alterClause,
			"--yes-i-know-what-i-am-doing",
			"--drop-old-table=false",
		}
		return g.pgoscBin, args, nil
	default:
		return "", nil, fmt.Errorf("unsupported engine: %s", engine)
	}
}

// ControlFlagPaths returns the cutover/panic flag paths for a plan.
func (g *GhostExecutor) ControlFlagPaths(planID uuid.UUID) (postpone string, panic string) {
	if g.flagDir == "" {
		g.flagDir = "."
	}
	postpone = fmt.Sprintf("%s/deeb-%s.postpone", g.flagDir, planID)
	panic = fmt.Sprintf("%s/deeb-%s.panic", g.flagDir, planID)
	return
}

// ParseGhostStatus identifies status transitions from tool stdout lines.
func ParseGhostStatus(line string) (string, bool) {
	l := strings.ToLower(line)
	switch {
	case strings.Contains(l, "postponing cut-over"), strings.Contains(l, "awaiting cutover"):
		return GhostStatusAwaiting, true
	case strings.Contains(l, "# done"):
		return GhostStatusCompleted, true
	case strings.Contains(l, "error"), strings.Contains(l, "panic"):
		return GhostStatusFailed, true
	case strings.Contains(l, "copy"):
		return GhostStatusCopying, true
	default:
		return "", false
	}
}

// alterTableRe matches a single `ALTER TABLE <name> <clauses>;` statement.
var alterTableRe = regexp.MustCompile(`(?is)^\s*ALTER\s+TABLE\s+(?:\x60?([\w.]+)\x60?|"([\w.]+)")\s+(.*?);?\s*$`)

// ParseAlterStatement extracts table name and ALTER clause from a single
// ALTER TABLE statement; rejects anything else (multi-statement or non-ALTER).
func ParseAlterStatement(sqlContent string) (table, alterClause string, err error) {
	m := alterTableRe.FindStringSubmatch(sqlContent)
	if m == nil {
		return "", "", fmt.Errorf("ghost executor supports only a single ALTER TABLE statement")
	}
	table = m[1]
	if table == "" {
		table = m[2]
	}
	alterClause = strings.TrimSpace(m[3])
	if table == "" || alterClause == "" {
		return "", "", fmt.Errorf("could not extract table/alter clause from statement")
	}
	if strings.Contains(table, ".") { // tool expects plain table name; schema comes from db flag
		parts := strings.Split(table, ".")
		table = parts[len(parts)-1]
	}
	return table, alterClause, nil
}

// GhostRunCallbacks receives stream events from Run.
type GhostRunCallbacks struct {
	OnLogLine     func(line string)
	OnStateChange func(status string)
}

// Run launches the tool with args built via BuildArgs and streams output.
func (g *GhostExecutor) Run(
	ctx context.Context,
	binPath string,
	args []string,
	callbacks GhostRunCallbacks,
) error {
	ctx, cancel := context.WithTimeout(ctx, g.maxDuration)
	defer cancel()

	cmd := exec.CommandContext(ctx, binPath, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to capture stdout: %w", err)
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start ghost tool: %w", err)
	}

	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if callbacks.OnLogLine != nil {
				callbacks.OnLogLine(line)
			}
			if status, ok := ParseGhostStatus(line); ok && callbacks.OnStateChange != nil {
				callbacks.OnStateChange(status)
			}
		}
	}()

	runErr := cmd.Wait()
	<-scanDone

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("ghost execution exceeded max duration (%s)", g.maxDuration)
	}
	if runErr != nil {
		return fmt.Errorf("ghost tool exited with error: %w", runErr)
	}
	return nil
}

// ApproveCutover removes the postpone flag (plan must be AwaitingCutover).
func (g *GhostExecutor) ApproveCutover(planID uuid.UUID) error {
	postpone, _ := g.ControlFlagPaths(planID)
	if err := os.Remove(postpone); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to approve cutover: %w", err)
	}
	return nil
}

// Abort creates the panic flag, asking the tool to abort cleanly pre-cutover.
func (g *GhostExecutor) Abort(planID uuid.UUID) error {
	_, panicPath := g.ControlFlagPaths(planID)
	f, err := os.Create(panicPath)
	if err != nil {
		return fmt.Errorf("failed to create panic flag: %w", err)
	}
	return f.Close()
}
