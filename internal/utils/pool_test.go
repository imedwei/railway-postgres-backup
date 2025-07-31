package utils

import (
	"errors"
	"os"
	"testing"
	"time"
)

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 10 {
		t.Errorf("expected MaxRetries to be 10, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 2*time.Second {
		t.Errorf("expected InitialDelay to be 2s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 60*time.Second {
		t.Errorf("expected MaxDelay to be 60s, got %v", config.MaxDelay)
	}
	if config.BackoffFactor != 2.0 {
		t.Errorf("expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}
}

func TestIsColdBootError(t *testing.T) {
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
			name:     "database starting up error",
			err:      errors.New("FATAL: the database system is starting up"),
			expected: true,
		},
		{
			name:     "SQLSTATE 57P03 error",
			err:      errors.New("ERROR: SQLSTATE 57P03"),
			expected: true,
		},
		{
			name:     "connection refused error",
			err:      errors.New("dial tcp 127.0.0.1:5432: connect: connection refused"),
			expected: true,
		},
		{
			name:     "no such host error",
			err:      errors.New("dial tcp: lookup postgres.railway.internal: no such host"),
			expected: true,
		},
		{
			name:     "ECONNREFUSED error",
			err:      errors.New("connect: ECONNREFUSED"),
			expected: true,
		},
		{
			name:     "timeout error",
			err:      errors.New("context deadline exceeded (timeout)"),
			expected: true,
		},
		{
			name:     "regular error",
			err:      errors.New("syntax error"),
			expected: false,
		},
		{
			name:     "authentication error",
			err:      errors.New("password authentication failed"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isColdBootError(tt.err)
			if result != tt.expected {
				t.Errorf("isColdBootError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestRetryConfigFromEnvironment(t *testing.T) {
	// Save original environment values
	originalMaxRetries := getEnv("DB_RETRY_MAX_ATTEMPTS")
	originalInitialDelay := getEnv("DB_RETRY_INITIAL_DELAY")
	originalMaxDelay := getEnv("DB_RETRY_MAX_DELAY")
	originalBackoffFactor := getEnv("DB_RETRY_BACKOFF_FACTOR")

	// Clean up environment after test
	defer func() {
		setEnv("DB_RETRY_MAX_ATTEMPTS", originalMaxRetries)
		setEnv("DB_RETRY_INITIAL_DELAY", originalInitialDelay)
		setEnv("DB_RETRY_MAX_DELAY", originalMaxDelay)
		setEnv("DB_RETRY_BACKOFF_FACTOR", originalBackoffFactor)
	}()

	// Set test environment values
	t.Setenv("DB_RETRY_MAX_ATTEMPTS", "5")
	t.Setenv("DB_RETRY_INITIAL_DELAY", "1")
	t.Setenv("DB_RETRY_MAX_DELAY", "30")
	t.Setenv("DB_RETRY_BACKOFF_FACTOR", "1.5")

	config := DefaultRetryConfig()

	if config.MaxRetries != 5 {
		t.Errorf("expected MaxRetries to be 5, got %d", config.MaxRetries)
	}
	if config.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay to be 1s, got %v", config.InitialDelay)
	}
	if config.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay to be 30s, got %v", config.MaxDelay)
	}
	if config.BackoffFactor != 1.5 {
		t.Errorf("expected BackoffFactor to be 1.5, got %f", config.BackoffFactor)
	}
}

// Helper functions for environment variable management
func getEnv(key string) string {
	val, _ := os.LookupEnv(key)
	return val
}

func setEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}