package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLinksOfOthersUntilSeen(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	recipes := e.createCollection(owner, space, "Recipes", "")
	before := e.createBookmark(owner, recipes)
	// Links from before joining are not news
	e.join(member, space, "CONTRIBUTOR", owner)
	added := e.createBookmark(owner, recipes)
	e.createBookmark(owner, recipes)
	e.createBookmark(member, recipes)

	spaceOf := func(s *session) *response {
		return e.req("GET", "/api/v1/spaces").as(s).expect(200)
	}
	spaceOf(member).has(fmt.Sprintf("#(id==%q).newCount", space), 2)
	// Own links are never new
	spaceOf(owner).has(fmt.Sprintf("#(id==%q).newCount", space), 1)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(member).expect(200).has("0.newCount", 2)

	list := e.req("GET", "/api/v1/collections/%s/bookmarks", recipes).as(member).expect(200)
	list.has(fmt.Sprintf("#(id==%q).isNew", added), true).has(fmt.Sprintf("#(id==%q).isNew", before), false)

	news := e.req("GET", "/api/v1/spaces/%s/news", space).as(member).expect(200)
	news.has("#", 1).has("0.collectionId", recipes).has("0.count", 2).has("0.addedBy", fmt.Sprintf(`[%q]`, owner.Username))

	// Listing does not mark anything as seen, opening the collection does
	spaceOf(member).has(fmt.Sprintf("#(id==%q).newCount", space), 2)
	e.req("POST", "/api/v1/collections/%s/seen", recipes).as(member).expect(204)
	spaceOf(member).has(fmt.Sprintf("#(id==%q).newCount", space), 0)
	e.req("GET", "/api/v1/spaces/%s/news", space).as(member).expect(200).has("#", 0)
	e.req("GET", "/api/v1/collections/%s/bookmarks", recipes).as(member).expect(200).
		has(fmt.Sprintf("#(id==%q).isNew", added), false)
}

func TestMarkingASpaceSeen(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "VIEWER", owner)
	e.createBookmark(owner, e.createCollection(owner, space, "A", ""))
	e.createBookmark(owner, e.createCollection(owner, space, "B", ""))

	e.req("GET", "/api/v1/spaces/%s/news", space).as(member).expect(200).has("#", 2)
	e.req("POST", "/api/v1/spaces/%s/seen", space).as(member).expect(204)
	e.req("GET", "/api/v1/spaces/%s/news", space).as(member).expect(200).has("#", 0)

	stranger := e.register()
	e.req("POST", "/api/v1/spaces/%s/seen", space).as(stranger).expect(403)
	e.req("GET", "/api/v1/spaces/%s/news", space).as(stranger).expect(403)
}

func TestNewLinksInPrivateCollectionsStayHidden(t *testing.T) {
	f := newPrivateFixture(t)
	gifts := f.createPrivate("Gift ideas", "", f.friend.UserID)
	inside := f.createCollection(f.friend, f.space, "Books", gifts)
	f.createBookmark(f.friend, inside)

	f.req("GET", "/api/v1/spaces").as(f.owner).expect(200).has(fmt.Sprintf("#(id==%q).newCount", f.space), 1)
	f.req("GET", "/api/v1/spaces").as(f.giftee).expect(200).has(fmt.Sprintf("#(id==%q).newCount", f.space), 0)
	f.req("GET", "/api/v1/spaces/%s/news", f.space).as(f.giftee).expect(200).has("#", 0)
	f.req("POST", "/api/v1/collections/%s/seen", inside).as(f.giftee).expect(404)
}

func TestReactions(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(viewer, space, "VIEWER", owner)
	collection := e.createCollection(owner, space, "Links", "")
	bookmark := e.createBookmark(owner, collection)
	react := func(s *session, emoji string, reacted bool) *request {
		return e.req("PUT", "/api/v1/bookmarks/%s/reactions", bookmark).as(s).
			json(fmt.Sprintf(`{"emoji":%q,"reacted":%t}`, emoji, reacted))
	}

	// Viewers may react too
	react(viewer, "❤️", true).expect(200).has("0.emoji", "❤️").has("0.count", 1).has("0.reacted", true)
	react(owner, "❤️", true).expect(200).has("0.count", 2).has("0.users", fmt.Sprintf(`[%q,%q]`, viewer.Username, owner.Username))
	// Reacting twice changes nothing
	react(owner, "❤️", true).expect(200).has("0.count", 2)
	react(owner, "🔥", true).expect(200).has("#", 2).has("1.emoji", "🔥").has("1.reacted", true)

	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(viewer).expect(200).
		has("0.reactions.#", 2).has("0.reactions.0.reacted", true).has("0.reactions.1.reacted", false)

	react(viewer, "❤️", false).expect(200).has("0.count", 1).has("0.reacted", false)
	react(owner, "🤡", true).expect(400)
	react(e.register(), "❤️", true).expect(403)

	// Leaving the space takes the reactions along
	e.req("POST", "/api/v1/spaces/%s/leave", space).as(viewer).expect(204)
	react(viewer, "👍", true).expect(403)
}

func TestSpaceBarKeepsOrderAndFolders(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	first := e.firstSpace(user)
	second := e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Second"}`).expect(201).str("id")
	third := e.req("POST", "/api/v1/spaces").as(user).json(`{"name":"Third"}`).expect(201).str("id")
	foreign := e.firstSpace(e.register())

	e.req("GET", "/api/v1/users/me/space-bar").as(user).expect(200).has("items", "[]")

	layout := fmt.Sprintf(`{"items":[{"spaceId":%q},{"folderId":"f1","name":" Work ","spaceIds":[%q,%q,%q]},{"spaceId":%q}]}`,
		third, first, foreign, third, second)
	// Unknown spaces and duplicates are dropped
	e.req("PUT", "/api/v1/users/me/space-bar").as(user).json(layout).expect(200).
		has("items.#", 3).has("items.1.name", "Work").has("items.1.spaceIds", fmt.Sprintf(`[%q]`, first))
	res := e.req("GET", "/api/v1/users/me/space-bar").as(user).expect(200)
	res.has("items.0.spaceId", third).has("items.2.spaceId", second)

	// Leaving a space removes it from the bar, an empty folder disappears
	e.req("POST", "/api/v1/spaces/%s/leave", first).as(user).expect(409) // last owner cannot leave
	e.req("DELETE", "/api/v1/spaces/%s", first).as(user).expect(204)
	e.req("GET", "/api/v1/users/me/space-bar").as(user).expect(200).has("items.#", 2)

	for _, body := range []string{
		`{"items":[{}]}`,
		fmt.Sprintf(`{"items":[{"spaceId":%q,"folderId":"x","name":"x","spaceIds":[]}]}`, second),
		`{"items":[{"folderId":"x","name":"  ","spaceIds":[]}]}`,
	} {
		e.req("PUT", "/api/v1/users/me/space-bar").as(user).json(body).expect(400)
	}
	require.NotEmpty(t, third)
}
