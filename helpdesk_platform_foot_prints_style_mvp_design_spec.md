# Helpdesk Platform – FootPrints-Style (MVP) Design Spec

**Owner:** You\
**Goal:** A modern, open-source, on‑prem/Kubernetes‑deployable helpdesk ticketing system modeled after BMC FootPrints workflows, tailored for IT Helpdesk/NOC/Field Techs.\
**MVP Scope:** Basic ticketing + SLAs + email/web channel + CSV reporting, with room to grow.

---

## 1) Product Overview

- **Primary audience:** Internal IT helpdesk (agents), requesters (employees), team leads/managers.
- **Scale (initial):** 1 agent, \~5 requesters (personal project); design to scale to small/mid teams later.
- **Channels:** Inbound **Email** + **Web portal**. Outbound **Email** + **Chat (later; optional Slack/Teams webhooks)**.
- **Core modules (MVP):**
  - Incident & Request Management ✔︎
  - Problem, Change, Release (scaffold only; post-MVP enable) ✔︎
  - Service Catalog (3 simple forms; no approvals) ✔︎
  - Knowledge Base (public + internal; minimal publish flow) ✔︎ ("maybe" → opt-in lightweight)
  - SLA Management ✔︎
  - Email-to-ticket ✔︎
  - Dashboards (good-to-have) ✔︎
  - **No** Discovery, Licenses in MVP.
- **Compliance/security:** None initially; **US data residency**. **Immutable audit logs**.

---

## 2) Opinionated Default Stack

### Go‑forward

- **Frontend:** React (Vite) SPA + PWA; Tailwind; React Router; React Query; Keyboard-first UX; Theming via CSS variables.
- **Backend:** Go (Gin/Fiber) REST + WebSockets; Modular monolith. Validation via go-playground/validator.
- **Auth:** OIDC (Authentik) via reverse proxy (oauth2-proxy / Traefik ForwardAuth). Optional LDAP sync job.
- **DB:** **Postgres** (JSONB for custom fields; **Postgres FTS** for search).
- **Cache/Queue:** **Valkey/Redis** (job queue, rate limiting, sessions, WS presence).
- **Object storage:** S3-compatible (MinIO) for attachments/inline images.
- **Email:** SMTP send; IMAP/POP3 poller (or direct Postfix/LSMTP webhook) for inbound → ticket.
- **Observability:** Prometheus (metrics), Grafana (dashboards), Loki (logs) optional.
- **Packaging/Deploy:** Docker images; Helm chart; K8s (ingress-nginx/Traefik); secrets via Sealed Secrets/External Secrets.

---

## 3) High-Level Architecture

```
[Requester Browser] --(HTTPS)--> [Web Portal (React SPA)] --(REST/WS)--> [API (Go)]
                                                   |                         |
                                                   |                         +--> [Postgres]
                                                   |                         +--> [Valkey/Redis]
                                                   |                         +--> [MinIO]
                                                   |                         +--> [Audit/WORM Store]
                                                   |
[Agent Console (React SPA)] <------ WebSockets <----+                         

[Email Inbound] -> [IMAP Poller/Mailhook] -> [Queue] -> [Ticket Ingest Worker] -> [API/DB]
[Outbound Email/Chat] <- [Notify Worker] <- [Queue] <- [Rules/SLA Engine]

[OIDC Provider (Authentik)] --(OIDC/OAuth2)--> [Ingress/ForwardAuth] --> [API]

[Prometheus/Grafana] <- metrics/logs <- [API/Workers/DB]
```

### Components

- **API Service:** AuthN (OIDC JWT), RBAC, tickets, comments, attachments, SLAs, notifications, search, exports.
- **Worker Service:** Email ingest, SLA timers, auto-ack/auto-close, scheduled jobs, CSAT link handling.
- **Web Portal:** Request submission, ticket view, KB browse, branded theme.
- **Agent Console:** Queue views, keyboard shortcuts, bulk edit, tabbed workspace, saved replies.
- **SLA Engine:** Multi-calendar aware stopwatch per ticket priority; pauses on configured states.
- **Search:** Postgres FTS (tsvector) on title, body, comments, categories; simple synonyms table.
- **Audit:** Append-only event log with hash-chaining (soft WORM) + periodic export to object storage.

---

## 4) Data Model (MVP)

> Postgres; snake\_case. **PKs** are UUID (v7 preferred). `created_at/updated_at` on all tables.

### 4.1 Identity & RBAC

- `users(id, external_id, email, display_name, active, locale, time_zone)`
- `roles(id, key)` with seeds: requester, agent, team\_lead, manager, admin
- `user_roles(user_id, role_id)`
- `teams(id, name, region_id, calendar_id)`
- `regions(id, name, calendar_id)`

### 4.2 Calendars & SLA

- `calendars(id, name, tz)`
- `business_hours(calendar_id, dow, start_sec, end_sec)`
- `holidays(calendar_id, date, label)`
- `sla_policies(id, name, priority, response_target_mins, resolution_target_mins, update_cadence_mins)`
- `ticket_sla_clocks(ticket_id, policy_id, response_elapsed_ms, resolution_elapsed_ms, last_started_at, paused, reason)`

### 4.3 Tickets & Workflow

- `tickets(id, number, title, description, requester_id, assignee_id, team_id, priority, urgency, category, subcategory, status, scheduled_at, due_at, source, csat_score, csat_token, deleted_at)`
- `ticket_custom_defs(id, key, label, type, required)`
- `ticket_custom_values(ticket_id, def_id, value_json)` **or** `tickets.custom_json` (JSONB; recommended simpler MVP)
- `ticket_comments(id, ticket_id, author_id, body_md, is_internal, via_email_msg_id)`
- `ticket_status_history(id, ticket_id, from_status, to_status, actor_id, note, at)`
- `ticket_watchers(user_id, ticket_id)`
- `attachments(id, ticket_id, uploader_id, object_key, filename, bytes, mime)` → stored in MinIO.

### 4.4 Email/Notifications/Integrations

- `email_inbound(id, raw_store_key, parsed_json, status, ticket_id)`
- `email_outbound(id, to_addr, subject, body_html, status, retries, ticket_id)`
- `webhooks(id, target_url, event_mask, secret, active)` (for GitHub or chat relays later)

### 4.5 Audit

- `audit_events(id, actor_type, actor_id, entity_type, entity_id, action, diff_json, ip, ua, at, hash, prev_hash)`

---

## 5) Workflow & States

**Default lifecycle** (configurable):

- `New → Assigned → Accepted → In Progress → Scheduled → Pending (Awaiting Info | Awaiting Callback | Awaiting Parts | Awaiting Approval) → Resolved → Closed`
- **SLA timers pause** in all `Pending` sub-states and `Scheduled`; resume on transition back to active states.

**Assignment:** Manual (MVP) + Round Robin to Team (toggle). Skill/load-based later.

**Automations (Day‑1):**

- **Auto‑ack**: on new ticket (email channel) → immediate acknowledgement.
- **Auto‑close**: if `Resolved` and requester no response after N days (config; default 3) → `Closed`.
- Duplicate detection (later): simple FTS + subject hash.
- Parent/child (later): relate dispatch subtasks; not in core MVP.

**Priorities & SLAs:**

- P1–P4 priorities with `response` and `resolution` targets; update cadence optional.
- Team & region **calendars** control working hours/holidays; SLA engine computes elapsed within business time only.

---

## 6) Channels

### 6.1 Web Portal (Requester)

- OIDC login, branded theme, create ticket (fields: title, description, category/subcategory, urgency, priority, custom fields), attachments.
- Ticket list & detail, status timeline, add comment (customer-visible), close ticket.
- KB browse/search (public/internal), simple markdown articles.
- Service Catalog: **3 fixed forms** implemented via JSON Schema + React Hook Form; no approvals.

### 6.2 Email

- **Inbound:** IMAP poller (label/folder based) or SMTP hook → parse, link to user via sender, create ticket or append comment via `[#TKT-1234]` in subject.
- **Outbound:** SMTP with DKIM/SPF support via infra; HTML templates; reply-to works for threading.
- **CSAT:** Links in resolution email to `/csat/{token}` form (score good/bad) + optional comment page.

### 6.3 “Chat” Outbound (Optional later)

- Slack/Teams webhook notifs for team channels; not a ticketing intake channel in MVP.

---

## 7) Search & Reporting

- **Search:** Postgres FTS; tsvector indexes on tickets(title, description, comments), category, requester/assignee.
- **Filters:** status, priority, team, assignee, date, SLA breach risk.
- **Dashboards (nice-to-have):** SLA attainment, mean/median resolution time, reopen rate, volume, backlog age.
- **Exports:** CSV on any list view; scheduled CSV email (post‑MVP).

---

## 8) Security, Auth, Audit

- **AuthN:** OIDC (Authentik). Service-to-service via PATs or JWT client creds.
- **AuthZ:** RBAC: requester, agent, team\_lead, manager, admin. No data partitioning required.
- **Audit:** Append-only event store with hash chain; nightly object-store snapshot.
- **PII:** Minimal; redact email bodies on request; retention policy below.

---

## 9) DevEx & Ops

- **Repo layout:** mono‑repo (`/api`, `/web`, `/worker`, `/helm`, `/infra`).
- **CI/CD:** GitHub Actions (lint, unit, build, image, Helm chart); image tags = `git sha`.
- **Observability:** Prom metrics for SLA lag, queue depth, mail errors, WS connections, 95/99 latencies.
- **Backups:** Postgres WAL + daily snapshot; MinIO versioning; restore docs.
- **Helm values:** external DB/S3/Redis endpoints; OIDC issuer/client; ingress hostnames; themes.

---

## 10) API Surface (MVP)

**REST (JWT bearer):**

- `POST /tickets` | `GET /tickets?filters` | `GET /tickets/{id}` | `PATCH /tickets/{id}` | `POST /tickets/{id}/comments` | `POST /tickets/{id}/attachments`
- `POST /search` (advanced)
- `GET /me` | `GET /teams` | `GET /slas` | `GET /kb?query`
- `POST /exports/tickets` (CSV async → download URL)
- `GET /csat/{token}` (public form)
- `POST /csat/{token}`
- `POST /webhooks/email-inbound` (if using SMTP hook)

**WebSockets:** presence, ticket updates, queue counters, typing, notifications.

---

## 11) Minimal Schema Snippets

```sql
create table tickets (
  id uuid primary key,
  number text unique not null,
  title text not null,
  description text,
  requester_id uuid not null references users(id),
  assignee_id uuid references users(id),
  team_id uuid references teams(id),
  priority smallint not null check (priority between 1 and 4),
  urgency smallint check (urgency between 1 and 4),
  category text,
  subcategory text,
  status text not null,
  scheduled_at timestamptz,
  due_at timestamptz,
  source text not null default 'web',
  custom_json jsonb not null default '{}',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index tickets_fts on tickets using gin (
  to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,''))
);
```

---

## 12) SLA Logic (Pseudo)

```text
on ticket_created:
  policy = select by priority/team
  clock.start(policy.response, policy.resolution)

on status_change:
  if status in {Pending.*, Scheduled}: clock.pause(reason=status)
  else: clock.resume()

cron (1m):
  for active clocks: add elapsed business-time since last_started_at per ticket.calendar
  if response_elapsed > target and not notified: flag breach_risk, notify
  if resolution_elapsed > target: breach; escalate
```

---

## 13) Theming & UX

- **Branding:** Org name/logo, primary/accent colors; dark mode.
- **Keyboard-first:** global quick‑open (`/`), new ticket (`n`), assign (`a`), reply (`r`), change status (`s`), next/prev (`j/k`).
- **Tabbed workspace:** multiple tickets side‑by‑side; keep draft replies per tab.
- **Bulk edits:** selection + mass status/assign/priority.
- **Saved replies/macros:** text snippets with placeholder variables.

---

## 14) Retention & Privacy

- **Ticket retention:** 6 months default (configurable).
- **Actions:** delete or anonymize requester data; keep audit hash chain.
- **CSAT tokens:** expire after 30 days.

---

## 15) MVP Backlog (Order of Execution)

1. **Scaffold** repo, CI, Docker, Helm; env wiring (OIDC, DB, Redis, MinIO).
2. **Auth & RBAC** (OIDC login, roles, seeds, `/me`).
3. **Tickets CRUD** (web + api); statuses; comments; attachments; audit events.
4. **Email outbound** (SMTP) + templates; **Auto‑ack** rule.
5. **Email inbound** poller → ticket create/append; subject tag parsing.
6. **SLA engine** + calendars (team/region) + pause/resume + breach flags.
7. **List views & filters**; Postgres FTS search.
8. **Agent console** (keyboard shortcuts, tabbed workspace, bulk edits, saved replies).
9. **Requester portal** (submit, track, KB minimal, service catalog 3 forms).
10. **CSV export** of tickets.
11. **CSAT 1‑click** + store + dashboard widget.
12. **Dashboards (basic)** via API aggregates; optional Grafana datasource.

---

## 16) Stretch (V2+)

- Skill/load-based assignment; on-call rotations.
- Problem/Change/Release fully enabled (relations, RFC templates).
- Parent/child tasks for field dispatches.
- Advanced automations (rules engine UI).
- OpenSearch for fuzzy/semantic search; embeddings for KB.
- Slack/Teams bot for notifications + quick actions.
- Mobile push (if/when native apps desired).
- Data warehouse exports; scheduled reports.

---

## 17) Open Source Plan

- **License:** AGPL‑3.0 or Apache‑2.0 (decide based on SaaS concerns).
- **Packages:** `/api`, `/web`, `/worker`, `/sdk` (TS client), `/helm`.
- **Docs:** Quickstart (minikube/k3s), Helm values samples, theming guide, contribution guide.

---

## 18) Risks & Mitigations

- **Email parsing complexity:** start simple (plain+HTML, small attachments), add edge cases later.
- **SLA calendar math:** leverage business-time libraries or thoroughly test custom implementation.
- **Audit immutability:** hash chain + offsite snapshots; consider WORM-capable bucket later.
- **Scope creep:** keep KB/catalog minimal; no approvals in MVP.

