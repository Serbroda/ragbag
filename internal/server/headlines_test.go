package server_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// setFeed gives the link a feed directly in the database: saving one through the API loads it, and tests never
// touch the network (a loopback feed is refused like any other loopback URL).
func (f *shareFixture) setFeed(collectionID string) string {
	f.t.Helper()
	id := f.req("GET", "/api/v1/collections/%s/bookmarks", collectionID).as(f.owner).expect(200).str("0.id")
	_, err := f.db.Exec(f.db.Rebind(`UPDATE bookmarks SET feed_url = ?, feed_filter = ? WHERE id = ?`),
		"http://127.0.0.1/feed", "Forever", id)
	require.NoError(f.t, err)
	return id
}

func TestHeadlinesNeedReadAccess(t *testing.T) {
	f := newShareFixture(t)
	withFeed := f.setFeed(f.shared)
	without := f.req("GET", "/api/v1/collections/%s/bookmarks", f.sub).as(f.owner).expect(200).str("0.id")

	f.req("GET", "/api/v1/bookmarks/%s", withFeed).as(f.viewer).expect(200).
		has("feedUrl", "http://127.0.0.1/feed").has("feedFilter", "Forever")
	// The feed cannot be loaded: no headlines, and the card says so instead of failing
	f.req("GET", "/api/v1/bookmarks/%s/headlines", withFeed).as(f.viewer).expect(200).
		has("failed", true).has("items.#", 0)
	f.req("GET", "/api/v1/bookmarks/%s/headlines", without).as(f.viewer).expect(200).
		has("failed", false).has("items.#", 0)

	outsider := f.register()
	f.req("GET", "/api/v1/bookmarks/%s/headlines", withFeed).as(outsider).expect(403)
	f.req("GET", "/api/v1/bookmarks/%s/headlines", withFeed).expect(401)
}

func TestSharedHeadlinesOnlyForLinksOfTheShare(t *testing.T) {
	f := newShareFixture(t)
	shared, deep := f.setFeed(f.shared), f.setFeed(f.deep)
	secret, other := f.setFeed(f.secret), f.setFeed(f.other)
	token := f.share()

	res := f.open(token, "").expect(200).has("bookmarks.0.headlines", true)
	require.NotContains(t, res.Body, "127.0.0.1/feed", "visitors do not need the feed's address")
	f.open(token, f.sub).expect(200).has("bookmarks.0.headlines", false)

	for _, id := range []string{shared, deep} {
		f.req("GET", "/api/v1/shares/%s/bookmarks/%s/headlines", token, id).expect(200).has("failed", true)
	}
	for _, id := range []string{secret, other, "00000000-0000-0000-0000-000000000000"} {
		f.req("GET", "/api/v1/shares/%s/bookmarks/%s/headlines", token, id).expect(404)
	}
	f.req("GET", "/api/v1/shares/%s/bookmarks/%s/headlines", "not-a-token", shared).expect(404)

	// Trashed links are gone from the share, their headlines too
	f.req("DELETE", "/api/v1/bookmarks/%s", deep).as(f.owner).expect(204)
	f.req("GET", "/api/v1/shares/%s/bookmarks/%s/headlines", token, deep).expect(404)
}

func TestSavingAFeed(t *testing.T) {
	f := newShareFixture(t)
	id := f.setFeed(f.shared)

	// A feed that cannot be found is refused right away rather than showing nothing later
	f.req("POST", "/api/v1/collections/%s/bookmarks", f.shared).as(f.owner).
		json(`{"url":"http://127.0.0.1/news","feedUrl":"http://127.0.0.1/news"}`).expect(400)
	f.req("PUT", "/api/v1/bookmarks/%s", id).as(f.owner).
		json(`{"url":"http://127.0.0.1/page","feedUrl":"http://127.0.0.1/other"}`).expect(400)

	// The stored address is kept without loading it again; the filter is tidied up
	f.req("PUT", "/api/v1/bookmarks/%s", id).as(f.owner).
		json(`{"url":"http://127.0.0.1/page","feedUrl":"http://127.0.0.1/feed","feedFilter":" Forever , ,Raid "}`).
		expect(200).has("feedUrl", "http://127.0.0.1/feed").has("feedFilter", "Forever, Raid")
	// Left out, both stay (e.g. moving or tagging several links)
	f.req("PUT", "/api/v1/bookmarks/%s", id).as(f.owner).json(`{"url":"http://127.0.0.1/page","tags":["b"]}`).
		expect(200).has("feedUrl", "http://127.0.0.1/feed").has("feedFilter", "Forever, Raid")
	f.req("PUT", "/api/v1/bookmarks/%s", id).as(f.owner).
		json(`{"url":"http://127.0.0.1/page","feedUrl":"","feedFilter":""}`).expect(200)
	res := f.req("GET", "/api/v1/bookmarks/%s", id).as(f.owner).expect(200)
	require.False(t, res.get("feedUrl").Exists())
	require.False(t, res.get("feedFilter").Exists())

	// Viewers cannot change it
	f.req("PUT", "/api/v1/bookmarks/%s", id).as(f.viewer).
		json(`{"url":"http://127.0.0.1/page","feedUrl":"http://127.0.0.1/feed"}`).expect(403)
}

func TestFeedsSurviveExportAndImport(t *testing.T) {
	f := newShareFixture(t)
	f.setFeed(f.deep)
	file := f.req("GET", "/api/v1/collections/%s/export", f.deep).as(f.owner).expect(200)
	file.has("collection.links.0.feed_url", "http://127.0.0.1/feed").has("collection.links.0.feed_filter", "Forever")

	target := f.createCollection(f.owner, f.space, "Imported", "")
	f.req("POST", "/api/v1/collections/%s/import", target).as(f.owner).json(file.Body).expect(200)
	f.req("GET", "/api/v1/collections/%s/bookmarks", target).as(f.owner).expect(200).
		has("0.feedUrl", "http://127.0.0.1/feed").has("0.feedFilter", "Forever")
}

func TestHeadlinesSectionOnTheDashboard(t *testing.T) {
	f := newShareFixture(t)
	shared, secret := f.setFeed(f.shared), f.setFeed(f.secret)
	plain := f.req("GET", "/api/v1/collections/%s/bookmarks", f.sub).as(f.owner).expect(200).str("0.id")

	// Without picked links: every link of the space with a feed the member can see
	f.req("PUT", "/api/v1/spaces/%s/dashboard", f.space).as(f.viewer).
		json(`{"sections":[{"title":"News","type":"HEADLINES"}]}`).expect(200).
		has("sections.0.type", "HEADLINES").has("sections.0.allFeeds", true).
		has("sections.0.bookmarks.#", 1).has("sections.0.bookmarks.0.id", shared)
	f.req("GET", "/api/v1/spaces/%s/dashboard", f.space).as(f.owner).expect(200).has("customized", false)

	// Picked links: those without a feed drop out
	f.req("PUT", "/api/v1/spaces/%s/dashboard", f.space).as(f.owner).
		json(`{"sections":[{"title":"News","type":"HEADLINES","bookmarkIds":["`+plain+`","`+secret+`"]}]}`).expect(200).
		has("sections.0.allFeeds", false).has("sections.0.bookmarks.#", 1).has("sections.0.bookmarks.0.id", secret)

	// The merged headlines leave out what the member cannot see; the feeds cannot be loaded in tests
	f.req("GET", "/api/v1/spaces/%s/headlines", f.space).as(f.viewer).
		query("bookmarkId", shared).query("bookmarkId", secret).query("bookmarkId", plain).
		expect(200).has("failed", true).has("items.#", 0)
	f.req("GET", "/api/v1/spaces/%s/headlines", f.space).as(f.viewer).query("bookmarkId", plain).
		expect(200).has("failed", false)
	f.req("GET", "/api/v1/spaces/%s/headlines", f.space).as(f.register()).query("bookmarkId", shared).expect(403)
}
