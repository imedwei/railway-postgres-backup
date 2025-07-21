// Package utils provides utility functions for the backup service.
package utils

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// ConnectionPool manages database connections.
type ConnectionPool struct {
	db *sql.DB
}

// NewConnectionPool creates a new connection pool from a database URL.
func NewConnectionPool(databaseURL string) (*ConnectionPool, error) {
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

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &ConnectionPool{db: db}, nil
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
