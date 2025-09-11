-- +goose Up
create table if not exists email_inbound (
    id uuid primary key default gen_random_uuid(),
    raw_store_key text not null,
    parsed_json jsonb not null,
    message_id text unique,
    status text not null default 'new',
    ticket_id uuid references tickets(id),
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists email_inbound;
