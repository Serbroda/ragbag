package server_test

import (
	"fmt"
	"testing"
)

func TestDefaultDashboardShowsFavoritesAndRecent(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	bookmark := e.createBookmark(owner, e.createCollection(owner, space, "Homelab", ""))

	res := e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(owner).expect(200).
		has("customized", false).has("sections.0.type", "FAVORITES").has("sections.0.bookmarks.#", 0).
		has("sections.1.type", "RECENT").has("sections.1.bookmarks.0.id", bookmark)
	// The frontend keys sections by id ?? index: default sections must not share an empty id
	if res.get("sections.0.id").Exists() || res.get("sections.1.id").Exists() {
		t.Fatalf("default sections must not have ids: %s", res.Body)
	}
}

func TestFavoritesArePersonal(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(viewer, space, "VIEWER", owner)
	collection := e.createCollection(owner, space, "Homelab", "")
	bookmark := e.createBookmark(owner, collection)

	// Viewers may star links even though they cannot edit them
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(viewer).expect(204)
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(viewer).expect(204)
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(viewer).expect(200).has("0.favorite", true)
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(owner).expect(200).has("0.favorite", false)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(viewer).expect(200).has("#", 1).has("0.favorite", true)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(viewer).expect(200).has("sections.0.bookmarks.0.id", bookmark)

	e.req("DELETE", "/api/v1/bookmarks/%s/favorite", bookmark).as(viewer).expect(204)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(viewer).expect(200).has("#", 0)
}

func TestCustomDashboardKeepsOrderAndIsPerUser(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	collection := e.createCollection(owner, space, "Homelab", "")
	first, second := e.createBookmark(owner, collection), e.createBookmark(owner, collection)

	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(owner).json(fmt.Sprintf(`{"sections":[
		{"title":"Homelab","type":"LINKS","bookmarkIds":[%q,%q,%q]},{"title":"Starred","type":"FAVORITES"}]}`,
		second, first, second)).expect(200).
		has("customized", true).has("sections.0.title", "Homelab").has("sections.0.bookmarks.#", 2).
		has("sections.0.bookmarks.0.id", second).has("sections.0.bookmarks.1.id", first).has("sections.1.type", "FAVORITES")

	other := e.register()
	e.join(other, space, "CONTRIBUTOR", owner)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(other).expect(200).has("customized", false)

	e.req("DELETE", "/api/v1/spaces/%s/dashboard", space).as(owner).expect(204)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(owner).expect(200).has("customized", false)
}

func TestDashboardValidation(t *testing.T) {
	e := newEnv(t)
	owner, stranger := e.register(), e.register()
	space := e.firstSpace(owner)
	foreign := e.createBookmark(stranger, e.createCollection(stranger, e.firstSpace(stranger), "Private", ""))

	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(owner).
		json(fmt.Sprintf(`{"sections":[{"title":"Sneaky","type":"LINKS","bookmarkIds":[%q]}]}`, foreign)).expect(400)
	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(owner).json(`{"sections":[{"title":" ","type":"RECENT"}]}`).expect(400)
	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(owner).json(`{"sections":[{"title":"x","type":"NOPE"}]}`).expect(400)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(stranger).expect(403)
}

func TestDeletingABookmarkRemovesItFromDashboardAndFavorites(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	bookmark := e.createBookmark(owner, e.createCollection(owner, space, "Homelab", ""))
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(owner).expect(204)
	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(owner).
		json(fmt.Sprintf(`{"sections":[{"title":"Pinned","type":"LINKS","bookmarkIds":[%q]}]}`, bookmark)).expect(200)

	e.req("DELETE", "/api/v1/bookmarks/%s", bookmark).as(owner).expect(204)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(owner).expect(200).
		has("customized", true).has("sections.0.bookmarks.#", 0)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(owner).expect(200).has("#", 0)
}

func TestLeavingASpaceRemovesPersonalData(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "VIEWER", owner)
	bookmark := e.createBookmark(owner, e.createCollection(owner, space, "Homelab", ""))
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(member).expect(204)
	e.req("PUT", "/api/v1/spaces/%s/dashboard", space).as(member).json(`{"sections":[{"title":"Mine","type":"RECENT"}]}`).expect(200)

	e.req("POST", "/api/v1/spaces/%s/leave", space).as(member).expect(204)
	e.join(member, space, "VIEWER", owner)
	e.req("GET", "/api/v1/spaces/%s/dashboard", space).as(member).expect(200).has("customized", false)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(member).expect(200).has("#", 0)
}
