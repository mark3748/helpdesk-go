# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview
Helpdesk-Go is a complete FootPrints-style ticketing system built in Go with PostgreSQL backend and Kubernetes deployment. The system consists of an API service, background worker, and two React frontends (internal workspace + customer portal).

## Development Commands

### Building
```bash
# Build API binary
cd cmd/api && go mod tidy && go build -o ../../bin/api

# Build Worker binary  
cd cmd/worker && go mod tidy && go build -o ../../bin/worker

# Build Docker images
docker build -f Dockerfile.api -t helpdesk-api .
docker build -f Dockerfile.worker -t helpdesk-worker .
```

### Testing
```bash
# Run all tests (required environment variable)
TEST_BYPASS_AUTH=true go test -cover ./...

# Test specific package
TEST_BYPASS_AUTH=true go test -cover ./internal/...
```

### Frontend Development
```bash
# Internal UI (agent/admin workspace)
cd web/internal && npm install && npm run dev

# Requester UI (customer portal)
cd web/requester && npm install && npm run dev

# Regenerate API types from OpenAPI spec
npm run gen:api
```

### Local Development Stack
```bash
# Start full stack with Docker Compose
docker compose up -d db redis api worker

# Include frontend dev servers
docker compose up -d db redis api worker internal requester
```

## Architecture

### Services
- **API** (`cmd/api`): Gin-based REST API (port 8080)
- **Worker** (`cmd/worker`): Background jobs for emails/SLA
- **Database**: PostgreSQL 14+ with goose migrations
- **Cache**: Redis for queues and caching
- **Storage**: MinIO/S3 or local filesystem for attachments

### Authentication
- **OIDC**: Production mode with JWKS validation  
- **Local**: Development mode (`AUTH_MODE=local`)
- **Test**: Bypass with `TEST_BYPASS_AUTH=true`

### Key Environment Variables
```bash
# Required for testing
export TEST_BYPASS_AUTH=true

# Local development
export DATABASE_URL=postgres://user:pass@localhost:5432/helpdesk?sslmode=disable
export AUTH_MODE=local
export AUTH_LOCAL_SECRET=dev-secret
export FILESTORE_PATH=$PWD/data  # or configure MinIO
```

## Code Patterns

### Tech Stack
- **Go 1.25+** with Gin framework
- **PostgreSQL** with pgx driver and JSONB support
- **Redis** for job queues and caching
- **React** frontends with TypeScript
- **Docker + Kubernetes** deployment

### Project Structure
- `cmd/` - Application entrypoints
- `internal/` - Private application code  
- `web/` - React frontends (internal/requester)
- `docs/` - OpenAPI specs and documentation
- `helm/` - Kubernetes deployment charts

### Testing Requirements
- Always use `TEST_BYPASS_AUTH=true` when running tests
- Maintain ≥70% test coverage across packages
- Run `go mod tidy` and verify clean git state

### Task Completion Checklist
1. Run tests: `TEST_BYPASS_AUTH=true go test -cover ./...`
2. Tidy modules: `go mod tidy`
3. Verify clean state: `git diff --exit-code go.mod go.sum`
4. For frontend changes: `npm run lint` and `npm run build`
5. For API changes: regenerate types with `npm run gen:api`

## Deployment
- **Docker Compose**: Local development (`docker-compose.yml`)
- **Helm**: Kubernetes production deployment (`helm/helpdesk/`)
- **Health checks**: `/healthz` (basic) and `/readyz` (full)