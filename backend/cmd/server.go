package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsxbet/deeb/internal/app"
	"github.com/nsxbet/deeb/internal/logger"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the Deeb API server",
	Long: `Start the Deeb API server which provides the REST API for managing
database changes. The server handles authentication, authorization, and
orchestrates the database change workflow.`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(serverCmd)
}

func runServer(cmd *cobra.Command, args []string) error {
	// Get config file from parent command
	configPath, _ := cmd.Flags().GetString("config")

	// Log config path
	logger.Info("Loading config from path", zap.String("config_path", configPath))

	// Create and initialize application
	application, err := app.New(version, configPath)
	if err != nil {
		return fmt.Errorf("failed to initialize application: %w", err)
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := application.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or error
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Error("Server error", zap.Error(err))
		return err
	case sig := <-quit:
		logger.Info("Received signal, shutting down gracefully", zap.String("signal", sig.String()))
	}

	// Graceful shutdown with timeout
	if err := application.Shutdown(application.Config.Server.ShutdownTimeout); err != nil {
		return fmt.Errorf("shutdown error: %w", err)
	}

	logger.Info("Server stopped successfully")
	return nil
}
