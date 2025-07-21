package utils

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateBackupFilename(t *testing.T) {
	timestamp := time.Date(2025, 1, 21, 10, 30, 45, 123000000, time.UTC)

	tests := []struct {
		name      string
		prefix    string
		pgVersion string
		want      string
	}{
		{
			name:      "with prefix and version",
			prefix:    "postgres",
			pgVersion: "PostgreSQL 15.2",
			want:      "postgres-pg15-2025-01-21T10-30-45-123Z.tar.gz",
		},
		{
			name:      "without prefix",
			prefix:    "",
			pgVersion: "PostgreSQL 16.1",
			want:      "backup-pg16-2025-01-21T10-30-45-123Z.tar.gz",
		},
		{
			name:      "prefix with trailing dash",
			prefix:    "postgres-",
			pgVersion: "PostgreSQL 17.0",
			want:      "postgres-pg17-2025-01-21T10-30-45-123Z.tar.gz",
		},
		{
			name:      "complex prefix",
			prefix:    "my-database-backup",
			pgVersion: "PostgreSQL 15.4",
			want:      "my-database-backup-pg15-2025-01-21T10-30-45-123Z.tar.gz",
		},
		{
			name:      "unknown version",
			prefix:    "test",
			pgVersion: "unknown",
			want:      "test-pgunknown-2025-01-21T10-30-45-123Z.tar.gz",
		},
		{
			name:      "empty version",
			prefix:    "test",
			pgVersion: "",
			want:      "test-pgunknown-2025-01-21T10-30-45-123Z.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateBackupFilename(tt.prefix, timestamp, tt.pgVersion)
			if got != tt.want {
				t.Errorf("GenerateBackupFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseBackupFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     time.Time
		wantErr  bool
	}{
		{
			name:     "valid with prefix",
			filename: "postgres-pg15-2025-01-21T10-30-45-123Z.tar.gz",
			want:     time.Date(2025, 1, 21, 10, 30, 45, 123000000, time.UTC),
			wantErr:  false,
		},
		{
			name:     "valid without prefix",
			filename: "backup-pg16-2025-01-21T10-30-45-123Z.tar.gz",
			want:     time.Date(2025, 1, 21, 10, 30, 45, 123000000, time.UTC),
			wantErr:  false,
		},
		{
			name:     "too short",
			filename: "backup.tar.gz",
			want:     time.Time{},
			wantErr:  true,
		},
		{
			name:     "invalid timestamp",
			filename: "backup-invalid-timestamp-here.tar.gz",
			want:     time.Time{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBackupFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseBackupFilename() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !got.Equal(tt.want) {
				t.Errorf("ParseBackupFilename() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that generate and parse are inverse operations
	prefixes := []string{"", "backup", "postgres-db", "my-app"}

	for _, prefix := range prefixes {
		t.Run("prefix="+prefix, func(t *testing.T) {
			original := time.Now().UTC().Truncate(time.Millisecond)
			filename := GenerateBackupFilename(prefix, original, "PostgreSQL 15.2")

			parsed, err := ParseBackupFilename(filename)
			if err != nil {
				t.Fatalf("Failed to parse generated filename: %v", err)
			}

			// Compare with millisecond precision
			if !parsed.Equal(original) {
				t.Errorf("Round trip failed: original=%v, parsed=%v", original, parsed)
			}
		})
	}
}

func TestGenerateBackupFilename_Format(t *testing.T) {
	// Test that the generated filename follows expected format
	timestamp := time.Now()
	filename := GenerateBackupFilename("test", timestamp, "PostgreSQL 16.1")

	// Should end with .tar.gz
	if !strings.HasSuffix(filename, ".tar.gz") {
		t.Errorf("Filename should end with .tar.gz, got: %s", filename)
	}

	// Should contain no colons (filesystem compatibility)
	if strings.Contains(filename, ":") {
		t.Errorf("Filename should not contain colons, got: %s", filename)
	}

	// Should start with prefix
	if !strings.HasPrefix(filename, "test-") {
		t.Errorf("Filename should start with prefix, got: %s", filename)
	}
}
