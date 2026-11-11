-- +goose Up
-- Smart collections hold no links of their own: they show what this search finds in the space
ALTER TABLE collections ADD COLUMN query TEXT;

-- +goose Down
ALTER TABLE collections DROP COLUMN query;
