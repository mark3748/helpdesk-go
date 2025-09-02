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
