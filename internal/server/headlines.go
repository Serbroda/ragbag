package server

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/fetch"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Headlines: a link may name a news feed whose latest headlines its card shows. The server loads a feed only
// when somebody looks at the link and keeps it for an hour, shared by everybody (the filter applies afterwards),
// so a collection full of news sites does not turn into a crawler. Nothing of a feed is stored in the database.

const (
	headlinesShown  = 5
	defaultMerged   = 15
	maxMergedFeeds  = 50
	parallelFeeds   = 8
	maxFeedFilter   = 100
	feedTTL         = time.Hour
	feedFailureTTL  = 10 * time.Minute
	maxCachedFeeds  = 2000
	feedLoadTimeout = 30 * time.Second
)

type feedEntry struct {
	feed   *fetch.Feed
	err    error
	loaded time.Time
	// done is closed once loaded; requests for a feed being loaded wait for it instead of loading it again
	done chan struct{}
}

func (e *feedEntry) fresh(at time.Time) bool {
	ttl := feedTTL
	if e.err != nil {
		ttl = feedFailureTTL
	}
	return at.Sub(e.loaded) < ttl
}

type feedCache struct {
	mu      sync.Mutex
	entries map[string]*feedEntry
}

func newFeedCache() *feedCache { return &feedCache{entries: map[string]*feedEntry{}} }

// get returns the cached feed or loads it, if allowed: loading counts against the user's (or visitor's) limit
// for user supplied URLs, cached feeds do not.
func (fc *feedCache) get(feedURL string, allowed func() bool, load func() (*fetch.Feed, error)) (*fetch.Feed, error) {
	fc.mu.Lock()
	if entry := fc.entries[feedURL]; entry != nil {
		select {
		case <-entry.done:
			if entry.fresh(time.Now()) {
				fc.mu.Unlock()
				return entry.feed, entry.err
			}
		default:
			// Being loaded by another request
			fc.mu.Unlock()
			<-entry.done
			return entry.feed, entry.err
		}
	}
	if !allowed() {
		fc.mu.Unlock()
		return nil, errFeedLimited
	}
	entry := &feedEntry{done: make(chan struct{})}
	fc.entries[feedURL] = entry
	fc.prune()
	fc.mu.Unlock()

	entry.feed, entry.err = load()
	entry.loaded = time.Now()
	close(entry.done)
	return entry.feed, entry.err
}

// put keeps a feed that was just loaded anyway (when saving a link).
func (fc *feedCache) put(feedURL string, feed *fetch.Feed) {
	entry := &feedEntry{feed: feed, loaded: time.Now(), done: make(chan struct{})}
	close(entry.done)
	fc.mu.Lock()
	fc.entries[feedURL] = entry
	fc.prune()
	fc.mu.Unlock()
}

// prune drops expired feeds once there are too many, then any loaded ones; called with the lock held.
func (fc *feedCache) prune() {
	if len(fc.entries) <= maxCachedFeeds {
		return
	}
	at := time.Now()
	for key, entry := range fc.entries {
		select {
		case <-entry.done:
			if !entry.fresh(at) || len(fc.entries) > maxCachedFeeds {
				delete(fc.entries, key)
			}
		default:
		}
	}
}

var errFeedLimited = errors.New("too many feeds loaded")

func (s *Server) GetBookmarkHeadlines(c echo.Context, bookmarkID api.ID) error {
	bookmark, err := s.requireBookmark(ctxOf(c), s.store, bookmarkID, currentUser(c), PermRead)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, s.headlines(ctxOf(c), bookmark, s.userMayFetch(currentUser(c)), headlinesShown))
}

// GetSpaceHeadlines merges the headlines of several links (a dashboard section), newest first. Links that are
// not of the space, hidden from the user or without a feed are left out.
func (s *Server) GetSpaceHeadlines(c echo.Context, spaceID api.ID, params api.GetSpaceHeadlinesParams) error {
	if _, err := s.requireSpace(ctxOf(c), s.store, spaceID, currentUser(c), PermRead); err != nil {
		return err
	}
	if len(params.BookmarkId) > maxMergedFeeds {
		return badRequest("At most %d links at once", maxMergedFeeds)
	}
	limit := defaultMerged
	if params.Limit != nil {
		limit = min(max(*params.Limit, 1), maxMergedFeeds)
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, spaceID, currentUser(c))
	if err != nil {
		return err
	}
	bookmarks, err := s.store.BookmarksByIDs(ctxOf(c), params.BookmarkId)
	if err != nil {
		return err
	}
	var sources []store.Bookmark
	for _, b := range bookmarks {
		if b.SpaceID == spaceID && !hidden[b.CollectionID] && b.FeedURL != nil {
			sources = append(sources, b)
		}
	}

	results := make([]api.HeadlinesDto, len(sources))
	allowed := s.userMayFetch(currentUser(c))
	var wg sync.WaitGroup
	slots := make(chan struct{}, parallelFeeds)
	for i := range sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i] = s.headlines(ctxOf(c), &sources[i], allowed, maxMergedFeeds)
			for j := range results[i].Items {
				results[i].Items[j].BookmarkId = &sources[i].ID
			}
		}()
	}
	wg.Wait()
	return c.JSON(http.StatusOK, mergeHeadlines(results, limit))
}

// mergeHeadlines puts the newest first (undated ones last) and shows an article reported by two feeds once.
func mergeHeadlines(results []api.HeadlinesDto, limit int) api.HeadlinesDto {
	merged := api.HeadlinesDto{Items: []api.HeadlineDto{}, Failed: len(results) > 0}
	var all []api.HeadlineDto
	for _, result := range results {
		merged.Failed = merged.Failed && result.Failed
		all = append(all, result.Items...)
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i].PublishedAt, all[j].PublishedAt
		if a == nil || b == nil {
			return a != nil
		}
		return a.After(*b)
	})
	seen := map[string]bool{}
	for _, item := range all {
		if seen[item.Url] {
			continue
		}
		seen[item.Url] = true
		merged.Items = append(merged.Items, item)
		if len(merged.Items) == limit {
			break
		}
	}
	return merged
}

// userMayFetch counts loading a feed against the user's limit for user supplied URLs.
func (s *Server) userMayFetch(userID string) func() bool {
	return func() bool { return s.limits == nil || s.limits.fetch.take(userID) == 0 }
}

// GetSharedHeadlines answers for links of a public link only; like there, every failure is the same 404.
func (s *Server) GetSharedHeadlines(c echo.Context, token string, bookmarkID api.ID) error {
	bookmark, err := s.store.BookmarkByID(ctxOf(c), bookmarkID)
	if err != nil {
		return orNotFound(err, shareNotFound)
	}
	if _, err := s.sharedCollection(c, token, &bookmark.CollectionID); err != nil {
		return err
	}
	c.Response().Header().Set("X-Robots-Tag", "noindex")
	// Visitors have no account: their loading counts per IP address
	allowed := func() bool { return s.limits == nil || s.limits.fetch.take("ip:"+clientKey(c)) == 0 }
	return c.JSON(http.StatusOK, s.headlines(ctxOf(c), bookmark, allowed, headlinesShown))
}

// headlines are the newest of the link's feed that pass its filter.
func (s *Server) headlines(ctx context.Context, bookmark *store.Bookmark, allowed func() bool, limit int) api.HeadlinesDto {
	dto := api.HeadlinesDto{Items: []api.HeadlineDto{}}
	if bookmark.FeedURL == nil {
		return dto
	}
	feedURL := *bookmark.FeedURL
	feed, err := s.feeds.get(feedURL, allowed, func() (*fetch.Feed, error) {
		// Not bound to this request: others may be waiting for the same feed
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), feedLoadTimeout)
		defer cancel()
		return s.fetch.Feed(ctx, feedURL)
	})
	if err != nil {
		dto.Failed = true
		return dto
	}
	dto.FeedTitle = nilIfEmpty(feed.Title)
	words := feedFilterWords(deref(bookmark.FeedFilter))
	for _, item := range feed.Items {
		if !matchesFeedFilter(item, words) {
			continue
		}
		dto.Items = append(dto.Items, api.HeadlineDto{Title: item.Title, Url: item.URL, PublishedAt: item.Published})
		if len(dto.Items) == limit {
			break
		}
	}
	return dto
}

// feedFilterWords splits the filter at commas; any of the words has to be in a headline or its categories.
func feedFilterWords(filter string) []string {
	var words []string
	for _, part := range strings.Split(filter, ",") {
		if word := strings.TrimSpace(part); word != "" {
			words = append(words, word)
		}
	}
	return words
}

func matchesFeedFilter(item fetch.FeedItem, words []string) bool {
	if len(words) == 0 {
		return true
	}
	text := strings.ToLower(item.Title + "\n" + strings.Join(item.Categories, "\n"))
	for _, word := range words {
		if strings.Contains(text, strings.ToLower(word)) {
			return true
		}
	}
	return false
}

// resolveFeed checks a feed address given when saving a link: a feed, or a page linking to one.
func (s *Server) resolveFeed(c echo.Context, address string) (string, error) {
	normalized, err := fetch.NormalizeURL(address)
	if err != nil {
		return "", badRequest("The feed address is not valid: %s", err.Error())
	}
	if err := s.limitFetch(c); err != nil {
		return "", err
	}
	feedURL, feed, err := s.fetch.FindFeed(ctxOf(c), normalized)
	if err != nil {
		return "", badRequest("No news feed (RSS or Atom) found at %s", normalized)
	}
	s.feeds.put(feedURL, feed)
	return feedURL, nil
}

func normalizeFeedFilter(filter *string) (*string, error) {
	words := feedFilterWords(deref(filter))
	if len(words) == 0 {
		return nil, nil
	}
	value := strings.Join(words, ", ")
	if len([]rune(value)) > maxFeedFilter {
		return nil, badRequest("The headline filter is too long (at most %d characters)", maxFeedFilter)
	}
	return &value, nil
}
