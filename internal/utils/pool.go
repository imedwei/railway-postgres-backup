// Package utils provides utility functions for the backup service.
package utils

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionPool manages database connections.
type ConnectionPool struct {
	db *sql.DB
}

// RetryConfig holds configuration for database connection retries
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay between retries
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Exponential backoff factor
}

// DefaultRetryConfig returns the default retry configuration
// Can be overridden with environment variables:
// - DB_RETRY_MAX_ATTEMPTS: Maximum number of retry attempts (default: 10)
// - DB_RETRY_INITIAL_DELAY: Initial delay in seconds (default: 2)
// - DB_RETRY_MAX_DELAY: Maximum delay in seconds (default: 60)
// - DB_RETRY_BACKOFF_FACTOR: Exponential backoff factor (default: 2.0)
func DefaultRetryConfig() RetryConfig {
	config := RetryConfig{
		MaxRetries:    10,               // Allow up to 10 retries
		InitialDelay:  2 * time.Second,  // Start with 2 second delay
		MaxDelay:      60 * time.Second, // Cap at 60 seconds
		BackoffFactor: 2.0,              // Double the delay each time
	}

	// Override with environment variables if set
	if maxRetries := os.Getenv("DB_RETRY_MAX_ATTEMPTS"); maxRetries != "" {
		if val, err := strconv.Atoi(maxRetries); err == nil && val > 0 {
			config.MaxRetries = val
		}
	}

	if initialDelay := os.Getenv("DB_RETRY_INITIAL_DELAY"); initialDelay != "" {
		if val, err := strconv.Atoi(initialDelay); err == nil && val > 0 {
			config.InitialDelay = time.Duration(val) * time.Second
		}
	}

	if maxDelay := os.Getenv("DB_RETRY_MAX_DELAY"); maxDelay != "" {
		if val, err := strconv.Atoi(maxDelay); err == nil && val > 0 {
			config.MaxDelay = time.Duration(val) * time.Second
		}
	}

	if backoffFactor := os.Getenv("DB_RETRY_BACKOFF_FACTOR"); backoffFactor != "" {
		if val, err := strconv.ParseFloat(backoffFactor, 64); err == nil && val > 1.0 {
			config.BackoffFactor = val
		}
	}

	return config
}

// NewConnectionPool creates a new connection pool from a database URL with default retry configuration.
func NewConnectionPool(databaseURL string) (*ConnectionPool, error) {
	return NewConnectionPoolWithRetry(context.Background(), databaseURL, DefaultRetryConfig())
}

// NewConnectionPoolWithRetry creates a new connection pool from a database URL with retry logic.
func NewConnectionPoolWithRetry(ctx context.Context, databaseURL string, retryConfig RetryConfig) (*ConnectionPool, error) {
	logger := slog.Default().With("component", "connection-pool")

	var attemptErrors []string
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			logger.Info("Retrying database connection",
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

		pool, err := tryDatabaseConnection(ctx, databaseURL)
		if err == nil {
			if attempt > 0 {
				logger.Info("Successfully connected to database",
					"attempts", attempt+1)
			}
			return pool, nil
		}

		// Record the error for this attempt
		attemptErrors = append(attemptErrors, fmt.Sprintf("attempt %d: %v", attempt+1, err))

		// Check if this is a cold boot error
		if isColdBootError(err) {
			logger.Warn("Database appears to be cold booting",
				"attempt", attempt+1,
				"error", err)
		} else {
			logger.Error("Failed to connect to database",
				"attempt", attempt+1,
				"error", err)
		}
	}

	return nil, fmt.Errorf("all database connection attempts failed after %d retries (errors: %v)",
		retryConfig.MaxRetries, attemptErrors)
}

// tryDatabaseConnection attempts to connect to the database with the given URL
func tryDatabaseConnection(ctx context.Context, databaseURL string) (*ConnectionPool, error) {
	// Parse URL to add connection pool parameters
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	// Add connection pool parameters
	q := u.Query()
	if q.Get("sslmode") == "" {
		q.Set("sslmode", "require")
	}
	if q.Get("connect_timeout") == "" {
		q.Set("connect_timeout", "10")
	}
	u.RawQuery = q.Encode()

	// Open database connection
	db, err := sql.Open("postgres", u.String())
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	// Test connection with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &ConnectionPool{db: db}, nil
}

// isColdBootError checks if the error indicates the database is still starting up
func isColdBootError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Check for common cold boot error messages
	return strings.Contains(errStr, "the database system is starting up") ||
		strings.Contains(errStr, "SQLSTATE 57P03") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "ECONNREFUSED") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "dial tcp")
}

// GetDatabaseInfo retrieves database information.
func (p *ConnectionPool) GetDatabaseInfo() (*DatabaseInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info := &DatabaseInfo{}

	// Get database name and version
	err := p.db.QueryRowContext(ctx, `
		SELECT current_database(), version()
	`).Scan(&info.Name, &info.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	// Get database size
	err = p.db.QueryRowContext(ctx, `
		SELECT pg_database_size(current_database())
	`).Scan(&info.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}

	return info, nil
}

// Close closes the connection pool.
func (p *ConnectionPool) Close() error {
	return p.db.Close()
}

// DatabaseInfo holds database metadata.
type DatabaseInfo struct {
	Name    string
	Version string
	Size    int64
}
