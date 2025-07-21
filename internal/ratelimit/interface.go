// Package ratelimit provides respawn protection for backup operations.
package ratelimit

import (
	"time"
)

// RateLimiter defines the interface for controlling backup frequency.
type RateLimiter interface {
	// ShouldBackup determines if a backup should proceed based on the last backup time.
	// Returns true if backup should proceed, false otherwise.
	// The string return value contains a human-readable reason when backup is skipped.
	ShouldBackup(lastBackup time.Time) (bool, string)

	// GetMinInterval returns the minimum time interval between backups.
	GetMinInterval() time.Duration
}

// Config holds configuration for rate limiting.
type Config struct {
	// MinInterval is the minimum time between backups.
	MinInterval time.Duration

	// ForceBackup overrides rate limiting when true.
	ForceBackup bool
}