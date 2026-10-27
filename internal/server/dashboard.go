package server

import (
	"net/http"
	"slices"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

const (
	maxSections        = 30
	maxLinksPerSection = 100
	recentCount        = 12
)

// GetDashboard returns the user's own layout, or favorites + recent until they customize it. Every member,
// viewers included, may arrange their own dashboard.
func (s *Server) GetDashboard(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	return s.dashboardView(c, spaceID)
}

func (s *Server) dashboardView(c echo.Context, spaceID string) error {
	userID := currentUser(c)
	sections, err := s.store.DashboardSections(ctxOf(c), userID, spaceID)
	if err != nil {
		return err
	}
	customized := len(sections) > 0
	if !customized {
		sections = []store.DashboardSection{
			{Title: "Favorites", Type: string(api.FAVORITES)},
			{Title: "Recently added", Type: string(api.RECENT)},
		}
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, userID)
	if err != nil {
		return err
	}

	result := api.DashboardDto{Customized: customized, Sections: []api.DashboardSectionDto{}}
	for _, section := range sections {
		var bookmarks []store.Bookmark
		switch api.DashboardSectionTypeEnum(section.Type) {
		case api.LINKS:
			pinned, err := s.store.BookmarksByIDs(ctxOf(c), section.BookmarkIDs)
			if err != nil {
				return err
			}
			// Pinned links stay pinned when their collection turns private, but only those who can see it get them
			byID := map[string]store.Bookmark{}
			for _, b := range pinned {
				if !hidden[b.CollectionID] {
					byID[b.ID] = b
				}
			}
			for _, id := range section.BookmarkIDs {
				if b, ok := byID[id]; ok {
					bookmarks = append(bookmarks, b)
				}
			}
		case api.FAVORITES:
			bookmarks, err = s.store.Favorites(ctxOf(c), userID, spaceID, keys(hidden), maxFavoritesListed)
		case api.RECENT:
			bookmarks, err = s.store.RecentBookmarks(ctxOf(c), spaceID, keys(hidden), recentCount)
		}
		if err != nil {
			return err
		}
		dtos, err := s.bookmarkDtos(c, bookmarks)
		if err != nil {
			return err
		}
		result.Sections = append(result.Sections, api.DashboardSectionDto{
			// Sections of the default layout have no id
			Id: nilIfEmpty(section.ID), Title: section.Title, Type: api.DashboardSectionTypeEnum(section.Type), Bookmarks: dtos,
		})
	}
	return c.JSON(http.StatusOK, result)
}

func (s *Server) UpdateDashboard(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	var req api.UpdateDashboardDto
	if err := bind(c, &req); err != nil {
		return err
	}
	if len(req.Sections) > maxSections {
		return badRequest("A dashboard can have at most %d sections", maxSections)
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}

	sections := make([]store.DashboardSection, 0, len(req.Sections))
	var referenced []string
	for _, section := range req.Sections {
		title := strings.TrimSpace(section.Title)
		if title == "" {
			return badRequest("Section title must not be empty")
		}
		if len([]rune(title)) > 80 {
			return badRequest("Section title must be at most 80 characters")
		}
		if !slices.Contains([]api.DashboardSectionTypeEnum{api.LINKS, api.FAVORITES, api.RECENT}, section.Type) {
			return badRequest("Section type is required")
		}
		var ids []string
		if section.Type == api.LINKS && section.BookmarkIds != nil {
			// Keep the first occurrence if a link was added twice
			for _, id := range *section.BookmarkIds {
				if !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
			}
			if len(ids) > maxLinksPerSection {
				return badRequest("A section can hold at most %d links", maxLinksPerSection)
			}
			for _, id := range ids {
				if !slices.Contains(referenced, id) {
					referenced = append(referenced, id)
				}
			}
		}
		sections = append(sections, store.DashboardSection{Title: title, Type: string(section.Type), BookmarkIDs: ids})
	}

	bookmarks, err := s.store.BookmarksByIDs(ctxOf(c), referenced)
	if err != nil {
		return err
	}
	valid := 0
	for _, b := range bookmarks {
		if b.SpaceID == spaceID && !hidden[b.CollectionID] {
			valid++
		}
	}
	if valid != len(referenced) {
		return badRequest("Dashboard links must be bookmarks of this space")
	}

	if err := s.store.InTx(ctxOf(c), func(tx *store.Store) error {
		return tx.ReplaceDashboard(ctxOf(c), currentUser(c), spaceID, sections)
	}); err != nil {
		return err
	}
	return s.dashboardView(c, spaceID)
}

func (s *Server) ResetDashboard(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	if err := s.store.DeleteDashboard(ctxOf(c), currentUser(c), spaceID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
