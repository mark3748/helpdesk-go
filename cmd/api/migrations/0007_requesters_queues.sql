-- +goose Up
create extension if not exists citext;

create table if not exists requesters (
    id uuid primary key default gen_random_uuid(),
    email citext,
    phone text,
    display_name text not null,
    profile jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create unique index if not exists requesters_email_idx on requesters (lower(email)) where email is not null;
create unique index if not exists requesters_phone_idx on requesters (phone) where phone is not null;

create table if not exists queues (
    id uuid primary key default gen_random_uuid(),
    name text not null unique,
    description text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

alter table tickets add column if not exists queue_id uuid references queues(id);

create table if not exists ticket_events (
    id bigserial primary key,
    ticket_id uuid not null references tickets(id) on delete cascade,
    actor_id uuid,
    type text not null,
    payload jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists ticket_events_ticket_created_idx on ticket_events (ticket_id, created_at desc);

alter table attachments rename column if exists mime to content_type;

create index if not exists tickets_updated_at_id_idx on tickets (updated_at desc, id);
create index if not exists tickets_status_queue_priority_idx on tickets (status, queue_id, priority);

-- +goose Down
DROP INDEX IF EXISTS tickets_status_queue_priority_idx;
DROP INDEX IF EXISTS tickets_updated_at_id_idx;
ALTER TABLE attachments RENAME COLUMN IF EXISTS content_type TO mime;
DROP TABLE IF EXISTS ticket_events;
ALTER TABLE tickets DROP COLUMN IF EXISTS queue_id;
DROP TABLE IF EXISTS queues;
DROP INDEX IF EXISTS requesters_phone_idx;
DROP INDEX IF EXISTS requesters_email_idx;
DROP TABLE IF EXISTS requesters;
