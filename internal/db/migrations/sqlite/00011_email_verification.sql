-- +goose Up
-- Null while a new account waits for its email address to be confirmed (only when SMTP is configured)
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP;
UPDATE users SET email_verified_at = created_at;

-- One pending confirmation link per account, stored as SHA-256 hash
CREATE TABLE email_verifications (
    user_id    TEXT PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE email_verifications;
ALTER TABLE users DROP COLUMN email_verified_at;
