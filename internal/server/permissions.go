package server

import (
	"context"
	"errors"
	"slices"

	"github.com/Serbroda/ragbag/internal/store"
)

type Role string

// Roles from most to least privileged.
const (
	RoleOwner       Role = "OWNER"
	RoleAdmin       Role = "ADMIN"
	RoleContributor Role = "CONTRIBUTOR"
	RoleViewer      Role = "VIEWER"
)

var roleOrder = []Role{RoleOwner, RoleAdmin, RoleContributor, RoleViewer}

type Permission string

const (
	PermRead              Permission = "READ"
	PermEditSpace         Permission = "EDIT_SPACE"
	PermCreateCollections Permission = "CREATE_COLLECTIONS"
	PermEditCollections   Permission = "EDIT_COLLECTIONS"
	PermDeleteCollections Permission = "DELETE_COLLECTIONS"
	PermCreateBookmarks   Permission = "CREATE_BOOKMARKS"
	PermEditBookmarks     Permission = "EDIT_BOOKMARKS"
	PermDeleteBookmarks   Permission = "DELETE_BOOKMARKS"
	PermDeleteSpace       Permission = "DELETE_SPACE"
	PermManageMembers     Permission = "MANAGE_MEMBERS"
	PermImportExport      Permission = "IMPORT_EXPORT"
)

// allPermissions in the order of the OpenAPI enum, which is also the order sent to the frontend.
var allPermissions = []Permission{
	PermRead, PermEditSpace, PermCreateCollections, PermEditCollections, PermDeleteCollections, PermCreateBookmarks,
	PermEditBookmarks, PermDeleteBookmarks, PermDeleteSpace, PermManageMembers, PermImportExport,
}

// rolePermissions is the single source of truth for what each role may do.
var rolePermissions = map[Role][]Permission{
	RoleOwner: allPermissions,
	RoleAdmin: slices.DeleteFunc(slices.Clone(allPermissions), func(p Permission) bool { return p == PermDeleteSpace }),
	RoleContributor: {PermRead, PermCreateCollections, PermEditCollections, PermDeleteCollections, PermCreateBookmarks,
		PermEditBookmarks, PermDeleteBookmarks},
	RoleViewer: {PermRead},
}

func (r Role) Has(p Permission) bool { return slices.Contains(rolePermissions[r], p) }

func (r Role) rank() int { return slices.Index(roleOrder, r) }

// AtLeast tells whether r is as privileged as other or more.
func (r Role) AtLeast(other Role) bool { return r.rank() <= other.rank() }

func validRole(r Role) bool { return slices.Contains(roleOrder, r) }

// requireSpace checks that the space exists (404) and the user's role grants the permission (403).
func (s *Server) requireSpace(ctx context.Context, tx *store.Store, spaceID, userID string, p Permission) (Role, error) {
	if _, err := tx.SpaceByID(ctx, spaceID); err != nil {
		return "", orNotFound(err, "Space not found: %s", spaceID)
	}
	role, err := tx.MemberRole(ctx, spaceID, userID)
	if errors.Is(err, store.ErrNotFound) {
		return "", forbidden("Not a member of this space")
	}
	if err != nil {
		return "", err
	}
	if !Role(role).Has(p) {
		return "", forbidden("Missing permission %s", p)
	}
	return Role(role), nil
}

// requireCollection also hides private collections: for everybody else they do not exist (404, not 403).
func (s *Server) requireCollection(ctx context.Context, tx *store.Store, collectionID, userID string, p Permission) (*store.Collection, Role, error) {
	collection, err := tx.CollectionByID(ctx, collectionID)
	if err != nil {
		return nil, "", orNotFound(err, "Collection not found: %s", collectionID)
	}
	hidden, err := s.hiddenCollections(ctx, tx, collection.SpaceID, userID)
	if err != nil {
		return nil, "", err
	}
	if hidden[collection.ID] {
		return nil, "", notFound("Collection not found: %s", collectionID)
	}
	role, err := s.requireSpace(ctx, tx, collection.SpaceID, userID, p)
	return collection, role, err
}

func (s *Server) requireBookmark(ctx context.Context, tx *store.Store, bookmarkID, userID string, p Permission) (*store.Bookmark, error) {
	bookmark, err := tx.BookmarkByID(ctx, bookmarkID)
	if err != nil {
		return nil, orNotFound(err, "Bookmark not found: %s", bookmarkID)
	}
	if _, _, err := s.requireCollection(ctx, tx, bookmark.CollectionID, userID, p); err != nil {
		if errors.As(err, new(*apiError)) && err.(*apiError).status == 404 {
			return nil, notFound("Bookmark not found: %s", bookmarkID)
		}
		return nil, err
	}
	return bookmark, nil
}

// hiddenCollections are the private collections of the space the user is not a member of, and everything
// below them. Visibility is inherited: a public collection inside a private one is private too.
func (s *Server) hiddenCollections(ctx context.Context, tx *store.Store, spaceID, userID string) (map[string]bool, error) {
	hidden := map[string]bool{}
	hasPrivate, err := tx.HasPrivateCollections(ctx, spaceID)
	if err != nil || !hasPrivate {
		return hidden, err
	}
	collections, err := tx.CollectionsBySpace(ctx, spaceID)
	if err != nil {
		return nil, err
	}
	memberOf, err := tx.PrivateCollectionsOf(ctx, spaceID, userID)
	if err != nil {
		return nil, err
	}
	byID := map[string]*store.Collection{}
	for i := range collections {
		byID[collections[i].ID] = &collections[i]
	}
	visible := map[string]bool{}
	var isVisible func(c *store.Collection, depth int) bool
	isVisible = func(c *store.Collection, depth int) bool {
		if known, ok := visible[c.ID]; ok {
			return known
		}
		result := !c.Restricted || slices.Contains(memberOf, c.ID)
		if result && c.ParentID != nil && depth < 1000 {
			if parent := byID[*c.ParentID]; parent != nil {
				result = isVisible(parent, depth+1)
			}
		}
		visible[c.ID] = result
		return result
	}
	for i := range collections {
		if !isVisible(&collections[i], 0) {
			hidden[collections[i].ID] = true
		}
	}
	return hidden, nil
}

func keys(set map[string]bool) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	return result
}
