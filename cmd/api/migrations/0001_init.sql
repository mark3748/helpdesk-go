-- +goose Up
create extension if not exists pgcrypto;
create extension if not exists "uuid-ossp";

-- sequence for ticket numbers
create sequence if not exists ticket_seq start 1;

-- core tables
create table if not exists users (
    id uuid primary key default gen_random_uuid(),
    external_id text,
    email text unique,
    display_name text,
    active boolean not null default true,
    locale text,
    time_zone text,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists teams (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    region_id uuid,
    calendar_id uuid,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists regions (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    calendar_id uuid,
    created_at timestamptz not null default now()
);

create table if not exists calendars (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    tz text not null
);

create table if not exists business_hours (
    calendar_id uuid not null references calendars(id) on delete cascade,
    dow smallint not null check (dow between 0 and 6),
    start_sec int not null,
    end_sec int not null,
    primary key (calendar_id, dow)
);

create table if not exists holidays (
    calendar_id uuid not null references calendars(id) on delete cascade,
    date date not null,
    label text,
    primary key (calendar_id, date)
);

create table if not exists sla_policies (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    priority smallint not null check (priority between 1 and 4),
    response_target_mins int not null,
    resolution_target_mins int not null,
    update_cadence_mins int,
    created_at timestamptz not null default now()
);

create table if not exists tickets (
    id uuid primary key default gen_random_uuid(),
    number text not null unique,
    title text not null,
    description text,
    requester_id uuid not null references users(id),
    assignee_id uuid references users(id),
    team_id uuid references teams(id),
    priority smallint not null check (priority between 1 and 4),
    urgency smallint check (urgency between 1 and 4),
    category text,
    subcategory text,
    status text not null default 'New',
    scheduled_at timestamptz,
    due_at timestamptz,
    source text not null default 'web',
    custom_json jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table if not exists ticket_comments (
    id uuid primary key default gen_random_uuid(),
    ticket_id uuid not null references tickets(id) on delete cascade,
    author_id uuid not null references users(id),
    body_md text not null,
    is_internal boolean not null default false,
    created_at timestamptz not null default now()
);

create table if not exists ticket_status_history (
    id uuid primary key default gen_random_uuid(),
    ticket_id uuid not null references tickets(id) on delete cascade,
    from_status text,
    to_status text not null,
    actor_id uuid,
    note text,
    at timestamptz not null default now()
);

create table if not exists attachments (
    id uuid primary key default gen_random_uuid(),
    ticket_id uuid not null references tickets(id) on delete cascade,
    uploader_id uuid not null references users(id),
    object_key text not null,
    filename text not null,
    bytes bigint not null,
    mime text,
    created_at timestamptz not null default now()
);

create table if not exists ticket_sla_clocks (
    ticket_id uuid primary key references tickets(id) on delete cascade,
    policy_id uuid references sla_policies(id),
    response_elapsed_ms bigint not null default 0,
    resolution_elapsed_ms bigint not null default 0,
    last_started_at timestamptz,
    paused boolean not null default false,
    reason text
);

create table if not exists audit_events (
    id uuid primary key default gen_random_uuid(),
    actor_type text,
    actor_id uuid,
    entity_type text,
    entity_id uuid,
    action text,
    diff_json jsonb,
    ip text,
    ua text,
    at timestamptz not null default now(),
    hash text,
    prev_hash text
);

-- FTS index
create index if not exists tickets_fts on tickets using gin (to_tsvector('english', coalesce(title,'') || ' ' || coalesce(description,'')));

-- +goose Down
drop index if exists tickets_fts;
drop table if exists audit_events;
drop table if exists ticket_sla_clocks;
drop table if exists attachments;
drop table if exists ticket_status_history;
drop table if exists ticket_comments;
drop table if exists tickets;
drop table if exists sla_policies;
drop table if exists holidays;
drop table if exists business_hours;
drop table if exists calendars;
drop table if exists regions;
drop table if exists teams;
drop table if exists users;
drop sequence if exists ticket_seq;
