-- +goose Up
-- The space invite an account registered with: it joins that space once its email address is confirmed. No
-- foreign key: a revoked invite (or a deleted space) just joins nothing
ALTER TABLE email_verifications ADD COLUMN invite_id TEXT;

-- +goose Down
ALTER TABLE email_verifications DROP COLUMN invite_id;
