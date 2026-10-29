package store

import (
	"context"
	"time"
)

// --- New links ---

// NewBookmark is a link another member added after the user last looked at its collection. Before the first
// visit the time the user joined the space counts, so joining does not turn everything into news.
type NewBookmark struct {
	ID            string    `db:"id"`
	SpaceID       string    `db:"space_id"`
	CollectionID  string    `db:"collection_id"`
	CreatedByName string    `db:"created_by_name"`
	CreatedAt     time.Time `db:"created_at"`
}

// NewBookmarks lists the links that are new for the user in one space, or in all their spaces if spaceID is
// empty. Hidden collections are not filtered here; callers drop them.
func (s *Store) NewBookmarks(ctx context.Context, userID, spaceID string) ([]NewBookmark, error) {
	query := `SELECT b.id, c.space_id, b.collection_id, u.username AS created_by_name, b.created_at
		FROM bookmarks b
		JOIN collections c ON c.id = b.collection_id
		JOIN users u ON u.id = b.created_by
		JOIN space_members m ON m.space_id = c.space_id AND m.user_id = ?
		LEFT JOIN collection_visits v ON v.collection_id = b.collection_id AND v.user_id = ?
		WHERE b.created_by <> ? AND b.created_at > COALESCE(v.seen_at, m.created_at)`
	args := []any{userID, userID, userID}
	if spaceID != "" {
		query += ` AND c.space_id = ?`
		args = append(args, spaceID)
	}
	var news []NewBookmark
	err := s.sel(ctx, &news, query+` ORDER BY b.created_at DESC, b.id`, args...)
	return news, err
}

// MarkSeen records that the user looked at the collections now.
func (s *Store) MarkSeen(ctx context.Context, userID string, collectionIDs []string) error {
	at := now()
	for _, id := range collectionIDs {
		if _, err := s.exec(ctx, `INSERT INTO collection_visits (user_id, collection_id, seen_at) VALUES (?, ?, ?)
			ON CONFLICT (user_id, collection_id) DO UPDATE SET seen_at = excluded.seen_at`, userID, id, at); err != nil {
			return err
		}
	}
	return nil
}

// --- Reactions ---

type Reaction struct {
	BookmarkID string `db:"bookmark_id"`
	UserID     string `db:"user_id"`
	Username   string `db:"username"`
	Emoji      string `db:"emoji"`
}

func (s *Store) AddReaction(ctx context.Context, bookmarkID, userID, emoji string) error {
	_, err := s.exec(ctx, `INSERT INTO bookmark_reactions (bookmark_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?)
		ON CONFLICT DO NOTHING`, bookmarkID, userID, emoji, now())
	return err
}

func (s *Store) RemoveReaction(ctx context.Context, bookmarkID, userID, emoji string) error {
	_, err := s.exec(ctx, `DELETE FROM bookmark_reactions WHERE bookmark_id = ? AND user_id = ? AND emoji = ?`,
		bookmarkID, userID, emoji)
	return err
}

// Reactions of the bookmarks, oldest first so the order of the emojis stays stable.
func (s *Store) Reactions(ctx context.Context, bookmarkIDs []string) ([]Reaction, error) {
	if len(bookmarkIDs) == 0 {
		return nil, nil
	}
	var reactions []Reaction
	err := s.selIn(ctx, &reactions, `SELECT r.bookmark_id, r.user_id, u.username, r.emoji FROM bookmark_reactions r
		JOIN users u ON u.id = r.user_id WHERE r.bookmark_id IN (?) ORDER BY r.created_at, u.username`, bookmarkIDs)
	return reactions, err
}

// --- Space bar ---

// SpaceBar returns the stored layout (JSON) or "" if the user never arranged the space bar.
func (s *Store) SpaceBar(ctx context.Context, userID string) (string, error) {
	var layouts []string
	if err := s.sel(ctx, &layouts, `SELECT layout FROM space_bars WHERE user_id = ?`, userID); err != nil {
		return "", err
	}
	if len(layouts) == 0 {
		return "", nil
	}
	return layouts[0], nil
}

func (s *Store) SetSpaceBar(ctx context.Context, userID, layout string) error {
	_, err := s.exec(ctx, `INSERT INTO space_bars (user_id, layout, updated_at) VALUES (?, ?, ?)
		ON CONFLICT (user_id) DO UPDATE SET layout = excluded.layout, updated_at = excluded.updated_at`,
		userID, layout, now())
	return err
}
