# Code Style and Conventions

## Go Code Style
- **Go Version**: 1.25+
- **Module**: `github.com/mark3748/helpdesk-go`
- **Framework**: Gin for HTTP routing
- **Database**: PostgreSQL with pgx driver
- **Logging**: zerolog structured logging
- **Error Handling**: Standard Go error patterns
- **Configuration**: Environment variables with fallback defaults

## Key Dependencies
- `github.com/gin-gonic/gin` - HTTP web framework
- `github.com/jackc/pgx/v5` - PostgreSQL driver
- `github.com/redis/go-redis/v9` - Redis client
- `github.com/pressly/goose/v3` - Database migrations
- `github.com/rs/zerolog` - Structured logging
- `github.com/golang-jwt/jwt/v5` - JWT authentication
- `github.com/minio/minio-go/v7` - S3/MinIO client

## Project Structure
- `cmd/` - Application entrypoints (api, worker)
- `internal/` - Private application code
- `web/` - Frontend applications (internal, requester)
- `docs/` - API documentation and specs
- `helm/` - Kubernetes deployment charts
- `docker/` - Docker-related files

## Authentication Patterns
- OIDC with JWKS validation (production)
- Local auth with HMAC-signed JWTs (development)
- Test bypass with `TEST_BYPASS_AUTH=true`
- Role-based access control (admin, manager, agent)

## Database Patterns
- goose migrations in embedded filesystem
- Context-aware database operations
- PostgreSQL-specific features (JSONB, full-text search)
- Connection pooling with pgx