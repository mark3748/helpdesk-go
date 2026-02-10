# Improvement Checklist

## Low Priority (Next Up)
- [ ] Worker SMTP: add STARTTLS/TLS support and dial/write timeouts; configurable `smtpSendMail` transport.

## Completed Improvements
### High Priority
- [x] Harden filesystem object store to prevent path traversal.
- [x] Fix `internal/sla/sla.go` DB loader bug.
- [x] Unify duplicated API composition (thin `main.go`).
- [x] Strengthen JWT/OIDC validation.
- [x] Consolidate login cookie handling (`hd_auth`).
- [x] Ensure `cmd/api/handlers/events.go` compiles cleanly.

### Medium Priority
- [x] Standardize rate limiting (Redis-backed).
- [x] Add context timeouts to DB, Redis, MinIO.
- [x] Improve JWKS handling (backoff, caching).
- [x] Helm: Secrets support and scheduling knobs.
- [x] Make Docker builds reproducible (vendor Swagger UI).
- [x] Multi-arch builds (AMD64/ARM64).
- [x] Expand tests: SLA calendars, JWT validation, upload keys.
- [x] Observability: unify request logging and Prometheus counters.
- [x] Tighten CORS headers.

### General / Infrastructure
- [x] Add readiness/liveness indicator for the worker.
- [x] Set default resource requests/limits in Helm.
- [x] Provide Makefile targets.
- [x] Documentation: auth modes, rate limiting, hardening.