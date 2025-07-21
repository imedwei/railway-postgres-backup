# Railway PostgreSQL Backup Development Guide

This document provides guidance for Claude (AI assistant) when working on this project.

## Project Overview

Railway PostgreSQL Backup is a production-ready backup service designed for Railway.app deployments. It supports backing up PostgreSQL databases to S3 or Google Cloud Storage with respawn protection to prevent accidental frequent backups.

## Key Principles

1. **Interface-driven design**: Always define interfaces first for testability
2. **Small commits**: Make focused, single-purpose commits
3. **Test everything**: Write unit tests for all new functionality
4. **Production readiness**: Include proper error handling, logging, and metrics
5. **Railway compatibility**: Designed for Railway's cron feature, not internal scheduling

## Development Workflow

### Before Making Changes

1. Run tests: `task test`
2. Format code: `task fmt`
3. Check current test coverage

### Making Changes

1. Create interfaces first when adding new functionality
2. Write unit tests alongside implementation
3. Use structured logging with `slog`
4. Add metrics for observable behavior
5. Handle errors properly with context
6. Follow existing code patterns

### Before Committing

1. Run `task fmt` to format code
2. Run `task test` to ensure all tests pass
3. Make small, focused commits with clear messages
4. Include emoji in commit messages: 🤖

## Architecture Notes

### Storage Interface
- All storage backends implement the `Storage` interface
- Storage operations are wrapped with retry logic
- Metrics track all storage operations

### Respawn Protection
- Prevents backups within `RESPAWN_PROTECTION_HOURS`
- Can be overridden with `FORCE_BACKUP=true`
- Checks last backup time from storage

### Health Checks
- `/health` - Detailed health status
- `/ready` - Simple readiness check
- `/live` - Simple liveness check
- Includes database and storage checks

### Metrics
- Prometheus format on `/metrics`
- Track backup attempts, duration, size
- Monitor storage operations
- Record rate limit blocks

## Environment Variables

### Required
- `DATABASE_URL` - PostgreSQL connection
- `STORAGE_PROVIDER` - Either `S3` or `GCS`

### Storage-specific
- S3: `S3_BUCKET`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`
- GCS: `GCS_BUCKET`, `GOOGLE_PROJECT_ID`, `GOOGLE_SERVICE_ACCOUNT_JSON`

### Optional
- `METRICS_PORT` - Enable metrics server
- `RESPAWN_PROTECTION_HOURS` - Default: 23
- `RETENTION_DAYS` - Auto-cleanup old backups

## Common Tasks

### Add a new storage backend
1. Create interface implementation in `internal/storage/`
2. Add to factory in `internal/storage/factory.go`
3. Add configuration in `internal/config/`
4. Write comprehensive tests

### Add new metrics
1. Define in `internal/metrics/metrics.go`
2. Record in appropriate locations
3. Document in README

### Modify backup process
1. Update `internal/backup/orchestrator.go`
2. Ensure metrics are recorded
3. Test respawn protection behavior

## Testing Guidelines

- Use table-driven tests for multiple scenarios
- Mock external dependencies
- Test error conditions
- Verify metrics are recorded
- Check log output when relevant

## Deployment

- Uses multi-stage Docker build
- Optimized for small image size
- Includes only necessary files
- Sets up non-root user

## Railway-specific Considerations

- No internal cron - rely on Railway's cron
- Single-run architecture
- Fast startup time important
- Environment variable configuration
- Graceful shutdown handling