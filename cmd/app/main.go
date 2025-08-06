package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"webhook-processor/config"
	"webhook-processor/internal/server"
	"webhook-processor/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := logger.NewLogger(cfg.LogLevel)

	// Initialize server
	srv := server.NewServer(cfg, logger)

	// Start server
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Shutdown server
	if err := srv.Shutdown(); err != nil {
		logger.Errorf("Server shutdown failed: %v", err)
	}
}
