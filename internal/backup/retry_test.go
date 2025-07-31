package backup

import (
	"errors"
	"os/exec"
	"testing"
	"time"
)

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "exec.ExitError with database starting up",
			err:      &exec.ExitError{Stderr: []byte("FATAL: the database system is starting up")},
			expected: true,
		},
		{
			name:     "exec.ExitError with SQLSTATE 57P03",
			err:      &exec.ExitError{Stderr: []byte("ERROR: SQLSTATE 57P03")},
			expected: true,
		},
		{
			name:     "exec.ExitError with connection refused",
			err:      &exec.ExitError{Stderr: []byte("psql: error: could not connect to server: Connection refused")},
			expected: true,
		},
		{
			name:     "exec.ExitError with no such host",
			err:      &exec.ExitError{Stderr: []byte("psql: error: could not translate host name to address: no such host")},
			expected: true,
		},
		{
			name:     "exec.ExitError with timeout",
			err:      &exec.ExitError{Stderr: []byte("psql: error: connection to server at \"localhost\", port 5432 failed: timeout expired")},
			expected: true,
		},
		{
			name:     "exec.ExitError with authentication failure",
			err:      &exec.ExitError{Stderr: []byte("psql: error: password authentication failed")},
			expected: false,
		},
		{
			name:     "regular error with connection refused",
			err:      errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"),
			expected: true,
		},
		{
			name:     "regular error with no retry pattern",
			err:      errors.New("syntax error at or near"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestDefaultPSQLRetryConfig(t *testing.T) {
	config := defaultPSQLRetryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries to be 5, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 2*time.Second {
		t.Errorf("expected InitialDelay to be 2s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay to be 30s, got %v", config.MaxDelay)
	}
	if config.BackoffFactor != 2.0 {
		t.Errorf("expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}
}
