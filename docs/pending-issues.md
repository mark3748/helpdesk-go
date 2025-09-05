# Improvement Checklist (Prioritized)

Track and check off pending improvements. Items are grouped by priority and reference files/areas to update. Keep this list current as work proceeds.

## High Priority

- [x] Harden filesystem object store to prevent path traversal in `cmd/api/app/app.go` (`FsObjectStore`). Align with safe path cleaning/prefix checks used in `cmd/api/main.go`’s `fsObjectStore`.
- [x] Fix `internal/sla/sla.go` DB loader bug (variable shadowing: `rows` vs `hrows`, incorrect defer). Add a unit test to cover holiday loading and ensure both query cursors close correctly.
- [x] Unify duplicated API composition: consolidate config, app wiring, and object store between `cmd/api/main.go` and modular packages (`cmd/api/app`, `cmd/api/auth`, feature handlers). Make `main.go` a thin bootstrapper.
- [x] Strengthen JWT/OIDC validation in `cmd/api/auth`: verify issuer (when configured) and enforce allowed signing algorithms; standard time-based claims validated by parser. Follow-up: audience config and clock skew.
- [x] Consolidate login cookie handling: prefer single `hd_auth` cookie with `HttpOnly`, `SameSite=Lax`, and `Secure` in prod; remove legacy `auth` cookie.
- [x] Ensure `cmd/api/handlers/events.go` compiles cleanly (brace/structure sanity) and matches tests; keep heartbeat/backpressure behavior intact.

Additional completions

- [x] Attachments: implement presign/finalize and internal upload flow in `cmd/api/attachments` and rewire routes; preserve MinIO redirect and filesystem serving; block traversal.
- [x] Tickets/Comments/Watchers/Metrics/Exports: rewire routes to modular handlers; remove legacy duplicates from `main.go`.
- [x] Auth: switch to `cmd/api/auth` middleware and role checks; remove legacy main auth handlers.
- [x] Align Go versions across codebase: `go.mod`, Dockerfiles, and CI (`.github/workflows/ci.yml`) to a single supported version.

## Medium Priority

- [ ] Standardize rate limiting: prefer Redis-backed limiter (`internal/ratelimit`) for login, ticket create, and attachments; remove ad‑hoc in‑memory limiter to ensure consistency across replicas.
- [ ] Add context timeouts to DB, Redis, MinIO, and JWKS operations; propagate request contexts to DB calls in handlers.
- [ ] Improve JWKS handling: periodic refresh with backoff/metrics; validate KID/alg robustly; fail closed with clear errors when JWKS unavailable.
- [x] Helm: move sensitive config to Kubernetes Secrets (DB URL, `AUTH_LOCAL_SECRET`, `SMTP_*`, `MINIO_*`); wire via `envFrom` in templates. Add `imagePullSecrets` and scheduling knobs. Add optional PVC for `FILESTORE_PATH`.
- [ ] Make Docker builds reproducible: vendor/pin Swagger UI assets instead of fetching at build time, or checksum‑verify downloads.
- [ ] Multi-arch builds: parameterize `GOARCH` and use Buildx matrix in CI for `linux/amd64,linux/arm64` images.
- [ ] Expand tests: path traversal attempts for object store; JWT claim validation; `s3` presign error paths; `/attachments/upload/:objectKey` validates keys; add SLA calendar loader tests (holidays/hours).
- [ ] Observability: unify structured request logging (use `cmd/api/app/middleware.go` logger everywhere), add Prometheus counters for ticket create/update, auth failures, and rate-limit rejections.
- [ ] Tighten CORS headers: minimize `Access-Control-Allow-Headers` to required set; keep `Vary: Origin`; document `ALLOWED_ORIGINS` usage and risks.

### Medium Implementation Plan (PR Checklist)

Tracking for the current branch `medium-priority-fixes`. See detailed stubs in `docs/plan-medium-prs.md`.

- [x] PR1 – Rate limiting standardization
  - [x] Ensure `internal/ratelimit` wraps: `POST /login`, `POST /tickets`, `POST /tickets/:id/attachments/presign`, `POST /tickets/:id/attachments`, `GET /tickets/:id/attachments/:attID` (download path gating optional)
  - [x] Remove any in‑memory limiters in modular handlers (confirm none remain)
  - [x] Add Prometheus counter `rate_limit_rejections_total{route=...}` and increment from limiter middleware
  - [x] Config notes: `RATE_LIMIT_LOGIN`, `RATE_LIMIT_TICKETS`, `RATE_LIMIT_ATTACHMENTS`
  - [x] Tests: burst requests hit 429 and counter increments

- [x] PR2 – Context timeouts (phase 1: DB)
  - [x] Add env knobs: `DB_TIMEOUT_MS`, default 5000
  - [x] Wrap DB calls in handlers with `context.WithTimeout` via `a.dbCtx(c)` helper
  - [x] Tests: simulated slow DB returns failure (readyz) using a slow DB stub

- [ ] PR3 – Timeouts (phase 2: Redis/MinIO) + upload key validation
  - [ ] Add `REDIS_TIMEOUT_MS` (2000) and `OBJECTSTORE_TIMEOUT_MS` (10000)
  - [ ] Apply to queue ops, limiter ping, presign/put/stat
  - [ ] Validate `/attachments/upload/:objectKey` requires UUID (return 400 otherwise)
  - [ ] Tests: Redis timeouts soft‑fail behavior; filesystem upload success + invalid key 400

- [ ] PR4 – JWKS hardening
  - [ ] Replace fixed ticker with jittered exponential backoff refresh; keep last‑good cache
  - [ ] Enforce allowed JWT algs; require/validate `kid` when present
  - [ ] Metrics: `jwks_refresh_total`, `jwks_refresh_errors_total`
  - [ ] `/readyz` fails when JWKS configured but cache empty
  - [ ] Tests: invalid alg/kid, clock skew, audience (when configured)

- [ ] PR5 – Reproducible Swagger UI assets
  - [ ] Vendor Swagger UI into `docker/swagger/` and COPY in `Dockerfile.api`
  - [ ] Remove any build‑time network fetches; checksum if download kept
  - [ ] Test: CI build offline; `/api/docs` serves UI

- [ ] PR6 – Multi‑arch Buildx CI
  - [ ] Update `.github/workflows/release.yml` to build `linux/amd64,linux/arm64`
  - [ ] Pass `--platform` via `docker/build-push-action`; parameterize `GOARCH`
  - [ ] Test: manifests created; push on tag

- [ ] PR7 – Observability counters
  - [ ] Counters: `tickets_created_total`, `tickets_updated_total`, `auth_failures_total`, `rate_limit_rejections_total`, `attachments_uploaded_total`
  - [ ] Increment at modular handler entry points
  - [ ] Tests: unit counter assertions with a test registry

- [ ] PR8 – CORS tightening
  - [ ] Restrict `Access-Control-Allow-Headers` to `Authorization, Content-Type, X-Requested-With`
  - [ ] Ensure `Vary: Origin` always set; 403 for disallowed origins
  - [ ] Docs: `ALLOWED_ORIGINS` risks and examples
  - [ ] Tests: OPTIONS preflight allowed/blocked

- [ ] PR9 – SLA tests expansion
  - [ ] Edge cases: holidays, non‑business hours, boundaries
  - [ ] Table‑driven tests in `internal/sla/*_test.go`


## Low Priority

- [ ] Worker SMTP: add STARTTLS/TLS support and dial/write timeouts; configurable `smtpSendMail` transport.
- [ ] Add a basic readiness/liveness indicator for the worker (e.g., Redis ping or lightweight HTTP endpoint) and wire probes if desired.
- [ ] Set default resource requests/limits in Helm for API, worker, and frontends; add examples in `helm/helpdesk/examples`.
- [ ] Provide Makefile targets for common workflows (`build`, `test`, `docker`, `lint-helm`).
- [ ] Documentation: expand auth modes section (cookie name/flags, OIDC claims), rate limiting behavior/dependencies, and deployment hardening tips.

## Notes

- Keep items scoped and linked to PRs. For multi-part refactors (e.g., unifying `main.go` with modular packages), split into small PRs to ease review.

## Recently Completed (This PR)

- [x] Ticket creation deduplication: Redis idempotency (resilient to Redis errors), DB advisory locks, and exact-content unique index (migration cleans existing dupes).
- [x] SSE stability: initial heartbeat and `X-Accel-Buffering: no`; graceful fallback when `http.Flusher` absent.
- [x] UI gating for attachments: `GET /api/features` exposes `attachments` flag; internal UI disables upload button with message when storage is not configured.
- [x] Helm: Secrets support (`values.secrets` + optional managed Secret), `imagePullSecrets`, `nodeSelector`/`tolerations`/`affinity`, and `persistence.enabled` PVC mounted at `FILESTORE_PATH`.
- [x] Tickets: return `201 Created` on success; include `description` in list/detail; auto-assign for `agent`/`admin` creators.
