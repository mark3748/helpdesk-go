# 0.4.3: Security Hardening + Production Polish

**Summary:**
- Enhanced security: Removed default credentials from Helm values and login UI in production builds
- Fixed CORS configuration for HTTPS deployments (resolves 403 authentication errors)
- Improved Helm template maintainability with cleaner conditional logic
- Chart version bumped to 0.4.3 with production-ready defaults

**Security Improvements:**
- Default `ADMIN_PASSWORD` removed from `helm/helpdesk/values.yaml` (now empty string)
- Login form no longer pre-fills credentials in production builds
- Development hints ("Dev default: admin / admin") hidden in production
- Development mode still shows defaults for easier local testing

**Fixes:**
- CORS `ALLOWED_ORIGINS` now correctly set to HTTPS for browser authentication
- Ingress template conditional logic refactored for better readability
- Resolved 403 Forbidden errors when accessing `/api/login` via HTTPS

**Testing:**
- Deploy with `helm upgrade --install helpdesk ./helm/helpdesk --values your-values.yaml`
- Verify login works at `https://your-domain.com` (no 403 errors)
- Confirm login form is clean (no pre-filled fields in production)
- Development mode: `npm run dev` still shows pre-filled admin/admin

**Migration:**
- Update your values file to explicitly set `ADMIN_PASSWORD` if using default charts
- Ensure `ALLOWED_ORIGINS` matches your HTTPS domain (not HTTP)

# Functional Dev Environment Release: Attachments, Comments, Swagger UI

**Summary:**
- Internal UI (replaces legacy Agent UI): Show attachments with Download/Delete; refresh list on upload/delete; comment posting fixed (derive author server‑side).
- Requester: Typed API usage and attachment actions; comment request no longer needs author_id.
- API: Serve Swagger UI locally at /docs; add GET attachment download endpoint; update OpenAPI to reflect server‑side comment author.

**Testing:**
- `docker compose build api && docker compose up -d api`
- web/internal:
    - `npm install`
    - `npm run gen:api`
    - Verify login persists; create ticket; add comment; upload and download attachment; delete attachment.
- web/requester:
    - `npm install`
    - `npm run gen:api`
    - Verify comments and attachments roundtrip with OIDC bearer token.

**Config notes:**
- /docs now uses packaged Swagger UI assets; no external CDN dependency.
- Download route streams from local filestore; redirects to MinIO if configured.
# Release Prep: Internal UI consolidation, Admin super-user, Helm frontends

Summary:
- Internal UI is now the primary frontend; legacy Agent UI and top-level shared module removed.
- Admin acts as a super-user across API and UI.
- Sidebar groups by role with expandable sections; Manager page included and visible to Admins.
- Helm now includes optional internal and requester frontends with clean path-based ingress.
- CI builds/pushes API, worker, internal-frontend, and optional requester-frontend images to GHCR.

Verification:
- docker compose up -d db redis api internal worker
- Login as admin/admin; verify Agent, Manager, Admin groups in sidebar and access to settings and queue manager.
- Helm: set image repositories/tags; internal UI served at /, API at /api, Swagger at /swagger.

# 0.4.0: Requester Local Auth + API/Chart polish

Summary:
- Requester portal gains a local-auth fallback (cookie-based). OIDC remains supported; app switches mode based on env.
- API mounts all routes under both `/...` and `/api/...` (dev proxies + clients), adds user settings, admin users/roles, and agent/manager metrics endpoints.
- Attachments work without MinIO using an internal presign/upload path; SSE continues heartbeating when Redis is down.
- Migrations cleaned up (removed duplicate 0006); `tickets.requester_id` now references `requesters(id)`.
- Helm chart bumped to 0.4.0; values include CORS `ALLOWED_ORIGINS` and auth toggles.

Highlights:
- New endpoints: `/me/profile` (GET/PATCH), `/me/password` (POST), `/metrics/agent`, `/metrics/manager`, `/users` (GET/POST), `/users/{id}` (GET), `/roles` (GET).
- Requester ticket create auto-links the current user as a requester when `requester_id` is omitted.
- Requester and Internal UIs proxy `/api` in dev and compose.

Upgrade notes:
- Ensure DB migrations applied through latest; `tickets.requester_id` must reference `requesters(id)`.
- For frontends calling the API from a different host, set `ALLOWED_ORIGINS`.
- For local dev without MinIO, set `FILESTORE_PATH` and ensure the path is writable.

Verification:
- Compose: `docker compose up -d db redis api internal requester worker`.
- Internal UI: login (local), create ticket, comments/attachments, admin pages, metrics pages.
- Requester: login (local), create ticket w/ attachment, view/detail comments.
