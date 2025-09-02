# Helpdesk (Go) — MVP Scaffold

A minimal FootPrints-style ticketing system scaffold in Go, with PostgreSQL migrations and a Helm chart for Kubernetes.

## What You Get
- Go API service (Gin): tickets, comments, attachments, watchers, exports, metrics.
- Embedded SQL migrations (goose) for Postgres (JSONB custom fields, FTS index, CSAT columns).
- Redis-driven worker: email notifications (SMTP), SLA clock updates, optional IMAP poller.
- Object storage: S3-compatible (MinIO) or local filesystem for attachments.
- Web apps: agent workspace (`web/agent`) and requester portal (`web/requester`).
- Dockerfiles for API & Worker and a Helm chart (deployments, service, ingress, config).

## Quickstart (local, Docker Compose-free)
1. Provision Postgres 14+ and create a db (e.g., `helpdesk`). Set env:
   ```
   export DATABASE_URL=postgres://user:pass@localhost:5432/helpdesk?sslmode=disable
   # Optional/Dev: enable local auth with cookie-based JWT
   export AUTH_MODE=local
   export AUTH_LOCAL_SECRET=dev-secret
   # Optional: use local filesystem for attachments (instead of MinIO)
   export FILESTORE_PATH=$PWD/data
   # Optional: Redis for worker/metrics (API logs if missing)
   export REDIS_ADDR=localhost:6379
   ```
2. Build and run API:
   ```bash
   cd cmd/api && go mod tidy && go build -o ../../bin/api
   ../../bin/api
   ```
3. Health check:
   ```
   curl http://localhost:8080/healthz
   ```

Notes:
- In `AUTH_MODE=local` and `ENV=dev`, an admin user is auto-seeded. Set `ADMIN_PASSWORD` to control it; otherwise a random password is generated and logged once.
- For MinIO, set `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL` and omit `FILESTORE_PATH`.

## Local Development

- Build binaries:
  ```bash
  cd cmd/api && go mod tidy && go build -o ../../bin/api
  cd cmd/worker && go mod tidy && go build -o ../../bin/worker
  ```
- Run tests (all packages):
  ```bash
  TEST_BYPASS_AUTH=true go test -cover ./...
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

## API Endpoints
- `GET /healthz`
- Auth (local mode): `POST /login`, `POST /logout`
- `GET /me` – authenticated user info and roles
- Tickets: `GET /tickets` (filters: `status,priority,team,assignee,search`), `POST /tickets`, `GET /tickets/:id`, `PATCH /tickets/:id`
- Comments: `GET /tickets/:id/comments`, `POST /tickets/:id/comments`
- Attachments: `GET /tickets/:id/attachments`, `POST /tickets/:id/attachments`, `DELETE /tickets/:id/attachments/:attID`
- Watchers: `GET /tickets/:id/watchers`, `POST /tickets/:id/watchers`, `DELETE /tickets/:id/watchers/:userID`
- Exports: `POST /exports/tickets` (CSV)
- CSAT: `GET /csat/:token?score=good|bad` (public)
- Metrics (agent role): `GET /metrics/sla`, `GET /metrics/resolution`, `GET /metrics/tickets`

See `docs/api.md` for detailed status codes, request/response bodies, and models. For tooling and client generation, use `docs/openapi.yaml`. A live documentation UI is served at `/docs` when the API is running; the spec is served at `/openapi.yaml` and is packaged in the Docker image. For metrics visualization guidance, see `docs/grafana.md`.

## Helm (Kubernetes)
1. Set values in `helm/helpdesk/values.yaml` (hostnames, secrets, external DB/Redis/MinIO).
2. Package & install:
   ```bash
   helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace
   ```

### Notes
- Auth supports OIDC (JWKS) and a dev-friendly local mode (`AUTH_MODE=local`). `TEST_BYPASS_AUTH=true` bypasses JWTs in tests.
- Worker consumes Redis jobs, sends SMTP email using templates in `cmd/worker/templates/`, updates SLA clocks, and can poll IMAP if configured.
- Object storage is optional; when unconfigured, set `FILESTORE_PATH` to store attachments locally.
- This is a starter kit—intended to be iterated on.

### Testing
- Unit tests can bypass JWT validation by setting `TEST_BYPASS_AUTH=true`. This injects a synthetic user with the `agent` role so auth-protected routes can be exercised without a JWKS.
- Handlers depend on database and object storage interfaces, enabling fakes in tests without external services.
- Run all tests from repo root: `go test ./...`

## Continuous Integration

GitHub Actions builds the API and worker images, runs tests (`TEST_BYPASS_AUTH=true go test ./...`), and lints/packages the Helm chart on every push and pull request.

## Docker Compose
- Start stack (Postgres, Redis, API, Worker):
  ```bash
  docker compose up -d db redis api worker
  ```
- Agent UI (optional dev server):
  ```bash
  docker compose up web
  ```
- Default ports: API `http://localhost:8080`, Agent UI `http://localhost:5173`, Postgres `5432`, Redis `6379`.
- Filesystem attachments are stored under `./data` (mounted to `/data`) when `FILESTORE_PATH` is used.
- Compose uses `AUTH_MODE=local` for quick start and seeds an admin (set `ADMIN_PASSWORD`).

## Environment Variables

API (cmd/api):
- `ADDR`: bind address (default `:8080`).
- `ENV`: `dev` or `prod`.
- `DATABASE_URL`: Postgres connection string.
- `REDIS_ADDR`: Redis address (optional but recommended).
- `OIDC_ISSUER`, `OIDC_JWKS_URL`: OIDC settings for JWT validation.
- `AUTH_MODE`: `oidc` or `local`.
- `AUTH_LOCAL_SECRET`: HMAC secret for local auth cookie JWTs.
- `ADMIN_PASSWORD`: initial admin password in dev local-auth mode.
- `FILESTORE_PATH`: local path for attachments (filesystem store).
- `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`: S3/MinIO settings.
- `TEST_BYPASS_AUTH`: set `true` in tests to bypass JWT and inject a test user.
- `OPENAPI_SPEC_PATH`: optional path to the OpenAPI spec for serving `/openapi.yaml` in local dev (default packaged in Docker at `/opt/helpdesk/docs/openapi.yaml`).

Worker (cmd/worker):
- `DATABASE_URL`, `REDIS_ADDR`, `ENV`.
- SMTP: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`.
- IMAP (optional): `IMAP_HOST`, `IMAP_USER`, `IMAP_PASS`, `IMAP_FOLDER`.
- MinIO/S3: `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`.

Web – Agent (web/agent):
- `VITE_API_TARGET`: API origin for dev proxy (defaults to `http://localhost:8080`).

Web – Requester (web/requester):
- `VITE_API_BASE`: API base path (default `/api`).
- `VITE_OIDC_AUTHORITY`: OIDC provider URL.
- `VITE_OIDC_CLIENT_ID`: OIDC client ID.

## Web Apps

Agent Workspace (React):
- Dev server:
  ```bash
  cd web/agent
  npm install
  # optional: point to a non-default API origin
  VITE_API_TARGET=http://localhost:8080 npm run dev
  ```
  Open `http://localhost:5173`. The dev server proxies `/api` to `VITE_API_TARGET`.

Requester Portal (React):
- Dev server:
  ```bash
  cd web/requester
  npm install
  export VITE_API_BASE=/api
  export VITE_OIDC_AUTHORITY=... 
  export VITE_OIDC_CLIENT_ID=...
  npm run dev
  ```
  See `web/requester/README.md` for details.

Compose-based UI:
- `docker compose up web` runs the agent dev server in a container with `VITE_API_TARGET` set to the API service.

## Troubleshooting
- Postgres connection/migrations: ensure `DATABASE_URL` is correct and the DB is reachable. Migrations auto-run at startup; logs will show goose errors if any.
- Redis unavailable: the API/worker log a ping error but continue; features that enqueue/process jobs may be no-ops until Redis is up.
- Attachments/uploads: configure either MinIO (`MINIO_*`) or a local path via `FILESTORE_PATH`. Permission issues on `FILESTORE_PATH` can cause 500s.
- Exports URL: ticket export uploads require an object store. With MinIO configured, the response includes a URL. With filesystem store, a public URL is not generated.
- Auth errors: for OIDC, set `OIDC_JWKS_URL` (and `OIDC_ISSUER` if enforcing issuer). For local auth, set `AUTH_LOCAL_SECRET` and optionally `ADMIN_PASSWORD`.
- Port conflicts: default ports are 8080 (API), 5173 (Agent UI), 5432 (Postgres), 6379 (Redis). Adjust `ADDR` or container port mappings as needed.
