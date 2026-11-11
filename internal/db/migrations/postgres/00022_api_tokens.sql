-- +goose Up
-- Personal API tokens for scripts and apps: sent as "Authorization: Bearer rb_…", only the SHA-256 hash is
-- stored. scope is READ or WRITE; they never reach the account, spaces' settings or members.
CREATE TABLE api_tokens (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   TEXT NOT NULL UNIQUE,
    scope        TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ
);
CREATE INDEX idx_api_tokens_user ON api_tokens (user_id);

-- +goose Down
DROP TABLE api_tokens;
