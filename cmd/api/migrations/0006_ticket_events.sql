-- +goose Up
create table if not exists ticket_events (
    id uuid primary key default gen_random_uuid(),
    ticket_id uuid references tickets(id) on delete cascade,
    action text not null,
    actor_id uuid,
    diff_json jsonb,
    at timestamptz not null default now()
);

-- +goose Down
drop table if exists ticket_events;
