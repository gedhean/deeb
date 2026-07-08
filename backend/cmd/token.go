package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/nsxbet/deeb/internal/auth"
	"github.com/nsxbet/deeb/internal/config"
	"github.com/nsxbet/deeb/internal/database"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/nsxbet/deeb/internal/models"
	"github.com/nsxbet/deeb/internal/repository"
	"github.com/nsxbet/deeb/internal/repository/postgres"
	"github.com/spf13/cobra"
)

// defaultTokenUserEmail is the system user API tokens bind to unless overridden.
// It is created by migration 000032 with the OrganizationAdmin role.
const defaultTokenUserEmail = "system@deeb1.dev"

var (
	tokenName      string
	tokenUserEmail string
	tokenExpiresIn time.Duration
)

// tokenCmd is the parent command for API token management.
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens (service accounts)",
	Long: `Manage long-lived API tokens used by unattended automation to authenticate
against the Deeb REST API. A token binds to a user and inherits that user's role.`,
}

var tokenIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issue a new API token",
	Long: `Issue a new API token bound to a user. The plaintext token is printed once and
never stored - save it immediately.`,
	RunE: runTokenIssue,
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API tokens",
	RunE:  runTokenList,
}

var tokenRevokeCmd = &cobra.Command{
	Use:   "revoke <id>",
	Short: "Revoke an API token",
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenRevoke,
}

func init() {
	rootCmd.AddCommand(tokenCmd)
	tokenCmd.AddCommand(tokenIssueCmd)
	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenRevokeCmd)

	tokenIssueCmd.Flags().StringVar(&tokenName, "name", "", "Human-readable name for the token (required)")
	tokenIssueCmd.Flags().StringVar(&tokenUserEmail, "user-email", defaultTokenUserEmail,
		"Email of the user the token binds to")
	tokenIssueCmd.Flags().DurationVar(&tokenExpiresIn, "expires-in", 0,
		"Optional token lifetime (e.g. 720h); zero means long-lived")
	_ = tokenIssueCmd.MarkFlagRequired("name")
}

// tokenRepos opens the database using the shared config and returns the repositories
// needed by the token commands, plus a close func.
func tokenRepos(cmd *cobra.Command) (repository.APITokenRepository, repository.UserRepository, func(), error) {
	configPath, _ := cmd.Flags().GetString("config")

	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := logger.Initialize(logger.Config{Level: cfg.Logging.Level, Format: cfg.Logging.Format}); err != nil {
		return nil, nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	db, err := database.Connect(cfg, logger.Get())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	closeFn := func() {
		if closeErr := db.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close database: %v\n", closeErr)
		}
	}

	return postgres.NewAPITokenRepository(db), postgres.NewUserRepository(db), closeFn, nil
}

func runTokenIssue(cmd *cobra.Command, _ []string) error {
	apiTokenRepo, userRepo, closeFn, err := tokenRepos(cmd)
	if err != nil {
		return err
	}
	defer closeFn()

	ctx := context.Background()

	user, err := userRepo.GetByEmail(ctx, tokenUserEmail)
	if err != nil {
		return fmt.Errorf("failed to find user %q: %w", tokenUserEmail, err)
	}

	plaintext, hash, prefix, err := auth.GenerateAPIToken()
	if err != nil {
		return err
	}

	token := &models.APIToken{
		UserID:      user.ID,
		Name:        tokenName,
		TokenPrefix: prefix,
		TokenHash:   hash,
	}
	if tokenExpiresIn > 0 {
		expiresAt := time.Now().Add(tokenExpiresIn)
		token.ExpiresAt = &expiresAt
	}

	if err := apiTokenRepo.Create(ctx, token); err != nil {
		return fmt.Errorf("failed to create api token: %w", err)
	}

	fmt.Printf("API token issued for %s (id: %s)\n", user.Email, token.ID)
	fmt.Println("Save this token now - it will not be shown again:")
	fmt.Println()
	fmt.Println("  " + plaintext)
	fmt.Println()

	return nil
}

func runTokenList(cmd *cobra.Command, _ []string) error {
	apiTokenRepo, _, closeFn, err := tokenRepos(cmd)
	if err != nil {
		return err
	}
	defer closeFn()

	tokens, err := apiTokenRepo.List(context.Background())
	if err != nil {
		return fmt.Errorf("failed to list api tokens: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w, "ID\tNAME\tPREFIX\tUSER_ID\tLAST_USED\tEXPIRES\tREVOKED")
	for _, t := range tokens {
		_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			t.ID, t.Name, t.TokenPrefix, t.UserID,
			formatTime(t.LastUsedAt), formatTime(t.ExpiresAt), formatTime(t.RevokedAt))
	}
	return w.Flush()
}

func runTokenRevoke(cmd *cobra.Command, args []string) error {
	id, err := uuid.Parse(args[0])
	if err != nil {
		return fmt.Errorf("invalid token id: %w", err)
	}

	apiTokenRepo, _, closeFn, err := tokenRepos(cmd)
	if err != nil {
		return err
	}
	defer closeFn()

	if err := apiTokenRepo.Revoke(context.Background(), id); err != nil {
		return fmt.Errorf("failed to revoke api token: %w", err)
	}

	fmt.Printf("Revoked api token %s\n", id)
	return nil
}

// formatTime renders an optional timestamp for tabular display.
func formatTime(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format(time.RFC3339)
}
