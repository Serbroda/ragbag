package store

import "context"

// The counts behind the quotas include trashed links and collections: they take up space until purged.

func (s *Store) CountLinksBy(ctx context.Context, userID string) (int, error) {
	var count int
	return count, s.get(ctx, &count, `SELECT COUNT(*) FROM bookmarks WHERE created_by = ?`, userID)
}

func (s *Store) CountCollections(ctx context.Context, spaceID string) (int, error) {
	var count int
	return count, s.get(ctx, &count, `SELECT COUNT(*) FROM collections WHERE space_id = ?`, spaceID)
}

func (s *Store) CountOwnedSpaces(ctx context.Context, userID string) (int, error) {
	var count int
	return count, s.get(ctx, &count, `SELECT COUNT(*) FROM space_members WHERE user_id = ? AND role = 'OWNER'`, userID)
}

// UploadedBytes is the size of the images a user uploaded (avatar, space icons, preview images).
func (s *Store) UploadedBytes(ctx context.Context, userID string) (int64, error) {
	var total int64
	return total, s.get(ctx, &total, `SELECT COALESCE(SUM(LENGTH(data)), 0) FROM images WHERE created_by = ?`, userID)
}
