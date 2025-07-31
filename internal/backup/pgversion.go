package backup

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// PGVersion represents a PostgreSQL version
type PGVersion struct {
	Major int
	Minor int
	Full  string
}

// RetryConfig holds configuration for command retries
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay between retries
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Exponential backoff factor
}

// defaultPSQLRetryConfig returns the default retry configuration for psql commands
func defaultPSQLRetryConfig() RetryConfig {
	config := RetryConfig{
		MaxRetries:    5,                // Fewer retries for psql commands
		InitialDelay:  2 * time.Second,  // Start with 2 second delay
		MaxDelay:      30 * time.Second, // Cap at 30 seconds
		BackoffFactor: 2.0,              // Double the delay each time
	}

	// Override with environment variables if set
	if maxRetries := os.Getenv("PSQL_RETRY_MAX_ATTEMPTS"); maxRetries != "" {
		if val, err := strconv.Atoi(maxRetries); err == nil && val > 0 {
			config.MaxRetries = val
		}
	}

	if initialDelay := os.Getenv("PSQL_RETRY_INITIAL_DELAY"); initialDelay != "" {
		if val, err := strconv.Atoi(initialDelay); err == nil && val > 0 {
			config.InitialDelay = time.Duration(val) * time.Second
		}
	}

	if maxDelay := os.Getenv("PSQL_RETRY_MAX_DELAY"); maxDelay != "" {
		if val, err := strconv.Atoi(maxDelay); err == nil && val > 0 {
			config.MaxDelay = time.Duration(val) * time.Second
		}
	}

	return config
}

// isRetryableError checks if an error from psql command should trigger a retry
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check exit error
	if exitErr, ok := err.(*exec.ExitError); ok {
		errOutput := string(exitErr.Stderr)
		// Check for common retryable error messages
		return strings.Contains(errOutput, "the database system is starting up") ||
			strings.Contains(errOutput, "SQLSTATE 57P03") ||
			strings.Contains(errOutput, "connection refused") ||
			strings.Contains(errOutput, "could not connect to server") ||
			strings.Contains(errOutput, "no such host") ||
			strings.Contains(errOutput, "timeout expired")
	}

	// Check error message
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout")
}

// ParsePGVersion parses a PostgreSQL version string
func ParsePGVersion(versionStr string) (*PGVersion, error) {
	// Match patterns like "PostgreSQL 16.2" or "PostgreSQL 14.11"
	re := regexp.MustCompile(`PostgreSQL (\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionStr)
	if len(matches) < 3 {
		return nil, fmt.Errorf("could not parse PostgreSQL version from: %s", versionStr)
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", matches[1])
	}

	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, fmt.Errorf("invalid minor version: %s", matches[2])
	}

	return &PGVersion{
		Major: major,
		Minor: minor,
		Full:  versionStr,
	}, nil
}

// findAvailablePSQL finds any available psql binary
func findAvailablePSQL() string {
	// Try versioned binaries first (newest to oldest)
	for _, v := range []int{17, 16, 15} {
		psqlBin := fmt.Sprintf("psql%d", v)
		if _, err := exec.LookPath(psqlBin); err == nil {
			return psqlBin
		}
	}
	
	// Fallback to plain psql
	return "psql"
}

// GetServerVersion gets the PostgreSQL server version with retry logic
func GetServerVersion(ctx context.Context, connectionURL string) (*PGVersion, error) {
	return GetServerVersionWithRetry(ctx, connectionURL, defaultPSQLRetryConfig())
}

// GetServerVersionWithRetry gets the PostgreSQL server version with configurable retry logic
func GetServerVersionWithRetry(ctx context.Context, connectionURL string, retryConfig RetryConfig) (*PGVersion, error) {
	// Try to find the best available psql binary
	psqlBin := findAvailablePSQL()
	return getServerVersionWithBinary(ctx, connectionURL, psqlBin, retryConfig)
}

// getServerVersionWithBinary gets the PostgreSQL server version using a specific psql binary
func getServerVersionWithBinary(ctx context.Context, connectionURL string, psqlBin string, retryConfig RetryConfig) (*PGVersion, error) {
	logger := slog.Default().With("component", "pgversion", "binary", psqlBin)

	var attemptErrors []string
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying PostgreSQL version check",
				"attempt", attempt,
				"max_retries", retryConfig.MaxRetries,
				"delay", delay)

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return nil, fmt.Errorf("context cancelled during retry after %d attempts: %w (previous errors: %v)", 
					attempt, ctx.Err(), attemptErrors)
			}

			// Calculate next delay with exponential backoff
			nextDelay := float64(delay) * retryConfig.BackoffFactor
			delay = time.Duration(math.Min(nextDelay, float64(retryConfig.MaxDelay)))
		}

		cmd := exec.CommandContext(ctx, psqlBin,
			"--no-password",
			"--tuples-only",
			"--no-align",
			"--command", "SELECT version();",
			connectionURL,
		)

		// Capture stderr for better error messages
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		output, err := cmd.Output()
		if err == nil {
			versionStr := strings.TrimSpace(string(output))
			version, parseErr := ParsePGVersion(versionStr)
			if parseErr == nil {
				if attempt > 0 {
					logger.Info("Successfully retrieved PostgreSQL version",
						"attempts", attempt+1,
						"version", version.Full)
				}
				return version, nil
			}
			err = parseErr
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			// Add stderr to the error for better debugging
			exitErr.Stderr = stderr.Bytes()
		}

		// Record the error for this attempt
		attemptErrors = append(attemptErrors, fmt.Sprintf("attempt %d: %v (stderr: %s)", attempt+1, err, stderr.String()))

		// Check if this is a connection error that we should retry
		if isRetryableError(err) {
			logger.Warn("Retryable error encountered",
				"attempt", attempt+1,
				"error", err,
				"stderr", stderr.String())
		} else {
			// If it's not retryable, return immediately
			return nil, fmt.Errorf("non-retryable error: %w (stderr: %s)", err, stderr.String())
		}
	}

	return nil, fmt.Errorf("failed to get server version after %d retries (errors: %v)",
		retryConfig.MaxRetries, attemptErrors)
}

// FindBestPGDump finds the best pg_dump binary for the given server version
func FindBestPGDump(serverVersion *PGVersion) (string, error) {
	// List of available PostgreSQL versions (only 15, 16, 17)
	availableVersions := []int{17, 16, 15}

	// For older versions, we'll use pg_dump15 as it should be backward compatible
	targetVersion := serverVersion.Major
	if targetVersion < 15 {
		targetVersion = 15
	}

	// First, try to find exact match
	pgDumpBin := fmt.Sprintf("pg_dump%d", targetVersion)
	if _, err := exec.LookPath(pgDumpBin); err == nil {
		return pgDumpBin, nil
	}

	// If no exact match, find the closest version that's >= server version
	for _, v := range availableVersions {
		if v >= targetVersion {
			pgDumpBin = fmt.Sprintf("pg_dump%d", v)
			if _, err := exec.LookPath(pgDumpBin); err == nil {
				return pgDumpBin, nil
			}
		}
	}

	// If still not found, try plain pg_dump
	if _, err := exec.LookPath("pg_dump"); err == nil {
		return "pg_dump", nil
	}

	// Last resort: try the newest available version
	for _, v := range availableVersions {
		pgDumpBin = fmt.Sprintf("pg_dump%d", v)
		if _, err := exec.LookPath(pgDumpBin); err == nil {
			return pgDumpBin, nil
		}
	}

	return "", fmt.Errorf("no suitable pg_dump found for PostgreSQL %d", serverVersion.Major)
}

// FindBestPSQL finds the best psql binary for the given server version
func FindBestPSQL(serverVersion *PGVersion) (string, error) {
	// List of available PostgreSQL versions (only 15, 16, 17)
	availableVersions := []int{17, 16, 15}

	// For older versions, we'll use psql15 as it should be backward compatible
	targetVersion := serverVersion.Major
	if targetVersion < 15 {
		targetVersion = 15
	}

	// First, try to find exact match
	psqlBin := fmt.Sprintf("psql%d", targetVersion)
	if _, err := exec.LookPath(psqlBin); err == nil {
		return psqlBin, nil
	}

	// If no exact match, find the closest version that's >= server version
	for _, v := range availableVersions {
		if v >= targetVersion {
			psqlBin = fmt.Sprintf("psql%d", v)
			if _, err := exec.LookPath(psqlBin); err == nil {
				return psqlBin, nil
			}
		}
	}

	// If still not found, try plain psql
	if _, err := exec.LookPath("psql"); err == nil {
		return "psql", nil
	}

	// Last resort: try the newest available version
	for _, v := range availableVersions {
		psqlBin = fmt.Sprintf("psql%d", v)
		if _, err := exec.LookPath(psqlBin); err == nil {
			return psqlBin, nil
		}
	}

	return "", fmt.Errorf("no suitable psql found for PostgreSQL %d", serverVersion.Major)
}
