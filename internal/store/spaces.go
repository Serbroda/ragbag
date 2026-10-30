package store

import (
	"context"
	"strings"
	"time"
)

type Space struct {
	ID          string  `db:"id"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	IconImageID *string `db:"icon_image_id"`
	// Discoverable spaces are listed for every account, which can join them with JoinRole
	Discoverable bool      `db:"discoverable"`
	JoinRole     string    `db:"join_role"`
	CreatedBy    string    `db:"created_by"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

const spaceColumns = `s.id, s.name, s.description, s.icon_image_id, s.discoverable, s.join_role, s.created_by, s.created_at,
	s.updated_at`

// SpaceWithRole is a space as one of its members sees it.
type SpaceWithRole struct {
	Space
	Role string `db:"role"`
}

func (s *Store) CreateSpace(ctx context.Context, name string, description *string, createdBy string) (*Space, error) {
	at := now()
	space := &Space{ID: newID(), Name: name, Description: description, JoinRole: "VIEWER", CreatedBy: createdBy,
		CreatedAt: at, UpdatedAt: at}
	_, err := s.exec(ctx, `INSERT INTO spaces (id, name, description, icon_image_id, discoverable, join_role, created_by,
		created_at, updated_at) VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?)`, space.ID, space.Name, space.Description, false,
		space.JoinRole, space.CreatedBy, at, at)
	return space, err
}

func (s *Store) SpaceByID(ctx context.Context, id string) (*Space, error) {
	var space Space
	return &space, s.get(ctx, &space, `SELECT `+spaceColumns+` FROM spaces s WHERE s.id = ?`, id)
}

func (s *Store) UpdateSpace(ctx context.Context, id, name string, description *string) error {
	_, err := s.exec(ctx, `UPDATE spaces SET name = ?, description = ?, updated_at = ? WHERE id = ?`, name, description, now(), id)
	return err
}

func (s *Store) SetSpaceDiscoverable(ctx context.Context, id string, discoverable bool, joinRole string) error {
	_, err := s.exec(ctx, `UPDATE spaces SET discoverable = ?, join_role = ?, updated_at = ? WHERE id = ?`,
		discoverable, joinRole, now(), id)
	return err
}

// DiscoverableSpace is a discoverable space as anybody sees it; Role is set if the user is a member already.
type DiscoverableSpace struct {
	Space
	MemberCount int     `db:"member_count"`
	Role        *string `db:"role"`
}

// DiscoverableSpaces lists the largest discoverable spaces matching the query (name or description).
func (s *Store) DiscoverableSpaces(ctx context.Context, userID, query string, limit int) ([]DiscoverableSpace, error) {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(strings.TrimSpace(query)))
	pattern := "%" + escaped + "%"
	var spaces []DiscoverableSpace
	err := s.sel(ctx, &spaces, `SELECT `+spaceColumns+`,
			(SELECT COUNT(*) FROM space_members a WHERE a.space_id = s.id) AS member_count, m.role
		FROM spaces s LEFT JOIN space_members m ON m.space_id = s.id AND m.user_id = ?
		WHERE s.discoverable = ? AND (LOWER(s.name) LIKE ? ESCAPE '\' OR LOWER(COALESCE(s.description, '')) LIKE ? ESCAPE '\')
		ORDER BY member_count DESC, LOWER(s.name), s.id LIMIT ?`, userID, true, pattern, pattern, limit)
	return spaces, err
}

func (s *Store) SetSpaceIcon(ctx context.Context, id string, imageID *string) error {
	_, err := s.exec(ctx, `UPDATE spaces SET icon_image_id = ?, updated_at = ? WHERE id = ?`, imageID, now(), id)
	return err
}

// DeleteSpace removes the space; members, collections, bookmarks and dashboards go with it (ON DELETE CASCADE).
func (s *Store) DeleteSpace(ctx context.Context, id string) error {
	if _, err := s.exec(ctx, `DELETE FROM invites WHERE target_type = 'SPACE' AND target_id = ?`, id); err != nil {
		return err
	}
	_, err := s.exec(ctx, `DELETE FROM spaces WHERE id = ?`, id)
	return err
}

func (s *Store) SpaceForMember(ctx context.Context, spaceID, userID string) (*SpaceWithRole, error) {
	var space SpaceWithRole
	return &space, s.get(ctx, &space, `SELECT `+spaceColumns+`, m.role FROM spaces s
		JOIN space_members m ON m.space_id = s.id WHERE s.id = ? AND m.user_id = ?`, spaceID, userID)
}

func (s *Store) SpacesForUser(ctx context.Context, userID string) ([]SpaceWithRole, error) {
	var spaces []SpaceWithRole
	err := s.sel(ctx, &spaces, `SELECT `+spaceColumns+`, m.role FROM spaces s
		JOIN space_members m ON m.space_id = s.id WHERE m.user_id = ?`, userID)
	return spaces, err
}

// --- Members ---

type Member struct {
	SpaceID       string  `db:"space_id"`
	UserID        string  `db:"user_id"`
	Role          string  `db:"role"`
	Username      string  `db:"username"`
	AvatarImageID *string `db:"avatar_image_id"`
}

const memberSelect = `SELECT m.space_id, m.user_id, m.role, u.username, u.avatar_image_id
	FROM space_members m JOIN users u ON u.id = m.user_id`

// MemberRole returns ErrNotFound if the user is not a member.
func (s *Store) MemberRole(ctx context.Context, spaceID, userID string) (string, error) {
	var role string
	return role, s.get(ctx, &role, `SELECT role FROM space_members WHERE space_id = ? AND user_id = ?`, spaceID, userID)
}

func (s *Store) Member(ctx context.Context, spaceID, userID string) (*Member, error) {
	var member Member
	return &member, s.get(ctx, &member, memberSelect+` WHERE m.space_id = ? AND m.user_id = ?`, spaceID, userID)
}

func (s *Store) Members(ctx context.Context, spaceID string) ([]Member, error) {
	var members []Member
	err := s.sel(ctx, &members, memberSelect+` WHERE m.space_id = ?`, spaceID)
	return members, err
}

func (s *Store) AddMember(ctx context.Context, spaceID, userID, role string) error {
	_, err := s.exec(ctx, `INSERT INTO space_members (space_id, user_id, role, created_at) VALUES (?, ?, ?, ?)`,
		spaceID, userID, role, now())
	return err
}

func (s *Store) SetMemberRole(ctx context.Context, spaceID, userID, role string) error {
	_, err := s.exec(ctx, `UPDATE space_members SET role = ? WHERE space_id = ? AND user_id = ?`, role, spaceID, userID)
	return err
}

func (s *Store) CountRole(ctx context.Context, spaceID, role string) (int, error) {
	var count int
	return count, s.get(ctx, &count, `SELECT COUNT(*) FROM space_members WHERE space_id = ? AND role = ?`, spaceID, role)
}

// RemoveMember also removes the personal data of the membership: dashboard, favorites and access to private
// collections of the space.
func (s *Store) RemoveMember(ctx context.Context, spaceID, userID string) error {
	statements := []string{
		`DELETE FROM dashboard_sections WHERE user_id = ? AND space_id = ?`,
		`DELETE FROM bookmark_favorites WHERE user_id = ? AND bookmark_id IN (
			SELECT b.id FROM bookmarks b JOIN collections c ON c.id = b.collection_id WHERE c.space_id = ?)`,
		`DELETE FROM bookmark_reactions WHERE user_id = ? AND bookmark_id IN (
			SELECT b.id FROM bookmarks b JOIN collections c ON c.id = b.collection_id WHERE c.space_id = ?)`,
		`DELETE FROM collection_visits WHERE user_id = ? AND collection_id IN (SELECT id FROM collections WHERE space_id = ?)`,
		`DELETE FROM collection_members WHERE user_id = ? AND collection_id IN (SELECT id FROM collections WHERE space_id = ?)`,
		`UPDATE users SET start_space_id = NULL WHERE id = ? AND start_space_id = ?`,
		`DELETE FROM space_members WHERE user_id = ? AND space_id = ?`,
	}
	for _, statement := range statements {
		if _, err := s.exec(ctx, statement, userID, spaceID); err != nil {
			return err
		}
	}
	// Private collections nobody can see anymore would be unreachable data
	_, err := s.exec(ctx, `DELETE FROM collections WHERE space_id = ? AND restricted = ?
		AND NOT EXISTS (SELECT 1 FROM collection_members cm WHERE cm.collection_id = collections.id)`, spaceID, true)
	return err
}
