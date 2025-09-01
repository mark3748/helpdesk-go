-- +goose Up
alter table tickets add column if not exists csat_token text unique;
alter table tickets add column if not exists csat_score text;

-- +goose Down
alter table tickets drop column if exists csat_score;
alter table tickets drop column if exists csat_token;
