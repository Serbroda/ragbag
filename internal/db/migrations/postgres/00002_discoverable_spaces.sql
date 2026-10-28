-- +goose Up
-- Discoverable spaces are listed for every account of the instance, which can join them with join_role
ALTER TABLE spaces ADD COLUMN discoverable BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE spaces ADD COLUMN join_role TEXT NOT NULL DEFAULT 'VIEWER';

-- +goose Down
ALTER TABLE spaces DROP COLUMN join_role;
ALTER TABLE spaces DROP COLUMN discoverable;
