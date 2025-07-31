package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"strings"
	"testing"
)

func TestNewPostgresBackup(t *testing.T) {
	tests := []struct {
		name          string
		connectionURL string
		pgDumpOptions string
		wantOptions   []string
	}{
		{
			name:          "no options",
			connectionURL: "postgres://localhost/test",
			pgDumpOptions: "",
			wantOptions:   []string{},
		},
		{
			name:          "with options",
			connectionURL: "postgres://localhost/test",
			pgDumpOptions: "--schema=public --exclude-table=logs",
			wantOptions:   []string{"--schema=public", "--exclude-table=logs"},
		},
		{
			name:          "with multiple spaces",
			connectionURL: "postgres://localhost/test",
			pgDumpOptions: "  --schema=public   --exclude-table=logs  ",
			wantOptions:   []string{"--schema=public", "--exclude-table=logs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := NewPostgresBackup(tt.connectionURL, tt.pgDumpOptions)

			if pb.connectionURL != tt.connectionURL {
				t.Errorf("connectionURL = %v, want %v", pb.connectionURL, tt.connectionURL)
			}

			if len(pb.pgDumpOptions) != len(tt.wantOptions) {
				t.Errorf("pgDumpOptions length = %v, want %v", len(pb.pgDumpOptions), len(tt.wantOptions))
				return
			}

			for i, opt := range pb.pgDumpOptions {
				if opt != tt.wantOptions[i] {
					t.Errorf("pgDumpOptions[%d] = %v, want %v", i, opt, tt.wantOptions[i])
				}
			}
			
			// Verify psqlBin is set (should be set even before version detection)
			if pb.psqlBin == "" {
				t.Error("psqlBin is empty")
			}
			
			// psqlBin should be one of the valid binaries
			validPSQLBinaries := map[string]bool{
				"psql":   true,
				"psql15": true,
				"psql16": true,
				"psql17": true,
			}
			if !validPSQLBinaries[pb.psqlBin] {
				t.Errorf("unexpected psqlBin: %s", pb.psqlBin)
			}
			
			// pgDumpBin should also be set (either versioned or default)
			if pb.pgDumpBin == "" {
				t.Error("pgDumpBin is empty")
			}
		})
	}
}

func TestPostgresBackup_Validate(t *testing.T) {
	pb := &PostgresBackup{}

	tests := []struct {
		name    string
		data    func() io.Reader
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid tar.gz",
			data: func() io.Reader {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)

				// Add a file
				hdr := &tar.Header{
					Name: "test.sql",
					Mode: 0600,
					Size: 12,
				}
				_ = tw.WriteHeader(hdr)
				_, _ = tw.Write([]byte("SELECT 1;\n"))

				_ = tw.Close()
				_ = gw.Close()

				return &buf
			},
			wantErr: false,
		},
		{
			name: "empty tar.gz",
			data: func() io.Reader {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				tw := tar.NewWriter(gw)
				_ = tw.Close()
				_ = gw.Close()
				return &buf
			},
			wantErr: true,
			errMsg:  "empty",
		},
		{
			name: "invalid gzip",
			data: func() io.Reader {
				return strings.NewReader("not a gzip file")
			},
			wantErr: true,
			errMsg:  "gzip",
		},
		{
			name: "invalid tar",
			data: func() io.Reader {
				var buf bytes.Buffer
				gw := gzip.NewWriter(&buf)
				_, _ = gw.Write([]byte("not a tar file"))
				_ = gw.Close()
				return &buf
			},
			wantErr: true,
			errMsg:  "tar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pb.Validate(context.Background(), tt.data())

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// Integration tests would require a real PostgreSQL instance
func TestPostgresBackup_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test would require:
	// 1. A running PostgreSQL instance
	// 2. Valid connection URL
	// 3. pg_dump and psql binaries available

	// Example:
	// pb := NewPostgresBackup("postgres://user:pass@localhost/testdb", "")
	//
	// reader, err := pb.Dump(context.Background())
	// if err != nil {
	//     t.Fatal(err)
	// }
	// defer reader.Close()
	//
	// // Validate the backup
	// data, _ := io.ReadAll(reader)
	// err = pb.Validate(context.Background(), bytes.NewReader(data))
	// if err != nil {
	//     t.Fatal(err)
	// }
}
