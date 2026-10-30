package store

import (
	"context"
	"time"
)

type Collection struct {
	ID          string  `db:"id"`
	SpaceID     string  `db:"space_id"`
	ParentID    *string `db:"parent_id"`
	Name        string  `db:"name"`
	Description *string `db:"description"`
	Restricted  bool    `db:"restricted"`
	// Position orders collections below the same parent
	Position  int    `db:"position"`
	CreatedBy string `db:"created_by"`
	// DeletedAt is set while the collection is in the trash (everything below it is hidden with it)
	DeletedAt *time.Time `db:"deleted_at"`
	DeletedBy *string    `db:"deleted_by"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

const collectionColumns = `id, space_id, parent_id, name, description, restricted, position, created_by, created_at,
	updated_at, deleted_at, deleted_by`

// CreateCollection appends the collection after its future siblings.
func (s *Store) CreateCollection(ctx context.Context, c *Collection, memberIDs []string) error {
	at := now()
	c.ID, c.CreatedAt, c.UpdatedAt = newID(), at, at
	position, err := s.nextPosition(ctx, c.SpaceID, c.ParentID)
	if err != nil {
		return err
	}
	c.Position = position
	if _, err := s.exec(ctx, `INSERT INTO collections (id, space_id, parent_id, name, description, restricted, position,
		created_by, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.SpaceID, c.ParentID, c.Name, c.Description, c.Restricted, c.Position, c.CreatedBy, at, at); err != nil {
		return err
	}
	return s.addCollectionMembers(ctx, c.ID, memberIDs)
}

func (s *Store) CollectionByID(ctx context.Context, id string) (*Collection, error) {
	var c Collection
	return &c, s.get(ctx, &c, `SELECT `+collectionColumns+` FROM collections WHERE id = ?`, id)
}

func (s *Store) CollectionsBySpace(ctx context.Context, spaceID string) ([]Collection, error) {
	var collections []Collection
	err := s.sel(ctx, &collections, `SELECT `+collectionColumns+` FROM collections WHERE space_id = ?
		ORDER BY position, created_at, id`, spaceID)
	return collections, err
}

func (s *Store) UpdateCollection(ctx context.Context, id, name string, description *string) error {
	_, err := s.exec(ctx, `UPDATE collections SET name = ?, description = ?, updated_at = ? WHERE id = ?`,
		name, description, now(), id)
	return err
}

// BookmarkCounts maps the collections of the space to the number of links directly in them (none: missing).
func (s *Store) BookmarkCounts(ctx context.Context, spaceID string) (map[string]int, error) {
	var rows []struct {
		CollectionID string `db:"collection_id"`
		Count        int    `db:"n"`
	}
	if err := s.sel(ctx, &rows, `SELECT b.collection_id, COUNT(*) AS n FROM bookmarks b
		JOIN collections c ON c.id = b.collection_id WHERE b.deleted_at IS NULL AND c.space_id = ? GROUP BY b.collection_id`, spaceID); err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[row.CollectionID] = row.Count
	}
	return counts, nil
}

// nextPosition is the position after the last collection below the parent (nil: top level).
func (s *Store) nextPosition(ctx context.Context, spaceID string, parentID *string) (int, error) {
	var positions []int
	var err error
	if parentID == nil {
		err = s.sel(ctx, &positions, `SELECT position FROM collections WHERE space_id = ? AND parent_id IS NULL`, spaceID)
	} else {
		err = s.sel(ctx, &positions, `SELECT position FROM collections WHERE space_id = ? AND parent_id = ?`, spaceID, *parentID)
	}
	next := 0
	for _, position := range positions {
		next = max(next, position+1)
	}
	return next, err
}

func (s *Store) MoveCollection(ctx context.Context, id string, parentID *string) error {
	_, err := s.exec(ctx, `UPDATE collections SET parent_id = ?, updated_at = ? WHERE id = ?`, parentID, now(), id)
	return err
}

// OrderCollections numbers the given collections (siblings) in this order.
func (s *Store) OrderCollections(ctx context.Context, ids []string) error {
	for position, id := range ids {
		if _, err := s.exec(ctx, `UPDATE collections SET position = ? WHERE id = ?`, position, id); err != nil {
			return err
		}
	}
	return nil
}

// TrashCollection moves the collection (and with it everything below) to the trash.
func (s *Store) TrashCollection(ctx context.Context, id, userID string) error {
	_, err := s.exec(ctx, `UPDATE collections SET deleted_at = ?, deleted_by = ? WHERE id = ?`, now(), userID, id)
	return err
}

func (s *Store) RestoreCollection(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `UPDATE collections SET deleted_at = NULL, deleted_by = NULL WHERE id = ?`, id)
	return err
}

// DeleteCollection removes it with everything below for good (ON DELETE CASCADE).
func (s *Store) DeleteCollection(ctx context.Context, id string) error {
	_, err := s.exec(ctx, `DELETE FROM collections WHERE id = ?`, id)
	return err
}

func (s *Store) SetCollectionAccess(ctx context.Context, id string, restricted bool, memberIDs []string) error {
	if _, err := s.exec(ctx, `UPDATE collections SET restricted = ?, updated_at = ? WHERE id = ?`, restricted, now(), id); err != nil {
		return err
	}
	if _, err := s.exec(ctx, `DELETE FROM collection_members WHERE collection_id = ?`, id); err != nil {
		return err
	}
	return s.addCollectionMembers(ctx, id, memberIDs)
}

func (s *Store) addCollectionMembers(ctx context.Context, collectionID string, userIDs []string) error {
	for _, userID := range userIDs {
		if _, err := s.exec(ctx, `INSERT INTO collection_members (collection_id, user_id) VALUES (?, ?)`, collectionID, userID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CollectionMemberIDs(ctx context.Context, collectionID string) ([]string, error) {
	var ids []string
	err := s.sel(ctx, &ids, `SELECT user_id FROM collection_members WHERE collection_id = ?`, collectionID)
	return ids, err
}

// PrivateCollectionsOf lists the private collections of the space the user is a member of.
func (s *Store) PrivateCollectionsOf(ctx context.Context, spaceID, userID string) ([]string, error) {
	var ids []string
	err := s.sel(ctx, &ids, `SELECT cm.collection_id FROM collection_members cm
		JOIN collections c ON c.id = cm.collection_id WHERE c.space_id = ? AND cm.user_id = ?`, spaceID, userID)
	return ids, err
}

// HasHiddenCandidates tells whether the space has private collections or collections in the trash, the two
// things that can hide a collection from everybody or some.
func (s *Store) HasHiddenCandidates(ctx context.Context, spaceID string) (bool, error) {
	var count int
	err := s.get(ctx, &count, `SELECT COUNT(*) FROM collections WHERE space_id = ? AND (restricted = ? OR deleted_at IS NOT NULL)`,
		spaceID, true)
	return count > 0, err
}
