-- +goose Up
-- Allow comment author_id to be nullable so external sources (email, discord) can add comments
alter table ticket_comments alter column author_id drop not null;

-- Add author_requester_id to link guest/external comment authors to requesters
alter table ticket_comments add column if not exists author_requester_id uuid references requesters(id) on delete set null;

-- Create table to map Discord user IDs to Helpdesk requester IDs
create table if not exists discord_user_mappings (
    discord_user_id text primary key,
    requester_id uuid not null references requesters(id) on delete cascade,
    created_at timestamptz not null default now()
);

-- Create table to map Discord thread IDs to Ticket IDs
create table if not exists discord_thread_mappings (
    discord_thread_id text primary key,
    ticket_id uuid not null references tickets(id) on delete cascade,
    channel_id text not null,
    created_at timestamptz not null default now()
);

-- Update tickets.source check constraint to include 'discord'
alter table tickets drop constraint if exists tickets_source_check;
alter table tickets add constraint tickets_source_check check (source in ('web', 'email', 'discord'));

-- +goose Down
alter table tickets drop constraint if exists tickets_source_check;
alter table tickets add constraint tickets_source_check check (source in ('web', 'email'));

drop table if exists discord_thread_mappings;
drop table if exists discord_user_mappings;
alter table ticket_comments drop column if exists author_requester_id;
alter table ticket_comments alter column author_id set not null;
