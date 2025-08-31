# Repository Guidelines

## Project Structure & Modules
- `cmd/api/`: Gin HTTP API, embedded SQL migrations in `migrations/`.
- `cmd/worker/`: Redis‑driven worker, SMTP email, optional IMAP poller; templates in `templates/`.
- `helm/helpdesk/`: Helm chart for API and worker.
- Root Dockerfiles: `Dockerfile.api`, `Dockerfile.worker`.
- Shared packages in `internal/` (e.g., `internal/sla`).
- Tests live next to code: `*_test.go` in `cmd/api`, `cmd/worker`, and `internal/`.

## Build, Test, and Development
- Build API: `cd cmd/api && go mod tidy && go build -o ../../bin/api`.
- Build worker: `cd cmd/worker && go mod tidy && go build -o ../../bin/worker`.
- Run API (requires Postgres): `DATABASE_URL=... REDIS_ADDR=... ../../bin/api`.
- Tests: `go test ./...` from repo root. Bypass JWT in tests with `TEST_BYPASS_AUTH=true`.
- Docker: `docker build -f Dockerfile.api -t helpdesk-api .` and similarly for worker.
- Helm (deploy): `helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace`.

## Coding Style & Naming
- Language: Go 1.23+ (module files set toolchain). Use `gofmt` defaults.
- Packages and files: lower_snake for files, `*_test.go` for tests. Handlers on `*App` receiver.
- Logging: `zerolog`. Avoid logging secrets.
- Design: small, cohesive funcs; dependency injection via interfaces (`DB`, `ObjectStore`).

## Testing Guidelines
- Framework: Go `testing` only. Name tests `TestXxx` in `*_test.go`.
- Run: `go test -cover ./...` from repo root.
- Patterns: use fakes for `DB` and stores; prefer table‑driven tests; assert JSON round‑trips for handlers.

## Commit & Pull Requests
- Commits: imperative, scoped messages (e.g., "api: add ticket PATCH validation").
- PRs: clear description, linked issue, test plan (commands + expected status codes), config notes (env vars), and screenshots/log excerpts when relevant.

## Security & Configuration Tips
- Config via env: `DATABASE_URL`, `REDIS_ADDR`, `OIDC_JWKS_URL`, `MINIO_*`, `SMTP_*`. Use `.env` locally; never commit secrets.
- Auth: use JWKS in prod; `TEST_BYPASS_AUTH=true` only for local/tests.
- Storage: MinIO optional; ensure bucket exists before enabling uploads.
