package server_test

import (
	"fmt"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

// apiToken creates a personal API token like the profile dialog and returns the token and its id.
func (e *env) apiToken(s *session, scope string) (string, string) {
	res := e.req("POST", "/api/v1/users/me/tokens").as(s).
		json(fmt.Sprintf(`{"name":"script","scope":%q,"currentPassword":%q}`, scope, testPassword)).expect(200)
	require.Regexp(e.t, `^rb_.{40,}$`, res.str("token"))
	return res.str("token"), res.str("id")
}

func TestAPITokensReadAndWriteLinksAndCollections(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	e.createBookmark(user, collection)
	read, _ := e.apiToken(user, "READ")
	write, _ := e.apiToken(user, "WRITE")

	// Spaces → collections → links, like the app
	e.req("GET", "/api/v1/users/me").bearer(read).expect(200).has("username", user.Username)
	e.req("GET", "/api/v1/spaces").bearer(read).expect(200).has("0.id", space)
	e.req("GET", "/api/v1/spaces/%s/collections", space).bearer(read).expect(200).has("0.name", "Links")
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).bearer(read).expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).bearer(read).query("q", "Example").expect(200).has("#", 1)

	add := `{"url":"https://example.org/from-a-script","title":"Scripted"}`
	denied := e.req("POST", "/api/v1/collections/%s/bookmarks", collection).bearer(read).json(add).expect(403)
	require.Contains(t, denied.str("detail"), "only reads")
	link := e.req("POST", "/api/v1/collections/%s/bookmarks", collection).bearer(write).json(add).expect(200).str("id")
	e.req("POST", "/api/v1/spaces/%s/collections", space).bearer(write).json(`{"name":"By script"}`).expect(200)
	e.req("DELETE", "/api/v1/bookmarks/%s", link).bearer(write).expect(204)

	// The list shows when a token was used, never the token
	list := e.req("GET", "/api/v1/users/me/tokens").as(user).expect(200).has("#", 2)
	require.NotEmpty(t, list.str("0.lastUsedAt"))
	require.NotContains(t, list.Body, read)
	require.NotContains(t, list.Body, "token_hash")
}

func TestAPITokensNeverReachAccountsOrSpaceManagement(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	write, _ := e.apiToken(user, "WRITE")
	password := fmt.Sprintf(`{"currentPassword":%q}`, testPassword)

	for _, call := range []struct{ method, path, body string }{
		// The account and its secrets
		{"GET", "/api/v1/users/me/tokens", ""},
		{"POST", "/api/v1/users/me/tokens", `{"name":"more","scope":"WRITE","currentPassword":"secret123"}`},
		{"POST", "/api/v1/users/me/two-factor/setup", password},
		{"PUT", "/api/v1/users/me/pin", `{"currentPassword":"secret123","pin":"1234"}`},
		{"POST", "/api/v1/users/me/unlock", `{"pin":"1234"}`},
		{"POST", "/api/v1/users/me/delete", `{"password":"secret123"}`},
		{"POST", "/api/auth/password", `{"currentPassword":"secret123","newPassword":"another-secret"}`},
		{"POST", "/api/auth/logout/all", ""},
		{"GET", "/api/v1/admin/users", ""},
		{"GET", "/api/v1/invites", ""},
		// Managing a space and who sees what
		{"POST", "/api/v1/spaces", `{"name":"New"}`},
		{"PUT", "/api/v1/spaces/" + space, `{"name":"Renamed"}`},
		{"DELETE", "/api/v1/spaces/" + space, ""},
		{"POST", "/api/v1/spaces/" + space + "/leave", ""},
		{"GET", "/api/v1/spaces/" + space + "/invites", ""},
		{"POST", "/api/v1/spaces/" + space + "/invites", `{"role":"VIEWER"}`},
		{"POST", "/api/v1/spaces/" + space + "/trash/purge", `{}`},
		{"GET", "/api/v1/spaces/discover", ""},
		{"PUT", "/api/v1/collections/" + collection + "/share", ""},
		{"PUT", "/api/v1/collections/" + collection + "/access", `{"restricted":true}`},
		{"PUT", "/api/v1/collections/" + collection + "/lock", ""},
	} {
		r := e.req(call.method, "%s", call.path).bearer(write)
		if call.body != "" {
			r = r.json(call.body)
		}
		require.Equal(t, 403, r.send().Status, "%s %s", call.method, call.path)
	}
	// Nothing happened
	e.req("GET", "/api/v1/spaces/%s", space).as(user).expect(200)
	e.req("GET", "/api/v1/users/me/tokens").as(user).expect(200).has("#", 1)
}

func TestAPITokensKeepLockedAndPrivateCollectionsHidden(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	member := e.register()
	e.join(member, space, "CONTRIBUTOR", owner)
	secret := e.req("POST", "/api/v1/spaces/%s/collections", space).as(owner).
		json(`{"name":"Secret","restricted":true}`).expect(200).str("id")
	token, _ := e.apiToken(member, "WRITE")
	e.req("GET", "/api/v1/collections/%s/bookmarks", secret).bearer(token).expect(404)

	// The member locks a collection for themselves: a token never unlocks it, not even with an unlock token
	locked := e.createCollection(member, space, "Locked", "")
	e.req("PUT", "/api/v1/users/me/pin").as(member).json(fmt.Sprintf(`{"currentPassword":%q,"pin":"1234"}`, testPassword)).expect(204)
	e.req("PUT", "/api/v1/collections/%s/lock", locked).as(member).expect(204)
	unlock := e.req("POST", "/api/v1/users/me/unlock").as(member).json(`{"pin":"1234"}`).expect(200).str("token")
	e.req("GET", "/api/v1/collections/%s", locked).as(member).header("X-Ragbag-Unlock", unlock).expect(200)
	e.req("GET", "/api/v1/collections/%s", locked).bearer(token).header("X-Ragbag-Unlock", unlock).expect(404)
}

func TestAPITokensAreCreatedWithThePasswordAndCanBeRevoked(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	create := func(body string) *request { return e.req("POST", "/api/v1/users/me/tokens").as(user).json(body) }
	create(`{"name":"x","scope":"WRITE","currentPassword":"wrong"}`).expect(400)
	create(`{"name":" ","scope":"WRITE","currentPassword":"secret123"}`).expect(400)
	create(`{"name":"x","scope":"ADMIN","currentPassword":"secret123"}`).expect(400)
	create(`{"name":"x","scope":"READ","expiresInDays":0,"currentPassword":"secret123"}`).expect(400)
	expiring := create(`{"name":"Thirty days","scope":"READ","expiresInDays":30,"currentPassword":"secret123"}`).expect(200)
	require.NotEmpty(t, expiring.str("expiresAt"))

	token, id := e.apiToken(user, "READ")
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200)
	// Somebody else cannot revoke it
	other := e.register()
	e.req("DELETE", "/api/v1/users/me/tokens/%s", id).as(other).expect(404)
	// A new password keeps scripts running; revoking stops them at once
	e.req("POST", "/api/auth/password").as(user).json(`{"currentPassword":"secret123","newPassword":"another-secret"}`).expect(200)
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200)
	fresh := e.login(user.Username, "another-secret")
	e.req("DELETE", "/api/v1/users/me/tokens/%s", id).as(fresh).expect(204)
	e.req("GET", "/api/v1/spaces").bearer(token).expect(401)
	e.req("GET", "/api/v1/spaces").bearer("rb_unknown").expect(401)
}

func TestBlockedAccountsLoseTheirAPITokens(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	token, _ := e.apiToken(user, "READ")
	admin := e.login("admin", "test123")
	e.req("PUT", "/api/v1/admin/users/%s/block", user.UserID).as(admin).json(`{"reason":"spam"}`).expect(200)
	e.req("GET", "/api/v1/spaces").bearer(token).expect(401)
}

func TestWithPlansAPITokensArePartOfPro(t *testing.T) {
	e := newEnv(t, withPlans(config.Quotas{}, config.Quotas{}))
	user := e.register()
	admin := e.login("admin", "test123")
	create := func() *request {
		return e.req("POST", "/api/v1/users/me/tokens").as(user).
			json(fmt.Sprintf(`{"name":"script","scope":"READ","currentPassword":%q}`, testPassword))
	}
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("apiTokens", false)
	e.req("GET", "/api/v1/users/me/plan").as(user).expect(200).has("freeApiTokens", false)
	require.Contains(t, create().expect(403).str("detail"), "part of Pro")
	// Instance admins are Pro
	e.req("GET", "/api/v1/users/me").as(admin).expect(200).has("apiTokens", true)

	e.req("PUT", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200)
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("apiTokens", true)
	created := create().expect(200)
	token := created.str("token")
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200)

	// Free again: the token rests, can still be seen and revoked, and works again with Pro
	e.req("DELETE", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200)
	require.Contains(t, e.req("GET", "/api/v1/spaces").bearer(token).expect(403).str("detail"), "part of Pro")
	e.req("GET", "/api/v1/users/me/tokens").as(user).expect(200).has("#", 1)
	e.req("PUT", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200)
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200)
	e.req("DELETE", "/api/v1/admin/users/%s/pro", user.UserID).as(admin).expect(200)
	e.req("DELETE", "/api/v1/users/me/tokens/%s", created.str("id")).as(user).expect(204)
}

func TestPlansCanGiveAPITokensToFreeAccounts(t *testing.T) {
	e := newEnv(t, withPlans(config.Quotas{}, config.Quotas{}), func(cfg *config.Config) { cfg.Plans.FreeAPITokens = true })
	user := e.register()
	e.req("GET", "/api/v1/users/me").as(user).expect(200).has("apiTokens", true)
	e.req("GET", "/api/v1/users/me/plan").as(user).expect(200).has("freeApiTokens", true)
	token, _ := e.apiToken(user, "READ")
	e.req("GET", "/api/v1/spaces").bearer(token).expect(200)
}

func TestWithoutPlansEverybodyHasAPITokens(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/v1/users/me").as(e.register()).expect(200).has("apiTokens", true)
}
