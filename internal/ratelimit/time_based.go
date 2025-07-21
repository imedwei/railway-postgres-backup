package ratelimit

import (
	"fmt"
	"time"
)

// TimeBasedLimiter implements RateLimiter with time-based rate limiting.
type TimeBasedLimiter struct {
	config Config
}

// NewTimeBasedLimiter creates a new time-based rate limiter.
func NewTimeBasedLimiter(config Config) *TimeBasedLimiter {
	return &TimeBasedLimiter{
		config: config,
	}
}

// ShouldBackup implements RateLimiter.
func (t *TimeBasedLimiter) ShouldBackup(lastBackup time.Time) (bool, string) {
	if t.config.ForceBackup {
		return true, "forced backup requested"
	}

	if lastBackup.IsZero() {
		return true, "no previous backup found"
	}

	timeSinceLastBackup := time.Since(lastBackup)
	if timeSinceLastBackup < t.config.MinInterval {
		timeUntilNextBackup := t.config.MinInterval - timeSinceLastBackup
		return false, fmt.Sprintf(
			"last backup was %s ago, next backup allowed in %s",
			formatDuration(timeSinceLastBackup),
			formatDuration(timeUntilNextBackup),
		)
	}

	return true, fmt.Sprintf("last backup was %s ago", formatDuration(timeSinceLastBackup))
}

// GetMinInterval implements RateLimiter.
func (t *TimeBasedLimiter) GetMinInterval() time.Duration {
	return t.config.MinInterval
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	}
	return fmt.Sprintf("%.1f hours", d.Hours())
}
