# Suggested Commands

## Building
```bash
# Build API binary
cd cmd/api && go mod tidy && go build -o ../../bin/api

# Build Worker binary
cd cmd/worker && go mod tidy && go build -o ../../bin/worker

# Build all Docker images
docker build -f Dockerfile.api -t helpdesk-api .
docker build -f Dockerfile.worker -t helpdesk-worker .
docker build -f Dockerfile.frontend-internal -t helpdesk-internal-frontend .
docker build -f Dockerfile.frontend-requester -t helpdesk-requester-frontend .
```

## Testing
```bash
# Run all tests with auth bypass
TEST_BYPASS_AUTH=true go test -cover ./...

# Run tests for specific package
TEST_BYPASS_AUTH=true go test -cover ./internal/...
```

## Frontend Development
```bash
# Internal UI dev server
cd web/internal && npm install && npm run dev

# Requester UI dev server  
cd web/requester && npm install && npm run dev

# Generate API types from OpenAPI spec
cd web/internal && npm run gen:api
cd web/requester && npm run gen:api
```

## Docker Compose (Local Development)
```bash
# Start full stack
docker compose up -d db redis api worker

# Start with UIs
docker compose up -d db redis api worker internal requester

# View logs
docker compose logs -f api
```

## Helm (Kubernetes)
```bash
# Lint chart
helm lint helm/helpdesk

# Package chart
helm package helm/helpdesk

# Install with local auth
helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace -f helm/helpdesk/examples/values-local-auth.yaml
```

## Module Management
```bash
# Tidy dependencies
go mod tidy

# Verify modules are clean
git diff --exit-code go.mod go.sum
```