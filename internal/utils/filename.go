// Package utils provides utility functions for the backup service.
package utils

import (
	"fmt"
	"strings"
	"time"
)

// GenerateBackupFilename creates a timestamped backup filename.
func GenerateBackupFilename(prefix string, timestamp time.Time) string {
	// Format: prefix-2006-01-02T15-04-05-000Z.tar.gz
	// Using dashes instead of colons for better filesystem compatibility
	// Format milliseconds manually to ensure 3 digits
	t := timestamp.UTC()
	ms := t.Nanosecond() / 1000000
	timeStr := fmt.Sprintf("%s-%03dZ", t.Format("2006-01-02T15-04-05"), ms)

	if prefix != "" {
		// Ensure prefix doesn't end with dash
		prefix = strings.TrimSuffix(prefix, "-")
		return fmt.Sprintf("%s-%s.tar.gz", prefix, timeStr)
	}

	return fmt.Sprintf("backup-%s.tar.gz", timeStr)
}

// ParseBackupFilename extracts the timestamp from a backup filename.
func ParseBackupFilename(filename string) (time.Time, error) {
	// Remove .tar.gz extension
	name := strings.TrimSuffix(filename, ".tar.gz")

	// Find the timestamp part (last 24 characters: 2006-01-02T15-04-05-000Z)
	if len(name) < 24 {
		return time.Time{}, fmt.Errorf("filename too short to contain timestamp")
	}

	timeStr := name[len(name)-24:]

	// Parse the custom format with milliseconds
	// Split the milliseconds part
	if len(timeStr) != 24 || !strings.HasSuffix(timeStr, "Z") {
		return time.Time{}, fmt.Errorf("invalid timestamp format")
	}

	// Extract parts
	datePart := timeStr[:19] // 2006-01-02T15-04-05
	msPart := timeStr[20:23] // 000

	// Parse milliseconds
	var ms int
	fmt.Sscanf(msPart, "%d", &ms)

	// Parse base time
	t, err := time.Parse("2006-01-02T15-04-05", datePart)
	if err != nil {
		return time.Time{}, err
	}

	// Add milliseconds
	return t.Add(time.Duration(ms) * time.Millisecond).UTC(), nil
}
