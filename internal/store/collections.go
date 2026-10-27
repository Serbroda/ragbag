package store

import (
	"context"
	"time"
)

type Collection struct {
	ID          string    `db:"id"`
	SpaceID     string    `db:"space_id"`
	ParentID    *string   `db:"parent_id"`
	Name        string    `db:"name"`
	Description *string   `db:"description"`
	Restricted  bool      `db:"restricted"`
	CreatedBy   string    `db:"created_by"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}

const collectionColumns = `id, space_id, parent_id, name, description, restricted, created_by, created_at, updated_at`

func (s *Store) CreateCollection(ctx context.Context, c *Collection, memberIDs []string) error {
	at := now()
	c.ID, c.CreatedAt, c.UpdatedAt = newID(), at, at
	if _, err := s.exec(ctx, `INSERT INTO collections (`+collectionColumns+`) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.SpaceID, c.ParentID, c.Name, c.Description, c.Restricted, c.CreatedBy, at, at); err != nil {
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
	err := s.sel(ctx, &collections, `SELECT `+collectionColumns+` FROM collections WHERE space_id = ? ORDER BY created_at, id`, spaceID)
	return collections, err
}

func (s *Store) RenameCollection(ctx context.Context, id, name string) error {
	_, err := s.exec(ctx, `UPDATE collections SET name = ?, updated_at = ? WHERE id = ?`, name, now(), id)
	return err
}

func (s *Store) MoveCollection(ctx context.Context, id string, parentID *string) error {
	_, err := s.exec(ctx, `UPDATE collections SET parent_id = ?, updated_at = ? WHERE id = ?`, parentID, now(), id)
	return err
}

// DeleteCollection removes it with everything below (ON DELETE CASCADE).
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

func (s *Store) HasPrivateCollections(ctx context.Context, spaceID string) (bool, error) {
	var count int
	err := s.get(ctx, &count, `SELECT COUNT(*) FROM collections WHERE space_id = ? AND restricted = ?`, spaceID, true)
	return count > 0, err
}
