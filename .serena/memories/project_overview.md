# Project Overview

## Purpose
Helpdesk-Go is a complete FootPrints-style ticketing system built in Go with PostgreSQL backend and Kubernetes deployment ready out of the box.

## Tech Stack
- **Backend**: Go 1.25+ with Gin web framework
- **Database**: PostgreSQL 14+ with goose migrations
- **Cache/Queue**: Redis for workers and caching
- **Object Storage**: S3-compatible (MinIO) or local filesystem
- **Frontend**: Two React apps (Internal workspace + Customer portal)
- **Authentication**: OIDC (JWKS) or local auth fallback
- **Deployment**: Docker + Kubernetes with Helm charts

## Key Components
- **API Service** (`cmd/api`): Main REST API with Gin framework
- **Worker Service** (`cmd/worker`): Background job processing (emails, SLA)
- **Internal UI** (`web/internal`): React workspace for agents/admins
- **Requester UI** (`web/requester`): React customer portal
- **Storage**: Attachments via MinIO/S3 or local filesystem

## Architecture
- Microservices architecture with API + Worker services
- Event-driven with Redis job queues
- Database-first with embedded migrations
- Role-based access control (Admin/Manager/Agent)
- Server-sent events for real-time updates