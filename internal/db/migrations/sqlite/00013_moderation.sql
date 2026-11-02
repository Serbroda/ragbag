-- +goose Up
-- Deleted accounts stay as an anonymous row (their links in shared spaces keep an author); blocked ones cannot
-- sign in and their public links stop working
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP;
ALTER TABLE users ADD COLUMN blocked_at TIMESTAMP;
ALTER TABLE users ADD COLUMN blocked_reason TEXT;

-- Public links disabled by the instance admins after a report; the space cannot share the collection again
ALTER TABLE collection_shares ADD COLUMN disabled_at TIMESTAMP;
ALTER TABLE collection_shares ADD COLUMN disabled_reason TEXT;

-- Reports of public links (notice and action, DSA art. 16): collection_id is the shared collection,
-- viewed_collection_id the (sub-)collection the reporter looked at. The name is a snapshot: the report outlives
-- the collection
CREATE TABLE abuse_reports (
    id                   TEXT PRIMARY KEY,
    share_token          TEXT NOT NULL,
    collection_id        TEXT REFERENCES collections (id) ON DELETE SET NULL,
    viewed_collection_id TEXT,
    collection_name      TEXT NOT NULL,
    category        TEXT NOT NULL,
    details         TEXT NOT NULL,
    reporter_name   TEXT,
    reporter_email  TEXT,
    created_at      TIMESTAMP NOT NULL,
    resolved_at     TIMESTAMP,
    resolved_by     TEXT REFERENCES users (id) ON DELETE SET NULL,
    resolution      TEXT,
    resolution_note TEXT
);
CREATE INDEX idx_abuse_reports_created ON abuse_reports (created_at);

-- +goose Down
DROP TABLE abuse_reports;
ALTER TABLE collection_shares DROP COLUMN disabled_reason;
ALTER TABLE collection_shares DROP COLUMN disabled_at;
ALTER TABLE users DROP COLUMN blocked_reason;
ALTER TABLE users DROP COLUMN blocked_at;
ALTER TABLE users DROP COLUMN deleted_at;
