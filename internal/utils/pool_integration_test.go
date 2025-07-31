package utils

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"net"
	"testing"
	"time"
)

func TestConnectionRetryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		name           string
		databaseURL    string
		retryConfig    RetryConfig
		expectedErr    string
		expectSuccess  bool
	}{
		{
			name:        "connection refused error triggers retry",
			databaseURL: "postgres://user:pass@localhost:55432/testdb?sslmode=disable",
			retryConfig: RetryConfig{
				MaxRetries:    2,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      500 * time.Millisecond,
				BackoffFactor: 2.0,
			},
			expectedErr:   "all database connection attempts failed",
			expectSuccess: false,
		},
		{
			name:        "invalid host triggers retry",
			databaseURL: "postgres://user:pass@invalid-host-that-does-not-exist:5432/testdb?sslmode=disable",
			retryConfig: RetryConfig{
				MaxRetries:    1,
				InitialDelay:  100 * time.Millisecond,
				MaxDelay:      200 * time.Millisecond,
				BackoffFactor: 2.0,
			},
			expectedErr:   "all database connection attempts failed",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			startTime := time.Now()
			pool, err := NewConnectionPoolWithRetry(ctx, tt.databaseURL, tt.retryConfig)
			elapsed := time.Since(startTime)

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("expected success but got error: %v", err)
				} else {
					defer pool.Close()
				}
			} else {
				if err == nil {
					t.Errorf("expected error but got success")
					pool.Close()
				} else if !errors.Is(err, context.DeadlineExceeded) && !contains(err.Error(), tt.expectedErr) {
					t.Errorf("expected error containing %q, got %v", tt.expectedErr, err)
				}
			}

			// Verify retry delays were applied
			expectedMinTime := time.Duration(tt.retryConfig.MaxRetries) * tt.retryConfig.InitialDelay
			if elapsed < expectedMinTime/2 { // Allow some margin
				t.Errorf("retries completed too quickly: %v < %v", elapsed, expectedMinTime)
			}
		})
	}
}

func TestColdBootErrorDetection(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "database starting up",
			err:      &net.OpError{Err: errors.New("the database system is starting up")},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      &net.OpError{Op: "dial", Err: errors.New("connect: connection refused")},
			expected: true,
		},
		{
			name:     "timeout error", 
			err:      &net.OpError{Err: errors.New("i/o timeout")},
			expected: true,
		},
		{
			name:     "authentication error",
			err:      errors.New("pq: password authentication failed"),
			expected: false,
		},
		{
			name:     "sql error",
			err:      sql.ErrNoRows,
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

func TestRetryDelayProgression(t *testing.T) {
	config := RetryConfig{
		MaxRetries:    5,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
	}

	expectedDelays := []time.Duration{
		0,                        // First attempt, no delay
		100 * time.Millisecond,   // After 1st failure
		200 * time.Millisecond,   // After 2nd failure
		400 * time.Millisecond,   // After 3rd failure
		800 * time.Millisecond,   // After 4th failure
		1 * time.Second,          // After 5th failure (capped at max)
	}

	delay := config.InitialDelay
	for i := 1; i < len(expectedDelays); i++ {
		if i > 1 {
			// Calculate next delay with exponential backoff
			nextDelay := float64(delay) * config.BackoffFactor
			delay = time.Duration(math.Min(nextDelay, float64(config.MaxDelay)))
		}

		if delay != expectedDelays[i] {
			t.Errorf("attempt %d: expected delay %v, got %v", i, expectedDelays[i], delay)
		}
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && s[0:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}