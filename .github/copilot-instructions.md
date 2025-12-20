# Helpdesk-Go Development Instructions

**CRITICAL**: Always follow these instructions first and only fallback to additional search and context gathering if the information here is incomplete or found to be in error.

## Project Overview
This is a Go-based helpdesk ticketing system with three main components:
- **API service** (Gin framework) with PostgreSQL, Redis, MinIO, JWT auth - located in `cmd/api/`
- **Worker service** for background jobs (email, notifications) - located in `cmd/worker/`
- **AuditCLI** for audit export job management - located in `cmd/auditcli/`
- Helm chart for Kubernetes deployment
- PostgreSQL with embedded migrations using goose
- Single Go module at repository root managing all components

## Bootstrap and Build Process

### Prerequisites - EXACT commands to install dependencies:
```bash
# Ensure Go 1.25+ is installed - check version
go version

# Install golangci-lint for linting (optional)
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.62.2
export PATH="/home/runner/go/bin:$PATH"
```

**Note**: This is a Go workspace/monorepo with a single `go.mod` at the repository root. All components share the same module dependencies.

### Build Commands - EXACT sequence with NEVER CANCEL warnings:
```bash
# Build from repository root using single go.mod
cd /home/runner/work/helpdesk-go/helpdesk-go

# Step 1: Download dependencies (shared across all components)
go mod download  # Downloads dependencies - takes ~30 seconds first time, <5 seconds if cached

# Step 2: Build API component
go build -o bin/api ./cmd/api  # NEVER CANCEL: First build takes 30-60 seconds, subsequent builds <1 second. Set timeout to 60+ minutes for safety.

# Step 3: Build Worker component
go build -o bin/worker ./cmd/worker  # NEVER CANCEL: First build takes 2-5 seconds, subsequent builds <1 second. Set timeout to 30+ minutes for safety.

# Step 4: Build AuditCLI component (optional)
go build -o bin/auditcli ./cmd/auditcli  # Quick build <1 second

# Verify binaries created
ls -lh bin/
# Should show: api (~27M), worker (~32M), and optionally auditcli (~14M) executables
```

### Testing - EXACT commands with timing:
```bash
# Run all tests from repository root
go test ./...  # Tests all packages

# Run API tests specifically
go test ./cmd/api -v  # NEVER CANCEL: Tests complete in <5 seconds but set timeout to 30+ minutes for safety.

# Run worker tests specifically  
go test ./cmd/worker -v  # Worker has basic tests

# Run with coverage
go test -cover ./...

# For development/testing: bypass JWT auth
export TEST_BYPASS_AUTH=true
go test ./cmd/api -v  # Injects synthetic user with 'agent' role
```

### Docker Build - EXACT commands:
```bash
# Build from repository root
cd /home/runner/work/helpdesk-go/helpdesk-go

# Build API Docker image - REQUIRES Go 1.25
docker build -f Dockerfile.api -t helpdesk-api .  # NEVER CANCEL: Takes 5-10 minutes first time. Set timeout to 60+ minutes.

# Build Worker Docker image
docker build -f Dockerfile.worker -t helpdesk-worker .  # NEVER CANCEL: Takes 5-10 minutes first time. Set timeout to 60+ minutes.

# Note: Docker builds may fail in restricted network environments due to Go proxy certificate issues
```

## Database Setup - EXACT requirements:
```bash
# CRITICAL: PostgreSQL 14+ required with specific connection string format
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"

# Database will be auto-migrated on first API startup using embedded goose migrations
# Migrations are in: cmd/api/migrations/ (e.g., 0001_init.sql, 0002_roles.sql, etc.)
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
ls -lh bin/
# Expected: api (~27M), worker (~32M), and optionally auditcli (~14M) executables

# 2. Test that API starts (will fail without database - this is expected)
./bin/api
# Expected: Error message about database connection refused

# 3. Test authentication bypass mode with API tests
export TEST_BYPASS_AUTH=true
go test ./cmd/api -v
# Expected: All tests pass, including synthetic user with "agent" role

# 4. Test worker startup (will continuously retry Redis connections)
export DATABASE_URL="postgres://user:pass@localhost:5432/helpdesk?sslmode=disable"
export REDIS_ADDR="localhost:6379"
./bin/worker
# Expected: "redis ping failed (queue not active yet)" followed by "worker started" then continuous Redis connection errors
# Note: Worker will retry indefinitely - this is normal behavior when Redis is unavailable

# 5. Test AuditCLI (requires Redis)
export REDIS_ADDR="localhost:6379"
./bin/auditcli run
# Expected: Prints a job UUID or connection error if Redis unavailable

# WITH REAL DATABASE (when available):
# 6. Test full API functionality
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
# Check formatting from repository root
gofmt -l ./cmd/api
gofmt -l ./cmd/worker
gofmt -l ./cmd/auditcli

# Fix formatting
gofmt -w ./cmd/api
gofmt -w ./cmd/worker
gofmt -w ./cmd/auditcli

# Or format all Go files
gofmt -w .
```

### Linting (optional):
```bash
# Run golangci-lint from repository root
golangci-lint run ./cmd/api/...
golangci-lint run ./cmd/worker/...
golangci-lint run ./cmd/auditcli/...

# Or lint entire project
golangci-lint run ./...
```

## Common Issues and Workarounds

### Build Failures:
- **"go.mod requires go >= 1.25"**: Ensure Go 1.25+ is installed; Update Dockerfiles to use `golang:1.25` base image
- **Certificate errors in Docker**: Network restrictions on Go proxy access - build locally first
- **Missing dependencies**: Run `go mod download` from repository root

### Runtime Failures:
- **Database connection refused**: Ensure PostgreSQL 14+ is running and DATABASE_URL is correct
- **Redis connection failed**: Worker logs error but continues - Redis only needed for job queue functionality
- **JWT validation errors**: Use `TEST_BYPASS_AUTH=true` for development/testing

### Test Failures:
- **Auth-related test failures**: Set `TEST_BYPASS_AUTH=true` environment variable
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
- `cmd/api/main.go` - API service main entry point and route definitions
- `cmd/api/migrations/` - Database schema migrations (goose format)
- `cmd/worker/main.go` - Worker service for background jobs
- `cmd/worker/templates/` - Email templates
- `cmd/auditcli/main.go` - CLI tool for audit export job management

### Configuration:
- `go.mod` - Root module with all dependencies (Go 1.25)
- `Dockerfile.api` - API container build (requires golang:1.25)
- `Dockerfile.worker` - Worker container build
- `Dockerfile.frontend-internal` - Internal web UI container
- `Dockerfile.frontend-requester` - Requester portal container
- `helm/helpdesk/` - Kubernetes deployment charts
- `docker-compose.yml` - Local development stack

### Repository Structure:
```
/cmd/
  /api/          # API service source (main.go + subdirectories)
  /worker/       # Worker service source
  /auditcli/     # Audit CLI tool
/internal/       # Shared internal packages (if any)
/web/
  /internal/     # Internal workspace UI (React)
  /requester/    # Customer portal UI (React)
/helm/helpdesk/  # Kubernetes deployment
/bin/            # Built binaries (gitignored)
/docs/           # API documentation and guides
go.mod           # Root Go module
```

## Environment Variables Reference
- `DATABASE_URL` - PostgreSQL connection string (REQUIRED)
- `REDIS_ADDR` - Redis server address (default: localhost:6379)
- `TEST_BYPASS_AUTH` - Set to "true" to bypass JWT validation in tests
- `ENV` - Environment mode ("dev" enables debug logging)
- `JWKS_URL` - OIDC JWKS endpoint for JWT validation
- `MINIO_*` - MinIO object storage configuration

**CRITICAL REMINDER**: NEVER CANCEL long-running builds or tests. API builds take 30-60 seconds on first run, Docker builds take 5-10 minutes. Always set appropriate timeouts (60+ minutes for builds, 30+ minutes for tests) and wait for completion.

## Additional Components

### AuditCLI Tool
The `auditcli` is a command-line utility for managing audit export jobs:
- **Location**: `cmd/auditcli/main.go`
- **Build**: `go build -o bin/auditcli ./cmd/auditcli`
- **Usage**: 
  - `./bin/auditcli run` - Queue a new audit export job (returns job ID)
  - `./bin/auditcli status <job_id>` - Check status of an audit export job
- **Requirements**: Redis connection via `REDIS_ADDR` environment variable
- **Integration**: Works with the worker service which processes queued audit jobs

### Web Frontends
The repository includes two React-based web applications:
- **Internal Workspace** (`web/internal/`) - Admin/agent interface for ticket management
- **Requester Portal** (`web/requester/`) - Customer-facing ticket submission portal

Build and development instructions are in respective README.md files within each directory.