-- +goose Up
create table if not exists webhooks (
    id uuid primary key default gen_random_uuid(),
    target_url text not null,
    event_mask int not null default 0,
    secret text,
    active boolean not null default true,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists webhooks;
