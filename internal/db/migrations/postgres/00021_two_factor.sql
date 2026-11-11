-- +goose Up
-- Two-factor authentication with an authenticator app (TOTP), off unless the user turns it on.
-- totp_secret is encrypted with a key derived from JWT_SECRET and waits for its first code until
-- totp_enabled_at is set; totp_recovery holds the hashes of the unused recovery codes (JSON),
-- totp_last_step the time window of the last accepted code (each works once).
ALTER TABLE users ADD COLUMN totp_secret TEXT;
ALTER TABLE users ADD COLUMN totp_enabled_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN totp_recovery TEXT;
ALTER TABLE users ADD COLUMN totp_last_step BIGINT NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN totp_failures INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN totp_blocked_until TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN totp_blocked_until;
ALTER TABLE users DROP COLUMN totp_failures;
ALTER TABLE users DROP COLUMN totp_last_step;
ALTER TABLE users DROP COLUMN totp_recovery;
ALTER TABLE users DROP COLUMN totp_enabled_at;
ALTER TABLE users DROP COLUMN totp_secret;
