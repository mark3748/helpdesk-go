# Deployment Architecture

## Services
- **API Service**: REST API with Gin framework (port 8080)
- **Worker Service**: Background job processor for emails/SLA
- **PostgreSQL**: Primary database (port 5432)
- **Redis**: Queue and caching (port 6379)
- **MinIO/S3**: Object storage for attachments (optional)

## Frontend Applications
- **Internal UI**: Agent/admin workspace (React, port 5173/5175)
- **Requester UI**: Customer portal (React, port 5174)

## Authentication Modes
- **OIDC**: Production mode with JWKS validation
- **Local**: Development mode with cookie-based JWTs

## Storage Options
- **MinIO/S3**: Production object storage
- **Filesystem**: Local development storage (`FILESTORE_PATH`)

## Key Environment Variables
- `DATABASE_URL`: PostgreSQL connection string
- `REDIS_ADDR`: Redis server address
- `AUTH_MODE`: `oidc` or `local`
- `AUTH_LOCAL_SECRET`: HMAC secret for local auth
- `ADMIN_PASSWORD`: Initial admin password
- `MINIO_*`: S3/MinIO configuration
- `FILESTORE_PATH`: Local filesystem storage path

## Deployment Methods
1. **Docker Compose**: Local development (`docker-compose.yml`)
2. **Kubernetes**: Production deployment via Helm chart
3. **Local Binaries**: Direct execution for development

## Health Checks
- `/healthz`: Basic health check
- `/readyz`: Readiness check (DB + object storage)