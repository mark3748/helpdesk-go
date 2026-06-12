-- +goose Up
alter table settings add column if not exists discord jsonb not null default '{}'::jsonb;

-- +goose Down
alter table settings drop column if exists discord;
