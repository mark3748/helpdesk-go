# Helpdesk (Go) — MVP Scaffold

A minimal FootPrints-style ticketing system scaffold in Go, with PostgreSQL migrations and a Helm chart for Kubernetes.

## What you get
- Go API service (Gin) with basic ticket endpoints
- Embedded SQL migrations (goose) for Postgres (JSONB custom fields, FTS index)
- Dockerfiles for API & Worker
- Helm chart skeleton (api + worker deployments, service, ingress, config)

## Quickstart (local, Docker Compose-free)
1. Provision Postgres 14+ and create a db (e.g., `helpdesk`). Save the URL:
   ```
   export DATABASE_URL=postgres://user:pass@localhost:5432/helpdesk?sslmode=disable
   ```
2. Build API:
   ```bash
   cd cmd/api
   go mod tidy
   go build -o ../../bin/api
   ../../bin/api
   ```
3. Hit health:
   ```
   curl http://localhost:8080/healthz
   ```

## Local Development

- Build binaries:
  ```bash
  cd cmd/api && go mod tidy && go build -o ../../bin/api
  cd cmd/worker && go mod tidy && go build -o ../../bin/worker
  ```
- Run unit tests (API only):
  ```bash
  cd cmd/api && go mod tidy && TEST_BYPASS_AUTH=true go test -v .
  ```
- Build Docker images:
  ```bash
  docker build -f Dockerfile.api -t helpdesk-api .
  docker build -f Dockerfile.worker -t helpdesk-worker .
  ```
- Helm chart checks:
  ```bash
  helm lint helm/helpdesk
  helm package helm/helpdesk
  ```

## API (MVP)
- `GET /healthz`
- `GET /me` (authenticated user info and roles)
- `GET /tickets`
- `POST /tickets`  (title, description, priority, urgency, category, subcategory, custom_json)
- `GET /tickets/:id`
- `PATCH /tickets/:id` (status, assignee_id, priority, urgency, scheduled_at, due_at, custom_json)
- `POST /tickets/:id/comments` (body_md, is_internal)

## Helm (Kubernetes)
1. Set values in `helm/helpdesk/values.yaml` (hostnames, secrets, external DB/Redis/MinIO).
2. Package & install:
   ```bash
   helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace
   ```

### Notes
- Auth uses OIDC JWTs with role checks (agents required for ticket updates).
- Worker service is scaffolded but not wired to a queue yet.
- This is a starter kit—intended to be iterated on.

### Testing
- Unit tests can bypass JWT validation by setting `TEST_BYPASS_AUTH=true`. This injects a synthetic user with the `agent` role so auth-protected routes can be exercised without a JWKS.
- Handlers depend on a database interface and an object storage interface, enabling mocks/fakes in tests without external services.
- Run all tests from repo root: `go test ./...`

## Continuous Integration

GitHub Actions builds the API and worker images, runs `cd cmd/api && go mod tidy && TEST_BYPASS_AUTH=true go test -v .`, and lints and packages the Helm chart on every push and pull request.
