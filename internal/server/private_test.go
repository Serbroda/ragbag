package server_test

import (
	"fmt"
	"strings"
	"testing"
)

type privateFixture struct {
	*env
	owner, giftee, friend *session
	space                 string
}

func newPrivateFixture(t *testing.T) *privateFixture {
	e := newEnv(t)
	f := &privateFixture{env: e, owner: e.register(), giftee: e.register(), friend: e.register()}
	f.space = e.firstSpace(f.owner)
	// The person the gifts are for is even an admin of the space
	e.join(f.giftee, f.space, "ADMIN", f.owner)
	e.join(f.friend, f.space, "CONTRIBUTOR", f.owner)
	return f
}

func (f *privateFixture) createPrivate(name, parentID string, memberIDs ...string) string {
	f.t.Helper()
	parent := ""
	if parentID != "" {
		parent = fmt.Sprintf(`,"parentId":%q`, parentID)
	}
	members := `"` + strings.Join(memberIDs, `","`) + `"`
	if len(memberIDs) == 0 {
		members = ""
	}
	return f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(fmt.Sprintf(`{"name":%q%s,"restricted":true,"memberIds":[%s]}`, name, parent, members)).
		expect(200).has("restricted", true).str("id")
}

func (f *privateFixture) tree(s *session) *response {
	return f.req("GET", "/api/v1/spaces/%s/collections", f.space).as(s).expect(200)
}

func TestPrivateCollectionsAreInvisibleToEveryoneElseIncludingAdmins(t *testing.T) {
	f := newPrivateFixture(t)
	gifts := f.createPrivate("Gift ideas", "", f.friend.UserID)
	sub := f.createCollection(f.friend, f.space, "Books", gifts)
	bookmark := f.createBookmark(f.friend, sub)

	f.tree(f.owner).has("0.name", "Gift ideas").has("0.restricted", true)
	f.tree(f.friend).has("0.children.0.name", "Books")
	f.tree(f.giftee).has("#", 0)

	// Direct access looks like the collection does not exist
	f.req("GET", "/api/v1/collections/%s", sub).as(f.giftee).expect(404)
	f.req("GET", "/api/v1/bookmarks/%s", bookmark).as(f.giftee).expect(404)
	f.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(f.giftee).expect(404)
	f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.giftee).
		json(fmt.Sprintf(`{"name":"Peek","parentId":%q}`, gifts)).expect(404)

	// ... and it never shows up in lists
	f.req("GET", "/api/v1/spaces/%s/bookmarks", f.space).as(f.giftee).expect(200).has("#", 0)
	f.req("GET", "/api/v1/spaces/%s/bookmarks/search", f.space).as(f.giftee).query("q", "Example").expect(200).has("#", 0)
	f.req("GET", "/api/v1/spaces/%s/dashboard", f.space).as(f.giftee).expect(200).has("sections.1.bookmarks.#", 0)
	f.req("PUT", "/api/v1/spaces/%s/dashboard", f.space).as(f.giftee).
		json(fmt.Sprintf(`{"sections":[{"title":"x","type":"LINKS","bookmarkIds":[%q]}]}`, bookmark)).expect(400)

	f.req("GET", "/api/v1/spaces/%s/bookmarks/search", f.space).as(f.friend).query("q", "Example").expect(200).has("#", 1)
}

func TestFavoritesAndPinsDisappearWhenACollectionTurnsPrivate(t *testing.T) {
	f := newPrivateFixture(t)
	shared := f.createCollection(f.owner, f.space, "Shared", "")
	bookmark := f.createBookmark(f.owner, shared)
	f.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(f.giftee).expect(204)

	f.req("PUT", "/api/v1/collections/%s/access", shared).as(f.owner).json(`{"restricted":true,"memberIds":[]}`).
		expect(200).has("members.#", 1).has("members.0.userId", f.owner.UserID).has("canManage", true)
	f.req("GET", "/api/v1/spaces/%s/favorites", f.space).as(f.giftee).expect(200).has("#", 0)

	f.req("PUT", "/api/v1/collections/%s/access", shared).as(f.owner).json(`{"restricted":false}`).expect(200).has("members.#", 0)
	f.req("GET", "/api/v1/spaces/%s/favorites", f.space).as(f.giftee).expect(200).has("#", 1)
}

func TestOnlyCreatorOrAdminsMakeASharedCollectionPrivate(t *testing.T) {
	f := newPrivateFixture(t)
	shared := f.createCollection(f.owner, f.space, "Team links", "")
	f.req("PUT", "/api/v1/collections/%s/access", shared).as(f.friend).json(`{"restricted":true}`).expect(403)
	f.req("GET", "/api/v1/collections/%s/access", shared).as(f.friend).expect(200).has("canManage", false)

	// Within a private collection, everyone who can see it may manage it
	gifts := f.createPrivate("Gift ideas", "", f.friend.UserID)
	f.req("PUT", "/api/v1/collections/%s/access", gifts).as(f.friend).
		json(fmt.Sprintf(`{"restricted":true,"memberIds":[%q]}`, f.owner.UserID)).expect(200)
	f.req("PUT", "/api/v1/collections/%s/access", gifts).as(f.friend).
		json(fmt.Sprintf(`{"restricted":true,"memberIds":[%q]}`, f.register().UserID)).expect(400)
}

func TestDeletingAParentNeverTakesHiddenCollectionsWithIt(t *testing.T) {
	f := newPrivateFixture(t)
	family := f.createCollection(f.owner, f.space, "Family", "")
	f.createPrivate("Gift ideas", family, f.friend.UserID)

	f.req("DELETE", "/api/v1/collections/%s", family).as(f.giftee).expect(409)
	f.req("DELETE", "/api/v1/collections/%s", family).as(f.owner).expect(204)
}

func TestLeavingRemovesAccessAndDeletesCollectionsNobodyCanSee(t *testing.T) {
	f := newPrivateFixture(t)
	mine := f.createPrivate("Only mine", "")
	ours := f.createPrivate("Ours", "", f.friend.UserID)

	f.req("POST", "/api/v1/spaces/%s/leave", f.space).as(f.friend).expect(204)
	f.join(f.friend, f.space, "CONTRIBUTOR", f.owner)
	f.tree(f.friend).has("#", 0)

	// Hand the space over and leave: nobody is left who could see "Only mine" or "Ours"
	f.req("PUT", "/api/v1/spaces/%s/members/%s", f.space, f.giftee.UserID).as(f.owner).json(`{"role":"OWNER"}`).expect(200)
	f.req("POST", "/api/v1/spaces/%s/leave", f.space).as(f.owner).expect(204)
	f.join(f.owner, f.space, "ADMIN", f.giftee)

	f.tree(f.owner).has("#", 0)
	f.req("GET", "/api/v1/collections/%s", mine).as(f.giftee).expect(404)
	f.req("GET", "/api/v1/collections/%s", ours).as(f.giftee).expect(404)
}
