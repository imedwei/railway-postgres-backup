# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of Railway PostgreSQL Backup
- Support for AWS S3 storage backend
- Support for Google Cloud Storage (GCS) backend
- Respawn protection to prevent frequent backups
- Prometheus metrics for monitoring
- Health check endpoints for Kubernetes/Railway
- Automatic cleanup of old backups based on retention policy
- Retry logic with exponential backoff
- Graceful shutdown handling
- Panic recovery
- Progress tracking for large backups
- Connection pooling for database operations
- Buffer pool for efficient memory usage
- Comprehensive test suite
- Docker multi-stage build
- GitHub Actions CI/CD pipeline
- Railway deployment configuration

### Security
- Non-root user in Docker container
- Secure handling of credentials
- Support for SSL/TLS database connections

## [1.0.0] - TBD

Initial stable release.