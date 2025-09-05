## Medium Priority Implementation Plan (PR Stubs)

Branch: `medium-priority-fixes`

This document outlines small, reviewable PRs to complete the Medium items in `docs/pending-issues.md`. Each PR lists scope, files, tests, and config notes.

### PR1 – Rate limiting standardization
- Scope: Ensure Redis-backed limiter is used for login, ticket create, and attachment flows.
- Files:
  - `cmd/api/main.go` – wrap routes with `internal/ratelimit` middleware and add rejection counter.
  - `cmd/api/app/middleware.go` (if needed) – metrics helper.
- Tests:
  - `cmd/api/main_test.go` and/or dedicated handler tests to assert 429 behavior and counter increments.
- Config:
  - `RATE_LIMIT_LOGIN`, `RATE_LIMIT_TICKETS`, `RATE_LIMIT_ATTACHMENTS`.

### PR2 – Context timeouts (DB)
- Scope: Introduce `DB_TIMEOUT_MS` and wrap DB calls with `context.WithTimeout`.
- Files: modular handlers under `cmd/api/*`, and helpers in `cmd/api/main.go`.
- Tests: simulate slow DB with context deadline exceeded; expect 504/500 mapping.

### PR3 – Timeouts (Redis/MinIO) + upload key validation
- Scope: Add `REDIS_TIMEOUT_MS`, `OBJECTSTORE_TIMEOUT_MS`; enforce UUID object key on `/attachments/upload/:objectKey`.
- Files: `cmd/api/attachments`, `cmd/api/main.go` queue/object store calls.
- Tests: invalid key returns 400; timeout soft-fail behavior (logged, request proceeds where safe).

### PR4 – JWKS hardening
- Scope: Backoff refresh with jitter, alg/kid validation, metrics, readyz dependency.
- Files: keyfunc wiring in `cmd/api/main.go` or extracted helper in `cmd/api/auth`.
- Tests: invalid alg/kid, skew, audience; readyz fails without cache.

### PR5 – Reproducible Swagger UI
- Scope: Vendor Swagger UI into `docker/swagger/` and COPY in `Dockerfile.api`.
- Files: `docker/swagger/*`, `Dockerfile.api`.
- Tests: CI build; `/api/docs` serves UI offline.

### PR6 – Multi-arch Buildx CI
- Scope: Release workflow builds `linux/amd64,linux/arm64`.
- Files: `.github/workflows/release.yml` (and `ci.yml` if needed), `Dockerfile.*` `--platform` args.
- Tests: manifests present; push on tag.

### PR7 – Observability counters
- Scope: Add counters for tickets, auth failures, rate-limit rejections, attachments.
- Files: `cmd/api/metrics` collectors; increments at handlers.
- Tests: unit tests using a Prometheus test registry.

### PR8 – CORS tightening
- Scope: Restrict `Access-Control-Allow-Headers`, ensure `Vary: Origin`, docs.
- Files: `cmd/api/main.go`, `README.md`/`docs/api.md`.
- Tests: OPTIONS preflight allowed vs blocked.

### PR9 – SLA tests expansion
- Scope: Holidays and business-hours edge cases.
- Files: `internal/sla/*_test.go`.
- Tests: table-driven scenarios for calendars.

---

General test guidance:
- Use `TEST_BYPASS_AUTH=true` for handler tests unless the test targets JWT behavior.
- Prefer table-driven tests and fakes for DB/object store; Redis can be a running service in integration tests via `docker-compose.yml`.

