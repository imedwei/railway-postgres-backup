// Package server provides HTTP server functionality for metrics and health checks.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/imedwei/railway-postgres-backup/internal/health"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server for metrics and health checks.
type Server struct {
	server  *http.Server
	logger  *slog.Logger
	checker *health.Checker
}

// Config holds server configuration.
type Config struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DefaultConfig returns default server configuration.
func DefaultConfig() Config {
	return Config{
		Port:            8080,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    10 * time.Second,
		ShutdownTimeout: 30 * time.Second,
	}
}

// New creates a new HTTP server.
func New(config Config, logger *slog.Logger) *Server {
	mux := http.NewServeMux()
	checker := health.NewChecker()

	// Set up routes
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", checker.Handler())
	mux.HandleFunc("/ready", health.ReadinessHandler())
	mux.HandleFunc("/live", health.LivenessHandler())

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      mux,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
	}

	return &Server{
		server:  server,
		logger:  logger,
		checker: checker,
	}
}

// RegisterHealthCheck registers a health check function.
func (s *Server) RegisterHealthCheck(name string, checkFunc func(context.Context) health.Check) {
	s.checker.RegisterCheck(name, checkFunc)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server", "addr", s.server.Addr)

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.server.Shutdown(ctx)
}
