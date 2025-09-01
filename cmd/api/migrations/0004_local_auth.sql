-- +goose Up
alter table users add column if not exists username text unique;
alter table users add column if not exists password_hash text;

create unique index if not exists users_username_idx on users((lower(username)));

-- +goose Down
drop index if exists users_username_idx;
alter table users drop column if exists password_hash;
alter table users drop column if exists username;

