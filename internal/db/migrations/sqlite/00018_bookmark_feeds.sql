-- +goose Up
-- A news feed whose latest headlines the card shows, optionally only those containing one of the filter's words
ALTER TABLE bookmarks ADD COLUMN feed_url TEXT;
ALTER TABLE bookmarks ADD COLUMN feed_filter TEXT;

-- +goose Down
ALTER TABLE bookmarks DROP COLUMN feed_filter;
ALTER TABLE bookmarks DROP COLUMN feed_url;
