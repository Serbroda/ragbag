-- +goose Up
-- Pro, granted by an instance admin (PLANS_ENABLED)
ALTER TABLE users ADD COLUMN pro_granted_at TIMESTAMP;

-- +goose Down
ALTER TABLE users DROP COLUMN pro_granted_at;
