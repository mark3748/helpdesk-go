.PHONY: help build test docker lint-helm clean dev-up dev-down

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build API and worker binaries"
	@echo "  test         - Run all tests with coverage"
	@echo "  docker       - Build all Docker images"
	@echo "  lint-helm    - Lint and package Helm chart"
	@echo "  clean        - Clean build artifacts"
	@echo "  dev-up       - Start development stack with Docker Compose"
	@echo "  dev-down     - Stop development stack"
	@echo "  frontend     - Install and build frontend applications"
	@echo "  gen-types    - Generate TypeScript types from OpenAPI spec"

# Build binaries
build:
	@echo "Building API binary..."
	cd cmd/api && go mod tidy && go build -o ../../bin/api
	@echo "Building worker binary..."
	cd cmd/worker && go mod tidy && go build -o ../../bin/worker
	@echo "Build complete!"

# Run tests
test:
	@echo "Running tests with coverage..."
	TEST_BYPASS_AUTH=true go test -cover ./...
	@echo "Tidying modules..."
	go mod tidy
	@echo "Checking for clean git state..."
	git diff --exit-code go.mod go.sum || (echo "Warning: go.mod or go.sum has changes" && exit 1)

# Build Docker images
docker:
	@echo "Building Docker images..."
	docker build -f Dockerfile.api -t helpdesk-api .
	docker build -f Dockerfile.worker -t helpdesk-worker .
	docker build -f Dockerfile.frontend-internal -t helpdesk-internal-frontend .
	docker build -f Dockerfile.frontend-requester -t helpdesk-requester-frontend .
	@echo "Docker images built successfully!"

# Lint and package Helm chart
lint-helm:
	@echo "Linting Helm chart..."
	helm lint helm/helpdesk
	@echo "Verifying ingress API mapping..."
	helm template test helm/helpdesk > rendered.yaml
	grep -A 5 'path: /api' rendered.yaml | grep 'name: test-helpdesk-api' || (echo "Error: API mapping verification failed" && exit 1)
	@echo "Packaging Helm chart..."
	helm package helm/helpdesk
	@echo "Helm chart validation complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f helpdesk-*.tgz
	rm -f rendered.yaml
	@echo "Clean complete!"

# Start development stack
dev-up:
	@echo "Starting development stack..."
	docker compose up -d db redis api worker
	@echo "Development stack started!"
	@echo "API available at: http://localhost:8080"
	@echo "Health check: curl http://localhost:8080/healthz"

# Stop development stack
dev-down:
	@echo "Stopping development stack..."
	docker compose down
	@echo "Development stack stopped!"

# Frontend development setup
frontend:
	@echo "Setting up internal UI..."
	cd web/internal && npm install && npm run build
	@echo "Setting up requester UI..."
	cd web/requester && npm install && npm run build
	@echo "Frontend setup complete!"

# Generate TypeScript types from OpenAPI spec
gen-types:
	@echo "Generating TypeScript types from OpenAPI spec..."
	@echo "Copying OpenAPI spec to frontend directories..."
	cp docs/openapi.yaml web/internal/
	cp docs/openapi.yaml web/requester/
	@echo "Generating types for internal frontend..."
	cd web/internal && npm run gen:api:file
	@echo "Generating types for requester frontend..."
	cd web/requester && npm run gen:api:file
	@echo "Cleaning up..."
	rm web/internal/openapi.yaml web/requester/openapi.yaml
	@echo "TypeScript types generation complete!"

# Development with frontends
dev-full:
	@echo "Starting full development stack with frontends..."
	docker compose up -d db redis api worker internal requester
	@echo "Full development stack started!"
	@echo "API: http://localhost:8080"
	@echo "Internal UI: http://localhost:5175"
	@echo "Requester UI: http://localhost:5174"