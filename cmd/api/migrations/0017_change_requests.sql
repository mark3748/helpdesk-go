-- +goose Up
create table if not exists change_requests (
    id uuid primary key default gen_random_uuid(),
    title text not null,
    description text,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists change_requests;
