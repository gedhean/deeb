// Package cmd provides the command-line interface for the Deeb application.
// It includes commands for running the server and managing database migrations.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.1.0"

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "deeb",
	Short: "Deeb - Secure Database Change Management",
	Long: `Deeb is a secure database change management tool that enables developers
to manage database changes without requiring direct database access.

The system provides issue-based change workflow with multi-approval and review
processes, automated policy validation, and full audit logging.`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
}

// GetVersion returns the application version
func GetVersion() string {
	return version
}
