-- +goose Up
-- +goose StatementBegin
-- Remove existing exact-content duplicates, keeping the oldest row per group.
WITH dups AS (
  SELECT id,
         row_number() OVER (
           PARTITION BY md5(lower(coalesce(title,'')) || '|' || coalesce(requester_id::text,'') || '|' || coalesce(description,''))
           ORDER BY created_at ASC, id ASC
         ) AS rn
  FROM tickets
)
DELETE FROM tickets t USING dups
WHERE t.id = dups.id AND dups.rn > 1;

-- Create the unique index using only IMMUTABLE functions
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
