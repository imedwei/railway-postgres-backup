// Package config handles application configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Database configuration
	DatabaseURL string

	// Storage provider configuration
	StorageProvider string // "s3" or "gcs"

	// S3 configuration
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	S3Bucket           string
	S3Region           string
	S3Endpoint         string // Optional custom endpoint

	// GCS configuration
	GCSBucket                string
	GoogleProjectID          string
	GoogleServiceAccountJSON string

	// Respawn protection
	RespawnProtectionHours int
	ForceBackup            bool

	// Backup options
	BackupFilePrefix string
	PGDumpOptions    string
	RetentionDays    int
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		StorageProvider: os.Getenv("STORAGE_PROVIDER"),

		// S3
		AWSAccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		AWSSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		S3Bucket:           os.Getenv("S3_BUCKET"),
		S3Region:           os.Getenv("S3_REGION"),
		S3Endpoint:         os.Getenv("S3_ENDPOINT"),

		// GCS
		GCSBucket:                os.Getenv("GCS_BUCKET"),
		GoogleProjectID:          os.Getenv("GOOGLE_PROJECT_ID"),
		GoogleServiceAccountJSON: os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON"),

		// Options
		BackupFilePrefix: os.Getenv("BACKUP_FILE_PREFIX"),
		PGDumpOptions:    os.Getenv("PG_DUMP_OPTIONS"),
	}

	// Parse numeric values with defaults
	cfg.RespawnProtectionHours = getEnvInt("RESPAWN_PROTECTION_HOURS", 6)
	cfg.RetentionDays = getEnvInt("RETENTION_DAYS", 0) // 0 means no retention policy
	cfg.ForceBackup = getEnvBool("FORCE_BACKUP", false)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.StorageProvider == "" {
		return fmt.Errorf("STORAGE_PROVIDER is required")
	}

	switch c.StorageProvider {
	case "s3":
		if err := c.validateS3(); err != nil {
			return err
		}
	case "gcs":
		if err := c.validateGCS(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid STORAGE_PROVIDER: %s (must be 's3' or 'gcs')", c.StorageProvider)
	}

	if c.RespawnProtectionHours < 0 {
		return fmt.Errorf("RESPAWN_PROTECTION_HOURS must be non-negative")
	}

	if c.RetentionDays < 0 {
		return fmt.Errorf("RETENTION_DAYS must be non-negative")
	}

	return nil
}

func (c *Config) validateS3() error {
	if c.AWSAccessKeyID == "" {
		return fmt.Errorf("AWS_ACCESS_KEY_ID is required for S3 storage")
	}
	if c.AWSSecretAccessKey == "" {
		return fmt.Errorf("AWS_SECRET_ACCESS_KEY is required for S3 storage")
	}
	if c.S3Bucket == "" {
		return fmt.Errorf("S3_BUCKET is required for S3 storage")
	}
	if c.S3Region == "" && c.S3Endpoint == "" {
		return fmt.Errorf("S3_REGION is required for S3 storage (unless S3_ENDPOINT is set)")
	}
	return nil
}

func (c *Config) validateGCS() error {
	if c.GCSBucket == "" {
		return fmt.Errorf("GCS_BUCKET is required for GCS storage")
	}
	if c.GoogleProjectID == "" {
		return fmt.Errorf("GOOGLE_PROJECT_ID is required for GCS storage")
	}
	if c.GoogleServiceAccountJSON == "" {
		return fmt.Errorf("GOOGLE_SERVICE_ACCOUNT_JSON is required for GCS storage")
	}
	return nil
}

// GetRespawnProtectionDuration returns the respawn protection as a Duration.
func (c *Config) GetRespawnProtectionDuration() time.Duration {
	return time.Duration(c.RespawnProtectionHours) * time.Hour
}

// getEnvInt gets an integer from environment variable with a default value.
func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

// getEnvBool gets a boolean from environment variable with a default value.
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
