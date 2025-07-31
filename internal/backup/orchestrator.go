package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/imedwei/railway-postgres-backup/internal/config"
	"github.com/imedwei/railway-postgres-backup/internal/metrics"
	"github.com/imedwei/railway-postgres-backup/internal/ratelimit"
	"github.com/imedwei/railway-postgres-backup/internal/storage"
	"github.com/imedwei/railway-postgres-backup/internal/utils"
)

// Orchestrator coordinates the backup process.
type Orchestrator struct {
	config      *config.Config
	storage     storage.Storage
	backup      Backup
	rateLimiter ratelimit.RateLimiter
	logger      *slog.Logger
}

// NewOrchestrator creates a new backup orchestrator.
func NewOrchestrator(cfg *config.Config, storage storage.Storage, backup Backup, logger *slog.Logger) *Orchestrator {
	// Create rate limiter
	rlConfig := ratelimit.Config{
		MinInterval: cfg.GetRespawnProtectionDuration(),
		ForceBackup: cfg.ForceBackup,
	}
	rateLimiter := ratelimit.NewTimeBasedLimiter(rlConfig)

	return &Orchestrator{
		config:      cfg,
		storage:     storage,
		backup:      backup,
		rateLimiter: rateLimiter,
		logger:      logger,
	}
}

// Run executes the backup process.
func (o *Orchestrator) Run(ctx context.Context) error {
	startTime := time.Now()
	o.logger.Info("Starting backup orchestration")

	// Initialize metrics
	metrics.Info.WithLabelValues("1.0.0", o.config.StorageProvider).Set(1)

	// Check respawn protection
	lastBackupTime, err := o.storage.GetLastBackupTime(ctx)
	if err != nil {
		o.logger.Warn("Failed to get last backup time, proceeding with backup", "error", err)
		// Continue with backup if we can't determine last backup time
	} else {
		shouldBackup, reason := o.rateLimiter.ShouldBackup(lastBackupTime)
		o.logger.Info("Rate limiter decision", "should_backup", shouldBackup, "reason", reason)

		if !shouldBackup {
			o.logger.Info("Skipping backup due to rate limiting", "reason", reason)
			metrics.RateLimitBlocked.Inc()
			return nil
		}
	}

	// Get database info
	info, err := o.backup.GetInfo(ctx)
	if err != nil {
		o.logger.Warn("Failed to get database info", "error", err)
		// Continue without info
		info = &DatabaseInfo{Name: "unknown", Size: 0, Version: "unknown"}
	} else {
		o.logger.Info("Database info",
			"name", info.Name,
			"size_bytes", info.Size,
			"version", info.Version,
		)
		metrics.DatabaseSize.Set(float64(info.Size))
	}

	// Generate backup filename and key
	timestamp := time.Now()
	filename := utils.GenerateBackupFilename(o.config.BackupFilePrefix, timestamp, info.Version)

	// Create storage key with year/month directory structure
	storageKey := fmt.Sprintf("%d/%02d/%s", timestamp.Year(), timestamp.Month(), filename)

	o.logger.Info("Generated backup filename", "filename", filename, "storage_key", storageKey)

	// Create backup
	o.logger.Info("Starting database dump")
	dumpTimer := metrics.BackupDuration.WithLabelValues("dump")
	dumpStart := time.Now()

	reader, err := o.backup.Dump(ctx)
	if err != nil {
		metrics.RecordBackupAttempt(false)
		return fmt.Errorf("failed to create backup: %w", err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			o.logger.Warn("Failed to close reader", "error", err)
		}
	}()

	dumpTimer.Observe(time.Since(dumpStart).Seconds())

	// Create a tee reader to count bytes
	pr, pw := io.Pipe()
	var bytesWritten int64

	go func() {
		defer func() {
			if err := pw.Close(); err != nil {
				o.logger.Warn("Failed to close pipe writer", "error", err)
			}
		}()

		buf := make([]byte, 32*1024) // 32KB buffer
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				bytesWritten += int64(n)
				if _, writeErr := pw.Write(buf[:n]); writeErr != nil {
					if closeErr := pw.CloseWithError(writeErr); closeErr != nil {
						o.logger.Warn("Failed to close pipe writer with error", "error", closeErr)
					}
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					if closeErr := pw.CloseWithError(err); closeErr != nil {
						o.logger.Warn("Failed to close pipe writer with error", "error", closeErr)
					}
					return
				}
				break
			}
		}
	}()

	// Prepare metadata
	metadata := map[string]string{
		"backup-timestamp": timestamp.Format(time.RFC3339),
		"database-name":    info.Name,
		"database-version": info.Version,
		"backup-tool":      "railway-postgres-backup",
	}

	// Upload to storage
	o.logger.Info("Starting upload to storage", "provider", o.config.StorageProvider)
	uploadTimer := metrics.BackupDuration.WithLabelValues("upload")
	uploadStart := time.Now()

	if err := o.storage.Upload(ctx, storageKey, pr, metadata); err != nil {
		metrics.RecordStorageOperation("upload", o.config.StorageProvider, false)
		metrics.RecordBackupAttempt(false)
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	uploadDuration := time.Since(uploadStart)
	uploadTimer.Observe(uploadDuration.Seconds())
	metrics.RecordStorageOperation("upload", o.config.StorageProvider, true)
	metrics.BackupSize.Set(float64(bytesWritten))
	metrics.LastBackupTimestamp.Set(float64(timestamp.Unix()))
	metrics.RecordBackupAttempt(true)

	o.logger.Info("Backup completed successfully",
		"filename", filename,
		"storage_key", storageKey,
		"bytes_written", bytesWritten,
		"upload_duration", uploadDuration,
		"bytes_per_second", float64(bytesWritten)/uploadDuration.Seconds(),
	)

	// Record total duration
	metrics.BackupDuration.WithLabelValues("total").Observe(time.Since(startTime).Seconds())

	// Optional: Clean up old backups if retention is configured
	if o.config.RetentionDays > 0 {
		if err := o.cleanupOldBackups(ctx); err != nil {
			o.logger.Warn("Failed to cleanup old backups", "error", err)
			// Don't fail the backup operation due to cleanup failure
		}
	}

	return nil
}

// cleanupOldBackups removes backups older than the retention period.
func (o *Orchestrator) cleanupOldBackups(ctx context.Context) error {
	o.logger.Info("Starting cleanup of old backups", "retention_days", o.config.RetentionDays)

	// Calculate cutoff time
	cutoff := time.Now().AddDate(0, 0, -o.config.RetentionDays)

	// List all backups
	objects, err := o.storage.List(ctx, o.config.BackupFilePrefix)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	var deleted int
	for _, obj := range objects {
		// Try to parse timestamp from filename
		backupTime, err := utils.ParseBackupFilename(obj.Key)
		if err != nil {
			o.logger.Warn("Failed to parse backup timestamp, using last modified time",
				"filename", obj.Key,
				"error", err,
			)
			backupTime = obj.LastModified
		}

		if backupTime.Before(cutoff) {
			o.logger.Info("Deleting old backup",
				"filename", obj.Key,
				"backup_time", backupTime,
				"age_days", int(time.Since(backupTime).Hours()/24),
			)

			if err := o.storage.Delete(ctx, obj.Key); err != nil {
				o.logger.Error("Failed to delete old backup",
					"filename", obj.Key,
					"error", err,
				)
				metrics.RecordStorageOperation("delete", o.config.StorageProvider, false)
				// Continue with other deletions
			} else {
				deleted++
				metrics.RecordStorageOperation("delete", o.config.StorageProvider, true)
				metrics.BackupsDeleted.Inc()
			}
		}
	}

	o.logger.Info("Cleanup completed", "deleted_count", deleted)
	return nil
}
