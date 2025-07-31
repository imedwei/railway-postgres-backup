package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"
)

// PostgresBackup implements the Backup interface for PostgreSQL databases.
type PostgresBackup struct {
	connectionURL string
	pgDumpOptions []string
	pgDumpBin     string
	psqlBin       string
	logger        *slog.Logger
}

// NewPostgresBackup creates a new PostgreSQL backup instance.
func NewPostgresBackup(connectionURL string, pgDumpOptions string) *PostgresBackup {
	// Parse pg_dump options from string
	var options []string
	if pgDumpOptions != "" {
		// Simple parsing - could be improved to handle quoted arguments
		options = strings.Fields(pgDumpOptions)
	}

	logger := slog.Default().With("component", "postgres-backup")

	// First, find an available psql binary for version detection
	availablePSQL := findAvailablePSQL()

	pb := &PostgresBackup{
		connectionURL: connectionURL,
		pgDumpOptions: options,
		logger:        logger,
		psqlBin:       availablePSQL, // Set initial psql binary
	}

	// Try to detect PostgreSQL version and find appropriate binaries
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if version, err := GetServerVersion(ctx, connectionURL); err == nil {
		logger.Info("Detected PostgreSQL version", "version", version.Full, "major", version.Major)

		if pgDumpBin, err := FindBestPGDump(version); err == nil {
			pb.pgDumpBin = pgDumpBin
			logger.Info("Selected pg_dump binary", "binary", pgDumpBin)
		}

		// Try to find a better psql binary based on the detected version
		if psqlBin, err := FindBestPSQL(version); err == nil {
			pb.psqlBin = psqlBin
			logger.Info("Selected psql binary", "binary", psqlBin)
		}
	} else {
		logger.Warn("Could not detect PostgreSQL version, using default binaries", "error", err)
	}

	// Fallback to default binaries if not set
	if pb.pgDumpBin == "" {
		pb.pgDumpBin = "pg_dump"
	}
	// psqlBin is already set from findAvailablePSQL()

	return pb
}

// Dump creates a backup of the PostgreSQL database.
func (p *PostgresBackup) Dump(ctx context.Context) (io.ReadCloser, error) {
	// Build pg_dump command
	args := []string{
		"--format=tar",
		"--verbose",
		"--no-password",
	}

	// Add custom options
	args = append(args, p.pgDumpOptions...)

	// Add connection URL last
	args = append(args, p.connectionURL)

	// Create command with the appropriate pg_dump binary
	cmd := exec.CommandContext(ctx, p.pgDumpBin, args...)

	// Set environment to avoid password prompts
	cmd.Env = append(os.Environ(), "PGPASSWORD=")

	// Get stdout pipe
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	// Get stderr for error messages
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pg_dump: %w", err)
	}

	// Create a pipe for gzip compression
	pr, pw := io.Pipe()

	// Start a goroutine to compress the output
	go func() {
		// Create gzip writer
		gw := gzip.NewWriter(pw)

		// Copy from pg_dump to gzip
		_, copyErr := io.Copy(gw, stdout)

		// Close gzip writer
		if closeErr := gw.Close(); closeErr != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to close gzip writer: %w", closeErr))
			return
		}

		// Wait for pg_dump to finish
		waitErr := cmd.Wait()

		// Close the pipe writer with appropriate error
		if copyErr != nil {
			_ = pw.CloseWithError(fmt.Errorf("failed to compress backup: %w", copyErr))
		} else if waitErr != nil {
			_ = pw.CloseWithError(fmt.Errorf("pg_dump failed: %w, stderr: %s", waitErr, stderr.String()))
		} else {
			_ = pw.Close()
		}
	}()

	return pr, nil
}

// Validate checks if a backup file is valid.
func (p *PostgresBackup) Validate(ctx context.Context, reader io.Reader) error {
	// Create gzip reader
	gr, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("invalid gzip format: %w", err)
	}
	defer func() {
		_ = gr.Close()
	}()

	// Create tar reader
	tr := tar.NewReader(gr)

	// Check if we can read at least one entry
	_, err = tr.Next()
	if err != nil {
		if err == io.EOF {
			return fmt.Errorf("backup archive is empty")
		}
		return fmt.Errorf("invalid tar format: %w", err)
	}

	// TODO: Could add more validation here, such as:
	// - Checking for specific PostgreSQL backup files
	// - Validating the structure of the backup
	// - Checking file sizes

	return nil
}

// GetInfo returns information about the database with retry logic.
func (p *PostgresBackup) GetInfo(ctx context.Context) (*DatabaseInfo, error) {
	return p.GetInfoWithRetry(ctx, defaultPSQLRetryConfig())
}

// GetInfoWithRetry returns information about the database with configurable retry logic.
func (p *PostgresBackup) GetInfoWithRetry(ctx context.Context, retryConfig RetryConfig) (*DatabaseInfo, error) {
	// Query to get database information
	query := `
		SELECT 
			current_database() as name,
			pg_database_size(current_database()) as size,
			version() as version
	`

	var attemptErrors []string
	delay := retryConfig.InitialDelay

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			p.logger.Info("Retrying database info query",
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

		// Use psql to execute the query
		cmd := exec.CommandContext(ctx, p.psqlBin,
			"--no-password",
			"--tuples-only",
			"--no-align",
			"--field-separator=|",
			"--command", query,
			p.connectionURL,
		)

		// Set environment
		cmd.Env = append(os.Environ(), "PGPASSWORD=")

		// Capture stderr for better error messages
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		// Execute command
		output, err := cmd.Output()
		if err == nil {
			// Parse output
			parts := strings.Split(strings.TrimSpace(string(output)), "|")
			if len(parts) != 3 {
				err = fmt.Errorf("unexpected output format from psql: %s", string(output))
			} else {
				// Parse size
				var size int64
				_, _ = fmt.Sscanf(parts[1], "%d", &size)

				if attempt > 0 {
					p.logger.Info("Successfully retrieved database info",
						"attempts", attempt+1)
				}

				return &DatabaseInfo{
					Name:    strings.TrimSpace(parts[0]),
					Size:    size,
					Version: strings.TrimSpace(parts[2]),
				}, nil
			}
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			// Add stderr to the error for better debugging
			exitErr.Stderr = stderr.Bytes()
		}

		// Record the error for this attempt
		attemptErrors = append(attemptErrors, fmt.Sprintf("attempt %d: %v (stderr: %s)", attempt+1, err, stderr.String()))

		// Check if this is a connection error that we should retry
		if isRetryableError(err) {
			p.logger.Warn("Retryable error encountered",
				"attempt", attempt+1,
				"error", err,
				"stderr", stderr.String())
		} else {
			// If it's not retryable, return immediately
			return nil, fmt.Errorf("non-retryable error: %w (stderr: %s)", err, stderr.String())
		}
	}

	return nil, fmt.Errorf("failed to get database info after %d retries (errors: %v)",
		retryConfig.MaxRetries, attemptErrors)
}
