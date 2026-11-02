-- +goose Up
-- When the account accepted the terms of use at registration (only instances that have terms)
ALTER TABLE users ADD COLUMN terms_accepted_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN terms_accepted_at;
