package server

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/fetch"
)

func TestFeedCacheLoadsEachFeedOnce(t *testing.T) {
	cache := newFeedCache()
	var loads atomic.Int32
	release := make(chan struct{})
	load := func() (*fetch.Feed, error) {
		loads.Add(1)
		<-release
		return &fetch.Feed{Title: "News"}, nil
	}
	always := func() bool { return true }

	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if feed, err := cache.get("https://a.example/feed", always, load); err != nil || feed.Title != "News" {
				t.Errorf("unexpected result: %v %v", feed, err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Errorf("loaded %d times for concurrent requests", loads.Load())
	}

	// Cached feeds do not count against the limit
	never := func() bool { return false }
	if _, err := cache.get("https://a.example/feed", never, load); err != nil {
		t.Errorf("a cached feed is served without loading: %v", err)
	}
	// A refused load is not cached: the next one with tokens left loads it
	if _, err := cache.get("https://b.example/feed", never, load); !errors.Is(err, errFeedLimited) {
		t.Errorf("expected the limit: %v", err)
	}
	if _, err := cache.get("https://b.example/feed", always, load); err != nil || loads.Load() != 2 {
		t.Errorf("expected a load: %v (%d loads)", err, loads.Load())
	}

	// Failures are kept for a while, then tried again
	failing := func() (*fetch.Feed, error) { loads.Add(1); return nil, errors.New("down") }
	cache.get("https://c.example/feed", always, failing)
	cache.get("https://c.example/feed", always, failing)
	if loads.Load() != 3 {
		t.Errorf("a failure is cached too (%d loads)", loads.Load())
	}
	cache.entries["https://c.example/feed"].loaded = time.Now().Add(-feedFailureTTL)
	cache.get("https://c.example/feed", always, failing)
	if loads.Load() != 4 {
		t.Errorf("an old failure is tried again (%d loads)", loads.Load())
	}
}

func TestFeedFilter(t *testing.T) {
	item := fetch.FeedItem{Title: "Darkmoon Faire comes to Elwynn Forest", Categories: []string{"WoW: Forever"}}
	for filter, expected := range map[string]bool{
		"":                true,
		"forever":         true,
		"raid, darkmoon ": true,
		"retail":          false,
		" , ":             true,
	} {
		if got := matchesFeedFilter(item, feedFilterWords(filter)); got != expected {
			t.Errorf("filter %q: got %v", filter, got)
		}
	}
}

func TestMergeHeadlines(t *testing.T) {
	at := func(hours int) *time.Time { value := time.Date(2026, 11, 9, hours, 0, 0, 0, time.UTC); return &value }
	wowhead := api.HeadlinesDto{Items: []api.HeadlineDto{
		{Title: "Darkmoon Faire", Url: "https://a.example/faire", PublishedAt: at(8)},
		{Title: "Raids", Url: "https://a.example/raids", PublishedAt: at(6)},
	}}
	buffed := api.HeadlinesDto{Items: []api.HeadlineDto{
		{Title: "Undated", Url: "https://b.example/undated"},
		{Title: "Tauren nerf", Url: "https://b.example/tauren", PublishedAt: at(10)},
		{Title: "Same article", Url: "https://a.example/raids", PublishedAt: at(5)},
	}}
	merged := mergeHeadlines([]api.HeadlinesDto{wowhead, buffed, {Failed: true}}, 10)
	var titles []string
	for _, item := range merged.Items {
		titles = append(titles, item.Title)
	}
	if strings.Join(titles, "|") != "Tauren nerf|Darkmoon Faire|Raids|Undated" || merged.Failed {
		t.Errorf("newest first, undated last, each article once: %v (failed %v)", titles, merged.Failed)
	}
	if len(mergeHeadlines([]api.HeadlinesDto{wowhead, buffed}, 2).Items) != 2 {
		t.Error("limited")
	}
	if !mergeHeadlines([]api.HeadlinesDto{{Failed: true}, {Failed: true}}, 5).Failed {
		t.Error("failed when every feed failed")
	}
}
