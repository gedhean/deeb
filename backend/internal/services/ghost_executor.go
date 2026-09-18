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
	"github.com/nsxbet/deeb/internal/parser"
	"github.com/nsxbet/deeb/internal/parser/base"
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
	ghostBin       string
	pgoscBin       string
	flagDir        string
	envRepo        repository.EnvironmentRepository
	databaseRepo   repository.DatabaseRepository
	dbInstanceRepo repository.DatabaseInstanceRepository
	maxDuration    time.Duration
}

// NewGhostExecutor creates a new GhostExecutor; flagDir may be empty to use
// the system temp directory (containers typically run with a read-only root
// filesystem, so a relative directory would fail). maxDuration of zero
// defaults to 6 hours.
func NewGhostExecutor(
	ghostBin, pgoscBin, flagDir string,
	envRepo repository.EnvironmentRepository,
	databaseRepo repository.DatabaseRepository,
	dbInstanceRepo repository.DatabaseInstanceRepository,
	maxDuration time.Duration,
) *GhostExecutor {
	if maxDuration == 0 {
		maxDuration = 6 * time.Hour
	}
	return &GhostExecutor{
		ghostBin:       ghostBin,
		pgoscBin:       pgoscBin,
		flagDir:        flagDir,
		envRepo:        envRepo,
		databaseRepo:   databaseRepo,
		dbInstanceRepo: dbInstanceRepo,
		maxDuration:    maxDuration,
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
		if err != nil {
			return fmt.Errorf("failed to resolve target database for ghost eligibility: %w", err)
		}
		dbInstance, err := g.dbInstanceRepo.GetByID(ctx, dbRow.DatabaseInstanceID)
		if err != nil {
			return fmt.Errorf("failed to resolve database instance for ghost eligibility: %w", err)
		}
		if dbInstance.EnvironmentID != nil {
			env, err := g.envRepo.GetByID(ctx, *dbInstance.EnvironmentID)
			if err != nil {
				return fmt.Errorf("failed to resolve environment for ghost eligibility: %w", err)
			}
			if env != nil && env.IsProduction {
				return fmt.Errorf("ghost executor not permitted on production environments")
			}
		}
	}

	return nil
}

// WriteGhostConfFile writes a minimal my.cnf-style credentials file for
// gh-ost under /tmp with a collision-free name (0600 perms). Passing
// credentials via --conf keeps the password out of the process argv, which
// is world-readable via /proc/<pid>/cmdline and ps. Caller must remove the
// file after the run.
func WriteGhostConfFile(planID uuid.UUID, user, password string) (string, error) {
	// /tmp/deeb-ghost-<plan>-<random>.cnf — unique per run, no name crashes
	f, err := os.CreateTemp("/tmp", fmt.Sprintf("deeb-ghost-%s-*.cnf", planID.String()))
	if err != nil {
		return "", fmt.Errorf("failed to create ghost conf file: %w", err)
	}
	path := f.Name()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to secure ghost conf file: %w", err)
	}
	content := fmt.Sprintf("[client]\nuser=%s\npassword=%s\n", user, password)
	if _, err := f.WriteString(content); err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to write ghost conf file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("failed to close ghost conf file: %w", err)
	}
	return path, nil
}

// BuildArgs composes tool invocation for the given engine. Returns the
// resolved binary path and the argument list. It does not launch the tool.
// Credentials are expected via confPath (--conf), never as a --password arg.
func (g *GhostExecutor) BuildArgs(
	engine models.DatabaseEngine,
	host, user, confPath, databaseName, table, alterClause string,
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
			"--conf=" + confPath,
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
			"--conf=" + confPath,
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
// It also ensures the flag directory exists (gh-ost and the executor both
// create files there, and containers may not have it pre-provisioned).
func (g *GhostExecutor) ControlFlagPaths(planID uuid.UUID) (postpone string, panic string) {
	if g.flagDir == "" {
		g.flagDir = os.TempDir()
	}
	_ = os.MkdirAll(g.flagDir, 0o755)
	postpone = fmt.Sprintf("%s/deeb-%s.postpone", g.flagDir, planID)
	panic = fmt.Sprintf("%s/deeb-%s.panic", g.flagDir, planID)
	return
}

// ghostCopyRe extracts the copy progress percentage from gh-ost status lines
// (e.g. "Copy: 23711127/23711127 45.2%; ..."). Used to render the plan's
// live progress bar.
var ghostCopyRe = regexp.MustCompile(`Copy:\s*([0-9.]+)%`)

// ParseGhostProgress extracts the numeric copy percent from a gh-ost status
// line, or returns nil when the line has no progress value.
func ParseGhostProgress(line string) *float64 {
	m := ghostCopyRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}
	var pct float64
	if _, err := fmt.Sscanf(m[1], "%f", &pct); err != nil {
		return nil
	}
	return &pct
}

// SanitizeArgsForLog returns a copy of args with secrets masked for logging.
// The executable args themselves stay intact — only the log copy is masked.
func SanitizeArgsForLog(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		if strings.HasPrefix(a, "--password=") {
			out[i] = "--password=***"
		} else {
			out[i] = a
		}
	}
	return out
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

// ParseAlterStatement extracts table name and ALTER clause from a single
// ALTER TABLE statement; rejects anything else (multi-statement or non-ALTER).
// Uses the project's SQL tokenizer/splitter (internal/parser) — no regex.
func ParseAlterStatement(sqlContent string) (table, alterClause string, err error) {
	result, err := parser.SplitSQL(sqlContent, parser.DialectMySQL)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse SQL: %w", err)
	}
	if result.TotalCount != 1 {
		return "", "", fmt.Errorf("ghost executor supports only a single ALTER TABLE statement (got %d)", result.TotalCount)
	}
	stmt := result.Statements[0].Text

	t := base.NewTokenizer(stmt)
	word := func() string {
		t.SkipBlank()
		start := t.Pos()
		for {
			c := t.Char(0)
			if c == base.EOFRune || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';' {
				break
			}
			t.Skip(1)
		}
		return t.GetString(start, t.Pos()-start)
	}

	if !strings.EqualFold(word(), "ALTER") {
		return "", "", fmt.Errorf("ghost executor supports only ALTER TABLE statements")
	}
	if !strings.EqualFold(word(), "TABLE") {
		return "", "", fmt.Errorf("ghost executor supports only ALTER TABLE statements")
	}

	// Table name: consume qualified components (quoted or unquoted) separated
	// by dots; the final component is the table gh-ost migrates. The database
	// qualifier comes from the --database flag, not here.
	readComponent := func() string {
		t.SkipBlank()
		start := t.Pos()
		if c := t.Char(0); c == '`' || c == '"' {
			t.ScanIdentifier(c)
		} else {
			for {
				c := t.Char(0)
				if c == base.EOFRune || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';' || c == '.' {
					break
				}
				t.Skip(1)
			}
		}
		return strings.Trim(t.GetString(start, t.Pos()-start), "`\"")
	}

	table = readComponent()
	for {
		t.SkipBlank()
		if t.Char(0) != '.' {
			break
		}
		t.Skip(1) // consume the separator
		next := readComponent()
		if next == "" {
			break
		}
		table = next
	}
	if table == "" {
		return "", "", fmt.Errorf("could not extract table name from statement")
	}

	// Everything after the table name is the ALTER clause.
	remaining := t.GetString(t.Pos(), uint(len(stmt))-t.Pos())
	alterClause = strings.TrimSpace(remaining)
	alterClause = strings.TrimSuffix(alterClause, ";")
	alterClause = strings.TrimSpace(alterClause)
	if alterClause == "" {
		return "", "", fmt.Errorf("could not extract alter clause from statement")
	}
	return table, alterClause, nil
}

// ClassifyGhostStatements splits SQL content and reports which statements
// gh-ost would run (ALTER TABLE) vs which fall back to direct execution.
// Informational only — used to warn users at plan creation.
func ClassifyGhostStatements(sqlContent string) (ghostCount, directCount int, err error) {
	result, err := parser.SplitSQL(sqlContent, parser.DialectMySQL)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse SQL: %w", err)
	}
	for _, stmt := range result.Statements {
		if stmt.Empty {
			continue
		}
		t := base.NewTokenizer(stmt.Text)
		t.SkipBlank()
		start := t.Pos()
		for {
			c := t.Char(0)
			if c == base.EOFRune || c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == ';' {
				break
			}
			t.Skip(1)
		}
		if strings.EqualFold(t.GetString(start, t.Pos()-start), "ALTER") {
			ghostCount++
		} else {
			directCount++
		}
	}
	return ghostCount, directCount, nil
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
