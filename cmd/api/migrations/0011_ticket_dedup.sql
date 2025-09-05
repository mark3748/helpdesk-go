-- +goose Up
-- NOTE: Replaced with a safe index to avoid IMMUTABLE function constraints on
-- some Postgres setups. Application-level advisory locks handle dedup.
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS tickets_title_lower_idx ON tickets (lower(title));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS tickets_title_lower_idx;
-- +goose StatementEnd
