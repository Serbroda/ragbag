-- +goose Up
-- Changes of the terms of use: from effective_at, accounts that accepted them before (or never) have to accept
-- them again before going on. Kept as a history of when which version applied.
CREATE TABLE terms_changes (
    id           TEXT PRIMARY KEY,
    effective_at TIMESTAMP NOT NULL,
    created_by   TEXT REFERENCES users (id) ON DELETE SET NULL,
    created_at   TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE terms_changes;
