package server_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSpaceTagsAreMergedAndCounted(t *testing.T) {
	f := newPrivateFixture(t)
	links := f.createCollection(f.owner, f.space, "Links", "")
	for _, body := range []string{
		`{"url":"http://127.0.0.1/1","tags":["Recipe","baking"]}`,
		`{"url":"http://127.0.0.1/2","tags":["recipe"]}`,
		`{"url":"http://127.0.0.1/3","tags":["recipe","go"]}`,
	} {
		f.req("POST", "/api/v1/collections/%s/bookmarks", links).as(f.owner).json(body).expect(200)
	}
	gifts := f.createPrivate("Gift ideas", "")
	f.req("POST", "/api/v1/collections/%s/bookmarks", gifts).as(f.owner).json(`{"url":"http://127.0.0.1/4","tags":["surprise"]}`).expect(200)

	f.req("GET", "/api/v1/spaces/%s/tags", f.space).as(f.owner).expect(200).
		has("0.tag", "recipe").has("0.count", 3).has("#(tag==\"surprise\").count", 1)
	// Tags of collections somebody cannot see are not suggested to them
	res := f.req("GET", "/api/v1/spaces/%s/tags", f.space).as(f.friend).expect(200)
	res.has("#", 3)
	if res.get("#(tag==\"surprise\")").Exists() {
		t.Fatal("tag of a private collection leaked")
	}
	f.req("GET", "/api/v1/spaces/%s/tags", f.space).as(f.register()).expect(403)
}

func TestLookingUpDuplicates(t *testing.T) {
	f := newPrivateFixture(t)
	links := f.createCollection(f.owner, f.space, "Links", "")
	f.req("POST", "/api/v1/collections/%s/bookmarks", links).as(f.owner).json(`{"url":"http://127.0.0.1/page","title":"Page"}`).expect(200)
	gifts := f.createPrivate("Gift ideas", "")
	f.req("POST", "/api/v1/collections/%s/bookmarks", gifts).as(f.owner).json(`{"url":"http://127.0.0.1/secret"}`).expect(200)

	lookup := func(s *session, url string) *response {
		return f.req("GET", "/api/v1/spaces/%s/bookmarks/lookup", f.space).as(s).query("url", url).expect(200)
	}
	// Normalized like when adding: no scheme is http(s)
	lookup(f.friend, "http://127.0.0.1/page").has("#", 1).has("0.title", "Page").has("0.collectionId", links)
	lookup(f.friend, "http://127.0.0.1/other").has("#", 0)
	lookup(f.friend, "not a url at all").has("#", 0)
	// A duplicate in a private collection is only reported to those who see it
	lookup(f.owner, "http://127.0.0.1/secret").has("#", 1)
	lookup(f.friend, "http://127.0.0.1/secret").has("#", 0)
}

func TestDeletedLinksGoToTheTrashAndComeBackComplete(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(viewer, space, "VIEWER", owner)
	links := e.createCollection(owner, space, "Links", "")
	bookmark := e.createBookmark(owner, links)
	e.createBookmark(owner, links)
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(owner).expect(204)
	e.req("PUT", "/api/v1/bookmarks/%s/reactions", bookmark).as(viewer).json(`{"emoji":"🔥","reacted":true}`).expect(200)

	e.req("DELETE", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(204)
	// Gone everywhere
	e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(404)
	e.req("GET", "/api/v1/collections/%s/bookmarks", links).as(owner).expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(owner).query("q", "Example").expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(owner).expect(200).has("0.bookmarkCount", 1)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/lookup", space).as(owner).query("url", "http://127.0.0.1/page").expect(200).has("#", 1)

	trash := e.req("GET", "/api/v1/spaces/%s/trash", space).as(viewer).expect(200)
	trash.has("bookmarks.#", 1).has("bookmarks.0.id", bookmark).has("bookmarks.0.collectionName", "Links").
		has("bookmarks.0.deletedBy", owner.Username).has("retentionDays", 30)

	restore := fmt.Sprintf(`{"bookmarkIds":[%q]}`, bookmark)
	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(viewer).json(restore).expect(403)
	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(owner).json(restore).expect(204)
	// Favorite and reaction are back with it
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(owner).expect(200).has("#", 1)
	e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(200).has("reactions.0.emoji", "🔥")
	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(owner).json(restore).expect(404)
}

func TestDeletedCollectionsHideEverythingBelowUntilRestored(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	parent := e.createCollection(owner, space, "Parent", "")
	child := e.createCollection(owner, space, "Child", parent)
	e.createBookmark(owner, parent)
	e.createBookmark(owner, child)
	token := e.req("PUT", "/api/v1/collections/%s/share", child).as(owner).expect(200).str("token")

	e.req("DELETE", "/api/v1/collections/%s", parent).as(owner).expect(204)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/collections/%s", child).as(owner).expect(404)
	e.req("GET", "/api/v1/spaces/%s/bookmarks", space).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/shares/%s", token).expect(404)
	e.req("GET", "/api/v1/spaces/%s/trash", space).as(owner).expect(200).
		has("collections.#", 1).has("collections.0.name", "Parent").has("collections.0.bookmarkCount", 2).
		has("collections.0.childCount", 1)

	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(owner).json(fmt.Sprintf(`{"collectionIds":[%q]}`, parent)).expect(204)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(owner).expect(200).has("0.children.0.name", "Child")
	e.req("GET", "/api/v1/shares/%s", token).expect(200)

	// A child restored while its parent stays in the trash comes back at the top level
	e.req("DELETE", "/api/v1/collections/%s", child).as(owner).expect(204)
	e.req("DELETE", "/api/v1/collections/%s", parent).as(owner).expect(204)
	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(owner).json(fmt.Sprintf(`{"collectionIds":[%q]}`, child)).expect(204)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(owner).expect(200).has("#", 1).has("0.name", "Child")
}

func TestPurgingTheTrash(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	links := e.createCollection(owner, space, "Links", "")
	first, second := e.createBookmark(owner, links), e.createBookmark(owner, links)
	e.req("DELETE", "/api/v1/bookmarks/%s", first).as(owner).expect(204)
	e.req("DELETE", "/api/v1/bookmarks/%s", second).as(owner).expect(204)

	e.req("POST", "/api/v1/spaces/%s/trash/purge", space).as(owner).json(fmt.Sprintf(`{"bookmarkIds":[%q]}`, first)).expect(204)
	e.req("GET", "/api/v1/spaces/%s/trash", space).as(owner).expect(200).has("bookmarks.#", 1)
	e.req("POST", "/api/v1/spaces/%s/trash/restore", space).as(owner).json(fmt.Sprintf(`{"bookmarkIds":[%q]}`, first)).expect(404)

	// After 30 days the job deletes the rest for good
	_, err := e.db.Exec(e.db.Rebind(`UPDATE bookmarks SET deleted_at = ? WHERE id = ?`), time.Now().Add(-31*24*time.Hour).UTC(), second)
	require.NoError(t, err)
	e.srv.PurgeOldTrash(context.Background())
	e.req("GET", "/api/v1/spaces/%s/trash", space).as(owner).expect(200).has("bookmarks.#", 0)

	e.req("DELETE", "/api/v1/collections/%s", links).as(owner).expect(204)
	e.req("POST", "/api/v1/spaces/%s/trash/purge", space).as(owner).json(`{"all":true}`).expect(204)
	e.req("GET", "/api/v1/spaces/%s/trash", space).as(owner).expect(200).has("collections.#", 0)
}

func TestTrashKeepsPrivateCollectionsPrivate(t *testing.T) {
	f := newPrivateFixture(t)
	gifts := f.createPrivate("Gift ideas", "", f.friend.UserID)
	f.createBookmark(f.friend, gifts)
	f.req("DELETE", "/api/v1/collections/%s", gifts).as(f.friend).expect(204)

	f.req("GET", "/api/v1/spaces/%s/trash", f.space).as(f.friend).expect(200).has("collections.#", 1)
	f.req("GET", "/api/v1/spaces/%s/trash", f.space).as(f.giftee).expect(200).has("collections.#", 0)
	// Admins cannot restore or purge it either
	f.req("POST", "/api/v1/spaces/%s/trash/restore", f.space).as(f.giftee).json(fmt.Sprintf(`{"collectionIds":[%q]}`, gifts)).expect(404)
	f.req("POST", "/api/v1/spaces/%s/trash/purge", f.space).as(f.giftee).json(`{"all":true}`).expect(204)
	f.req("GET", "/api/v1/spaces/%s/trash", f.space).as(f.friend).expect(200).has("collections.#", 1)
}
