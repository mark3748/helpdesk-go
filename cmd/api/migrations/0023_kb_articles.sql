-- +goose Up
create table if not exists kb_articles (
    id uuid primary key default gen_random_uuid(),
    slug text unique not null,
    title text not null,
    body_md text not null,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

-- +goose Down
drop table if exists kb_articles;
