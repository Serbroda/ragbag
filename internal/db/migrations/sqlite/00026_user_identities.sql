-- +goose Up
-- Accounts connected to the single sign-on provider (OIDC_ISSUER): its issuer and subject identify the person,
-- email is what the provider said when connecting (shown in the settings). Accounts made by single sign-on have
-- an empty password_hash until they set a password.
CREATE TABLE user_identities (
    issuer     TEXT NOT NULL,
    subject    TEXT NOT NULL,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    email      TEXT,
    created_at TIMESTAMP NOT NULL,
    PRIMARY KEY (issuer, subject)
);
CREATE UNIQUE INDEX idx_user_identities_user ON user_identities (user_id, issuer);

-- +goose Down
DROP TABLE user_identities;
