-- +goose Up
-- Headline sections: full_width keeps one out of the columns of its neighbors, headline_count is how many
-- headlines it shows before "Show more" (null: the default).
ALTER TABLE dashboard_sections ADD COLUMN full_width BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE dashboard_sections ADD COLUMN headline_count INTEGER;

-- +goose Down
ALTER TABLE dashboard_sections DROP COLUMN headline_count;
ALTER TABLE dashboard_sections DROP COLUMN full_width;
