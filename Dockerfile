# Build stage
FROM golang:1.24.5-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o postgres-backup ./cmd/backup

# Final stage
FROM alpine:latest

# Install runtime dependencies and PostgreSQL client versions 15, 16, 17
# Note: Alpine supports multiple PostgreSQL client versions side by side
RUN apk add --no-cache \
    postgresql15-client \
    postgresql16-client \
    postgresql17-client \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 -S backup && \
    adduser -u 1000 -S backup -G backup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/postgres-backup .

# Change ownership
RUN chown -R backup:backup /app

USER backup

ENTRYPOINT ["./postgres-backup"]
