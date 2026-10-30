-- +goose Up
-- The space to open on start (null: the last visited one); deleting the space unsets it
ALTER TABLE users ADD COLUMN start_space_id TEXT REFERENCES spaces (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE users DROP COLUMN start_space_id;
