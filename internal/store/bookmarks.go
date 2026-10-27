package store

import (
	"context"
	"sort"
	"strings"
	"time"
)

type Bookmark struct {
	ID            string    `db:"id"`
	CollectionID  string    `db:"collection_id"`
	SpaceID       string    `db:"space_id"`
	URL           string    `db:"url"`
	Title         *string   `db:"title"`
	Description   *string   `db:"description"`
	OgImage       *string   `db:"og_image"`
	Favicon       *string   `db:"favicon"`
	Canonical     *string   `db:"canonical"`
	CreatedBy     string    `db:"created_by"`
	CreatedByName string    `db:"created_by_name"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
	Tags          []string  `db:"-"`
}

const (
	bookmarkColumns = `b.id, b.collection_id, c.space_id, b.url, b.title, b.description, b.og_image, b.favicon,
		b.canonical, b.created_by, u.username AS created_by_name, b.created_at, b.updated_at`
	bookmarkJoins  = ` JOIN collections c ON c.id = b.collection_id JOIN users u ON u.id = b.created_by`
	bookmarkSelect = `SELECT ` + bookmarkColumns + ` FROM bookmarks b` + bookmarkJoins
)

// CreateBookmark keeps a preset CreatedAt (imports), otherwise uses the current time.
func (s *Store) CreateBookmark(ctx context.Context, b *Bookmark) error {
	at := now()
	b.ID, b.UpdatedAt = newID(), at
	if b.CreatedAt.IsZero() || b.CreatedAt.After(at) {
		b.CreatedAt = at
	}
	if _, err := s.exec(ctx, `INSERT INTO bookmarks (id, collection_id, url, title, description, og_image, favicon,
		canonical, created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		b.ID, b.CollectionID, b.URL, b.Title, b.Description, b.OgImage, b.Favicon, b.Canonical, b.CreatedBy,
		b.CreatedAt.UTC(), at); err != nil {
		return err
	}
	return s.setTags(ctx, b.ID, b.Tags)
}

func (s *Store) UpdateBookmark(ctx context.Context, b *Bookmark) error {
	if _, err := s.exec(ctx, `UPDATE bookmarks SET url = ?, title = ?, description = ?, og_image = ?, favicon = ?,
		canonical = ?, updated_at = ? WHERE id = ?`,
		b.URL, b.Title, b.Description, b.OgImage, b.Favicon, b.Canonical, now(), b.ID); err != nil {
		return err
	}
	return s.setTags(ctx, b.ID, b.Tags)
}

func (s *Store) DeleteBookmark(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM bookmarks WHERE id = ?`, id)
	return err
}

func (s *Store) setTags(ctx context.Context, bookmarkID string, tags []string) error {
	if _, err := s.exec(ctx, `DELETE FROM bookmark_tags WHERE bookmark_id = ?`, bookmarkID); err != nil {
		return err
	}
	for _, tag := range tags {
		if _, err := s.exec(ctx, `INSERT INTO bookmark_tags (bookmark_id, tag) VALUES (?, ?) ON CONFLICT DO NOTHING`,
			bookmarkID, tag); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) BookmarkByID(ctx context.Context, id string) (*Bookmark, error) {
	var b Bookmark
	if err := s.get(ctx, &b, bookmarkSelect+` WHERE b.id = ?`, id); err != nil {
		return nil, err
	}
	bookmarks := []Bookmark{b}
	if err := s.loadTags(ctx, bookmarks); err != nil {
		return nil, err
	}
	return &bookmarks[0], nil
}

func (s *Store) BookmarksByIDs(ctx context.Context, ids []string) ([]Bookmark, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var bookmarks []Bookmark
	if err := s.selIn(ctx, &bookmarks, bookmarkSelect+` WHERE b.id IN (?)`, ids); err != nil {
		return nil, err
	}
	return bookmarks, s.loadTags(ctx, bookmarks)
}

// BookmarksInCollection lists the newest first.
func (s *Store) BookmarksInCollection(ctx context.Context, collectionID string) ([]Bookmark, error) {
	var bookmarks []Bookmark
	if err := s.sel(ctx, &bookmarks, bookmarkSelect+` WHERE b.collection_id = ? ORDER BY b.created_at DESC, b.id`,
		collectionID); err != nil {
		return nil, err
	}
	return bookmarks, s.loadTags(ctx, bookmarks)
}

// RecentBookmarks lists the newest links of the space outside the hidden collections.
func (s *Store) RecentBookmarks(ctx context.Context, spaceID string, hidden []string, limit int) ([]Bookmark, error) {
	query, args := bookmarkSelect+` WHERE c.space_id = ?`, []any{spaceID}
	query, args = excludeCollections(query, args, hidden)
	var bookmarks []Bookmark
	if err := s.selIn(ctx, &bookmarks, query+` ORDER BY b.created_at DESC, b.id LIMIT ?`, append(args, limit)...); err != nil {
		return nil, err
	}
	return bookmarks, s.loadTags(ctx, bookmarks)
}

// SearchBookmarks matches title, URL, description and tags case-insensitively.
func (s *Store) SearchBookmarks(ctx context.Context, spaceID, query string, hidden []string) ([]Bookmark, error) {
	// Treat LIKE wildcards in user input literally
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(query))
	pattern := "%" + escaped + "%"
	sqlQuery, args := bookmarkSelect+` WHERE c.space_id = ? AND (
		LOWER(b.title) LIKE ? ESCAPE '\' OR LOWER(b.url) LIKE ? ESCAPE '\' OR LOWER(b.description) LIKE ? ESCAPE '\'
		OR EXISTS (SELECT 1 FROM bookmark_tags t WHERE t.bookmark_id = b.id AND LOWER(t.tag) LIKE ? ESCAPE '\'))`,
		[]any{spaceID, pattern, pattern, pattern, pattern}
	sqlQuery, args = excludeCollections(sqlQuery, args, hidden)
	var bookmarks []Bookmark
	if err := s.selIn(ctx, &bookmarks, sqlQuery+` ORDER BY b.created_at DESC, b.id LIMIT 200`, args...); err != nil {
		return nil, err
	}
	return bookmarks, s.loadTags(ctx, bookmarks)
}

func (s *Store) URLsInCollection(ctx context.Context, collectionID string) ([]string, error) {
	var urls []string
	err := s.sel(ctx, &urls, `SELECT url FROM bookmarks WHERE collection_id = ?`, collectionID)
	return urls, err
}

func excludeCollections(query string, args []any, hidden []string) (string, []any) {
	if len(hidden) == 0 {
		return query, args
	}
	return query + ` AND b.collection_id NOT IN (?)`, append(args, hidden)
}

func (s *Store) loadTags(ctx context.Context, bookmarks []Bookmark) error {
	if len(bookmarks) == 0 {
		return nil
	}
	ids := make([]string, len(bookmarks))
	for i, b := range bookmarks {
		ids[i] = b.ID
	}
	var rows []struct {
		BookmarkID string `db:"bookmark_id"`
		Tag        string `db:"tag"`
	}
	if err := s.selIn(ctx, &rows, `SELECT bookmark_id, tag FROM bookmark_tags WHERE bookmark_id IN (?)`, ids); err != nil {
		return err
	}
	byBookmark := map[string][]string{}
	for _, row := range rows {
		byBookmark[row.BookmarkID] = append(byBookmark[row.BookmarkID], row.Tag)
	}
	for i := range bookmarks {
		tags := byBookmark[bookmarks[i].ID]
		sort.Slice(tags, func(a, b int) bool { return strings.ToLower(tags[a]) < strings.ToLower(tags[b]) })
		bookmarks[i].Tags = tags
	}
	return nil
}

// --- Favorites ---

func (s *Store) AddFavorite(ctx context.Context, userID, bookmarkID string) error {
	_, err := s.exec(ctx, `INSERT INTO bookmark_favorites (user_id, bookmark_id, created_at) VALUES (?, ?, ?)
		ON CONFLICT DO NOTHING`, userID, bookmarkID, now())
	return err
}

func (s *Store) RemoveFavorite(ctx context.Context, userID, bookmarkID string) error {
	_, err := s.exec(ctx, `DELETE FROM bookmark_favorites WHERE user_id = ? AND bookmark_id = ?`, userID, bookmarkID)
	return err
}

// FavoriteIDs tells which of the bookmarks the user starred.
func (s *Store) FavoriteIDs(ctx context.Context, userID string, bookmarkIDs []string) (map[string]bool, error) {
	favorites := map[string]bool{}
	if len(bookmarkIDs) == 0 {
		return favorites, nil
	}
	var ids []string
	if err := s.selIn(ctx, &ids, `SELECT bookmark_id FROM bookmark_favorites WHERE user_id = ? AND bookmark_id IN (?)`,
		userID, bookmarkIDs); err != nil {
		return nil, err
	}
	for _, id := range ids {
		favorites[id] = true
	}
	return favorites, nil
}

// Favorites lists the user's starred links of the space, most recently starred first.
func (s *Store) Favorites(ctx context.Context, userID, spaceID string, hidden []string, limit int) ([]Bookmark, error) {
	query := `SELECT ` + bookmarkColumns + ` FROM bookmark_favorites f JOIN bookmarks b ON b.id = f.bookmark_id` +
		bookmarkJoins
	query, args := excludeCollections(query+` WHERE f.user_id = ? AND c.space_id = ?`, []any{userID, spaceID}, hidden)
	var bookmarks []Bookmark
	if err := s.selIn(ctx, &bookmarks, query+` ORDER BY f.created_at DESC LIMIT ?`, append(args, limit)...); err != nil {
		return nil, err
	}
	return bookmarks, s.loadTags(ctx, bookmarks)
}

// --- Dashboard ---

type DashboardSection struct {
	ID          string   `db:"id"`
	Title       string   `db:"title"`
	Type        string   `db:"type"`
	Position    int      `db:"position"`
	BookmarkIDs []string `db:"-"`
}

func (s *Store) DashboardSections(ctx context.Context, userID, spaceID string) ([]DashboardSection, error) {
	var sections []DashboardSection
	if err := s.sel(ctx, &sections, `SELECT id, title, type, position FROM dashboard_sections
		WHERE user_id = ? AND space_id = ? ORDER BY position`, userID, spaceID); err != nil || len(sections) == 0 {
		return sections, err
	}
	ids := make([]string, len(sections))
	for i, section := range sections {
		ids[i] = section.ID
	}
	var items []struct {
		SectionID  string `db:"section_id"`
		BookmarkID string `db:"bookmark_id"`
	}
	if err := s.selIn(ctx, &items, `SELECT section_id, bookmark_id FROM dashboard_items WHERE section_id IN (?)
		ORDER BY position`, ids); err != nil {
		return nil, err
	}
	index := map[string]int{}
	for i, section := range sections {
		index[section.ID] = i
	}
	for _, item := range items {
		sections[index[item.SectionID]].BookmarkIDs = append(sections[index[item.SectionID]].BookmarkIDs, item.BookmarkID)
	}
	return sections, nil
}

func (s *Store) ReplaceDashboard(ctx context.Context, userID, spaceID string, sections []DashboardSection) error {
	if err := s.DeleteDashboard(ctx, userID, spaceID); err != nil {
		return err
	}
	at := now()
	for position, section := range sections {
		id := newID()
		if _, err := s.exec(ctx, `INSERT INTO dashboard_sections (id, user_id, space_id, title, type, position, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, id, userID, spaceID, section.Title, section.Type, position, at); err != nil {
			return err
		}
		for itemPosition, bookmarkID := range section.BookmarkIDs {
			if _, err := s.exec(ctx, `INSERT INTO dashboard_items (section_id, bookmark_id, position) VALUES (?, ?, ?)`,
				id, bookmarkID, itemPosition); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) DeleteDashboard(ctx context.Context, userID, spaceID string) error {
	_, err := s.exec(ctx, `DELETE FROM dashboard_sections WHERE user_id = ? AND space_id = ?`, userID, spaceID)
	return err
}

// --- Invites ---

type Invite struct {
	ID            string     `db:"id"`
	Token         string     `db:"token"`
	TargetType    string     `db:"target_type"`
	TargetID      string     `db:"target_id"`
	Role          string     `db:"role"`
	CreatedBy     string     `db:"created_by"`
	CreatedByName string     `db:"created_by_name"`
	ExpiresAt     *time.Time `db:"expires_at"`
	MaxUses       int        `db:"max_uses"`
	UsedCount     int        `db:"used_count"`
	CreatedAt     time.Time  `db:"created_at"`
}

// Usable is false once the invite expired or was used up.
func (i *Invite) Usable() bool {
	if i.ExpiresAt != nil && i.ExpiresAt.Before(time.Now()) {
		return false
	}
	return i.MaxUses < 0 || i.UsedCount < i.MaxUses
}

const inviteSelect = `SELECT i.id, i.token, i.target_type, i.target_id, i.role, i.created_by, u.username AS created_by_name,
	i.expires_at, i.max_uses, i.used_count, i.created_at FROM invites i JOIN users u ON u.id = i.created_by`

func (s *Store) CreateInvite(ctx context.Context, invite *Invite) error {
	invite.ID, invite.CreatedAt = newID(), now()
	_, err := s.exec(ctx, `INSERT INTO invites (id, token, target_type, target_id, role, created_by, expires_at, max_uses,
		used_count, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?)`, invite.ID, invite.Token, invite.TargetType,
		invite.TargetID, invite.Role, invite.CreatedBy, invite.ExpiresAt, invite.MaxUses, invite.CreatedAt)
	return err
}

func (s *Store) InviteByToken(ctx context.Context, token string) (*Invite, error) {
	var invite Invite
	return &invite, s.get(ctx, &invite, inviteSelect+` WHERE i.token = ?`, token)
}

func (s *Store) InvitesForSpace(ctx context.Context, spaceID string) ([]Invite, error) {
	var invites []Invite
	err := s.sel(ctx, &invites, inviteSelect+` WHERE i.target_type = 'SPACE' AND i.target_id = ? ORDER BY i.created_at DESC`, spaceID)
	return invites, err
}

func (s *Store) InvitesByCreator(ctx context.Context, userID string) ([]Invite, error) {
	var invites []Invite
	err := s.sel(ctx, &invites, inviteSelect+` WHERE i.created_by = ? ORDER BY i.created_at DESC`, userID)
	return invites, err
}

// UseInvite counts a use unless the invite is used up meanwhile (no row changes then).
func (s *Store) UseInvite(ctx context.Context, id string) (bool, error) {
	changed, err := s.exec(ctx, `UPDATE invites SET used_count = used_count + 1
		WHERE id = ? AND (max_uses < 0 OR used_count < max_uses)`, id)
	return changed == 1, err
}

func (s *Store) DeleteInvite(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM invites WHERE id = ?`, id)
	return err
}
