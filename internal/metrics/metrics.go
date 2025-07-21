// Package metrics provides Prometheus metrics for the backup service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// BackupAttempts tracks the total number of backup attempts.
	BackupAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "postgres_backup_attempts_total",
		Help: "Total number of backup attempts",
	}, []string{"status"})

	// BackupDuration tracks the duration of backup operations.
	BackupDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "postgres_backup_duration_seconds",
		Help:    "Duration of backup operations in seconds",
		Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17min
	}, []string{"phase"})

	// BackupSize tracks the size of backups.
	BackupSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "postgres_backup_size_bytes",
		Help: "Size of the last backup in bytes",
	})

	// DatabaseSize tracks the size of the database.
	DatabaseSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "postgres_database_size_bytes",
		Help: "Size of the database in bytes",
	})

	// StorageOperations tracks storage operations.
	StorageOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "postgres_backup_storage_operations_total",
		Help: "Total number of storage operations",
	}, []string{"operation", "provider", "status"})

	// RateLimitBlocked tracks rate limit blocks.
	RateLimitBlocked = promauto.NewCounter(prometheus.CounterOpts{
		Name: "postgres_backup_rate_limit_blocked_total",
		Help: "Total number of backups blocked by rate limiting",
	})

	// LastBackupTimestamp tracks when the last successful backup occurred.
	LastBackupTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "postgres_backup_last_success_timestamp",
		Help: "Unix timestamp of the last successful backup",
	})

	// BackupsDeleted tracks the number of old backups deleted.
	BackupsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "postgres_backup_deleted_total",
		Help: "Total number of old backups deleted",
	})

	// Info provides static information about the service.
	Info = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "postgres_backup_info",
		Help: "Information about the backup service",
	}, []string{"version", "storage_provider"})
)

// RecordBackupAttempt records a backup attempt with its status.
func RecordBackupAttempt(success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	BackupAttempts.WithLabelValues(status).Inc()
}

// RecordStorageOperation records a storage operation.
func RecordStorageOperation(operation, provider string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	StorageOperations.WithLabelValues(operation, provider, status).Inc()
}
