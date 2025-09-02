# Functional Dev Environment Release: Attachments, Comments, Swagger UI

**Summary:**
- Agent: Show attachments with Download/Delete; refresh list on upload/delete; comment posting fixed (derive author server‑side).
- Requester: Typed API usage and attachment actions; comment request no longer needs author_id.
- API: Serve Swagger UI locally at /docs; add GET attachment download endpoint; update OpenAPI to reflect server‑side comment author.

**Testing:**
- `docker compose build api && docker compose up -d api`
- web/agent:
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
