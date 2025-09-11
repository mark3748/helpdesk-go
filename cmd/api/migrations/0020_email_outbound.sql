-- +goose Up
create table if not exists email_outbound (
    id uuid primary key default gen_random_uuid(),
    to_addr text not null,
    subject text,
    body_html text,
    status text not null,
    retries int not null default 0,
    ticket_id uuid references tickets(id),
    created_at timestamptz not null default now()
);

-- +goose Down
drop table if exists email_outbound;
