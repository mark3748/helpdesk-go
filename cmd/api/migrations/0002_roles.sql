-- +goose Up
create table if not exists roles (
    id uuid primary key default gen_random_uuid(),
    name text not null unique
);

create table if not exists user_roles (
    user_id uuid not null references users(id) on delete cascade,
    role_id uuid not null references roles(id) on delete cascade,
    primary key (user_id, role_id)
);

-- seed roles
insert into roles (id, name) values
    (gen_random_uuid(), 'agent'),
    (gen_random_uuid(), 'requester'),
    (gen_random_uuid(), 'admin'),
    (gen_random_uuid(), 'manager')
    on conflict do nothing;

-- seed dev user with agent role
insert into users (id, external_id, email, display_name)
values (gen_random_uuid(), 'agent-1', 'agent@example.com', 'Agent One')
    on conflict (email) do nothing;

insert into user_roles (user_id, role_id)
select u.id, r.id from users u, roles r
where u.email = 'agent@example.com' and r.name = 'agent'
    on conflict do nothing;

-- +goose Down

drop table if exists user_roles;
drop table if exists roles;
