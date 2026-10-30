package server

import (
	"context"
	"log/slog"
	"net/http"
	"sort"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// The trash: deleting only marks links and collections (deleted_at), hiddenCollections and the bookmark joins
// leave them out everywhere, and PurgeOldTrash deletes them for good after trashRetention. The trash shows what
// the user could see before it was deleted: private and locked collections stay out of it for everybody else.

const trashRetention = 30 * 24 * time.Hour

// trashView is the space's trash as the current user may see it.
type trashView struct {
	collections []*store.Collection
	bookmarks   []store.TrashedBookmark
	all         []store.Collection
	byID        map[string]*store.Collection
}

func (s *Server) loadTrash(c echo.Context, tx *store.Store, spaceID string) (*trashView, error) {
	// Only what is private or locked counts here, not the trash itself
	hidden, err := s.hiddenCollectionsOf(ctxOf(c), tx, spaceID, currentUser(c), false)
	if err != nil {
		return nil, err
	}
	collections, err := tx.CollectionsBySpace(ctxOf(c), spaceID)
	if err != nil {
		return nil, err
	}
	view := &trashView{all: collections, byID: map[string]*store.Collection{}}
	for i := range collections {
		collection := &collections[i]
		view.byID[collection.ID] = collection
		if collection.DeletedAt != nil && !hidden[collection.ID] {
			view.collections = append(view.collections, collection)
		}
	}
	sort.SliceStable(view.collections, func(i, j int) bool {
		return view.collections[i].DeletedAt.After(*view.collections[j].DeletedAt)
	})
	bookmarks, err := tx.TrashedBookmarks(ctxOf(c), spaceID)
	if err != nil {
		return nil, err
	}
	for _, bookmark := range bookmarks {
		if !hidden[bookmark.CollectionID] {
			view.bookmarks = append(view.bookmarks, bookmark)
		}
	}
	return view, nil
}

// descendants lists the collections below the given one.
func (v *trashView) descendants(id string) []string {
	result := []string{}
	for _, collection := range v.all {
		for parent := collection.ParentID; parent != nil; {
			if *parent == id {
				result = append(result, collection.ID)
				break
			}
			next := v.byID[*parent]
			if next == nil {
				break
			}
			parent = next.ParentID
		}
	}
	return result
}

func (s *Server) GetTrash(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	view, err := s.loadTrash(c, s.store, spaceID)
	if err != nil {
		return err
	}
	counts, err := s.store.BookmarkCounts(ctxOf(c), spaceID)
	if err != nil {
		return err
	}
	deleters := []string{}
	for _, collection := range view.collections {
		if collection.DeletedBy != nil {
			deleters = append(deleters, *collection.DeletedBy)
		}
	}
	names, err := s.store.DeleterNames(ctxOf(c), deleters)
	if err != nil {
		return err
	}
	result := api.TrashDto{Collections: []api.TrashedCollectionDto{}, Bookmarks: []api.TrashedBookmarkDto{},
		RetentionDays: int(trashRetention.Hours() / 24)}
	for _, collection := range view.collections {
		below := view.descendants(collection.ID)
		links := counts[collection.ID]
		for _, id := range below {
			links += counts[id]
		}
		dto := api.TrashedCollectionDto{Id: collection.ID, Name: collection.Name, BookmarkCount: links,
			ChildCount: len(below), DeletedAt: *collection.DeletedAt}
		if collection.ParentID != nil {
			if parent := view.byID[*collection.ParentID]; parent != nil {
				dto.ParentName = &parent.Name
			}
		}
		if collection.DeletedBy != nil {
			if name, ok := names[*collection.DeletedBy]; ok {
				dto.DeletedBy = &name
			}
		}
		result.Collections = append(result.Collections, dto)
	}
	for _, bookmark := range view.bookmarks {
		collectionName := bookmark.CollectionName
		result.Bookmarks = append(result.Bookmarks, api.TrashedBookmarkDto{Id: bookmark.ID, Url: bookmark.URL,
			Title: bookmark.Title, Favicon: bookmark.Favicon, CollectionId: bookmark.CollectionID,
			CollectionName: &collectionName, DeletedAt: bookmark.DeletedAt, DeletedBy: bookmark.DeletedByName})
	}
	return c.JSON(http.StatusOK, result)
}

// trashSelection resolves the requested ids against what the user may see in the trash.
func (s *Server) trashSelection(c echo.Context, view *trashView, req api.TrashSelectionDto, role Role) (
	collections []*store.Collection, bookmarks []string, err error) {
	inTrash := map[string]*store.Collection{}
	for _, collection := range view.collections {
		inTrash[collection.ID] = collection
	}
	trashedLinks := map[string]bool{}
	for _, bookmark := range view.bookmarks {
		trashedLinks[bookmark.ID] = true
	}
	var collectionIDs, bookmarkIDs []string
	if req.CollectionIds != nil {
		collectionIDs = *req.CollectionIds
	}
	if req.BookmarkIds != nil {
		bookmarkIDs = *req.BookmarkIds
	}
	if req.All != nil && *req.All {
		collectionIDs, bookmarkIDs = nil, nil
		for _, collection := range view.collections {
			collectionIDs = append(collectionIDs, collection.ID)
		}
		for _, bookmark := range view.bookmarks {
			bookmarkIDs = append(bookmarkIDs, bookmark.ID)
		}
	}
	if len(collectionIDs) > 0 && !role.Has(PermDeleteCollections) {
		return nil, nil, forbidden("Missing permission %s", PermDeleteCollections)
	}
	if len(bookmarkIDs) > 0 && !role.Has(PermDeleteBookmarks) {
		return nil, nil, forbidden("Missing permission %s", PermDeleteBookmarks)
	}
	for _, id := range collectionIDs {
		collection := inTrash[id]
		if collection == nil {
			return nil, nil, notFound("Not in the trash: %s", id)
		}
		collections = append(collections, collection)
	}
	for _, id := range bookmarkIDs {
		if !trashedLinks[id] {
			return nil, nil, notFound("Not in the trash: %s", id)
		}
		bookmarks = append(bookmarks, id)
	}
	return collections, bookmarks, nil
}

func (s *Server) RestoreFromTrash(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	var req api.TrashSelectionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	req.All = nil
	var restored int
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		view, err := s.loadTrash(c, tx, spaceID)
		if err != nil {
			return err
		}
		collections, bookmarks, err := s.trashSelection(c, view, req, role)
		if err != nil {
			return err
		}
		for _, collection := range collections {
			if err := tx.RestoreCollection(ctxOf(c), collection.ID); err != nil {
				return err
			}
			collection.DeletedAt = nil
		}
		// A collection whose parent is still in the trash would stay invisible: it comes back at the top level
		for _, collection := range collections {
			if !trashedAbove(view, collection) {
				continue
			}
			collection.ParentID = nil
			if err := tx.MoveCollection(ctxOf(c), collection.ID, nil); err != nil {
				return err
			}
			if err := s.placeCollection(c, tx, collection, nil); err != nil {
				return err
			}
		}
		for _, id := range bookmarks {
			if err := tx.RestoreBookmark(ctxOf(c), id); err != nil {
				return err
			}
		}
		restored = len(collections) + len(bookmarks)
		return nil
	})
	if err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "restored from the trash", "space", spaceID, "items", restored)
	return c.NoContent(http.StatusNoContent)
}

func trashedAbove(view *trashView, collection *store.Collection) bool {
	for parent, depth := collection.ParentID, 0; parent != nil && depth < 1000; depth++ {
		above := view.byID[*parent]
		if above == nil {
			return true
		}
		if above.DeletedAt != nil {
			return true
		}
		parent = above.ParentID
	}
	return false
}

func (s *Server) PurgeTrash(c echo.Context, spaceID api.ID) error {
	role, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	var req api.TrashSelectionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	var purged int
	err = s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		view, err := s.loadTrash(c, tx, spaceID)
		if err != nil {
			return err
		}
		collections, bookmarks, err := s.trashSelection(c, view, req, role)
		if err != nil {
			return err
		}
		hidden, err := s.hiddenCollectionsOf(ctxOf(c), tx, spaceID, currentUser(c), false)
		if err != nil {
			return err
		}
		for _, collection := range collections {
			// Never takes other members' private collections along
			for _, id := range view.descendants(collection.ID) {
				if hidden[id] {
					return conflict("“%s” contains collections you cannot see and cannot be deleted for good", collection.Name)
				}
			}
		}
		for _, id := range bookmarks {
			if err := tx.PurgeBookmark(ctxOf(c), id); err != nil {
				return err
			}
		}
		for _, collection := range collections {
			if err := tx.DeleteCollection(ctxOf(c), collection.ID); err != nil {
				return err
			}
		}
		purged = len(collections) + len(bookmarks)
		return nil
	})
	if err != nil {
		return err
	}
	audit(c, slog.LevelInfo, "deleted from the trash for good", "space", spaceID, "items", purged)
	return c.NoContent(http.StatusNoContent)
}

// PurgeOldTrash deletes what has been in the trash longer than trashRetention; main runs it every hour.
func (s *Server) PurgeOldTrash(ctx context.Context) {
	purged, err := s.store.PurgeTrash(ctx, time.Now().Add(-trashRetention))
	if err != nil {
		slog.Error("purging the trash failed", "error", err)
		return
	}
	if purged > 0 {
		slog.Info("purged the trash", "items", purged)
	}
	// Uploads of links deleted for good (or replaced) are no longer needed
	if orphans, err := s.store.DeleteOrphanImages(ctx, time.Now().Add(-time.Hour)); err != nil {
		slog.Error("deleting unused images failed", "error", err)
	} else if orphans > 0 {
		slog.Info("deleted unused images", "images", orphans)
	}
}
