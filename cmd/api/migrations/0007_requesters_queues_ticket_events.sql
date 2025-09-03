-- +goose Up
create table if not exists requesters (
    id uuid primary key default gen_random_uuid(),
    email text unique,
    name text,
    created_at timestamptz not null default now()
);

create table if not exists queues (
    id uuid primary key default gen_random_uuid(),
    name text not null,
    created_at timestamptz not null default now()
);

alter table tickets
    add column if not exists queue_id uuid references queues(id);
alter table tickets
    drop constraint if exists tickets_requester_id_fkey;
alter table tickets
    add constraint tickets_requester_id_fkey foreign key (requester_id) references requesters(id);

create table if not exists ticket_events (
    id uuid primary key default gen_random_uuid(),
    ticket_id uuid not null references tickets(id) on delete cascade,
    event_type text not null,
    payload jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists tickets_queue_id_idx on tickets(queue_id);
create index if not exists tickets_requester_id_idx on tickets(requester_id);
create index if not exists ticket_events_ticket_id_idx on ticket_events(ticket_id);

-- +goose Down
create or replace function __drop_if_exists(name text) returns void as $$
begin
    execute format('drop table if exists %s', name);
end; $$ language plpgsql;

-- drop indexes and tables
 drop index if exists ticket_events_ticket_id_idx;
 drop index if exists tickets_queue_id_idx;
 drop index if exists tickets_requester_id_idx;
 select __drop_if_exists('ticket_events');
 alter table tickets drop column if exists queue_id;
 select __drop_if_exists('queues');
 alter table tickets drop constraint if exists tickets_requester_id_fkey;
 select __drop_if_exists('requesters');
 drop function if exists __drop_if_exists;
