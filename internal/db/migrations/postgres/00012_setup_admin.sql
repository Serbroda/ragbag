-- +goose Up
-- The account created on the first start of an empty instance: an instance admin while ADMIN_USERS is empty,
-- with a random password (in a file next to the data) that has to be changed on the first login
ALTER TABLE users ADD COLUMN setup_admin BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE users DROP COLUMN must_change_password;
ALTER TABLE users DROP COLUMN setup_admin;
