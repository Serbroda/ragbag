package store

import (
	"context"
	"time"
)

// TrashedBookmark is a link in the trash with who deleted it.
type TrashedBookmark struct {
	ID             string    `db:"id"`
	CollectionID   string    `db:"collection_id"`
	CollectionName string    `db:"collection_name"`
	URL            string    `db:"url"`
	Title          *string   `db:"title"`
	Favicon        *string   `db:"favicon"`
	DeletedAt      time.Time `db:"deleted_at"`
	DeletedByName  *string   `db:"deleted_by_name"`
}

// TrashedBookmarks lists the links of the space that were deleted one by one, the most recent first.
func (s *Store) TrashedBookmarks(ctx context.Context, spaceID string) ([]TrashedBookmark, error) {
	var bookmarks []TrashedBookmark
	err := s.sel(ctx, &bookmarks, `SELECT b.id, b.collection_id, c.name AS collection_name, b.url, b.title, b.favicon,
			b.deleted_at, u.username AS deleted_by_name
		FROM bookmarks b JOIN collections c ON c.id = b.collection_id LEFT JOIN users u ON u.id = b.deleted_by
		WHERE c.space_id = ? AND b.deleted_at IS NOT NULL ORDER BY b.deleted_at DESC, b.id`, spaceID)
	return bookmarks, err
}

// TrashedBookmark returns a link in the trash with its space, or ErrNotFound.
func (s *Store) TrashedBookmark(ctx context.Context, id string) (*Bookmark, error) {
	var b Bookmark
	return &b, s.get(ctx, &b, `SELECT `+bookmarkColumns+` FROM bookmarks b JOIN collections c ON c.id = b.collection_id
		JOIN users u ON u.id = b.created_by WHERE b.id = ? AND b.deleted_at IS NOT NULL`, id)
}

// DeleterNames maps user ids to usernames for the "deleted by" of trashed collections.
func (s *Store) DeleterNames(ctx context.Context, ids []string) (map[string]string, error) {
	names := map[string]string{}
	users, err := s.UsersByIDs(ctx, ids)
	for _, user := range users {
		names[user.ID] = user.Username
	}
	return names, err
}

// DeleteOrphanImages removes uploads nothing points to any more (links deleted for good, images replaced),
// leaving fresh ones alone in case they are about to be used.
func (s *Store) DeleteOrphanImages(ctx context.Context, before time.Time) (int64, error) {
	return s.exec(ctx, `DELETE FROM images WHERE created_at < ?
		AND id NOT IN (SELECT avatar_image_id FROM users WHERE avatar_image_id IS NOT NULL)
		AND id NOT IN (SELECT icon_image_id FROM spaces WHERE icon_image_id IS NOT NULL)
		AND id NOT IN (SELECT image_id FROM bookmarks WHERE image_id IS NOT NULL)`, before.UTC())
}

// PurgeTrash deletes what has been in the trash since before the given time, for good.
func (s *Store) PurgeTrash(ctx context.Context, before time.Time) (int64, error) {
	links, err := s.exec(ctx, `DELETE FROM bookmarks WHERE deleted_at < ?`, before.UTC())
	if err != nil {
		return 0, err
	}
	collections, err := s.exec(ctx, `DELETE FROM collections WHERE deleted_at < ?`, before.UTC())
	return links + collections, err
}
