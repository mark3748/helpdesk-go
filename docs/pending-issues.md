# Improvement Checklist (Prioritized)

Track and check off pending improvements. Items are grouped by priority and reference files/areas to update. Keep this list current as work proceeds.

## High Priority

- [ ] Harden filesystem object store to prevent path traversal in `cmd/api/app/app.go` (`FsObjectStore`). Align with safe path cleaning/prefix checks used in `cmd/api/main.go`’s `fsObjectStore`.
- [ ] Fix `internal/sla/sla.go` DB loader bug (variable shadowing: `rows` vs `hrows`, incorrect defer). Add a unit test to cover holiday loading and ensure both query cursors close correctly.
- [ ] Unify duplicated API composition: consolidate config, app wiring, and object store between `cmd/api/main.go` and modular packages (`cmd/api/app`, `cmd/api/auth`, feature handlers). Make `main.go` a thin bootstrapper.
- [ ] Strengthen JWT/OIDC validation in `cmd/api/auth`: verify issuer/audience, enforce allowed signing algorithms, handle clock skew, and ensure `exp`/`nbf` checks. Document required claims.
- [ ] Consolidate login cookie handling: prefer single `hd_auth` cookie with `HttpOnly`, `SameSite=Lax`, and `Secure` in prod; remove legacy `auth` cookie. Ensure consistent behavior across local and OIDC modes.
- [ ] Ensure `cmd/api/handlers/events.go` compiles cleanly (brace/structure sanity) and matches tests; keep heartbeat/backpressure behavior intact.
- [ ] Align Go versions across codebase: `go.mod`, Dockerfiles, and CI (`.github/workflows/ci.yml`) to a single supported version.

## Medium Priority

- [ ] Standardize rate limiting: prefer Redis-backed limiter (`internal/ratelimit`) for login, ticket create, and attachments; remove ad‑hoc in‑memory limiter to ensure consistency across replicas.
- [ ] Add context timeouts to DB, Redis, MinIO, and JWKS operations; propagate request contexts to DB calls in handlers.
- [ ] Improve JWKS handling: periodic refresh with backoff/metrics; validate KID/alg robustly; fail closed with clear errors when JWKS unavailable.
- [ ] Helm: move sensitive config to Kubernetes Secrets (DB URL, `AUTH_LOCAL_SECRET`, `SMTP_*`, `MINIO_*`); wire via `valueFrom` in templates.
- [ ] Make Docker builds reproducible: vendor/pin Swagger UI assets instead of fetching at build time, or checksum‑verify downloads.
- [ ] Multi-arch builds: parameterize `GOARCH` and use Buildx matrix in CI for `linux/amd64,linux/arm64` images.
- [ ] Expand tests: path traversal attempts for object store; JWT claim validation; `s3` presign error paths; `/attachments/upload/:objectKey` validates keys; add SLA calendar loader tests (holidays/hours).
- [ ] Observability: unify structured request logging (use `cmd/api/app/middleware.go` logger everywhere), add Prometheus counters for ticket create/update, auth failures, and rate-limit rejections.
- [ ] Tighten CORS headers: minimize `Access-Control-Allow-Headers` to required set; keep `Vary: Origin`; document `ALLOWED_ORIGINS` usage and risks.

## Low Priority

- [ ] Worker SMTP: add STARTTLS/TLS support and dial/write timeouts; configurable `smtpSendMail` transport.
- [ ] Add a basic readiness/liveness indicator for the worker (e.g., Redis ping or lightweight HTTP endpoint) and wire probes if desired.
- [ ] Set default resource requests/limits in Helm for API, worker, and frontends; add examples in `helm/helpdesk/examples`.
- [ ] Provide Makefile targets for common workflows (`build`, `test`, `docker`, `lint-helm`).
- [ ] Documentation: expand auth modes section (cookie name/flags, OIDC claims), rate limiting behavior/dependencies, and deployment hardening tips.

## Notes

- Keep items scoped and linked to PRs. For multi-part refactors (e.g., unifying `main.go` with modular packages), split into small PRs to ease review.

