-- +goose Up
-- Deleting moves links and collections to the trash: they are only marked and purged after 30 days
ALTER TABLE bookmarks ADD COLUMN deleted_at TIMESTAMP;
ALTER TABLE bookmarks ADD COLUMN deleted_by TEXT REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE collections ADD COLUMN deleted_at TIMESTAMP;
ALTER TABLE collections ADD COLUMN deleted_by TEXT REFERENCES users (id) ON DELETE SET NULL;
CREATE INDEX idx_bookmarks_deleted ON bookmarks (deleted_at);
CREATE INDEX idx_collections_deleted ON collections (deleted_at);

-- +goose Down
DROP INDEX idx_collections_deleted;
DROP INDEX idx_bookmarks_deleted;
ALTER TABLE collections DROP COLUMN deleted_by;
ALTER TABLE collections DROP COLUMN deleted_at;
ALTER TABLE bookmarks DROP COLUMN deleted_by;
ALTER TABLE bookmarks DROP COLUMN deleted_at;
