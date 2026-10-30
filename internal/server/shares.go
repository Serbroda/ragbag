package server

import (
	"errors"
	"net/http"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Public read-only links: anybody with the link sees the shared collection and its sub-collections, nothing else
// of the space. Private collections can neither be shared nor be seen through a link, which is checked on every
// request: a shared collection that becomes private (or is moved below a private one) is unreachable meanwhile.

const shareNotFound = "This link does not exist or was revoked"

// spaceCollections lists the collections of a space in tree order and indexes them by id.
func (s *Server) spaceCollections(c echo.Context, spaceID string) ([]store.Collection, map[string]*store.Collection, error) {
	collections, err := s.store.CollectionsBySpace(ctxOf(c), spaceID)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]*store.Collection, len(collections))
	for i := range collections {
		byID[collections[i].ID] = &collections[i]
	}
	return collections, byID, nil
}

// privateInPath tells whether the collection or one of its ancestors is private or in the trash.
func privateInPath(byID map[string]*store.Collection, collection *store.Collection) bool {
	for current, depth := collection, 0; current != nil && depth < 1000; depth++ {
		if current.Restricted || current.DeletedAt != nil {
			return true
		}
		if current.ParentID == nil {
			return false
		}
		current = byID[*current.ParentID]
	}
	return false
}

func (s *Server) GetCollectionShare(c echo.Context, collectionID api.ID) error {
	_, role, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	share, err := s.store.ShareOf(ctxOf(c), collectionID)
	if errors.Is(err, store.ErrNotFound) {
		return c.JSON(http.StatusOK, api.CollectionShareDto{Shared: false})
	}
	if err != nil {
		return err
	}
	dto := api.CollectionShareDto{Shared: true, CreatedAt: &share.CreatedAt}
	// Only those who manage the link get it; viewers merely learn that the collection is public
	if role.Has(PermEditCollections) {
		dto.Token = &share.Token
	}
	return c.JSON(http.StatusOK, dto)
}

func (s *Server) ShareCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermEditCollections)
	if err != nil {
		return err
	}
	_, byID, err := s.spaceCollections(c, collection.SpaceID)
	if err != nil {
		return err
	}
	if privateInPath(byID, collection) {
		return badRequest("Private collections cannot be shared publicly")
	}
	share := &store.Share{CollectionID: collectionID, Token: randomToken(), CreatedBy: currentUser(c)}
	if err := s.store.ReplaceShare(ctxOf(c), share); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.CollectionShareDto{Shared: true, Token: &share.Token, CreatedAt: &share.CreatedAt})
}

func (s *Server) UnshareCollection(c echo.Context, collectionID api.ID) error {
	if _, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermEditCollections); err != nil {
		return err
	}
	if err := s.store.DeleteShare(ctxOf(c), collectionID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetSharedCollection answers the public link. Every failure is the same 404, so links cannot be probed.
func (s *Server) GetSharedCollection(c echo.Context, token string, params api.GetSharedCollectionParams) error {
	dto, err := s.sharedCollection(c, token, params.CollectionId)
	if err != nil {
		return err
	}
	c.Response().Header().Set("X-Robots-Tag", "noindex")
	return c.JSON(http.StatusOK, dto)
}

// sharedCollection is what a public link shows; the page and its link preview (meta.go) both go through here.
func (s *Server) sharedCollection(c echo.Context, token string, collectionID *string) (*api.SharedCollectionDto, error) {
	share, err := s.store.ShareByToken(ctxOf(c), token)
	if err != nil {
		return nil, orNotFound(err, shareNotFound)
	}
	root, err := s.store.CollectionByID(ctxOf(c), share.CollectionID)
	if err != nil {
		return nil, orNotFound(err, shareNotFound)
	}
	collections, byID, err := s.spaceCollections(c, root.SpaceID)
	if err != nil {
		return nil, err
	}
	if privateInPath(byID, root) {
		return nil, notFound(shareNotFound)
	}

	// The requested collection must be the shared one or below it, with nothing private in between
	target := root
	if collectionID != nil && *collectionID != root.ID {
		target = byID[*collectionID]
		if target == nil {
			return nil, notFound(shareNotFound)
		}
	}
	path := []*store.Collection{}
	for current := target; ; {
		if current.Restricted || current.DeletedAt != nil {
			return nil, notFound(shareNotFound)
		}
		path = append([]*store.Collection{current}, path...)
		if current.ID == root.ID {
			break
		}
		if current.ParentID == nil || len(path) > 1000 {
			return nil, notFound(shareNotFound)
		}
		current = byID[*current.ParentID]
		if current == nil {
			return nil, notFound(shareNotFound)
		}
	}

	counts, err := s.store.BookmarkCounts(ctxOf(c), root.SpaceID)
	if err != nil {
		return nil, err
	}
	// In the order of the app's collection tree
	publicChildren := map[string][]*store.Collection{}
	for i := range collections {
		collection := &collections[i]
		if collection.ParentID != nil && !collection.Restricted && collection.DeletedAt == nil {
			publicChildren[*collection.ParentID] = append(publicChildren[*collection.ParentID], collection)
		}
	}
	ref := func(collection *store.Collection) api.SharedCollectionRefDto {
		return api.SharedCollectionRefDto{Id: collection.ID, Name: collection.Name,
			BookmarkCount: counts[collection.ID], ChildCount: len(publicChildren[collection.ID])}
	}
	dto := api.SharedCollectionDto{Id: target.ID, Name: target.Name, Description: target.Description,
		Path: []api.SharedCollectionRefDto{}, Children: []api.SharedCollectionRefDto{}, Bookmarks: []api.SharedBookmarkDto{}}
	for _, collection := range path {
		dto.Path = append(dto.Path, ref(collection))
	}
	for _, child := range publicChildren[target.ID] {
		dto.Children = append(dto.Children, ref(child))
	}

	bookmarks, err := s.store.BookmarksInCollection(ctxOf(c), target.ID)
	if err != nil {
		return nil, err
	}
	for _, b := range bookmarks {
		tags, createdAt := b.Tags, b.CreatedAt
		if tags == nil {
			tags = []string{}
		}
		// The preview image as the app shows it: an upload, an own address, the page's, or none
		image, imageID := b.OgImage, b.ImageID
		if b.ImageURL != nil {
			image = b.ImageURL
		}
		if b.ImageHidden {
			image, imageID = nil, nil
		}
		// Deliberately without who added it, favorites or reactions
		dto.Bookmarks = append(dto.Bookmarks, api.SharedBookmarkDto{Id: b.ID, Url: b.URL, Title: b.Title,
			Description: b.Description, Tags: tags, OgImage: image, CustomImageId: imageID, Favicon: b.Favicon,
			CreatedAt: &createdAt})
	}
	return &dto, nil
}
