// Package storage defines the interface for backup storage providers.
package storage

import (
	"context"
	"io"
	"time"
)

// Storage defines the interface for backup storage operations.
type Storage interface {
	// Upload stores a backup file with the given key.
	Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error

	// Delete removes a backup file with the given key.
	Delete(ctx context.Context, key string) error

	// List returns all backup files matching the given prefix.
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)

	// GetLastBackupTime retrieves the timestamp of the most recent backup.
	GetLastBackupTime(ctx context.Context) (time.Time, error)
}

// ObjectInfo contains information about a stored backup.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	Metadata     map[string]string
}
