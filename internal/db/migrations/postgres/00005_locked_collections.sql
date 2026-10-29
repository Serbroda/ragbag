-- +goose Up
-- A personal PIN unlocks the collections a user locked for themselves (only a BCrypt hash is stored)
ALTER TABLE users ADD COLUMN pin_hash TEXT;
ALTER TABLE users ADD COLUMN pin_failures INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN pin_blocked_until TIMESTAMPTZ;

-- Collections a user hides from their own view until they unlock with the PIN; others are not affected
CREATE TABLE collection_locks (
    user_id       TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    collection_id TEXT NOT NULL REFERENCES collections (id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (user_id, collection_id)
);

-- +goose Down
DROP TABLE collection_locks;
ALTER TABLE users DROP COLUMN pin_blocked_until;
ALTER TABLE users DROP COLUMN pin_failures;
ALTER TABLE users DROP COLUMN pin_hash;
