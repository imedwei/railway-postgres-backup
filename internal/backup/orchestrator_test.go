package backup

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/imedwei/railway-postgres-backup/internal/config"
	"github.com/imedwei/railway-postgres-backup/internal/storage"
)

// Mock implementations for testing

type mockBackup struct {
	dumpErr   error
	dumpData  string
	infoErr   error
	info      *DatabaseInfo
	validated bool
}

func (m *mockBackup) Dump(ctx context.Context) (io.ReadCloser, error) {
	if m.dumpErr != nil {
		return nil, m.dumpErr
	}
	return io.NopCloser(strings.NewReader(m.dumpData)), nil
}

func (m *mockBackup) Validate(ctx context.Context, reader io.Reader) error {
	m.validated = true
	return nil
}

func (m *mockBackup) GetInfo(ctx context.Context) (*DatabaseInfo, error) {
	if m.infoErr != nil {
		return nil, m.infoErr
	}
	if m.info != nil {
		return m.info, nil
	}
	return &DatabaseInfo{
		Name:    "testdb",
		Size:    1024 * 1024,
		Version: "PostgreSQL 16.0",
	}, nil
}

type mockStorage struct {
	uploadErr    error
	uploadCalled bool
	uploadKey    string
	metadata     map[string]string
	lastBackup   time.Time
	listResult   []storage.ObjectInfo
	deleteCalls  []string
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	m.uploadCalled = true
	m.uploadKey = key
	m.metadata = metadata

	// Consume the reader
	_, _ = io.ReadAll(reader)

	return m.uploadErr
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.deleteCalls = append(m.deleteCalls, key)
	return nil
}

func (m *mockStorage) List(ctx context.Context, prefix string) ([]storage.ObjectInfo, error) {
	return m.listResult, nil
}

func (m *mockStorage) GetLastBackupTime(ctx context.Context) (time.Time, error) {
	return m.lastBackup, nil
}

func TestOrchestrator_Run(t *testing.T) {
	// Create logger that discards output
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name          string
		config        *config.Config
		mockBackup    *mockBackup
		mockStorage   *mockStorage
		wantErr       bool
		wantUpload    bool
		checkMetadata bool
	}{
		{
			name: "successful backup",
			config: &config.Config{
				StorageProvider:        "s3",
				BackupFilePrefix:       "test",
				RespawnProtectionHours: 6,
			},
			mockBackup: &mockBackup{
				dumpData: "backup data",
			},
			mockStorage: &mockStorage{
				lastBackup: time.Now().Add(-7 * time.Hour), // Old enough
			},
			wantErr:    false,
			wantUpload: true,
		},
		{
			name: "respawn protection blocks backup",
			config: &config.Config{
				StorageProvider:        "s3",
				BackupFilePrefix:       "test",
				RespawnProtectionHours: 6,
				ForceBackup:            false,
			},
			mockBackup: &mockBackup{
				dumpData: "backup data",
			},
			mockStorage: &mockStorage{
				lastBackup: time.Now().Add(-1 * time.Hour), // Too recent
			},
			wantErr:    false,
			wantUpload: false,
		},
		{
			name: "force backup overrides protection",
			config: &config.Config{
				StorageProvider:        "s3",
				BackupFilePrefix:       "test",
				RespawnProtectionHours: 6,
				ForceBackup:            true,
			},
			mockBackup: &mockBackup{
				dumpData: "backup data",
			},
			mockStorage: &mockStorage{
				lastBackup: time.Now().Add(-1 * time.Hour), // Too recent
			},
			wantErr:    false,
			wantUpload: true,
		},
		{
			name: "dump failure",
			config: &config.Config{
				StorageProvider:        "s3",
				RespawnProtectionHours: 6,
			},
			mockBackup: &mockBackup{
				dumpErr: errors.New("dump failed"),
			},
			mockStorage: &mockStorage{
				lastBackup: time.Time{}, // No previous backup
			},
			wantErr:    true,
			wantUpload: false,
		},
		{
			name: "upload failure",
			config: &config.Config{
				StorageProvider:        "s3",
				RespawnProtectionHours: 6,
			},
			mockBackup: &mockBackup{
				dumpData: "backup data",
			},
			mockStorage: &mockStorage{
				lastBackup: time.Time{}, // No previous backup
				uploadErr:  errors.New("upload failed"),
			},
			wantErr:    true,
			wantUpload: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := NewOrchestrator(tt.config, tt.mockStorage, tt.mockBackup, logger)

			err := orchestrator.Run(context.Background())

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.mockStorage.uploadCalled != tt.wantUpload {
				t.Errorf("Upload called = %v, want %v", tt.mockStorage.uploadCalled, tt.wantUpload)
			}

			if tt.wantUpload && tt.mockStorage.uploadCalled {
				// Check filename format
				if !strings.HasSuffix(tt.mockStorage.uploadKey, ".tar.gz") {
					t.Errorf("Upload key should end with .tar.gz, got %v", tt.mockStorage.uploadKey)
				}

				// Check metadata
				if tt.mockStorage.metadata["backup-tool"] != "railway-postgres-backup" {
					t.Errorf("Missing or incorrect backup-tool metadata")
				}
			}
		})
	}
}

func TestOrchestrator_CleanupOldBackups(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	now := time.Now()
	oldBackup1 := now.AddDate(0, 0, -10)  // 10 days old
	oldBackup2 := now.AddDate(0, 0, -8)   // 8 days old
	recentBackup := now.AddDate(0, 0, -2) // 2 days old

	mockStorage := &mockStorage{
		listResult: []storage.ObjectInfo{
			{
				Key:          "test-" + oldBackup1.Format("2006-01-02T15-04-05-000Z") + ".tar.gz",
				LastModified: oldBackup1,
			},
			{
				Key:          "test-" + oldBackup2.Format("2006-01-02T15-04-05-000Z") + ".tar.gz",
				LastModified: oldBackup2,
			},
			{
				Key:          "test-" + recentBackup.Format("2006-01-02T15-04-05-000Z") + ".tar.gz",
				LastModified: recentBackup,
			},
		},
	}

	cfg := &config.Config{
		StorageProvider:  "s3",
		BackupFilePrefix: "test",
		RetentionDays:    7, // Keep backups for 7 days
	}

	orchestrator := NewOrchestrator(cfg, mockStorage, &mockBackup{}, logger)

	err := orchestrator.cleanupOldBackups(context.Background())
	if err != nil {
		t.Fatalf("cleanupOldBackups() error = %v", err)
	}

	// Should have deleted 2 old backups
	if len(mockStorage.deleteCalls) != 2 {
		t.Errorf("Expected 2 deletions, got %d", len(mockStorage.deleteCalls))
	}

	// Check that the correct backups were deleted
	deletedKeys := make(map[string]bool)
	for _, key := range mockStorage.deleteCalls {
		deletedKeys[key] = true
	}

	if !deletedKeys[mockStorage.listResult[0].Key] {
		t.Errorf("Expected oldest backup to be deleted")
	}
	if !deletedKeys[mockStorage.listResult[1].Key] {
		t.Errorf("Expected 8-day old backup to be deleted")
	}
	if deletedKeys[mockStorage.listResult[2].Key] {
		t.Errorf("Recent backup should not be deleted")
	}
}

func TestNewOrchestrator(t *testing.T) {
	cfg := &config.Config{
		StorageProvider:        "s3",
		RespawnProtectionHours: 6,
		ForceBackup:            false,
	}

	storage := &mockStorage{}
	backup := &mockBackup{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	orchestrator := NewOrchestrator(cfg, storage, backup, logger)

	if orchestrator == nil {
		t.Fatal("NewOrchestrator returned nil")
	}

	if orchestrator.config != cfg {
		t.Error("Config not set correctly")
	}

	if orchestrator.storage != storage {
		t.Error("Storage not set correctly")
	}

	if orchestrator.backup != backup {
		t.Error("Backup not set correctly")
	}

	if orchestrator.rateLimiter == nil {
		t.Error("Rate limiter not initialized")
	}
}
