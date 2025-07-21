package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/imedwei/railway-postgres-backup/internal/config"
)

// RetryConfig holds retry configuration for storage operations.
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryableStorage wraps a Storage implementation with retry logic.
type RetryableStorage struct {
	storage Storage
	config  RetryConfig
}

// NewRetryableStorage creates a new storage wrapper with retry logic.
func NewRetryableStorage(storage Storage, config RetryConfig) *RetryableStorage {
	return &RetryableStorage{
		storage: storage,
		config:  config,
	}
}

// Upload implements Storage.Upload with retry logic.
func (r *RetryableStorage) Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	return r.retry(ctx, func() error {
		return r.storage.Upload(ctx, key, reader, metadata)
	})
}

// Delete implements Storage.Delete with retry logic.
func (r *RetryableStorage) Delete(ctx context.Context, key string) error {
	return r.retry(ctx, func() error {
		return r.storage.Delete(ctx, key)
	})
}

// List implements Storage.List with retry logic.
func (r *RetryableStorage) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var result []ObjectInfo
	err := r.retry(ctx, func() error {
		var err error
		result, err = r.storage.List(ctx, prefix)
		return err
	})
	return result, err
}

// GetLastBackupTime implements Storage.GetLastBackupTime with retry logic.
func (r *RetryableStorage) GetLastBackupTime(ctx context.Context) (time.Time, error) {
	var result time.Time
	err := r.retry(ctx, func() error {
		var err error
		result, err = r.storage.GetLastBackupTime(ctx)
		return err
	})
	return result, err
}

// retry executes a function with exponential backoff retry logic.
func (r *RetryableStorage) retry(ctx context.Context, fn func() error) error {
	delay := r.config.InitialDelay

	for attempt := 1; attempt <= r.config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		// Check if this is the last attempt
		if attempt == r.config.MaxAttempts {
			return fmt.Errorf("operation failed after %d attempts: %w", r.config.MaxAttempts, err)
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		// Calculate next delay with exponential backoff
		delay = time.Duration(float64(delay) * r.config.Multiplier)
		if delay > r.config.MaxDelay {
			delay = r.config.MaxDelay
		}
	}

	return nil
}

// NewStorage creates a storage provider based on configuration.
func NewStorage(ctx context.Context, cfg *config.Config) (Storage, error) {
	var storage Storage
	var err error

	switch cfg.StorageProvider {
	case "s3":
		s3Config := S3Config{
			AccessKeyID:     cfg.AWSAccessKeyID,
			SecretAccessKey: cfg.AWSSecretAccessKey,
			Region:          cfg.S3Region,
			Bucket:          cfg.S3Bucket,
			Endpoint:        cfg.S3Endpoint,
			Prefix:          cfg.BackupFilePrefix,
			ObjectLock:      false,                // Could be made configurable
			UsePathStyle:    cfg.S3Endpoint != "", // Use path style for custom endpoints
		}
		storage, err = NewS3Storage(ctx, s3Config)

	case "gcs":
		// Validate service account JSON
		if err := ValidateServiceAccountJSON(cfg.GoogleServiceAccountJSON); err != nil {
			return nil, fmt.Errorf("invalid GCS service account: %w", err)
		}

		gcsConfig := GCSConfig{
			Bucket:             cfg.GCSBucket,
			ProjectID:          cfg.GoogleProjectID,
			ServiceAccountJSON: cfg.GoogleServiceAccountJSON,
			Prefix:             cfg.BackupFilePrefix,
		}
		storage, err = NewGCSStorage(ctx, gcsConfig)

	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.StorageProvider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s storage: %w", cfg.StorageProvider, err)
	}

	// Wrap with retry logic
	return NewRetryableStorage(storage, DefaultRetryConfig()), nil
}
