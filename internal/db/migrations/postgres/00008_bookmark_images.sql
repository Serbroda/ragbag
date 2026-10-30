-- +goose Up
-- Own preview images: an upload (images table) or an address; or none at all. Otherwise og_image is shown.
ALTER TABLE bookmarks ADD COLUMN image_id TEXT REFERENCES images (id) ON DELETE SET NULL;
ALTER TABLE bookmarks ADD COLUMN image_url TEXT;
ALTER TABLE bookmarks ADD COLUMN image_hidden BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE bookmarks DROP COLUMN image_hidden;
ALTER TABLE bookmarks DROP COLUMN image_url;
ALTER TABLE bookmarks DROP COLUMN image_id;
