package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/imedwei/railway-postgres-backup/internal/backup"
	"github.com/imedwei/railway-postgres-backup/internal/config"
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

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Shutdown signal received")
		cancel()
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
	os.Exit(0)
}
