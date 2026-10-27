package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type exchangeFixture struct {
	*env
	owner       *session
	space, root string
}

func newExchangeFixture(t *testing.T) *exchangeFixture {
	e := newEnv(t)
	f := &exchangeFixture{env: e, owner: e.register()}
	f.space = e.firstSpace(f.owner)
	f.root = e.createCollection(f.owner, f.space, "Homelab", "")
	e.createBookmark(f.owner, f.root)
	e.req("POST", "/api/v1/collections/%s/bookmarks", f.root).as(f.owner).
		json(`{"url":"http://127.0.0.1/newest","title":"Newest"}`).expect(200)
	child := e.createCollection(f.owner, f.space, "Monitoring", f.root)
	e.createBookmark(f.owner, child)
	return f
}

func (f *exchangeFixture) export(s *session, collectionID string) *request {
	return f.req("GET", "/api/v1/collections/%s/export", collectionID).as(s)
}

func (f *exchangeFixture) importFile(s *session, collectionID, file string, preview bool) *request {
	return f.req("POST", "/api/v1/collections/%s/import", collectionID).as(s).query("preview", fmt.Sprint(preview)).json(file)
}

func TestExportIsVersionedAndContainsTheTree(t *testing.T) {
	f := newExchangeFixture(t)
	res := f.export(f.owner, f.root).expect(200)
	res.has("format", "ragbag-collection").has("format_version", 1).has("app_version", "test").
		has("collection.name", "Homelab").
		has("collection.links.0.url", "http://127.0.0.1/page").has("collection.links.0.tags.0", "a").
		has("collection.links.1.url", "http://127.0.0.1/newest").
		has("collection.collections.0.name", "Monitoring").has("collection.collections.0.links.#", 1)
	require.NotEmpty(t, res.str("exported_at"))
}

func TestImportRoundTripWithPreviewAndDuplicates(t *testing.T) {
	f := newExchangeFixture(t)
	file := f.export(f.owner, f.root).expect(200).Body
	other := f.register()
	otherSpace := f.firstSpace(other)
	target := f.createCollection(other, otherSpace, "Imported", "")

	f.importFile(other, target, file, true).expect(200).
		has("preview", true).has("collectionsCreated", 1).has("bookmarksImported", 3).has("appVersion", "test")
	// The preview changed nothing
	f.req("GET", "/api/v1/collections/%s/bookmarks", target).as(other).expect(200).has("#", 0)
	f.req("GET", "/api/v1/spaces/%s/collections", otherSpace).as(other).expect(200).has("0.children.#", 0)

	f.importFile(other, target, file, false).expect(200).has("preview", false).has("bookmarksImported", 3)
	imported := f.req("GET", "/api/v1/collections/%s/bookmarks", target).as(other).expect(200)
	imported.has("0.title", "Newest").has("1.title", "Example")
	// Imported links keep their original date
	source := f.req("GET", "/api/v1/collections/%s/bookmarks", f.root).as(f.owner).expect(200)
	require.Equal(t, source.str("0.createdAt"), imported.str("0.createdAt"))
	f.req("GET", "/api/v1/spaces/%s/collections", otherSpace).as(other).expect(200).has("0.children.0.name", "Monitoring")

	// Importing again merges the sub-collection and skips known links
	f.importFile(other, target, file, false).expect(200).
		has("collectionsCreated", 0).has("collectionsMerged", 1).has("bookmarksImported", 0).has("bookmarksSkipped", 3)
}

func TestImportRejectsForeignAndNewerFilesAndSkipsInvalidLinks(t *testing.T) {
	f := newExchangeFixture(t)
	f.importFile(f.owner, f.root, `{"format":"jucosuma","format_version":2}`, false).expect(400)
	f.importFile(f.owner, f.root, `[]`, false).expect(400)
	f.importFile(f.owner, f.root, `{"format":"ragbag-collection"}`, false).expect(400)
	res := f.importFile(f.owner, f.root, `{"format":"ragbag-collection","format_version":99}`, false).expect(400)
	require.Contains(t, res.str("detail"), "newer version")

	f.importFile(f.owner, f.root, `{"format":"ragbag-collection","format_version":1,"collection":{"name":"x","links":[
		{"url":"javascript:alert(1)"},{"url":"example.org","favicon":"javascript:alert(1)"}]}}`, false).expect(200).
		has("bookmarksImported", 1).has("bookmarksInvalid", 1)
	links := f.req("GET", "/api/v1/collections/%s/bookmarks", f.root).as(f.owner).expect(200)
	imported := links.get(`#(url=="https://example.org")`)
	require.True(t, imported.Exists(), links.Body)
	require.False(t, imported.Get("favicon").Exists())
}

func TestPrivateCollectionsStayPrivateAndHiddenOnesAreNotExported(t *testing.T) {
	f := newExchangeFixture(t)
	member := f.register()
	f.join(member, f.space, "ADMIN", f.owner)
	secret := f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(fmt.Sprintf(`{"name":"Gifts for %s","parentId":%q,"restricted":true}`, member.Username, f.root)).expect(200).str("id")
	f.createBookmark(f.owner, secret)

	// The admin cannot see the private collection, so it is not part of their export
	f.export(member, f.root).expect(200).has("collection.collections.#", 1)
	ownerExport := f.export(f.owner, f.root).expect(200).Body
	require.Equal(t, int64(1), gjson.Get(ownerExport, `collection.collections.#(private==true)#|#`).Int())

	other := f.register()
	otherSpace := f.firstSpace(other)
	target := f.createCollection(other, otherSpace, "Copy", "")
	f.importFile(other, target, ownerExport, false).expect(200).has("collectionsCreated", 2)
	f.req("GET", "/api/v1/spaces/%s/collections", otherSpace).as(other).expect(200).
		has("0.children.#(restricted==true).name", "Gifts for "+member.Username)
}

func TestContributorsMayNeitherExportNorImport(t *testing.T) {
	f := newExchangeFixture(t)
	contributor := f.register()
	f.join(contributor, f.space, "CONTRIBUTOR", f.owner)
	f.export(contributor, f.root).expect(403)
	f.importFile(contributor, f.root, f.export(f.owner, f.root).expect(200).Body, true).expect(403)
}
