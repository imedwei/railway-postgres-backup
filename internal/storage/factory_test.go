package storage

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

// mockStorage is a mock implementation for testing retry logic.
type mockStorage struct {
	uploadCalls int
	uploadErr   error
	deleteCalls int
	deleteErr   error
	listCalls   int
	listErr     error
	listResult  []ObjectInfo
	timeCalls   int
	timeErr     error
	timeResult  time.Time
}

func (m *mockStorage) Upload(ctx context.Context, key string, reader io.Reader, metadata map[string]string) error {
	m.uploadCalls++
	return m.uploadErr
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	m.deleteCalls++
	return m.deleteErr
}

func (m *mockStorage) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	m.listCalls++
	return m.listResult, m.listErr
}

func (m *mockStorage) GetLastBackupTime(ctx context.Context) (time.Time, error) {
	m.timeCalls++
	return m.timeResult, m.timeErr
}

func TestRetryableStorage_Upload(t *testing.T) {
	tests := []struct {
		name        string
		uploadErr   error
		maxAttempts int
		wantCalls   int
		wantErr     bool
	}{
		{
			name:        "success on first attempt",
			uploadErr:   nil,
			maxAttempts: 3,
			wantCalls:   1,
			wantErr:     false,
		},
		{
			name:        "failure after max attempts",
			uploadErr:   errors.New("upload failed"),
			maxAttempts: 3,
			wantCalls:   3,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockStorage{uploadErr: tt.uploadErr}
			config := RetryConfig{
				MaxAttempts:  tt.maxAttempts,
				InitialDelay: 1 * time.Millisecond,
				MaxDelay:     10 * time.Millisecond,
				Multiplier:   2.0,
			}
			retryable := NewRetryableStorage(mock, config)

			err := retryable.Upload(context.Background(), "test.tar.gz", nil, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("Upload() error = %v, wantErr %v", err, tt.wantErr)
			}
			if mock.uploadCalls != tt.wantCalls {
				t.Errorf("Upload() calls = %v, want %v", mock.uploadCalls, tt.wantCalls)
			}
		})
	}
}

func TestRetryableStorage_ContextCancellation(t *testing.T) {
	mock := &mockStorage{uploadErr: errors.New("upload failed")}
	config := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}
	retryable := NewRetryableStorage(mock, config)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := retryable.Upload(ctx, "test.tar.gz", nil, nil)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}
	// Should have attempted at least once but not max attempts
	if mock.uploadCalls >= config.MaxAttempts {
		t.Errorf("Upload() should have been cancelled, but made %v calls", mock.uploadCalls)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("MaxAttempts = %v, want 3", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, want 1s", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay = %v, want 10s", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", cfg.Multiplier)
	}
}
