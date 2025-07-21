package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/imedwei/railway-postgres-backup/internal/backup"
	"github.com/imedwei/railway-postgres-backup/internal/config"
	"github.com/imedwei/railway-postgres-backup/internal/health"
	"github.com/imedwei/railway-postgres-backup/internal/server"
	"github.com/imedwei/railway-postgres-backup/internal/storage"
)

func main() {
	// Set up logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Log startup
	logger.Info("Railway PostgreSQL Backup Service starting")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Log configuration (without sensitive data)
	logger.Info("Configuration loaded",
		"storage_provider", cfg.StorageProvider,
		"backup_prefix", cfg.BackupFilePrefix,
		"respawn_protection_hours", cfg.RespawnProtectionHours,
		"force_backup", cfg.ForceBackup,
		"retention_days", cfg.RetentionDays,
	)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start metrics server if enabled
	var httpServer *server.Server
	var wg sync.WaitGroup

	if metricsPort := os.Getenv("METRICS_PORT"); metricsPort != "" {
		port, err := strconv.Atoi(metricsPort)
		if err != nil {
			logger.Warn("Invalid METRICS_PORT, using default", "error", err)
			port = 8080
		}

		serverConfig := server.DefaultConfig()
		serverConfig.Port = port
		httpServer = server.New(serverConfig, logger)

		// Register health checks
		httpServer.RegisterHealthCheck("storage", func(ctx context.Context) health.Check {
			// Simple check - in production, you might ping the storage backend
			return health.Check{
				Status:    health.StatusHealthy,
				Timestamp: time.Now(),
				Details:   map[string]interface{}{"provider": cfg.StorageProvider},
			}
		})

		// Start server in background
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := httpServer.Start(); err != nil {
				logger.Error("HTTP server failed", "error", err)
			}
		}()

		// Give server time to start
		time.Sleep(100 * time.Millisecond)
	}

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()

		// Shutdown HTTP server
		if httpServer != nil {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil {
				logger.Error("HTTP server shutdown failed", "error", err)
			}
		}
	}()

	// Create storage provider
	storageProvider, err := storage.NewStorage(ctx, cfg)
	if err != nil {
		logger.Error("Failed to create storage provider", "error", err)
		os.Exit(1)
	}

	// Create backup provider
	backupProvider := backup.NewPostgresBackup(cfg.DatabaseURL, cfg.PGDumpOptions)

	// Create and run orchestrator
	orchestrator := backup.NewOrchestrator(cfg, storageProvider, backupProvider, logger)

	if err := orchestrator.Run(ctx); err != nil {
		logger.Error("Backup failed", "error", err)
		os.Exit(1)
	}

	logger.Info("Backup completed successfully")

	// Wait for HTTP server to finish if it was started
	wg.Wait()

	os.Exit(0)
}
