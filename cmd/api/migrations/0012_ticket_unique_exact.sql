-- +goose Up
-- Enforce exact-content dedup for tickets: blocks inserting two tickets with
-- the same (normalized title, requester_id, description) at any time.
-- Uses only IMMUTABLE functions to satisfy index expression rules.
-- +goose StatementBegin
CREATE UNIQUE INDEX IF NOT EXISTS tickets_unique_exact_idx ON tickets (
  md5(
    lower(coalesce(title, '')) || '|' ||
    coalesce(requester_id::text, '') || '|' ||
    coalesce(description, '')
  )
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS tickets_unique_exact_idx;
-- +goose StatementEnd

