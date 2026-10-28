package server_test

import (
	"testing"
)

func TestSpacesAreNotDiscoverableByDefault(t *testing.T) {
	f := newSpaceFixture(t)
	outsider := f.register()
	f.req("GET", "/api/v1/spaces/%s", f.space).as(f.owner).expect(200).has("discoverable", false).has("joinRole", "VIEWER")
	f.req("GET", "/api/v1/spaces/discover").as(outsider).expect(200).has("#", 0)
	// Invite-only spaces cannot be joined, and do not reveal that they exist
	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(outsider).expect(404)
	f.req("POST", "/api/v1/spaces/%s/join", "00000000-0000-0000-0000-000000000000").as(outsider).expect(404)
}

func TestDiscoverabilityNeedsManageMembers(t *testing.T) {
	f := newSpaceFixture(t)
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.contributor).json(`{"name":"Space","discoverable":true}`).expect(403)
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.admin).json(`{"name":"Space","discoverable":true,"joinRole":"CONTRIBUTOR"}`).
		expect(200).has("discoverable", true).has("joinRole", "CONTRIBUTOR")
	// Leaving the fields out keeps them
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.owner).json(`{"name":"Renamed"}`).
		expect(200).has("name", "Renamed").has("discoverable", true).has("joinRole", "CONTRIBUTOR")
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.owner).json(`{"name":"Space","joinRole":"OWNER"}`).expect(400)
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.owner).json(`{"name":"Space","joinRole":"ADMIN"}`).expect(400)
}

func TestDiscoverAndJoinSpace(t *testing.T) {
	f := newSpaceFixture(t)
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.owner).
		json(`{"name":"Homelab 100%","description":"Self-hosting links","discoverable":true,"joinRole":"CONTRIBUTOR"}`).expect(200)
	secret := f.createCollection(f.owner, f.space, "Secret", "")
	f.req("PUT", "/api/v1/collections/%s/access", secret).as(f.owner).json(`{"restricted":true,"memberIds":[]}`).expect(200)
	visible := f.createCollection(f.owner, f.space, "Visible", "")

	outsider := f.register()
	f.req("GET", "/api/v1/spaces/discover").as(outsider).expect(200).
		has("#", 1).has("0.name", "Homelab 100%").has("0.memberCount", 4).has("0.joinRole", "CONTRIBUTOR").
		has("0.description", "Self-hosting links")
	res := f.req("GET", "/api/v1/spaces/discover").as(outsider).expect(200)
	if res.get("0.role").Exists() {
		t.Fatalf("outsider must not have a role: %s", res.Body)
	}
	f.req("GET", "/api/v1/spaces/discover").as(outsider).query("q", "SELF-HOST").expect(200).has("#", 1)
	f.req("GET", "/api/v1/spaces/discover").as(outsider).query("q", "100%").expect(200).has("#", 1)
	f.req("GET", "/api/v1/spaces/discover").as(outsider).query("q", "_").expect(200).has("#", 0)
	f.req("GET", "/api/v1/spaces/discover").as(outsider).query("q", "gifts").expect(200).has("#", 0)

	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(outsider).expect(200).
		has("role", "CONTRIBUTOR").has(`permissions.#(=="CREATE_BOOKMARKS")`, "CREATE_BOOKMARKS")
	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(outsider).expect(200).has("role", "CONTRIBUTOR")
	f.req("GET", "/api/v1/spaces/discover").as(outsider).expect(200).has("0.memberCount", 5).has("0.role", "CONTRIBUTOR")

	// Joining never changes an existing membership
	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(f.admin).expect(200).has("role", "ADMIN")
	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(f.viewer).expect(200).has("role", "VIEWER")

	// Private collections stay private for people who joined
	f.req("GET", "/api/v1/collections/%s", secret).as(outsider).expect(404)
	f.req("GET", "/api/v1/collections/%s", visible).as(outsider).expect(200)

	// Back to invite only: no longer listed or joinable, members stay
	f.req("PUT", "/api/v1/spaces/%s", f.space).as(f.owner).json(`{"name":"Homelab","discoverable":false}`).expect(200)
	latecomer := f.register()
	f.req("GET", "/api/v1/spaces/discover").as(latecomer).expect(200).has("#", 0)
	f.req("POST", "/api/v1/spaces/%s/join", f.space).as(latecomer).expect(404)
	f.req("GET", "/api/v1/spaces/%s", f.space).as(outsider).expect(200).has("role", "CONTRIBUTOR")
}

func TestDiscoverNeedsLogin(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/v1/spaces/discover").expect(401)
}
