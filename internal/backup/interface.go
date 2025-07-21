// Package backup defines the interface for database backup operations.
package backup

import (
	"context"
	"io"
)

// Backup defines the interface for database backup operations.
type Backup interface {
	// Dump creates a backup of the database and returns a reader for the backup data.
	Dump(ctx context.Context) (io.ReadCloser, error)

	// Validate checks if a backup file is valid.
	Validate(ctx context.Context, reader io.Reader) error

	// GetInfo returns information about the database being backed up.
	GetInfo(ctx context.Context) (*DatabaseInfo, error)
}

// DatabaseInfo contains information about the database.
type DatabaseInfo struct {
	Name    string
	Size    int64
	Version string
}
