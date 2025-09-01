# Helpdesk-Go Development Instructions

**CRITICAL**: Always follow these instructions first and only fallback to additional search and context gathering if the information here is incomplete or found to be in error.

## Project Overview
This is a Go-based helpdesk ticketing system with two main components:
- **API service** (Gin framework) with PostgreSQL, Redis, MinIO, JWT auth
- **Worker service** for background jobs (email, notifications)
- Helm chart for Kubernetes deployment
- PostgreSQL with embedded migrations using goose

## Bootstrap and Build Process

### Prerequisites - EXACT commands to install dependencies:
```bash
# Ensure Go 1.23+ is installed - check version
go version

# Install golangci-lint for linting (optional)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.2
export PATH="/home/runner/go/bin:$PATH"
```

### Build Commands - EXACT sequence with NEVER CANCEL warnings:
```bash
# Step 1: Build API component
cd api/cmd/api
go mod tidy  # Downloads dependencies - takes ~30 seconds first time, <5 seconds if cached
time go build -o ../../../bin/api  # NEVER CANCEL: First build takes 32 seconds, subsequent builds <1 second. Set timeout to 60+ minutes for safety.

# Step 2: Build Worker component  
cd ../../../worker/cmd/worker
go mod tidy  # Fast if API dependencies already downloaded, <5 seconds
time go build -o ../../../bin/worker  # NEVER CANCEL: First build takes 2.5 seconds, subsequent builds <1 second. Set timeout to 30+ minutes for safety.

# Verify binaries created
ls -lah ../../bin/
# Should show: api (~22M) and worker (~19M) executables
```

### Testing - EXACT commands with timing:
```bash
# Run API tests - all should pass
cd api/cmd/api
time go test -v .  # NEVER CANCEL: Tests complete in <1 second but set timeout to 30+ minutes for safety.

# No worker tests exist currently
```

### Docker Build - EXACT commands:
```bash
# Build API Docker image - REQUIRES Go 1.23
time docker build -f Dockerfile.api -t helpdesk-api .  # NEVER CANCEL: Takes 5-10 minutes first time. Set timeout to 60+ minutes.

# Build Worker Docker image
time docker build -f Dockerfile.worker -t helpdesk-worker .  # NEVER CANCEL: Takes 5-10 minutes first time. Set timeout to 60+ minutes.

# Note: Docker builds may fail in restricted network environments due to Go proxy certificate issues
```

## Database Setup - EXACT requirements:
```bash
# CRITICAL: PostgreSQL 14+ required with specific connection string format
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"

# Database will be auto-migrated on first API startup using embedded goose migrations
# Migrations are in: api/cmd/api/migrations/ (0001_init.sql, 0002_roles.sql, 0003_watchers.sql)
```

## Running the Applications

### API Service:
```bash
# CRITICAL: Must set DATABASE_URL before running
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"

# Run API (will auto-migrate database on startup)
./bin/api
# Default port: 8080
# Health check: curl http://localhost:8080/healthz

# FOR TESTING: Bypass JWT auth (injects synthetic user with 'agent' role)
export TEST_BYPASS_AUTH=true
./bin/api
```

### Worker Service:
```bash
# Worker requires DATABASE_URL and Redis connection
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"
export REDIS_ADDR="localhost:6379"

./bin/worker
# Worker polls Redis queue for email jobs
```

## Manual Validation - MANDATORY testing scenarios:

### After Building - ALWAYS run these validation steps:
```bash
# 1. Verify binaries were created successfully
ls -lah bin/
# Expected: api (~22M) and worker (~19M) executables

# 2. Test that API starts (will fail without database - this is expected)
./bin/api
# Expected: Error message about database connection refused

# 3. Test authentication bypass mode with API tests
cd api/cmd/api
export TEST_BYPASS_AUTH=true
go test -v .
# Expected: All tests pass, including synthetic user with "agent" role

# 4. Test worker startup (will continuously retry Redis connections)
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"
export REDIS_ADDR="localhost:6379"
./bin/worker
# Expected: "redis ping failed (queue not active yet)" followed by "worker started" then continuous Redis connection errors
# Note: Worker will retry indefinitely - this is normal behavior when Redis is unavailable

# WITH REAL DATABASE (when available):
# 5. Test full API functionality
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"
export TEST_BYPASS_AUTH=true
./bin/api &
curl http://localhost:8080/healthz
# Expected: {"ok":true}
curl http://localhost:8080/me  
# Expected: JSON with synthetic user containing "agent" role
curl http://localhost:8080/tickets
# Expected: Empty array [] or ticket list
```

## Linting and Code Quality

### Format Code:
```bash
# Check formatting
cd api/cmd/api && gofmt -l .
cd ../../../worker/cmd/worker && gofmt -l .

# Fix formatting
cd api/cmd/api && gofmt -w .
cd ../../../worker/cmd/worker && gofmt -w .
```

### Linting (optional):
```bash
# Run golangci-lint (may have version compatibility issues)
cd api/cmd/api && golangci-lint run .
cd ../../../worker/cmd/worker && golangci-lint run .
```

## Common Issues and Workarounds

### Build Failures:
- **"go.mod requires go >= 1.23"**: Update Dockerfiles to use `golang:1.23-alpine`
- **Certificate errors in Docker**: Network restrictions on Go proxy access - build locally first
- **Missing dependencies**: Run `go mod tidy` in each component directory

### Runtime Failures:
- **Database connection refused**: Ensure PostgreSQL 14+ is running and DATABASE_URL is correct
- **Redis connection failed**: Worker logs error but continues - Redis only needed for job queue functionality
- **JWT validation errors**: Use `TEST_BYPASS_AUTH=true` for development/testing

### Test Failures:
- **"not enough arguments in call to NewApp"**: Fixed - tests now include Redis client parameter
- All tests should pass in bypass auth mode

## API Endpoints Reference
- `GET /healthz` - Health check (no auth required)
- `GET /me` - User info (requires auth or TEST_BYPASS_AUTH=true)  
- `GET /tickets` - List tickets (requires 'agent' role)
- `POST /tickets` - Create ticket (requires 'agent' role)
- `GET /tickets/:id` - Get ticket details
- `PATCH /tickets/:id` - Update ticket (requires 'agent' role)
- `POST /tickets/:id/comments` - Add comment

## Key Project Files and Locations

### Frequently Modified:
- `api/cmd/api/main.go` - API service main entry point and route definitions
- `api/cmd/api/migrations/` - Database schema migrations
- `worker/cmd/worker/main.go` - Worker service for background jobs
- `worker/cmd/worker/templates/` - Email templates

### Configuration:
- `api/cmd/api/go.mod` - API dependencies (Go 1.23)
- `worker/cmd/worker/go.mod` - Worker dependencies (Go 1.23)  
- `Dockerfile.api` - API container build (requires golang:1.23-alpine)
- `Dockerfile.worker` - Worker container build
- `helm/helpdesk/` - Kubernetes deployment charts

### Repository Structure:
```
/api/cmd/api/          # API service source
/worker/cmd/worker/    # Worker service source  
/helm/helpdesk/        # Kubernetes deployment
/bin/                  # Built binaries (gitignored)
```

## Environment Variables Reference
- `DATABASE_URL` - PostgreSQL connection string (REQUIRED)
- `REDIS_ADDR` - Redis server address (default: localhost:6379)
- `TEST_BYPASS_AUTH` - Set to "true" to bypass JWT validation in tests
- `ENV` - Environment mode ("dev" enables debug logging)
- `JWKS_URL` - OIDC JWKS endpoint for JWT validation
- `MINIO_*` - MinIO object storage configuration

**CRITICAL REMINDER**: NEVER CANCEL long-running builds or tests. API builds take 32 seconds, Docker builds take 5-10 minutes. Always set appropriate timeouts (60+ minutes for builds, 30+ minutes for tests) and wait for completion.