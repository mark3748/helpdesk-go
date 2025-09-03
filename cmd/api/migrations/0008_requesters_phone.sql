-- +goose Up
alter table requesters add column if not exists phone text;

-- +goose Down
alter table requesters drop column if exists phone;
