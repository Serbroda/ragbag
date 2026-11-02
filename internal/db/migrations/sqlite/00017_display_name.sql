-- +goose Up
-- Shown instead of the username where set (any case, spaces, emojis); not unique, the username stays the login
ALTER TABLE users ADD COLUMN display_name TEXT;

-- +goose Down
ALTER TABLE users DROP COLUMN display_name;
