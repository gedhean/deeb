package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/nsxbet/deeb/internal/app"
	"github.com/nsxbet/deeb/internal/logger"
	"go.uber.org/zap"
)

const version = "0.1.0"

func main() {
	// Create and initialize application
	application, err := app.New(version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		if err := application.Start(); err != nil {
			logger.Error("Server error", zap.Error(err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown with timeout
	if err := application.Shutdown(application.Config.Server.ShutdownTimeout); err != nil {
		fmt.Fprintf(os.Stderr, "Shutdown error: %v\n", err)
		os.Exit(1)
	}
}
