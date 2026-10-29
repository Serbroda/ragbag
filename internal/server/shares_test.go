package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type shareFixture struct {
	*env
	owner, viewer          *session
	space                  string
	shared, sub, deep      string
	secret, insideSecret   string
	other                  string
	spaceName, ownerSecret string
}

// WoW Forever (shared) › Addons › UI, plus a private "Raid notes" inside and an unrelated collection.
func newShareFixture(t *testing.T) *shareFixture {
	e := newEnv(t)
	f := &shareFixture{env: e, owner: e.register(), viewer: e.register()}
	f.space = e.firstSpace(f.owner)
	f.spaceName = e.req("GET", "/api/v1/spaces/%s", f.space).as(f.owner).expect(200).str("name")
	e.join(f.viewer, f.space, "VIEWER", f.owner)
	f.shared = e.createCollection(f.owner, f.space, "WoW Forever", "")
	f.sub = e.createCollection(f.owner, f.space, "Addons", f.shared)
	f.deep = e.createCollection(f.owner, f.space, "UI", f.sub)
	f.secret = e.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(fmt.Sprintf(`{"name":"Raid notes","parentId":%q,"restricted":true}`, f.shared)).expect(200).str("id")
	f.insideSecret = e.createCollection(f.owner, f.space, "Loot", f.secret)
	f.other = e.createCollection(f.owner, f.space, "Taxes", "")
	for _, collection := range []string{f.shared, f.sub, f.deep, f.secret, f.insideSecret, f.other} {
		e.createBookmark(f.owner, collection)
	}
	return f
}

func (f *shareFixture) share() string {
	f.t.Helper()
	return f.req("PUT", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(200).has("shared", true).str("token")
}

// open requests the link without being signed in.
func (f *shareFixture) open(token, collectionID string) *request {
	r := f.req("GET", "/api/v1/shares/%s", token)
	if collectionID != "" {
		r = r.query("collectionId", collectionID)
	}
	return r
}

func TestSharedCollectionIsPublicWithoutTheRestOfTheSpace(t *testing.T) {
	f := newShareFixture(t)
	token := f.share()

	res := f.open(token, "").expect(200)
	res.has("name", "WoW Forever").has("path.#", 1).has("bookmarks.#", 1).has("bookmarks.0.title", "Example")
	// The private sub-collection is not even listed
	res.has("children.#", 1).has("children.0.name", "Addons").has("children.0.childCount", 1).has("children.0.bookmarkCount", 1)
	// Nothing about the space or its members
	require.NotContains(t, res.Body, f.owner.Username)
	require.NotContains(t, res.Body, f.spaceName)
	require.NotContains(t, res.Body, "Raid notes")
	require.False(t, res.get("bookmarks.0.createdBy").Exists())
	require.False(t, res.get("bookmarks.0.reactions").Exists())
	require.Equal(t, "noindex", res.Header.Get("X-Robots-Tag"))

	f.open(token, f.deep).expect(200).has("name", "UI").has("path.#", 3).has("path.1.name", "Addons")

	// Private collections, everything below them and the rest of the space stay closed
	for _, collection := range []string{f.secret, f.insideSecret, f.other, "00000000-0000-0000-0000-000000000000"} {
		f.open(token, collection).expect(404)
	}
	f.open("not-a-token", "").expect(404)

	page := f.req("GET", "/app/share/%s", token).send()
	require.Equal(t, "noindex, nofollow", page.Header.Get("X-Robots-Tag"))
}

func TestShareLinksCanBeRevokedAndReplaced(t *testing.T) {
	f := newShareFixture(t)
	first := f.share()
	second := f.share()
	require.NotEqual(t, first, second)
	f.open(first, "").expect(404)
	f.open(second, "").expect(200)

	f.req("GET", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(200).has("token", second)
	f.req("DELETE", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(204)
	f.open(second, "").expect(404)
	f.req("GET", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(200).has("shared", false)

	// Deleting the collection deletes the link
	third := f.share()
	f.req("DELETE", "/api/v1/collections/%s", f.shared).as(f.owner).expect(204)
	f.open(third, "").expect(404)
}

func TestPrivateCollectionsCannotBeSharedAndHideTheirLinks(t *testing.T) {
	f := newShareFixture(t)
	f.req("PUT", "/api/v1/collections/%s/share", f.secret).as(f.owner).expect(400)
	f.req("PUT", "/api/v1/collections/%s/share", f.insideSecret).as(f.owner).expect(400)

	token := f.share()
	f.req("PUT", "/api/v1/collections/%s/access", f.shared).as(f.owner).json(`{"restricted":true,"memberIds":[]}`).expect(200)
	f.open(token, "").expect(404)
	f.open(token, f.sub).expect(404)
	f.req("PUT", "/api/v1/collections/%s/access", f.shared).as(f.owner).json(`{"restricted":false}`).expect(200)
	f.open(token, "").expect(200)

	// Moved below a private collection: closed as well
	vault := f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(`{"name":"Vault","restricted":true}`).expect(200).str("id")
	f.req("POST", "/api/v1/collections/%s/move", f.shared).as(f.owner).json(fmt.Sprintf(`{"parentId":%q}`, vault)).expect(200)
	f.open(token, "").expect(404)
}

func TestOnlyEditorsManageShareLinks(t *testing.T) {
	f := newShareFixture(t)
	token := f.share()
	f.req("PUT", "/api/v1/collections/%s/share", f.shared).as(f.viewer).expect(403)
	f.req("DELETE", "/api/v1/collections/%s/share", f.shared).as(f.viewer).expect(403)
	// Viewers see that it is public, but not the link
	res := f.req("GET", "/api/v1/collections/%s/share", f.shared).as(f.viewer).expect(200).has("shared", true)
	require.False(t, res.get("token").Exists())
	f.req("GET", "/api/v1/collections/%s/share", f.shared).as(f.register()).expect(403)

	f.req("GET", "/api/v1/collections/%s", f.shared).as(f.viewer).expect(200).has("shared", true)
	tree := f.req("GET", "/api/v1/spaces/%s/collections", f.space).as(f.viewer).expect(200)
	tree.has(fmt.Sprintf("#(id==%q).shared", f.shared), true).has(fmt.Sprintf("#(id==%q).shared", f.other), false)
	require.GreaterOrEqual(t, len(token), 40, "links must not be guessable")
}
