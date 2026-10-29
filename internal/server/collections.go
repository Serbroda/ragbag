package server

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

const maxDescriptionLength = 500

func collectionDto(c *store.Collection) api.CollectionDto {
	restricted := c.Restricted
	return api.CollectionDto{Id: c.ID, Name: c.Name, Description: c.Description, ParentId: c.ParentID, Restricted: &restricted}
}

// GetCollections returns the tree of all collections the user can see.
func (s *Server) GetCollections(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	collections, err := s.store.CollectionsBySpace(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	counts, err := s.store.BookmarkCounts(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	news, err := s.visibleNews(ctxOf(c), currentUser(c), spaceID)
	if err != nil {
		return err
	}
	newByCollection := newCounts(news, func(n store.NewBookmark) string { return n.CollectionID })
	shared, err := s.store.SharedCollections(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	byParent := map[string][]*store.Collection{}
	for i := range collections {
		collection := &collections[i]
		if hidden[collection.ID] {
			continue
		}
		parent := ""
		if collection.ParentID != nil {
			parent = *collection.ParentID
		}
		byParent[parent] = append(byParent[parent], collection)
	}
	var build func(parent string) []api.CollectionDto
	build = func(parent string) []api.CollectionDto {
		nodes := []api.CollectionDto{}
		for _, collection := range byParent[parent] {
			node := collectionDto(collection)
			children := build(collection.ID)
			count, newCount := counts[collection.ID], newByCollection[collection.ID]
			isShared := shared[collection.ID]
			node.Children, node.BookmarkCount, node.NewCount, node.Shared = &children, &count, &newCount, &isShared
			nodes = append(nodes, node)
		}
		return nodes
	}
	return c.JSON(http.StatusOK, build(""))
}

func (s *Server) GetCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	dto := collectionDto(collection)
	_, err = s.store.ShareOf(ctxOf(c), collectionID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	isShared := err == nil
	dto.Shared = &isShared
	return c.JSON(http.StatusOK, dto)
}

func (s *Server) CreateCollection(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermCreateCollections); err != nil {
		return err
	}
	var req api.CreateCollectionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	name, err := requireName(req.Name, "Collection")
	if err != nil {
		return err
	}
	collection := &store.Collection{SpaceID: spaceID, Name: name, CreatedBy: currentUser(c)}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if req.ParentId != nil {
			parent, _, err := s.requireCollection(ctxOf(c), tx, *req.ParentId, currentUser(c), PermRead)
			if err != nil {
				return err
			}
			if parent.SpaceID != spaceID {
				return badRequest("Parent collection belongs to a different space")
			}
			collection.ParentID = &parent.ID
		}
		var memberIDs []string
		if req.Restricted != nil && *req.Restricted {
			collection.Restricted = true
			if memberIDs, err = s.requireSpaceMembers(c, tx, spaceID, req.MemberIds); err != nil {
				return err
			}
		}
		return tx.CreateCollection(ctxOf(c), collection, memberIDs)
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, collectionDto(collection))
}

// requireSpaceMembers returns the given users plus the current one, who must never lock themselves out.
func (s *Server) requireSpaceMembers(c echo.Context, tx *store.Store, spaceID string, userIDs *[]api.ID) ([]string, error) {
	ids := []string{currentUser(c)}
	if userIDs != nil {
		for _, id := range *userIDs {
			if !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
	}
	for _, id := range ids {
		if _, err := tx.MemberRole(ctxOf(c), spaceID, id); err != nil {
			return nil, orBadRequest(err, "Only members of the space can be given access: %s", id)
		}
	}
	return ids, nil
}

func orBadRequest(err error, format string, args ...any) error {
	if errors.Is(err, store.ErrNotFound) {
		return badRequest(format, args...)
	}
	return err
}

func (s *Server) UpdateCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermEditCollections)
	if err != nil {
		return err
	}
	var req api.UpdateCollectionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	name, err := requireName(req.Name, "Collection")
	if err != nil {
		return err
	}
	if req.Description != nil {
		description := strings.TrimSpace(*req.Description)
		if len([]rune(description)) > maxDescriptionLength {
			return badRequest("Collection description must be at most %d characters", maxDescriptionLength)
		}
		collection.Description = nilIfEmpty(description)
	}
	if err := s.store.UpdateCollection(ctxOf(c), collectionID, name, collection.Description); err != nil {
		return err
	}
	collection.Name = name
	return c.JSON(http.StatusOK, collectionDto(collection))
}

func (s *Server) MoveCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermEditCollections)
	if err != nil {
		return err
	}
	var req api.MoveCollectionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		if req.ParentId == nil {
			collection.ParentID = nil
			return tx.MoveCollection(ctxOf(c), collectionID, nil)
		}
		if *req.ParentId == collectionID {
			return badRequest("A collection cannot be its own parent")
		}
		parent, _, err := s.requireCollection(ctxOf(c), tx, *req.ParentId, currentUser(c), PermRead)
		if err != nil {
			return err
		}
		if parent.SpaceID != collection.SpaceID {
			return badRequest("Parent collection belongs to a different space")
		}
		for ancestor := parent; ; {
			if ancestor.ID == collectionID {
				return badRequest("A collection cannot be moved into its own descendant")
			}
			if ancestor.ParentID == nil {
				break
			}
			if ancestor, err = tx.CollectionByID(ctxOf(c), *ancestor.ParentID); err != nil {
				return err
			}
		}
		collection.ParentID = &parent.ID
		return tx.MoveCollection(ctxOf(c), collectionID, &parent.ID)
	})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, collectionDto(collection))
}

func (s *Server) DeleteCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermDeleteCollections)
	if err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		// Deleting would silently take other members' private collections with it
		hidden, err := s.hiddenCollections(ctxOf(c), tx, collection.SpaceID, currentUser(c))
		if err != nil {
			return err
		}
		if len(hidden) > 0 {
			subtree, err := s.subtree(c, tx, collection)
			if err != nil {
				return err
			}
			for _, id := range subtree {
				if hidden[id] {
					return conflict("This collection contains private collections of other members and cannot be deleted")
				}
			}
		}
		return tx.DeleteCollection(ctxOf(c), collectionID)
	})
	if err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// subtree lists the ids of the collection and everything below it.
func (s *Server) subtree(c echo.Context, tx *store.Store, root *store.Collection) ([]string, error) {
	collections, err := tx.CollectionsBySpace(ctxOf(c), root.SpaceID)
	if err != nil {
		return nil, err
	}
	children := map[string][]string{}
	for _, collection := range collections {
		if collection.ParentID != nil {
			children[*collection.ParentID] = append(children[*collection.ParentID], collection.ID)
		}
	}
	ids := []string{root.ID}
	for i := 0; i < len(ids); i++ {
		ids = append(ids, children[ids[i]]...)
	}
	return ids, nil
}

// --- Who can see a collection ---

func (s *Server) GetCollectionAccess(c echo.Context, collectionID api.ID) error {
	collection, role, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	return s.accessView(c, collection, role)
}

func (s *Server) UpdateCollectionAccess(c echo.Context, collectionID api.ID) error {
	collection, role, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermEditCollections)
	if err != nil {
		return err
	}
	if !s.canManageAccess(c, collection, role) {
		return forbidden("Only the creator of the collection or space admins can change who sees it")
	}
	var req api.UpdateCollectionAccessDto
	if err := bind(c, &req); err != nil {
		return err
	}
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		var memberIDs []string
		if req.Restricted {
			if memberIDs, err = s.requireSpaceMembers(c, tx, collection.SpaceID, req.MemberIds); err != nil {
				return err
			}
		}
		return tx.SetCollectionAccess(ctxOf(c), collectionID, req.Restricted, memberIDs)
	})
	if err != nil {
		return err
	}
	collection.Restricted = req.Restricted
	return s.accessView(c, collection, role)
}

// canManageAccess: private collections are managed by the people who can see them. Making a shared collection
// private hides it from everybody else, so only its creator or a space admin may do that.
func (s *Server) canManageAccess(c echo.Context, collection *store.Collection, role Role) bool {
	if !role.Has(PermEditCollections) {
		return false
	}
	return collection.Restricted || collection.CreatedBy == currentUser(c) || role.Has(PermEditSpace)
}

func (s *Server) accessView(c echo.Context, collection *store.Collection, role Role) error {
	memberIDs, err := s.store.CollectionMemberIDs(ctxOf(c), collection.ID)
	if err != nil {
		return err
	}
	users, err := s.store.UsersByIDs(ctxOf(c), memberIDs)
	if err != nil {
		return err
	}
	members := make([]api.CollectionAccessMemberDto, 0, len(users))
	for _, user := range users {
		members = append(members, api.CollectionAccessMemberDto{UserId: user.ID, Username: user.Username, AvatarId: user.AvatarImageID})
	}
	sort.Slice(members, func(i, j int) bool { return members[i].Username < members[j].Username })
	return c.JSON(http.StatusOK, api.CollectionAccessDto{
		Restricted: collection.Restricted, Members: members, CanManage: s.canManageAccess(c, collection, role),
	})
}
