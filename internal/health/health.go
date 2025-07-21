// Package health provides health check functionality.
package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusHealthy indicates the component is healthy.
	StatusHealthy Status = "healthy"
	// StatusUnhealthy indicates the component is unhealthy.
	StatusUnhealthy Status = "unhealthy"
)

// Check represents a health check result.
type Check struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// Checker performs health checks.
type Checker struct {
	mu     sync.RWMutex
	checks map[string]func(context.Context) Check
}

// NewChecker creates a new health checker.
func NewChecker() *Checker {
	return &Checker{
		checks: make(map[string]func(context.Context) Check),
	}
}

// RegisterCheck registers a health check function.
func (c *Checker) RegisterCheck(name string, checkFunc func(context.Context) Check) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.checks[name] = checkFunc
}

// CheckHealth performs all registered health checks.
func (c *Checker) CheckHealth(ctx context.Context) map[string]Check {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make(map[string]Check)
	for name, checkFunc := range c.checks {
		results[name] = checkFunc(ctx)
	}
	return results
}

// Handler returns an HTTP handler for health checks.
func (c *Checker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		results := c.CheckHealth(ctx)

		// Determine overall status
		overallStatus := StatusHealthy
		for _, check := range results {
			if check.Status == StatusUnhealthy {
				overallStatus = StatusUnhealthy
				break
			}
		}

		// Prepare response
		response := struct {
			Status    Status           `json:"status"`
			Checks    map[string]Check `json:"checks"`
			Timestamp time.Time        `json:"timestamp"`
		}{
			Status:    overallStatus,
			Checks:    results,
			Timestamp: time.Now(),
		}

		// Set appropriate status code
		if overallStatus == StatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Write JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			// Log error but don't try to write another response
			// as headers are already sent
			_ = err
		}
	}
}

// ReadinessHandler returns a simple readiness check handler.
func ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	}
}

// LivenessHandler returns a simple liveness check handler.
func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("alive\n"))
	}
}
