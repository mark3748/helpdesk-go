# Helpdesk (Go) — MVP Scaffold

A minimal FootPrints-style ticketing system scaffold in Go, with PostgreSQL migrations and a Helm chart for Kubernetes.

## What you get
- Go API service (Gin) with basic ticket endpoints
- Embedded SQL migrations (goose) for Postgres (JSONB custom fields, FTS index)
- Dockerfiles for API & Worker
- Helm chart skeleton (api + worker deployments, service, ingress, config)

## Quickstart (local, Docker Compose-free)
1. Provision Postgres 14+ and create a db (e.g., `helpdesk`). Save the URL:
   ```
   export DATABASE_URL=postgres://user:pass@localhost:5432/helpdesk?sslmode=disable
   ```
2. Build API:
   ```bash
   cd api/cmd/api
   go mod tidy
   go build -o ../../../bin/api
   ../../../bin/api
   ```
3. Hit health:
   ```
   curl http://localhost:8080/healthz
   ```

## API (MVP)
- `GET /healthz`
- `GET /me` (authenticated user info and roles)
- `GET /tickets`
- `POST /tickets`  (title, description, priority, urgency, category, subcategory, custom_json)
- `GET /tickets/:id`
- `PATCH /tickets/:id` (status, assignee_id, priority, urgency, scheduled_at, due_at, custom_json)
- `POST /tickets/:id/comments` (body_md, is_internal)

## Helm (Kubernetes)
1. Set values in `helm/helpdesk/values.yaml` (hostnames, secrets, external DB/Redis/MinIO).
2. Package & install:
   ```bash
   helm upgrade --install helpdesk ./helm/helpdesk -n helpdesk --create-namespace
   ```

### Notes
- Auth uses OIDC JWTs with role checks (agents required for ticket updates).
- Worker service is scaffolded but not wired to a queue yet.
- This is a starter kit—intended to be iterated on.
