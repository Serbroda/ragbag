package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

const testPassword = "secret123"

func (e *env) setPin(s *session, pin string) {
	e.t.Helper()
	e.req("PUT", "/api/v1/users/me/pin").as(s).json(fmt.Sprintf(`{"currentPassword":%q,"pin":%q}`, testPassword, pin)).expect(204)
}

func (e *env) unlock(s *session, pin string) string {
	e.t.Helper()
	return e.req("POST", "/api/v1/users/me/unlock").as(s).json(fmt.Sprintf(`{"pin":%q}`, pin)).expect(200).str("token")
}

func TestSettingAPin(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("hasPin", false)
	e.req("PUT", "/api/v1/users/me/pin").as(user).json(`{"currentPassword":"wrong-password","pin":"1234"}`).expect(400)
	for _, pin := range []string{"123", "123456789", "12a4", " 1234"} {
		e.req("PUT", "/api/v1/users/me/pin").as(user).json(fmt.Sprintf(`{"currentPassword":%q,"pin":%q}`, testPassword, pin)).expect(400)
	}
	e.setPin(user, "1234")
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("hasPin", true)
}

func TestLockedCollectionsDisappearOnlyForTheirOwnerUntilUnlocked(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "CONTRIBUTOR", owner)
	spicy := e.createCollection(owner, space, "Spicy", "")
	deeper := e.createCollection(owner, space, "Deeper", spicy)
	normal := e.createCollection(owner, space, "Normal", "")
	link := e.createBookmark(owner, spicy)
	e.createBookmark(owner, deeper)
	e.createBookmark(owner, normal)
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", link).as(owner).expect(204)

	// Locking needs a PIN, otherwise the collection could never be seen again
	e.req("PUT", "/api/v1/collections/%s/lock", spicy).as(owner).expect(400)
	e.setPin(owner, "2468")
	e.req("PUT", "/api/v1/collections/%s/lock", spicy).as(owner).expect(204)
	// The other member adds something to it; for the owner that must not show up anywhere
	e.createBookmark(member, deeper)

	tree := func(s *session, unlock string) *response {
		return e.req("GET", "/api/v1/spaces/%s/collections", space).as(s).header("X-Ragbag-Unlock", unlock).expect(200)
	}
	tree(owner, "").has("#", 1).has("0.name", "Normal")
	for _, path := range []string{"/api/v1/collections/%s", "/api/v1/collections/%s/bookmarks"} {
		e.req("GET", path, spicy).as(owner).expect(404)
		e.req("GET", path, deeper).as(owner).expect(404)
	}
	e.req("GET", "/api/v1/bookmarks/%s", link).as(owner).expect(404)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(owner).query("q", "Example").expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/bookmarks", space).as(owner).expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/spaces/%s/news", space).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/spaces").as(owner).expect(200).has(fmt.Sprintf("#(id==%q).newCount", space), 0)

	// Nobody else is affected
	tree(member, "").has("#", 2)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(member).query("q", "Example").expect(200).has("#", 4)

	// Unlocked, everything is back and marked
	e.req("POST", "/api/v1/users/me/unlock").as(owner).json(`{"pin":"1111"}`).expect(400)
	unlock := e.unlock(owner, "2468")
	tree(owner, unlock).has("#", 2).has("#(name==\"Spicy\").locked", true).has("#(name==\"Normal\").locked", false).
		has("#(name==\"Spicy\").children.0.name", "Deeper")
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(owner).header("X-Ragbag-Unlock", unlock).
		query("q", "Example").expect(200).has("#", 4)
	e.req("GET", "/api/v1/spaces/%s/news", space).as(owner).header("X-Ragbag-Unlock", unlock).expect(200).has("#", 1)
	e.req("POST", "/api/v1/users/me/unlock/renew").as(owner).header("X-Ragbag-Unlock", unlock).expect(200)
	e.req("POST", "/api/v1/users/me/unlock/renew").as(owner).expect(400)

	// Another user's unlock token unlocks nothing
	e.setPin(member, "1357")
	tree(owner, e.unlock(member, "1357")).has("#", 1)

	// Removing the lock needs the collection to be visible, i.e. unlocked
	e.req("DELETE", "/api/v1/collections/%s/lock", spicy).as(owner).expect(404)
	e.req("DELETE", "/api/v1/collections/%s/lock", spicy).as(owner).header("X-Ragbag-Unlock", unlock).expect(204)
	tree(owner, "").has("#", 2)
}

func TestLoggingOutEverywhereLocksAgain(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	e.setPin(user, "2468")
	e.req("PUT", "/api/v1/collections/%s/lock", e.createCollection(user, space, "Spicy", "")).as(user).expect(204)
	unlock := e.unlock(user, "2468")

	e.req("GET", "/api/auth/logout/all").as(user).expect(204)
	again := e.login(user.Username, testPassword)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(again).header("X-Ragbag-Unlock", unlock).expect(200).has("#", 0)
}

func TestWrongPinsPauseUnlocking(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.setPin(user, "2468")
	for i := 1; i < 5; i++ {
		e.req("POST", "/api/v1/users/me/unlock").as(user).json(`{"pin":"0000"}`).expect(400)
	}
	e.req("POST", "/api/v1/users/me/unlock").as(user).json(`{"pin":"0000"}`).expect(429)
	// Even the right PIN has to wait now
	e.req("POST", "/api/v1/users/me/unlock").as(user).json(`{"pin":"2468"}`).expect(429)
	// A new PIN (needs the password) starts over
	e.setPin(user, "1357")
	require.NotEmpty(t, e.unlock(user, "1357"))
}
