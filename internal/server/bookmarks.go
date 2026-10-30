package server

import (
	"net/http"
	"sort"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/fetch"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

const (
	maxTitleLength     = 160
	maxFavoritesListed = 200
	defaultRecentLimit = 20
)

func bookmarkDto(b *store.Bookmark, favorite bool) api.BookmarkDto {
	tags := b.Tags
	if tags == nil {
		tags = []string{}
	}
	collectionID, createdAt, createdBy := b.CollectionID, b.CreatedAt, b.CreatedByName
	return api.BookmarkDto{
		Id: b.ID, CollectionId: &collectionID, Url: b.URL, Title: b.Title, Description: b.Description, Tags: &tags,
		OgImage: b.OgImage, Favicon: b.Favicon, Canonical: b.Canonical, CreatedAt: &createdAt, CreatedBy: &createdBy,
		Favorite: &favorite,
	}
}

// bookmarkDtos maps a list for the current user, resolving all favorite flags with one query.
func (s *Server) bookmarkDtos(c echo.Context, bookmarks []store.Bookmark) ([]api.BookmarkDto, error) {
	ids := make([]string, len(bookmarks))
	for i, b := range bookmarks {
		ids[i] = b.ID
	}
	favorites, err := s.store.FavoriteIDs(ctxOf(c), currentUser(c), ids)
	if err != nil {
		return nil, err
	}
	reactions, err := s.reactionDtos(c, ids)
	if err != nil {
		return nil, err
	}
	result := make([]api.BookmarkDto, len(bookmarks))
	for i := range bookmarks {
		result[i] = bookmarkDto(&bookmarks[i], favorites[bookmarks[i].ID])
		list := reactions[bookmarks[i].ID]
		if list == nil {
			list = []api.ReactionDto{}
		}
		result[i].Reactions = &list
	}
	return result, nil
}

func (s *Server) respondBookmarks(c echo.Context, bookmarks []store.Bookmark, err error) error {
	if err != nil {
		return err
	}
	dtos, err := s.bookmarkDtos(c, bookmarks)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dtos)
}

func (s *Server) respondBookmark(c echo.Context, bookmarkID string) error {
	bookmark, err := s.store.BookmarkByID(ctxOf(c), bookmarkID)
	if err != nil {
		return err
	}
	dtos, err := s.bookmarkDtos(c, []store.Bookmark{*bookmark})
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, dtos[0])
}

// GetBookmarks lists a collection and flags the links that are new for the user. Listing does not mark them
// as seen; the frontend does that separately once it has shown them.
func (s *Server) GetBookmarks(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	bookmarks, err := s.store.BookmarksInCollection(ctxOf(c), collectionID)
	if err != nil {
		return err
	}
	news, err := s.store.NewBookmarks(ctxOf(c), currentUser(c), collection.SpaceID)
	if err != nil {
		return err
	}
	isNew := map[string]bool{}
	for _, item := range news {
		isNew[item.ID] = true
	}
	dtos, err := s.bookmarkDtos(c, bookmarks)
	if err != nil {
		return err
	}
	for i := range dtos {
		flag := isNew[dtos[i].Id]
		dtos[i].IsNew = &flag
	}
	return c.JSON(http.StatusOK, dtos)
}

func (s *Server) GetBookmark(c echo.Context, bookmarkID api.ID) error {
	if _, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermRead); err != nil {
		return err
	}
	return s.respondBookmark(c, bookmarkID)
}

func (s *Server) GetRecentBookmarks(c echo.Context, spaceID api.ID, params api.GetRecentBookmarksParams) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	limit := defaultRecentLimit
	if params.Limit != nil {
		limit = min(max(*params.Limit, 1), 100)
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	bookmarks, err := s.store.RecentBookmarks(ctxOf(c), spaceID, keys(hidden), limit)
	return s.respondBookmarks(c, bookmarks, err)
}

func (s *Server) SearchBookmarks(c echo.Context, spaceID api.ID, params api.SearchBookmarksParams) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	query := strings.TrimSpace(params.Q)
	if query == "" {
		return c.JSON(http.StatusOK, []api.BookmarkDto{})
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	bookmarks, err := s.store.SearchBookmarks(ctxOf(c), spaceID, query, keys(hidden))
	return s.respondBookmarks(c, bookmarks, err)
}

// GetSpaceTags lists the tags for suggestions, case variants merged under their most used spelling.
func (s *Server) GetSpaceTags(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	counts, err := s.store.TagCounts(ctxOf(c), spaceID, keys(hidden))
	if err != nil {
		return err
	}
	type merged struct {
		spelling string
		best     int
		total    int
	}
	byKey := map[string]*merged{}
	for _, count := range counts {
		key := strings.ToLower(count.Tag)
		entry := byKey[key]
		if entry == nil {
			entry = &merged{}
			byKey[key] = entry
		}
		entry.total += count.Count
		if count.Count > entry.best || (count.Count == entry.best && count.Tag < entry.spelling) {
			entry.spelling, entry.best = count.Tag, count.Count
		}
	}
	result := make([]api.TagCountDto, 0, len(byKey))
	for _, entry := range byKey {
		result = append(result, api.TagCountDto{Tag: entry.spelling, Count: entry.total})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return strings.ToLower(result[i].Tag) < strings.ToLower(result[j].Tag)
	})
	return c.JSON(http.StatusOK, result)
}

// LookupBookmarks finds links with the same URL, so adding one twice can be pointed out.
func (s *Server) LookupBookmarks(c echo.Context, spaceID api.ID, params api.LookupBookmarksParams) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	url, err := fetch.NormalizeURL(params.Url)
	if err != nil {
		// Not a URL (yet): nothing can match
		return c.JSON(http.StatusOK, []api.BookmarkDto{})
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	bookmarks, err := s.store.BookmarksByURL(ctxOf(c), spaceID, url, keys(hidden))
	return s.respondBookmarks(c, bookmarks, err)
}

func (s *Server) CreateBookmark(c echo.Context, collectionID api.ID) error {
	if _, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermCreateBookmarks); err != nil {
		return err
	}
	var req api.CreateBookmarkDto
	if err := bind(c, &req); err != nil {
		return err
	}
	url, err := fetch.NormalizeURL(req.Url)
	if err != nil {
		return badRequest("%s", err.Error())
	}
	metadata := s.fetch.Metadata(ctxOf(c), url)
	bookmark := &store.Bookmark{
		CollectionID: collectionID,
		URL:          url,
		Title:        ptr(truncate(firstNonEmpty(deref(req.Title), metadata.Title, url), maxTitleLength)),
		Description:  nilIfEmpty(firstNonEmpty(deref(req.Description), metadata.Description)),
		OgImage:      nilIfEmpty(metadata.OgImage),
		Favicon:      nilIfEmpty(metadata.Favicon),
		Canonical:    nilIfEmpty(metadata.Canonical),
		CreatedBy:    currentUser(c),
		Tags:         normalizeTags(req.Tags),
	}
	if err := s.store.CreateBookmark(ctxOf(c), bookmark); err != nil {
		return err
	}
	return s.respondBookmark(c, bookmark.ID)
}

func (s *Server) UpdateBookmark(c echo.Context, bookmarkID api.ID) error {
	bookmark, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermEditBookmarks)
	if err != nil {
		return err
	}
	var req api.UpdateBookmarkDto
	if err := bind(c, &req); err != nil {
		return err
	}
	url, err := fetch.NormalizeURL(req.Url)
	if err != nil {
		return badRequest("%s", err.Error())
	}
	if req.CollectionId != nil && *req.CollectionId != bookmark.CollectionID {
		// Moving adds the link to the target, so it needs the same permission as creating it there
		target, _, err := s.requireCollection(ctxOf(c), s.store, *req.CollectionId, currentUser(c), PermCreateBookmarks)
		if err != nil {
			return err
		}
		if target.SpaceID != bookmark.SpaceID {
			return badRequest("Links can only be moved within their space")
		}
		bookmark.CollectionID = target.ID
	}
	urlChanged := url != bookmark.URL
	bookmark.URL = url
	bookmark.Title = ptr(truncate(firstNonEmpty(deref(req.Title), deref(bookmark.Title)), maxTitleLength))
	bookmark.Description = nilIfEmpty(deref(req.Description))
	if req.Tags != nil {
		bookmark.Tags = normalizeTags(req.Tags)
	}
	if urlChanged {
		metadata := s.fetch.Metadata(ctxOf(c), url)
		bookmark.OgImage, bookmark.Favicon, bookmark.Canonical =
			nilIfEmpty(metadata.OgImage), nilIfEmpty(metadata.Favicon), nilIfEmpty(metadata.Canonical)
		bookmark.Title = ptr(truncate(firstNonEmpty(deref(bookmark.Title), metadata.Title, url), maxTitleLength))
		bookmark.Description = nilIfEmpty(firstNonEmpty(deref(bookmark.Description), metadata.Description))
	}
	if err := s.store.UpdateBookmark(ctxOf(c), bookmark); err != nil {
		return err
	}
	return s.respondBookmark(c, bookmarkID)
}

func (s *Server) DeleteBookmark(c echo.Context, bookmarkID api.ID) error {
	if _, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermDeleteBookmarks); err != nil {
		return err
	}
	// Into the trash; restoring brings back favorites, reactions and pins as well
	if err := s.store.DeleteBookmark(ctxOf(c), bookmarkID, currentUser(c)); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// FetchBookmarkMetadata loads title & co. for the add/edit dialogs without saving anything.
func (s *Server) FetchBookmarkMetadata(c echo.Context, collectionID api.ID) error {
	_, role, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	if !role.Has(PermCreateBookmarks) && !role.Has(PermEditBookmarks) {
		return forbidden("Missing permission to add or edit links")
	}
	var req api.FetchMetadataRequest
	if err := bind(c, &req); err != nil {
		return err
	}
	url, err := fetch.NormalizeURL(req.Url)
	if err != nil {
		return badRequest("%s", err.Error())
	}
	metadata := s.fetch.Metadata(ctxOf(c), url)
	keywords := metadata.Keywords
	if keywords == nil {
		keywords = []string{}
	}
	return c.JSON(http.StatusOK, api.BookmarkMetadataDto{
		Url: url, Found: metadata.Found(), Title: nilIfEmpty(metadata.Title), Description: nilIfEmpty(metadata.Description),
		OgImage: nilIfEmpty(metadata.OgImage), Favicon: nilIfEmpty(metadata.Favicon), Canonical: nilIfEmpty(metadata.Canonical),
		Keywords: &keywords,
	})
}

// --- Favorites (personal) ---

func (s *Server) GetFavorites(c echo.Context, spaceID api.ID) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	bookmarks, err := s.store.Favorites(ctxOf(c), currentUser(c), spaceID, keys(hidden), maxFavoritesListed)
	return s.respondBookmarks(c, bookmarks, err)
}

// AddFavorite: anyone who can see a bookmark may star it, viewers included.
func (s *Server) AddFavorite(c echo.Context, bookmarkID api.ID) error {
	if _, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermRead); err != nil {
		return err
	}
	if err := s.store.AddFavorite(ctxOf(c), currentUser(c), bookmarkID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) RemoveFavorite(c echo.Context, bookmarkID api.ID) error {
	if _, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermRead); err != nil {
		return err
	}
	if err := s.store.RemoveFavorite(ctxOf(c), currentUser(c), bookmarkID); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// --- helpers ---

func normalizeTags(tags *[]string) []string {
	result := []string{}
	if tags == nil {
		return result
	}
	seen := map[string]bool{}
	for _, tag := range *tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return result
}

func truncate(value string, maxLength int) string {
	runes := []rune(value)
	if len(runes) <= maxLength {
		return value
	}
	return string(runes[:maxLength-1]) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ptr[T any](value T) *T { return &value }

func deref(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nilIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
