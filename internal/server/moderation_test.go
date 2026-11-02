package server_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func (e *env) deleteAccount(s *session, password string) *request {
	return e.req("POST", "/api/v1/users/me/delete").as(s).json(fmt.Sprintf(`{"password":%q}`, password))
}

func TestDeletingTheOwnAccount(t *testing.T) {
	e := newEnv(t)
	user, friend := e.register(), e.register()
	ownSpace := e.firstSpace(user)
	e.createBookmark(user, e.createCollection(user, ownSpace, "Mine", ""))
	// A space shared with somebody else, where the user added a link and reacted
	shared := e.firstSpace(friend)
	e.join(user, shared, "CONTRIBUTOR", friend)
	links := e.createCollection(friend, shared, "Links", "")
	bookmark := e.createBookmark(user, links)
	e.req("PUT", "/api/v1/bookmarks/%s/reactions", bookmark).as(user).json(`{"emoji":"👍","reacted":true}`).expect(200)

	e.deleteAccount(user, "wrong-password").expect(400)
	res := e.deleteAccount(user, "secret123").expect(204)
	require.Equal(t, -1, res.cookieNamed("refresh_token").MaxAge, "signed out")

	e.req("GET", "/api/v1/users/me").as(user).expect(401)
	e.req("POST", "/api/auth/login").json(fmt.Sprintf(`{"username":%q,"password":"secret123"}`, user.Username)).expect(401)
	e.req("POST", "/api/auth/refresh").cookie(user.Refresh).expect(401)

	// The own space is gone, the link in the shared one stays without the name
	var spaces int
	require.NoError(t, e.db.Get(&spaces, e.db.Rebind(`SELECT COUNT(*) FROM spaces WHERE id = ?`), ownSpace))
	require.Zero(t, spaces)
	list := e.req("GET", "/api/v1/collections/%s/bookmarks", links).as(friend).expect(200)
	list.has("#", 1).has("0.reactions.#", 0)
	require.True(t, strings.HasPrefix(list.str("0.createdBy"), "deleted-"), list.Body)
	e.req("GET", "/api/v1/spaces/%s/members", shared).as(friend).expect(200).has("#", 1)

	// Name and email address are free again; nobody can register the anonymous name
	admin := e.login("admin", "test123")
	require.False(t, e.req("GET", "/api/v1/admin/users").as(admin).expect(200).
		get(fmt.Sprintf(`#(username==%q)`, user.Username)).Exists())
	e.req("POST", "/api/auth/register").json(fmt.Sprintf(`{"username":%q,"email":"%s@example.com","password":"secret123"}`,
		user.Username, user.Username)).expect(200)
	e.req("POST", "/api/auth/register").json(`{"username":"deleted-abc","email":"x@example.com","password":"secret123"}`).
		expect(400)
}

func TestTheLastOwnerOfASharedSpaceHandsItOverFirst(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	name := e.req("GET", "/api/v1/spaces/%s", space).as(owner).expect(200).str("name")
	e.join(member, space, "ADMIN", owner)

	res := e.deleteAccount(owner, "secret123").expect(409)
	require.Contains(t, res.Body, name)
	e.req("GET", "/api/v1/users/me").as(owner).expect(200)

	e.req("PUT", "/api/v1/spaces/%s/members/%s", space, member.UserID).as(owner).json(`{"role":"OWNER"}`).expect(200)
	e.deleteAccount(owner, "secret123").expect(204)
	e.req("GET", "/api/v1/spaces/%s/members", space).as(member).expect(200).has("#", 1).has("0.role", "OWNER")
}

func TestAdminsDeleteAccountsAndHandOverTheirSpaces(t *testing.T) {
	e := newEnv(t)
	admin, user, viewer, helper := e.login("admin", "test123"), e.register(), e.register(), e.register()
	space := e.firstSpace(user)
	e.join(viewer, space, "VIEWER", user)
	e.join(helper, space, "ADMIN", user)

	e.req("DELETE", "/api/v1/admin/users/%s", user.UserID).as(viewer).expect(403)
	adminID := e.req("GET", "/api/v1/users/me").as(admin).expect(200).str("id")
	e.req("DELETE", "/api/v1/admin/users/%s", adminID).as(admin).expect(400)

	e.req("DELETE", "/api/v1/admin/users/%s", user.UserID).as(admin).expect(204)
	e.req("GET", "/api/v1/users/me").as(user).expect(401)
	// The highest role takes over
	members := e.req("GET", "/api/v1/spaces/%s/members", space).as(viewer).expect(200)
	members.has("#", 2).has(fmt.Sprintf(`#(userId==%q).role`, helper.UserID), "OWNER")
	e.req("DELETE", "/api/v1/admin/users/%s", user.UserID).as(admin).expect(404)
}

func TestBlockedAccounts(t *testing.T) {
	f := newShareFixture(t)
	token := f.share()
	admin := f.login("admin", "test123")
	block := func(s *session, userID string) *request {
		return f.req("PUT", "/api/v1/admin/users/%s/block", userID).as(s).json(`{"reason":"Spam links"}`)
	}

	block(f.viewer, f.owner.UserID).expect(403)
	block(admin, f.owner.UserID).expect(200).has("blockedReason", "Spam links")
	f.req("GET", "/api/v1/admin/users").as(admin).expect(200).
		has(fmt.Sprintf(`#(id==%q).blockedReason`, f.owner.UserID), "Spam links")

	// Signed out everywhere, no new login, the public links stop working
	f.req("GET", "/api/v1/users/me").as(f.owner).expect(401)
	f.req("POST", "/api/auth/refresh").cookie(f.owner.Refresh).expect(401)
	res := f.req("POST", "/api/auth/login").json(fmt.Sprintf(`{"username":%q,"password":"secret123"}`, f.owner.Username)).
		expect(403)
	require.Contains(t, res.Body, "Spam links")
	f.open(token, "").expect(404)
	// The space goes on for its other members
	f.req("GET", "/api/v1/spaces/%s", f.space).as(f.viewer).expect(200)

	f.req("DELETE", "/api/v1/admin/users/%s/block", f.owner.UserID).as(admin).expect(200)
	f.login(f.owner.Username, "secret123")
	f.open(token, "").expect(200)

	// Not the own account, not other admins
	adminID := f.req("GET", "/api/v1/users/me").as(admin).expect(200).str("id")
	block(admin, adminID).expect(400)
}

func TestReportingPublicLinks(t *testing.T) {
	f := newShareFixture(t)
	box := &mailbox{}
	f.srv.UseMailer(box)
	token := f.share()
	admin := f.login("admin", "test123")
	report := func(body string) *request { return f.req("POST", "/api/v1/shares/%s/reports", token).json(body) }

	// No account needed, but a description and the confirmation; only what the link shows
	report(`{"category":"ILLEGAL","details":"too short","goodFaith":true}`).expect(400)
	report(`{"category":"ILLEGAL","details":"This collection links to pirated films.","goodFaith":false}`).expect(400)
	report(`{"category":"NONSENSE","details":"This collection links to pirated films.","goodFaith":true}`).expect(400)
	f.req("POST", "/api/v1/shares/%s/reports", "unknown").
		json(`{"category":"ILLEGAL","details":"This collection links to pirated films.","goodFaith":true}`).expect(404)
	report(fmt.Sprintf(`{"category":"ILLEGAL","collectionId":%q,"details":"This collection links to pirated films.","goodFaith":true}`,
		f.secret)).expect(404)
	report(fmt.Sprintf(`{"category":"COPYRIGHT","collectionId":%q,"details":"The UI addon is a pirated copy.",
		"name":"Jane","email":"jane@example.com","goodFaith":true}`, f.deep)).expect(204)

	// The reporter gets a confirmation, the admins the report
	require.Eventually(t, func() bool { return box.count() == 2 }, 5*time.Second, 10*time.Millisecond)
	recipients := []string{box.messages[0].To, box.messages[1].To}
	require.ElementsMatch(t, []string{"jane@example.com", "admin@example.com"}, recipients)

	f.req("GET", "/api/v1/admin/reports").as(f.owner).expect(403)
	f.req("GET", "/api/v1/users/me").as(admin).expect(200).has("openReports", 1)
	reports := f.req("GET", "/api/v1/admin/reports").as(admin).expect(200)
	reports.has("#", 1).has("0.collectionName", "WoW Forever / Addons / UI").has("0.viewedCollectionId", f.deep).
		has("0.spaceName", f.spaceName).has("0.sharedBy", f.owner.Username).has("0.linkDisabled", false)
	id := reports.str("0.id")

	// Disabled: the link is gone and the space cannot share the collection again
	f.req("POST", "/api/v1/admin/reports/%s/resolve", id).as(admin).
		json(`{"action":"DISABLE_LINK","note":"Copyright infringement"}`).expect(200).
		has("linkDisabled", true).has("resolution", "DISABLE_LINK").has("resolvedBy", "admin")
	f.open(token, "").expect(404)
	f.req("GET", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(200).
		has("shared", false).has("disabled", true).has("disabledReason", "Copyright infringement")
	f.req("GET", "/api/v1/collections/%s", f.shared).as(f.owner).expect(200).has("shared", false)
	f.req("DELETE", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(204)
	f.req("PUT", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(403)
	f.req("GET", "/api/v1/users/me").as(admin).expect(200).has("openReports", 0)

	// Both sides learn the decision with the reasons
	require.Eventually(t, func() bool { return box.count() == 4 }, 5*time.Second, 10*time.Millisecond)
	for _, msg := range box.messages[2:] {
		require.Contains(t, msg.Text, "Copyright infringement")
	}
	require.ElementsMatch(t, []string{"jane@example.com", f.owner.Username + "@example.com"},
		[]string{box.messages[2].To, box.messages[3].To})

	// Enabled again after all
	f.req("POST", "/api/v1/admin/reports/%s/resolve", id).as(admin).json(`{"action":"ENABLE_LINK"}`).expect(200).
		has("linkDisabled", false)
	f.open(token, "").expect(200)
}

func TestReportsAreRateLimited(t *testing.T) {
	e := newEnv(t, withRateLimits)
	owner := e.register()
	collection := e.createCollection(owner, e.firstSpace(owner), "Public", "")
	token := e.req("PUT", "/api/v1/collections/%s/share", collection).as(owner).expect(200).str("token")
	body := `{"category":"SPAM","details":"Nothing but advertising here.","goodFaith":true}`
	for range 5 {
		e.req("POST", "/api/v1/shares/%s/reports", token).json(body).expect(204)
	}
	e.req("POST", "/api/v1/shares/%s/reports", token).json(body).expect(429)
}
