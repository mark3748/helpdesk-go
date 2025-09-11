-- +goose Up
create table if not exists releases (
    id uuid primary key default gen_random_uuid(),
    version text not null,
    notes text,
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists releases;
