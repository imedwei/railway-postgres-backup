# Railway Postgres 🥇 Daily Backups

A production-ready PostgreSQL backup service designed for Railway.app deployments with support for S3 and Google Cloud Storage.

## Features

- **Multi-Storage Support**: Back up to Amazon S3 or Google Cloud Storage
- **Respawn Protection**: Prevents frequent backups from container restarts
- **Railway Integration**: Works seamlessly with Railway's cron feature
- **Monitoring**: Prometheus metrics and health check endpoints
- **Production Ready**: Retry logic, graceful shutdown, panic recovery
- **Flexible Configuration**: Environment variable based configuration
- **Backup Management**: Automatic cleanup of old backups based on retention policy
- **PostgreSQL Version Support**: Automatically detects and uses the correct pg_dump version for PostgreSQL 15, 16, and 17

## Quick Start

### Railway Deployment

1. Deploy to Railway using the template:
   
   [![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/postgres-daily-backups?referralCode=66q-h8)

2. Configure environment variables (see Configuration section)

3. Set up Railway cron schedule (e.g., `0 3 * * *` for daily at 3 AM)

### Docker

```bash
docker run -e DATABASE_URL=postgres://... \
           -e STORAGE_PROVIDER=S3 \
           -e S3_BUCKET=my-bucket \
           -e AWS_ACCESS_KEY_ID=... \
           -e AWS_SECRET_ACCESS_KEY=... \
           ghcr.io/imedwei/railway-postgres-backup:latest
```

## Configuration

### Required Environment Variables

| Variable | Description |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL connection string |
| `STORAGE_PROVIDER` | Storage backend: `S3` or `GCS` |

### S3 Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `S3_BUCKET` | S3 bucket name | Yes |
| `AWS_ACCESS_KEY_ID` | AWS access key | Yes |
| `AWS_SECRET_ACCESS_KEY` | AWS secret key | Yes |
| `AWS_REGION` | AWS region | No (default: us-east-1) |
| `S3_ENDPOINT` | Custom S3 endpoint | No |
| `S3_PATH_STYLE` | Use path-style URLs | No (default: false) |
| `S3_PREFIX` | Key prefix for backups | No |

### GCS Configuration

| Variable | Description | Required |
|----------|-------------|----------|
| `GCS_BUCKET` | GCS bucket name | Yes |
| `GOOGLE_PROJECT_ID` | GCP project ID | Yes |
| `GOOGLE_SERVICE_ACCOUNT_JSON` | Service account JSON | Yes |
| `GCS_PREFIX` | Object prefix for backups | No |

### Backup Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `BACKUP_FILE_PREFIX` | Prefix for backup filenames | backup |
| `PG_DUMP_OPTIONS` | Additional pg_dump options | |
| `RESPAWN_PROTECTION_HOURS` | Minimum hours between backups | 23 |
| `FORCE_BACKUP` | Skip respawn protection | false |
| `RETENTION_DAYS` | Days to keep old backups | 0 (disabled) |

### Monitoring Configuration

| Variable | Description | Default |
|----------|-------------|---------|
| `METRICS_PORT` | Port for metrics/health endpoints | (disabled) |

## Monitoring

When `METRICS_PORT` is set, the following endpoints are available:

- `/metrics` - Prometheus metrics
- `/health` - Health check with detailed status
- `/ready` - Readiness probe
- `/live` - Liveness probe

### Available Metrics

- `postgres_backup_attempts_total` - Total backup attempts
- `postgres_backup_duration_seconds` - Backup duration by phase
- `postgres_backup_size_bytes` - Size of last backup
- `postgres_database_size_bytes` - Current database size
- `postgres_backup_storage_operations_total` - Storage operations
- `postgres_backup_rate_limit_blocked_total` - Rate limited backups
- `postgres_backup_last_success_timestamp` - Last successful backup time

## Respawn Protection

The service includes respawn protection to prevent excessive backups when Railway restarts containers. By default, backups are only allowed once every 23 hours. This can be configured with `RESPAWN_PROTECTION_HOURS` or overridden with `FORCE_BACKUP=true`.

## PostgreSQL Version Compatibility

The service automatically detects your PostgreSQL server version and uses the appropriate `pg_dump` client:

- **PostgreSQL 17**: Uses `pg_dump17`
- **PostgreSQL 16**: Uses `pg_dump16`
- **PostgreSQL 15**: Uses `pg_dump15`
- **PostgreSQL < 15**: Uses `pg_dump15` (backward compatible)

This ensures maximum compatibility and prevents version mismatch errors during backups.

## Development

### Prerequisites

- Go 1.24.3+
- Task (taskfile.dev)
- PostgreSQL client tools (pg_dump)
- AWS CLI (for S3 testing)
- gcloud CLI (for GCS testing)

### Setup

```bash
# Clone the repository
git clone https://github.com/imedwei/railway-postgres-backup
cd railway-postgres-backup

# Install dependencies
go mod download

# Run tests
task test

# Build
task build

# Run locally
export DATABASE_URL=postgres://localhost/mydb
export STORAGE_PROVIDER=S3
export S3_BUCKET=my-backup-bucket
export AWS_ACCESS_KEY_ID=...
export AWS_SECRET_ACCESS_KEY=...
./bin/backup
```

### Testing

```bash
# Run all tests
task test

# Run tests with coverage
task test:coverage

# Run specific package tests
go test ./internal/storage/...
```

### Project Structure

```
.
├── cmd/backup/          # Main application entry point
├── internal/
│   ├── backup/          # Backup orchestration and PostgreSQL
│   ├── config/          # Configuration management
│   ├── health/          # Health check implementation
│   ├── metrics/         # Prometheus metrics
│   ├── ratelimit/       # Respawn protection
│   ├── server/          # HTTP server for metrics
│   ├── storage/         # Storage backends (S3, GCS)
│   └── utils/           # Utility functions
├── Dockerfile           # Multi-stage Docker build
├── Taskfile.yml         # Task automation
└── go.mod               # Go module definition
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`task test`)
5. Format your code (`task fmt`)
6. Commit your changes (small, focused commits)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

## License

MIT License - see [LICENSE](LICENSE) file for details

## Acknowledgments

- Based on [railwayapp-templates/postgres-s3-backups](https://github.com/railwayapp-templates/postgres-s3-backups)
- Built for [Railway.app](https://railway.app) deployments
