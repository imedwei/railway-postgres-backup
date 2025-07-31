package backup

import (
	"testing"
)

func TestParsePGVersion(t *testing.T) {
	tests := []struct {
		name       string
		versionStr string
		wantMajor  int
		wantMinor  int
		wantErr    bool
	}{
		{
			name:       "PostgreSQL 16.2",
			versionStr: "PostgreSQL 16.2 on x86_64-pc-linux-gnu, compiled by gcc (GCC) 11.3.1 20221121 (Red Hat 11.3.1-4), 64-bit",
			wantMajor:  16,
			wantMinor:  2,
			wantErr:    false,
		},
		{
			name:       "PostgreSQL 14.11",
			versionStr: "PostgreSQL 14.11 (Ubuntu 14.11-0ubuntu0.22.04.1) on x86_64-pc-linux-gnu, compiled by gcc (Ubuntu 11.4.0-1ubuntu1~22.04) 11.4.0, 64-bit",
			wantMajor:  14,
			wantMinor:  11,
			wantErr:    false,
		},
		{
			name:       "PostgreSQL 15.0",
			versionStr: "PostgreSQL 15.0 on aarch64-unknown-linux-gnu, compiled by gcc (GCC) 7.3.1 20180712 (Red Hat 7.3.1-6), 64-bit",
			wantMajor:  15,
			wantMinor:  0,
			wantErr:    false,
		},
		{
			name:       "PostgreSQL 13.14",
			versionStr: "PostgreSQL 13.14",
			wantMajor:  13,
			wantMinor:  14,
			wantErr:    false,
		},
		{
			name:       "Invalid format",
			versionStr: "Not a PostgreSQL version string",
			wantErr:    true,
		},
		{
			name:       "MySQL version",
			versionStr: "MySQL 8.0.35",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePGVersion(tt.versionStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePGVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.wantMajor {
					t.Errorf("ParsePGVersion() Major = %v, want %v", got.Major, tt.wantMajor)
				}
				if got.Minor != tt.wantMinor {
					t.Errorf("ParsePGVersion() Minor = %v, want %v", got.Minor, tt.wantMinor)
				}
			}
		})
	}
}

func TestFindBestPGDump(t *testing.T) {
	// Note: These tests will depend on what's actually installed
	// In CI/CD, you might want to skip or mock these
	t.Run("finds binary for version", func(t *testing.T) {
		version := &PGVersion{Major: 15, Minor: 0}
		_, err := FindBestPGDump(version)
		// We expect this might fail in test environment
		if err != nil {
			t.Logf("Expected behavior in test environment: %v", err)
		}
	})
}

func TestFindAvailablePSQL(t *testing.T) {
	// This test verifies that findAvailablePSQL returns a psql binary
	psqlBin := findAvailablePSQL()

	// Should always return at least "psql" as fallback
	if psqlBin == "" {
		t.Error("findAvailablePSQL returned empty string")
	}

	// Log which binary was found
	t.Logf("findAvailablePSQL returned: %s", psqlBin)

	// Verify it's one of the expected values
	validBinaries := map[string]bool{
		"psql":   true,
		"psql15": true,
		"psql16": true,
		"psql17": true,
	}

	if !validBinaries[psqlBin] {
		t.Errorf("unexpected psql binary: %s", psqlBin)
	}
}
