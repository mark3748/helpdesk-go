-- +goose Up
create table if not exists ticket_watchers (
    ticket_id uuid not null references tickets(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    added_at timestamptz not null default now(),
    primary key (ticket_id, user_id)
);

-- +goose Down
drop table if exists ticket_watchers;
