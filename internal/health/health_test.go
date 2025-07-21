package health

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChecker(t *testing.T) {
	checker := NewChecker()

	// Register a healthy check
	checker.RegisterCheck("test-healthy", func(ctx context.Context) Check {
		return Check{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"test": "value"},
		}
	})

	// Register an unhealthy check
	checker.RegisterCheck("test-unhealthy", func(ctx context.Context) Check {
		return Check{
			Status:    StatusUnhealthy,
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"error": "test error"},
		}
	})

	// Perform health check
	results := checker.CheckHealth(context.Background())

	// Verify results
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	if results["test-healthy"].Status != StatusHealthy {
		t.Errorf("Expected test-healthy to be healthy")
	}

	if results["test-unhealthy"].Status != StatusUnhealthy {
		t.Errorf("Expected test-unhealthy to be unhealthy")
	}
}

func TestHealthHandler(t *testing.T) {
	checker := NewChecker()

	// Register checks
	checker.RegisterCheck("healthy", func(ctx context.Context) Check {
		return Check{
			Status:    StatusHealthy,
			Timestamp: time.Now(),
		}
	})

	checker.RegisterCheck("unhealthy", func(ctx context.Context) Check {
		return Check{
			Status:    StatusUnhealthy,
			Timestamp: time.Now(),
		}
	})

	// Create request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Record response
	rr := httptest.NewRecorder()
	handler := checker.Handler()
	handler.ServeHTTP(rr, req)

	// Check status code (should be 503 due to unhealthy check)
	if status := rr.Code; status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusServiceUnavailable)
	}

	// Check response body
	var response struct {
		Status    Status           `json:"status"`
		Checks    map[string]Check `json:"checks"`
		Timestamp time.Time        `json:"timestamp"`
	}

	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != StatusUnhealthy {
		t.Errorf("Expected overall status to be unhealthy")
	}

	if len(response.Checks) != 2 {
		t.Errorf("Expected 2 checks in response, got %d", len(response.Checks))
	}
}

func TestReadinessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := ReadinessHandler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "ready\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestLivenessHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/live", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := LivenessHandler()
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := "alive\n"
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}
