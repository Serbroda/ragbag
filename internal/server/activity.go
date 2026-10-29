package server

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// --- New links ---

// visibleNews lists the links that are new for the user, in one space or in all of them (spaceID ""), without
// those in collections hidden from the user.
func (s *Server) visibleNews(ctx context.Context, userID, spaceID string) ([]store.NewBookmark, error) {
	news, err := s.store.NewBookmarks(ctx, userID, spaceID)
	if err != nil || len(news) == 0 {
		return news, err
	}
	hiddenBySpace := map[string]map[string]bool{}
	visible := news[:0]
	for _, item := range news {
		hidden, ok := hiddenBySpace[item.SpaceID]
		if !ok {
			if hidden, err = s.hiddenCollections(ctx, s.store, item.SpaceID, userID); err != nil {
				return nil, err
			}
			hiddenBySpace[item.SpaceID] = hidden
		}
		if !hidden[item.CollectionID] {
			visible = append(visible, item)
		}
	}
	return visible, nil
}

// newCounts counts the visible new links per key, e.g. per space or per collection.
func newCounts(news []store.NewBookmark, key func(store.NewBookmark) string) map[string]int {
	counts := map[string]int{}
	for _, item := range news {
		counts[key(item)]++
	}
	return counts
}

func (s *Server) MarkCollectionSeen(c echo.Context, collectionID api.ID) error {
	if _, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead); err != nil {
		return err
	}
	if err := s.store.MarkSeen(ctxOf(c), currentUser(c), []string{collectionID}); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// MarkSpaceSeen marks the collections with new links as seen; the others have nothing to catch up on.
func (s *Server) MarkSpaceSeen(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	news, err := s.visibleNews(ctxOf(c), currentUser(c), spaceID)
	if err != nil {
		return err
	}
	ids := sortedKeys(newCounts(news, func(n store.NewBookmark) string { return n.CollectionID }))
	if err := s.store.MarkSeen(ctxOf(c), currentUser(c), ids); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetSpaceNews summarizes the new links per collection: how many, by whom, the most recent first.
func (s *Server) GetSpaceNews(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	news, err := s.visibleNews(ctxOf(c), currentUser(c), spaceID)
	if err != nil {
		return err
	}
	// news is ordered newest first, so the first link of a collection is its latest
	byCollection := map[string]*api.CollectionNewsDto{}
	result := []*api.CollectionNewsDto{}
	for _, item := range news {
		entry := byCollection[item.CollectionID]
		if entry == nil {
			entry = &api.CollectionNewsDto{CollectionId: item.CollectionID, AddedBy: []string{}, LatestAt: item.CreatedAt}
			byCollection[item.CollectionID] = entry
			result = append(result, entry)
		}
		entry.Count++
		if !slices.Contains(entry.AddedBy, item.CreatedByName) {
			entry.AddedBy = append(entry.AddedBy, item.CreatedByName)
		}
	}
	return c.JSON(http.StatusOK, result)
}

// --- Reactions ---

// reactionEmojis is the fixed set to react with (no free choice, so there is nothing to moderate). The
// frontend shows the same list in this order.
var reactionEmojis = []string{"👍", "❤️", "😂", "🔥", "😮", "🙏"}

func (s *Server) SetReaction(c echo.Context, bookmarkID api.ID) error {
	if _, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermRead); err != nil {
		return err
	}
	var req api.SetReactionDto
	if err := bind(c, &req); err != nil {
		return err
	}
	if !slices.Contains(reactionEmojis, req.Emoji) {
		return badRequest("Unknown reaction, use one of %s", strings.Join(reactionEmojis, " "))
	}
	var err error
	if req.Reacted {
		err = s.store.AddReaction(ctxOf(c), bookmarkID, currentUser(c), req.Emoji)
	} else {
		err = s.store.RemoveReaction(ctxOf(c), bookmarkID, currentUser(c), req.Emoji)
	}
	if err != nil {
		return err
	}
	reactions, err := s.reactionDtos(c, []string{bookmarkID})
	if err != nil {
		return err
	}
	result := reactions[bookmarkID]
	if result == nil {
		result = []api.ReactionDto{}
	}
	return c.JSON(http.StatusOK, result)
}

// reactionDtos groups the reactions per bookmark and emoji, in the order the emojis were first used.
func (s *Server) reactionDtos(c echo.Context, bookmarkIDs []string) (map[string][]api.ReactionDto, error) {
	reactions, err := s.store.Reactions(ctxOf(c), bookmarkIDs)
	if err != nil {
		return nil, err
	}
	result := map[string][]api.ReactionDto{}
	for _, reaction := range reactions {
		list := result[reaction.BookmarkID]
		i := slices.IndexFunc(list, func(r api.ReactionDto) bool { return r.Emoji == reaction.Emoji })
		if i < 0 {
			list = append(list, api.ReactionDto{Emoji: reaction.Emoji, Users: []string{}})
			i = len(list) - 1
		}
		list[i].Count++
		list[i].Users = append(list[i].Users, reaction.Username)
		list[i].Reacted = list[i].Reacted || reaction.UserID == currentUser(c)
		result[reaction.BookmarkID] = list
	}
	return result, nil
}

// --- Space bar ---

const (
	maxSpaceBarItems   = 500
	maxFolderNameRunes = 40
)

func (s *Server) GetSpaceBar(c echo.Context) error {
	layout, err := s.store.SpaceBar(ctxOf(c), currentUser(c))
	if err != nil {
		return err
	}
	bar := api.SpaceBarDto{Items: []api.SpaceBarItemDto{}}
	if layout != "" {
		// A layout that no longer parses is simply forgotten
		_ = json.Unmarshal([]byte(layout), &bar)
	}
	cleaned, err := s.cleanSpaceBar(c, bar)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cleaned)
}

func (s *Server) UpdateSpaceBar(c echo.Context) error {
	var req api.SpaceBarDto
	if err := bind(c, &req); err != nil {
		return err
	}
	if len(req.Items) > maxSpaceBarItems {
		return badRequest("The space bar can hold at most %d items", maxSpaceBarItems)
	}
	for _, item := range req.Items {
		isSpace, isFolder := item.SpaceId != nil, item.FolderId != nil
		if isSpace == isFolder {
			return badRequest("Each item is either a space (spaceId) or a folder (folderId, name, spaceIds)")
		}
		if isFolder {
			name := strings.TrimSpace(deref(item.Name))
			if name == "" || len([]rune(name)) > maxFolderNameRunes || len(*item.FolderId) > 64 {
				return badRequest("Folder names have 1 to %d characters", maxFolderNameRunes)
			}
		}
	}
	cleaned, err := s.cleanSpaceBar(c, req)
	if err != nil {
		return err
	}
	layout, err := json.Marshal(cleaned)
	if err != nil {
		return err
	}
	if err := s.store.SetSpaceBar(ctxOf(c), currentUser(c), string(layout)); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, cleaned)
}

// cleanSpaceBar keeps only spaces the user is a member of, each once, and drops folders left empty.
func (s *Server) cleanSpaceBar(c echo.Context, bar api.SpaceBarDto) (api.SpaceBarDto, error) {
	spaces, err := s.store.SpacesForUser(ctxOf(c), currentUser(c))
	if err != nil {
		return bar, err
	}
	member := map[string]bool{}
	for _, space := range spaces {
		member[space.ID] = true
	}
	placed := map[string]bool{}
	keep := func(id string) bool {
		if !member[id] || placed[id] {
			return false
		}
		placed[id] = true
		return true
	}
	cleaned := api.SpaceBarDto{Items: []api.SpaceBarItemDto{}}
	for _, item := range bar.Items {
		switch {
		case item.SpaceId != nil:
			if keep(*item.SpaceId) {
				cleaned.Items = append(cleaned.Items, api.SpaceBarItemDto{SpaceId: item.SpaceId})
			}
		case item.FolderId != nil && item.SpaceIds != nil:
			ids := []api.ID{}
			for _, id := range *item.SpaceIds {
				if keep(id) {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				name := strings.TrimSpace(deref(item.Name))
				cleaned.Items = append(cleaned.Items, api.SpaceBarItemDto{FolderId: item.FolderId, Name: &name, SpaceIds: &ids})
			}
		}
	}
	return cleaned, nil
}

// sortedKeys keeps the order of writes stable.
func sortedKeys(set map[string]int) []string {
	result := make([]string, 0, len(set))
	for key := range set {
		result = append(result, key)
	}
	sort.Strings(result)
	return result
}
