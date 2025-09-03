-- +goose Up
create table if not exists settings (
    id smallint primary key default 1,
    storage jsonb not null default '{}'::jsonb,
    oidc jsonb not null default '{}'::jsonb,
    mail jsonb not null default '{}'::jsonb,
    log_path text not null default '/config/logs',
    last_test timestamptz
);
insert into settings (id) values (1) on conflict do nothing;

-- +goose Down
drop table if exists settings;
