# Railway PostgreSQL Backup

A Go-based PostgreSQL backup service designed for Railway deployments, supporting both AWS S3 and Google Cloud Storage (GCS) with built-in respawn protection.

## Features

- 🚀 Single-run architecture optimized for Railway cron jobs
- 💾 Support for AWS S3 and Google Cloud Storage
- 🛡️ Built-in respawn protection to prevent accidental frequent backups
- 🔧 Configurable backup options via environment variables
- 📊 Structured logging with detailed progress reporting
- 🐳 Optimized Docker image for fast cold starts
- 🧪 Comprehensive test coverage

## Quick Start

### Deploy on Railway

[![Deploy on Railway](https://railway.app/button.svg)](https://railway.app/template/railway-postgres-backup)

### Environment Variables

#### Core Configuration
- `DATABASE_URL` - PostgreSQL connection string
- `STORAGE_PROVIDER` - Storage backend (`s3` or `gcs`)

#### AWS S3 Configuration
- `AWS_ACCESS_KEY_ID` - AWS access key
- `AWS_SECRET_ACCESS_KEY` - AWS secret key
- `S3_BUCKET` - S3 bucket name
- `S3_REGION` - AWS region

#### Google Cloud Storage Configuration
- `GCS_BUCKET` - GCS bucket name
- `GOOGLE_PROJECT_ID` - Google Cloud project ID
- `GOOGLE_SERVICE_ACCOUNT_JSON` - Service account credentials JSON

#### Respawn Protection
- `RESPAWN_PROTECTION_HOURS` - Minimum hours between backups (default: 6)
- `FORCE_BACKUP` - Override respawn protection (`true` to force backup)

#### Additional Options
- `BACKUP_FILE_PREFIX` - Prefix for backup files
- `PG_DUMP_OPTIONS` - Additional pg_dump options
- `RETENTION_DAYS` - Days to retain old backups

## Development

### Prerequisites
- Go 1.21+
- Task (https://taskfile.dev)
- Docker (for containerized testing)

### Building
```bash
task build
```

### Testing
```bash
task test
task test:coverage
```

### Running Locally
```bash
task dev
```

## Architecture

This service follows a single-run architecture designed specifically for Railway cron jobs. It performs a backup on each run and exits, relying on Railway's cron scheduler for timing.

### Respawn Protection

To prevent accidental frequent backups (e.g., from container restarts), the service includes respawn protection that checks the timestamp of the last successful backup before proceeding.

## Migration from TypeScript Version

If you're migrating from the original TypeScript version, the environment variables have been updated:

- `AWS_S3_BUCKET` → `S3_BUCKET`
- `AWS_S3_REGION` → `S3_REGION`
- `BACKUP_CRON_SCHEDULE` → Use Railway's cron configuration
- `RUN_ON_STARTUP` → No longer needed (always runs on startup)
- `SINGLE_SHOT_MODE` → No longer needed (always single shot)

## License

MIT
