-- +goose Up
alter table tickets drop constraint if exists tickets_requester_id_fkey;
alter table tickets add constraint tickets_requester_id_fkey foreign key (requester_id) references requesters(id);

-- +goose Down
alter table tickets drop constraint if exists tickets_requester_id_fkey;
alter table tickets add constraint tickets_requester_id_fkey foreign key (requester_id) references users(id);
