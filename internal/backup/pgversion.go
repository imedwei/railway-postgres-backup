package backup

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// PGVersion represents a PostgreSQL version
type PGVersion struct {
	Major int
	Minor int
	Full  string
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

// GetServerVersion gets the PostgreSQL server version
func GetServerVersion(ctx context.Context, connectionURL string) (*PGVersion, error) {
	cmd := exec.CommandContext(ctx, "psql",
		"--no-password",
		"--tuples-only",
		"--no-align",
		"--command", "SELECT version();",
		connectionURL,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	versionStr := strings.TrimSpace(string(output))
	return ParsePGVersion(versionStr)
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