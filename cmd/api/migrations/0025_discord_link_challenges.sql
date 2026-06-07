-- +goose Up
create table if not exists discord_link_challenges (
    id uuid primary key default gen_random_uuid(),
    discord_user_id text not null,
    email text not null,
    token_hash bytea not null unique,
    expires_at timestamptz not null,
    consumed_at timestamptz,
    created_at timestamptz not null default now()
);

create index if not exists discord_link_challenges_user_created_idx
    on discord_link_challenges(discord_user_id, created_at desc);

create index if not exists discord_link_challenges_email_created_idx
    on discord_link_challenges(email, created_at desc);

-- +goose Down
drop table if exists discord_link_challenges;
