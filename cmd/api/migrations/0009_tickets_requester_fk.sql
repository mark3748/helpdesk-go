-- +goose Up
insert into requesters (id, email, name)
    select distinct u.id, u.email, u.display_name
    from tickets t
    join users u on t.requester_id = u.id
    where not exists (
        select 1 from requesters r where r.id = u.id
    );

alter table tickets drop constraint if exists tickets_requester_id_fkey;
alter table tickets add constraint tickets_requester_id_fkey foreign key (requester_id) references requesters(id);

-- +goose Down
alter table tickets drop constraint if exists tickets_requester_id_fkey;
alter table tickets add constraint tickets_requester_id_fkey foreign key (requester_id) references users(id);
