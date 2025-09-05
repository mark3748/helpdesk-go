# API Reference

This document describes the HTTP API exposed by the Helpdesk service. Unless noted as public, endpoints require authentication. In production, use OIDC (JWKS). For development, `AUTH_MODE=local` enables cookie-based login via `/login`.

Base URL examples:
- Local API: `http://localhost:8080`
- Agent dev server proxy: requests to `/api/...` are proxied to the API in dev.

## CORS
Browser clients must originate from an allowed origin. Configure `ALLOWED_ORIGINS`
as a comma-separated list, for example:

`ALLOWED_ORIGINS=https://helpdesk.example.com,https://portal.example.com`

Preflight responses only permit `Authorization`, `Content-Type`, and
`X-Requested-With` headers. Using broad or wildcard origins can let malicious sites
read authenticated responses; limit the list to trusted domains.

## Authentication
- OIDC (default): Send `Authorization: Bearer <JWT>`. The API validates against `OIDC_JWKS_URL` and optional `OIDC_ISSUER`.
- Local (dev): `POST /login` issues an HttpOnly cookie. Include cookie on subsequent requests. `POST /logout` clears it.

## Conventions
- Content type: JSON unless specified.
- Time format: RFC3339.
- Errors: `{ "error": "message" }` or validation errors `{ "errors": { "field": "message" } }`.

## Endpoints

Health
- GET `/livez` → 200 OK `{ "ok": true }`
- GET `/readyz` → 200 OK `{ "ok": true }` | 500
- GET `/healthz` → 200 OK `{ "ok": true }`

Auth (local mode only)
- POST `/login` body `{ username, password }` → 200 OK `{ ok:true }` | 400 | 401 | 500
- POST `/logout` → 200 OK `{ ok:true }`

User
- GET `/me` → 200 `{ id, external_id, email, display_name, roles }` | 401

Requesters
- POST `/requesters` body `{ email, display_name }` → 201 `{ id, email, display_name }` | 400 | 500
- GET `/requesters/:id` → 200 `{ id, email, display_name }` | 404
- PATCH `/requesters/:id` body `{ email?, display_name? }` → 200 `{ id, email, display_name }` | 400 | 404 | 500

Tickets
- GET `/tickets` query `status,priority,team,assignee,search` → 200 `[Ticket]` | 500
- POST `/tickets` body `{ title, description, requester_id, priority, urgency?, category?, subcategory?, custom_json? }` → 201 `{ id, number, status }` | 400 | 500
  - `urgency` 1-4
  - `custom_json` object of additional fields
- GET `/tickets/:id` → 200 `Ticket` | 404
- PATCH `/tickets/:id` (agent role) body partial `{ status?, assignee_id?, priority?, urgency?, scheduled_at?, due_at?, custom_json? }` → 200 `{ ok:true }` | 400 | 500

Comments
- GET `/tickets/:id/comments` → 200 `[Comment]` | 500
- POST `/tickets/:id/comments` body `{ body_md, is_internal, author_id }` → 201 `{ id }` | 400 | 500

Attachments
- GET `/tickets/:id/attachments` → 200 `[{ id, filename, bytes, mime, created_at }]` | 500
- POST `/tickets/:id/attachments/presign` `{ filename, bytes, mime? }` → 201 `{ upload_url, headers, attachment_id }` | 400 | 500
- POST `/tickets/:id/attachments` `{ attachment_id, filename, bytes, mime? }` → 201 `{ id }` | 400 | 500
- DELETE `/tickets/:id/attachments/:attID` → 200 `{ ok:true }` | 404 | 500

Watchers
- GET `/tickets/:id/watchers` → 200 `[user_id]` | 500
- POST `/tickets/:id/watchers` body `{ user_id }` → 201 `{ ok:true }` | 400 | 500
- DELETE `/tickets/:id/watchers/:userID` → 200 `{ ok:true }` | 500

Customer Satisfaction (CSAT)
- GET `/csat/:token` (public) → 200 HTML form | 500
- POST `/csat/:token` score=good|bad → 200 `{ ok:true }` | 400 | 404 | 500

Exports
- POST `/exports/tickets` (agent role) body `{ ids: [uuid] }` → 200 `{ url }` | 400 | 500
  - Requires configured object store. For MinIO/S3, `url` points to the uploaded CSV. With filesystem store, prefer fetching the file via your own mechanism since no HTTP endpoint serves it.

Metrics (agent role)
- GET `/metrics/sla` → 200 `{ total, met, sla_attainment }` | 500
- GET `/metrics/resolution` → 200 `{ avg_resolution_ms }` | 500
- GET `/metrics/tickets` → 200 `{ daily: [{ day, count }] }` | 500
- GET `/metrics` → Prometheus metrics (no auth)

Events
- GET `/events` (SSE) → stream of `ticket_created`, `ticket_updated`, `queue_changed`
  - `queue_changed` requires `admin` role
  - Heartbeat comments (`:hb`) sent ~every 30s keep the connection alive

## Models

Ticket
- Fields: `id, number, title, description, requester_id, assignee_id?, team_id?, priority, urgency?, category?, subcategory?, status, scheduled_at?, due_at?, source, custom_json, created_at, updated_at, sla?`

Comment
- Fields: `id, ticket_id, author_id, body_md, is_internal, created_at`

Requester
- Fields: `id, email, display_name`

