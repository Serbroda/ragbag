-- +goose Up
-- Manual order of collections below the same parent (shared by all members of the space)
ALTER TABLE collections ADD COLUMN position INTEGER NOT NULL DEFAULT 0;
-- Existing collections keep the order they had so far: by creation
UPDATE collections SET position = (
    SELECT COUNT(*) FROM collections c2
    WHERE c2.space_id = collections.space_id
      AND COALESCE(c2.parent_id, '') = COALESCE(collections.parent_id, '')
      AND (c2.created_at < collections.created_at OR (c2.created_at = collections.created_at AND c2.id < collections.id))
);

-- +goose Down
ALTER TABLE collections DROP COLUMN position;
