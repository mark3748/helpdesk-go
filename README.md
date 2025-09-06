# Helpdesk (Go) — MVP Scaffold

A minimal FootPrints-style ticketing system scaffold in Go, with PostgreSQL migrations and a Helm chart for Kubernetes.

## What You Get
- Go API service (Gin): tickets, comments, attachments, watchers, exports, metrics.
- Embedded SQL migrations (goose) for Postgres (JSONB custom fields, FTS index, CSAT columns).
- Redis-driven worker: email notifications (SMTP), SLA clock updates, optional IMAP poller.
- Object storage: S3-compatible (MinIO) or local filesystem for attachments.
- Web apps: internal workspace (`web/internal`) and requester portal (`web/requester`).
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
- CSAT: `GET /csat/:token` form, `POST /csat/:token` score=good|bad (public)
- Metrics (agent role): `GET /metrics/sla`, `GET /metrics/resolution`, `GET /metrics/tickets`
- Prometheus metrics: `GET /metrics` (no auth)
- Events: `GET /events` (SSE) with heartbeat comments `:hb` ~every 30s

See `docs/api.md` for detailed status codes, request/response bodies, and models. For tooling and client generation, use `docs/openapi.yaml`. A live documentation UI is served at `/docs` when the API is running; the spec is served at `/openapi.yaml` and is packaged in the Docker image. For metrics visualization guidance, see `docs/grafana.md`.

## Helm (Kubernetes)
1. Set values in `helm/helpdesk/values.yaml` (hostnames, secrets, external DB/Redis/MinIO).
2. Package & install:
   ```bash
   helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace
   ```

Examples:
- Local auth, both frontends, /api proxy for requester:
  ```bash
  helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk \
    -f helm/helpdesk/examples/values-local-auth.yaml
  ```
- To supply secrets out-of-band, create a `Secret` with the local auth keys and point the chart to it:
  ```bash
  kubectl create secret generic helpdesk-auth \
    --from-literal=AUTH_LOCAL_SECRET=change-me \
    --from-literal=ADMIN_PASSWORD=admin
  helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace \
    -f helm/helpdesk/examples/values-local-auth.yaml \
    --set secrets.name=helpdesk-auth --set secrets.enabled=true
  ```
- OIDC (Authentik/Keycloak), both frontends, CORS + JWKS configured:
  ```bash
  helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk \
    -f helm/helpdesk/examples/values-oidc.yaml
  ```

### Notes
- Auth supports OIDC (JWKS) and a dev-friendly local mode (`AUTH_MODE=local`). `TEST_BYPASS_AUTH=true` bypasses JWTs in tests.
- Worker consumes Redis jobs, sends SMTP email using templates in `cmd/worker/templates/`, updates SLA clocks, and can poll IMAP if configured.
- Object storage is optional; when unconfigured, set `FILESTORE_PATH` to store attachments locally.
- This is a starter kit—intended to be iterated on.

### Secrets and Sensitive Config
- The chart can read sensitive env vars from a Kubernetes Secret. Enable and either let the chart create one or reference an existing Secret:
  ```yaml
  secrets:
    enabled: true
    # Use an existing Secret by name (skip creation):
    # name: helpdesk-secrets
    data:
      DATABASE_URL: "postgres://user:pass@postgres:5432/helpdesk?sslmode=disable"
      AUTH_LOCAL_SECRET: "change-me"
      ADMIN_PASSWORD: "admin"
      # Optional:
      # REDIS_ADDR: "redis-master.helpdesk.svc.cluster.local:6379"
      # MINIO_ENDPOINT: "minio.helpdesk.svc:9000"
      # MINIO_ACCESS_KEY: "..."
      # MINIO_SECRET_KEY: "..."
      # MINIO_BUCKET: "attachments"
  ```
- You can combine `secrets.data` with `env:` in `values.yaml`. Explicit `env:` keys win if duplicated.

### Private Registry Pull
If your images are in a private registry (e.g., GHCR), add an imagePullSecret and reference it:
```yaml
imagePullSecrets:
  - ghcr-pull
```

### Persistence for Attachments (Filesystem Store)
If you are not using MinIO/S3, you can enable a PVC and mount it at `FILESTORE_PATH`:
```yaml
persistence:
  enabled: true
  mountPath: "/data"
  size: 5Gi
#  existingClaim: helpdesk-data   # optionally use an existing PVC
```
Then set the app to use that path (via Secret or `env:`):
```yaml
secrets:
  enabled: true
  data:
    FILESTORE_PATH: "/data"
```
On startup, `/readyz` verifies the object store (MinIO bucket exists or filesystem path is writable).

### Feature Flags (/features)
The API exposes `GET /api/features` to advertise simple capabilities to the UI. Current fields:
- `attachments`: true when object storage is configured (MinIO or filesystem). The internal UI disables the upload button when `attachments=false` and avoids presign calls.

### SSE (Events)
`GET /api/events` streams Server-Sent Events with heartbeat comments (`:hb`) roughly every 30s. For Traefik/Nginx ingress, ensure streaming is not buffered and timeouts are sufficient. The API sets `X-Accel-Buffering: no` and sends an initial heartbeat immediately. If streaming is not possible in some dev proxies, the UI falls back to polling.

### Testing
- Unit tests can bypass JWT validation by setting `TEST_BYPASS_AUTH=true`. This injects a synthetic user with the `agent` role so auth-protected routes can be exercised without a JWKS.
- Handlers depend on database and object storage interfaces, enabling fakes in tests without external services.
- Run all tests from repo root: `go test ./...`
- The project targets **≥70%** test coverage across packages; pull requests should not drop below this threshold.

## Continuous Integration

GitHub Actions builds the API and worker images, runs tests (`TEST_BYPASS_AUTH=true go test ./...`), and lints/packages the Helm chart on every push and pull request.

## Docker Compose
- Start stack (Postgres, Redis, API, Worker):
  ```bash
  docker compose up -d db redis api worker
  ```
- Internal UI (dev server):
  ```bash
  docker compose up internal
  ```
  Runs at http://localhost:5175 with an `/api` proxy to the API service.

- Requester UI (dev server):
  ```bash
  docker compose up requester
  ```
  Runs at http://localhost:5174 with an `/api` proxy to the API service.
  - OIDC: set `VITE_OIDC_AUTHORITY` and `VITE_OIDC_CLIENT_ID` to enable real login.
  - Local auth fallback: when OIDC is unset, a simple login form posts to `/api/login` and uses cookie auth.
- Default ports: API `http://localhost:8080`, Internal UI `http://localhost:5175`, Postgres `5432`, Redis `6379`.
- Filesystem attachments are stored under `./data` (mounted to `/data`) when `FILESTORE_PATH` is used.
- Compose uses `AUTH_MODE=local` for quick start and seeds an admin (set `ADMIN_PASSWORD`).

## Environment Variables

API (cmd/api):
- `ADDR`: bind address (default `:8080`).
- `ENV`: `dev` or `prod`.
- `DATABASE_URL`: Postgres connection string.
- `REDIS_ADDR`: Redis address (optional but recommended).
- `OIDC_ISSUER`, `OIDC_JWKS_URL`: OIDC settings for JWT validation.
- `OIDC_GROUP_CLAIM`: JWT claim name containing group roles (default `groups`).
- `AUTH_MODE`: `oidc` or `local`.
- `AUTH_LOCAL_SECRET`: HMAC secret for local auth cookie JWTs.
- `ADMIN_PASSWORD`: initial admin password in dev local-auth mode.
- `FILESTORE_PATH`: local path for attachments (filesystem store).
- `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`: S3/MinIO settings.
- `REDIS_TIMEOUT_MS`: per-call Redis timeout in milliseconds (default 2000). Applies to readiness ping and queue operations.
- `OBJECTSTORE_TIMEOUT_MS`: per-call object store timeout in milliseconds (default 10000). Applies to MinIO/S3 presign/put/stat and filesystem operations.
- `ALLOWED_ORIGINS`: comma-separated origins allowed for cross-origin requests (default none).
  Example: `ALLOWED_ORIGINS=https://helpdesk.example.com,https://portal.example.com`.
  Avoid broad patterns or untrusted origins; permissive values let other sites read authenticated responses.
- `TEST_BYPASS_AUTH`: set `true` in tests to bypass JWT and inject a test user.
- `OPENAPI_SPEC_PATH`: optional path to the OpenAPI spec for serving `/openapi.yaml` in local dev (default packaged in Docker at `/opt/helpdesk/docs/openapi.yaml`).
- `LOG_PATH`: directory for API log output (default system temp dir, e.g. `/tmp`). Falls back to stdout if unwritable.
- `RATE_LIMIT_LOGIN`: max login/logout requests per minute per IP (default unlimited).
- `rate_limit_rejections_total{route=...}`: Prometheus counter exported by the API indicating the number of requests rejected by rate limiting for a given route label (e.g., `login`, `tickets_create`, `attachments_presign`).
- `RATE_LIMIT_TICKETS`: max ticket creation requests per minute per user.
- `RATE_LIMIT_ATTACHMENTS`: max attachment upload/download requests per minute per user.

Worker (cmd/worker):
- `DATABASE_URL`, `REDIS_ADDR`, `ENV`.
- SMTP: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASS`, `SMTP_FROM`.
- IMAP (optional): `IMAP_HOST`, `IMAP_USER`, `IMAP_PASS`, `IMAP_FOLDER`.
- MinIO/S3: `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`.
- `LOG_PATH`: directory for worker log output (default system temp dir, e.g. `/tmp`). Falls back to stdout if unwritable.

Web – Internal (web/internal):
- `VITE_API_TARGET`: API origin for dev proxy (defaults to `http://localhost:8080`).

Web – Requester (web/requester):
- `VITE_API_BASE`: API base path (default `/api`).
- `VITE_OIDC_AUTHORITY`: OIDC provider URL.
- `VITE_OIDC_CLIENT_ID`: OIDC client ID.

## Web Apps

Internal Workspace (React):
- Dev server:
  ```bash
  cd web/internal
  npm install
  # optional: point to a non-default API origin
  VITE_API_TARGET=http://localhost:8080 npm run dev
  ```
  Default dev port is `5173`. When using docker-compose, the internal UI runs on `http://localhost:5175`.

Requester Portal (React):
- Dev server:
  ```bash
  cd web/requester
  npm install
  export VITE_API_BASE=/api
  # OIDC mode
  # export VITE_OIDC_AUTHORITY=...
  # export VITE_OIDC_CLIENT_ID=...
  npm run dev
  ```
  See `web/requester/README.md` for details.

Compose-based UI:
- `docker compose up internal` runs the internal dev server in a container with `VITE_API_TARGET` set to the API service.

## Troubleshooting
- Postgres connection/migrations: ensure `DATABASE_URL` is correct and the DB is reachable. Migrations auto-run at startup; logs will show goose errors if any.
- Redis unavailable: the API/worker log a ping error but continue; features that enqueue/process jobs may be no-ops until Redis is up.
- Attachments/uploads: configure either MinIO (`MINIO_*`) or a local path via `FILESTORE_PATH`. Permission issues on `FILESTORE_PATH` can cause 500s. In local dev without MinIO, the API uses an internal upload URL for presigned uploads.
- Compose data dir permissions (uploads 500): the API image runs as a nonroot user (UID 65532). If `./data` is owned by `root:root`, the API can’t write and uploads will 500. Fix by aligning ownership:
  ```bash
  mkdir -p data
  sudo chown -R 65532:65532 data
  sudo chmod -R u+rwX data
  ```
  Dev-only quick fix:
  ```bash
  chmod -R 777 data
  ```
  Ephemeral alternative: set `FILESTORE_PATH=/tmp/files` for the `api` service and remove the `./data:/data` volume (attachments won’t persist across restarts). On SELinux, keep the `:Z` label on bind mounts.
- Compose data dir permissions (uploads 500): the API image runs as a nonroot user (UID 65532). If `./data` is owned by `root:root`, the API can’t write and uploads will 500. Fix by aligning ownership:
  ```bash
  mkdir -p data
  sudo chown -R 65532:65532 data
  sudo chmod -R u+rwX data
  ```
  Dev-only quick fix:
  ```bash
  chmod -R 777 data
  ```
  Ephemeral alternative: set `FILESTORE_PATH=/tmp/files` for the `api` service and remove the `./data:/data` volume (attachments won’t persist across restarts). On SELinux, keep the `:Z` label on bind mounts.
- Exports URL: ticket export uploads require an object store. With MinIO configured, the response includes a URL. With filesystem store, a public URL is not generated.

## Recent Changes / Merge Notes
- Migrations: removed duplicate goose migration `0006_ticket_events.sql`; `ticket_events` unified to `(event_type, payload, created_at)`.
- Requesters: `tickets.requester_id` now references `requesters(id)`. The API auto-creates a matching `requesters` row for the current user when `requester_id` is omitted (requester portal flow).
- Events: SSE endpoint (`/events`) heartbeats even when Redis is unavailable.
- Attachments: filesystem store now supports presign + direct upload via an internal endpoint; MinIO continues to use S3 presigned URLs.
- Admin endpoints: `/users`, `/roles`, `/users/:id`, `/users/:id/roles` wired for internal UI.
- User settings: `/me/profile` (GET/PATCH) and `/me/password` (POST) for local auth.
- API prefix: all routes are mounted at both `/...` and `/api/...` for dev proxies and clients.

Breaking considerations:
- If you have a pre-existing DB, run the migrations in order. Ensure `0007_requesters_queues_ticket_events.sql` applies cleanly and `tickets.requester_id` points to `requesters`. For dev data, a fresh compose up is easiest.

Security/config:
- In prod, set `ALLOWED_ORIGINS` to the specific UI origins that may call the API.
  Avoid wildcards or public origins to prevent cross-site request forgery and data leaks.
  Set `LOG_PATH` to a writable directory. Use secure cookies and OIDC.
- Auth errors: for OIDC, set `OIDC_JWKS_URL` (and `OIDC_ISSUER` if enforcing issuer). For local auth, set `AUTH_LOCAL_SECRET` and optionally `ADMIN_PASSWORD`.
- Port conflicts: default ports are 8080 (API), 5173 (Internal UI dev), 5175 (Compose internal UI), 5432 (Postgres), 6379 (Redis). Adjust `ADDR` or container port mappings as needed.
