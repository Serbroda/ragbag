package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFeatureFlagsReportTheVersion(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/app/config/features").expect(200).has("version", "test").has("registration_enabled", "true")
}

func TestLoginWithWrongPassword(t *testing.T) {
	e := newEnv(t)
	e.req("POST", "/api/auth/login").json(`{"username":"admin","password":"wrong"}`).expect(401).has("detail", "Invalid credentials")
}

func TestTokensCannotBeMixedUp(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	// A refresh token is no access token ...
	e.req("GET", "/api/v1/users/me").bearer(alice.Refresh.Value).expect(401)
	// ... and the other way round
	e.req("POST", "/api/auth/refresh").cookie(&http.Cookie{Name: "refresh_token", Value: alice.Token}).expect(401)
	e.req("POST", "/api/auth/refresh").expect(401)
	e.req("POST", "/api/auth/refresh").cookie(alice.Refresh).expect(200).str("access_token")
}

func TestLogoutAllRevokesEveryToken(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	e.req("GET", "/api/auth/logout/all").as(alice).expect(204)
	e.req("POST", "/api/auth/refresh").cookie(alice.Refresh).expect(401)
	e.req("GET", "/api/v1/users/me").as(alice).expect(401)
}

func TestLogoutClearsTheCookieWithoutToken(t *testing.T) {
	e := newEnv(t)
	res := e.req("POST", "/api/auth/logout").expect(204)
	require.Contains(t, res.Header.Get("Set-Cookie"), "Max-Age=0")
}

func TestProtectedEndpointsNeedAToken(t *testing.T) {
	e := newEnv(t)
	e.req("GET", "/api/v1/spaces").expect(401).has("status", 401)
	e.req("GET", "/api/v1/users/me").bearer("not-a-jwt").expect(401)
	e.req("GET", "/api/app/config/features").expect(200).has("registration_enabled", "true")
}

func TestRefreshCookieIsSecureBehindHttpsProxies(t *testing.T) {
	e := newEnv(t)
	login := `{"username":"admin","password":"test123"}`
	// What nginx proxy manager sends for https://ragbag.example.com
	behindProxy := e.req("POST", "/api/auth/login").json(login).
		header("Origin", "https://ragbag.example.com").header("X-Forwarded-Proto", "https").expect(200)
	require.True(t, behindProxy.cookieNamed("refresh_token").Secure)
	require.Equal(t, http.SameSiteStrictMode, behindProxy.cookieNamed("refresh_token").SameSite)

	plain := e.req("POST", "/api/auth/login").json(login).header("Origin", "http://192.168.2.10:8080").expect(200)
	require.False(t, plain.cookieNamed("refresh_token").Secure)
}

func TestRegistration(t *testing.T) {
	e := newEnv(t)
	e.req("POST", "/api/auth/register").json(`{"username":"shorty","email":"s@example.com","password":"1234"}`).
		expect(400).has("detail", "Passwords must have at least 8 characters")
	e.req("POST", "/api/auth/register").json(`{"username":"Newbie","email":"n@example.com","password":"secret123"}`).
		expect(200).has("username", "newbie")
	e.req("POST", "/api/auth/register").json(`{"username":"newbie","email":"other@example.com","password":"secret123"}`).
		expect(409)
	newbie := e.login("NEWBIE", "secret123")
	e.req("GET", "/api/v1/spaces").as(newbie).expect(200).has("0.name", "Newbie's Space").has("0.role", "OWNER")
}

func TestBookmarkOfOtherUserIsInaccessible(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.register(), e.register()
	bookmark := e.createBookmark(alice, e.createCollection(alice, e.firstSpace(alice), "Private", ""))

	e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(bob).expect(403)
	e.req("PUT", "/api/v1/bookmarks/%s", bookmark).as(bob).json(`{"url":"http://127.0.0.1/evil","title":"pwned"}`).expect(403)
	e.req("DELETE", "/api/v1/bookmarks/%s", bookmark).as(bob).expect(403)
	e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(alice).expect(200).has("title", "Example")
}

func TestCollectionTreeOperations(t *testing.T) {
	e := newEnv(t)
	alice, bob := e.register(), e.register()
	space := e.firstSpace(alice)
	parent := e.createCollection(alice, space, "Parent", "")
	child := e.createCollection(alice, space, "Child", parent)

	e.req("GET", "/api/v1/collections/%s", child).as(alice).expect(200).has("parentId", parent)
	e.req("POST", "/api/v1/collections/%s/move", parent).as(alice).json(fmt.Sprintf(`{"parentId":%q}`, child)).expect(400)
	e.req("POST", "/api/v1/collections/%s/move", parent).as(alice).json(fmt.Sprintf(`{"parentId":%q}`, parent)).expect(400)
	bobs := e.createCollection(bob, e.firstSpace(bob), "Bobs", "")
	e.req("POST", "/api/v1/collections/%s/move", parent).as(alice).json(fmt.Sprintf(`{"parentId":%q}`, bobs)).expect(403)
	e.req("POST", "/api/v1/collections/%s/move", child).as(alice).json(`{"parentId":null}`).expect(200)
	e.req("GET", "/api/v1/spaces/%s/collections", space).as(alice).expect(200).has("#", 2)
	e.req("PUT", "/api/v1/collections/%s", child).as(alice).json(`{"name":"  Renamed "}`).expect(200).has("name", "Renamed")
	e.req("POST", "/api/v1/spaces/%s/collections", space).as(alice).json(`{"name":"  "}`).expect(400)
}

func TestDeletingCascades(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	space := e.firstSpace(alice)
	parent := e.createCollection(alice, space, "Parent", "")
	child := e.createCollection(alice, space, "Child", parent)
	e.createBookmark(alice, parent)
	bookmark := e.createBookmark(alice, child)
	e.req("PUT", "/api/v1/bookmarks/%s/favorite", bookmark).as(alice).expect(204)

	e.req("DELETE", "/api/v1/collections/%s", parent).as(alice).expect(204)
	e.req("GET", "/api/v1/bookmarks/%s", bookmark).as(alice).expect(404)
	e.req("GET", "/api/v1/spaces/%s/favorites", space).as(alice).expect(200).has("#", 0)

	e.createBookmark(alice, e.createCollection(alice, space, "Again", ""))
	e.req("DELETE", "/api/v1/spaces/%s", space).as(alice).expect(204)
	e.req("GET", "/api/v1/spaces").as(alice).expect(200).has("#", 0)
}

func TestLastOwnerCannotLeave(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	e.req("POST", "/api/v1/spaces/%s/leave", e.firstSpace(alice)).as(alice).expect(409)
}

func TestAcceptingAnInviteNeverDowngrades(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	space := e.firstSpace(alice)
	e.req("POST", "/api/v1/invites/%s", e.invite(alice, space, "VIEWER")).as(alice).expect(200).has("alreadyMember", true)
	e.req("DELETE", "/api/v1/spaces/%s", space).as(alice).expect(204)
}

// --- Bookmarks ---

func TestNonHttpURLsAreRejected(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	collection := e.createCollection(user, e.firstSpace(user), "Links", "")
	for _, url := range []string{"javascript:alert(1)", "data:text/html,<script>alert(1)</script>", "ftp://example.com"} {
		e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(fmt.Sprintf(`{"url":%q}`, url)).expect(400)
	}
	bookmark := e.createBookmark(user, collection)
	e.req("PUT", "/api/v1/bookmarks/%s", bookmark).as(user).json(`{"url":"javascript:alert(1)"}`).expect(400)
}

func TestBookmarkDefaultsAndUpdates(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	collection := e.createCollection(user, e.firstSpace(user), "Links", "")
	res := e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).
		json(`{"url":"127.0.0.1/docs","title":"Docs","tags":[" b ","a","b",""]}`).expect(200)
	res.has("url", "https://127.0.0.1/docs").has("createdBy", user.Username).has("tags", `["a","b"]`).has("favorite", false)
	require.NotEmpty(t, res.str("createdAt"))

	id := res.str("id")
	long := strings.Repeat("x", 200)
	e.req("PUT", "/api/v1/bookmarks/%s", id).as(user).json(fmt.Sprintf(`{"url":"https://127.0.0.1/docs","title":%q}`, long)).
		expect(200).has("tags", `["a","b"]`)
	title := e.req("GET", "/api/v1/bookmarks/%s", id).as(user).expect(200).str("title")
	require.Equal(t, 160, len([]rune(title)))
	require.True(t, strings.HasSuffix(title, "…"))

	// Without a title the URL is used
	e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(`{"url":"http://127.0.0.1/x"}`).
		expect(200).has("title", "http://127.0.0.1/x")
}

func TestRecentBookmarksAreNewestFirstAndLimited(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	e.createBookmark(user, collection)
	e.createBookmark(user, e.createCollection(user, space, "Other", ""))
	newest := e.createBookmark(user, collection)

	e.req("GET", "/api/v1/spaces/%s/bookmarks", space).as(user).query("limit", "2").expect(200).has("#", 2).has("0.id", newest)
}

func TestSearchTreatsWildcardsLiterally(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	e.createBookmark(user, e.createCollection(user, space, "Links", ""))
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", "%").expect(200).has("#", 0)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", "EXAMP").expect(200).has("#", 1)
	// Tags are searched too
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", "a").expect(200).has("#", 1)
	e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", " ").expect(200).has("#", 0)
}

func TestFetchMetadata(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	// Loopback is refused by the SSRF guard, so this only checks normalization and the empty result
	e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", collection).as(user).json(`{"url":" 127.0.0.1/page "}`).
		expect(200).has("url", "https://127.0.0.1/page").has("found", false)
	e.req("GET", "/api/v1/collections/%s/bookmarks", collection).as(user).expect(200).has("#", 0)
	e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", collection).as(user).json(`{"url":"javascript:alert(1)"}`).expect(400)

	viewer := e.register()
	e.join(viewer, space, "VIEWER", user)
	e.req("POST", "/api/v1/collections/%s/bookmarks/metadata", collection).as(viewer).json(`{"url":"https://example.com"}`).expect(403)
}

func TestMalformedBodiesAreBadRequests(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	e.req("POST", "/api/v1/spaces").as(user).json(`{"name":`).expect(400).has("detail", "Malformed request body")
	e.req("GET", "/api/v1/nothing-here").as(user).expect(404)
}

func TestCollectionDescriptionAndBookmarkCounts(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	parent := e.createCollection(user, space, "Parent", "")
	child := e.createCollection(user, space, "Child", parent)
	e.createBookmark(user, child)
	e.createBookmark(user, child)

	e.req("GET", "/api/v1/spaces/%s/collections", space).as(user).expect(200).
		has("0.bookmarkCount", 0).has("0.children.0.bookmarkCount", 2)

	e.req("PUT", "/api/v1/collections/%s", parent).as(user).json(`{"name":"Parent","description":" Things "}`).
		expect(200).has("description", "Things")
	// Without a description it is kept, an empty one removes it
	e.req("PUT", "/api/v1/collections/%s", parent).as(user).json(`{"name":"Renamed"}`).expect(200).has("description", "Things")
	e.req("PUT", "/api/v1/collections/%s", parent).as(user).json(`{"name":"Renamed","description":""}`).expect(200)
	require.False(t, e.req("GET", "/api/v1/collections/%s", parent).as(user).expect(200).get("description").Exists())
	e.req("PUT", "/api/v1/collections/%s", parent).as(user).
		json(fmt.Sprintf(`{"name":"Renamed","description":%q}`, strings.Repeat("x", 501))).expect(400)
}

func TestMovingBookmarks(t *testing.T) {
	e := newEnv(t)
	owner, viewer := e.register(), e.register()
	space := e.firstSpace(owner)
	from := e.createCollection(owner, space, "From", "")
	to := e.createCollection(owner, space, "To", "")
	bookmark := e.createBookmark(owner, from)
	move := func(s *session, target string) *request {
		return e.req("PUT", "/api/v1/bookmarks/%s", bookmark).as(s).
			json(fmt.Sprintf(`{"url":"http://127.0.0.1/page","title":"Example","tags":["a"],"collectionId":%q}`, target))
	}

	move(owner, to).expect(200).has("collectionId", to).has("title", "Example").has("tags", `["a"]`)
	e.req("GET", "/api/v1/collections/%s/bookmarks", from).as(owner).expect(200).has("#", 0)
	e.req("GET", "/api/v1/collections/%s/bookmarks", to).as(owner).expect(200).has("#", 1)

	// Only within the space, and only with the permissions of the user
	other := e.req("POST", "/api/v1/spaces").as(owner).json(`{"name":"Other"}`).expect(201).str("id")
	move(owner, e.createCollection(owner, other, "Elsewhere", "")).expect(400)
	e.join(viewer, space, "VIEWER", owner)
	move(viewer, from).expect(403)
}

func TestSearchingForATagMatchesItExactly(t *testing.T) {
	e := newEnv(t)
	user := e.register()
	space := e.firstSpace(user)
	collection := e.createCollection(user, space, "Links", "")
	for _, body := range []string{
		`{"url":"http://127.0.0.1/1","title":"Recipe collection","tags":["food"]}`,
		`{"url":"http://127.0.0.1/2","title":"Pie","tags":["Recipe"]}`,
		`{"url":"http://127.0.0.1/3","title":"Other","tags":["recipes"]}`,
	} {
		e.req("POST", "/api/v1/collections/%s/bookmarks", collection).as(user).json(body).expect(200)
	}
	search := func(q string) *response {
		return e.req("GET", "/api/v1/spaces/%s/bookmarks/search", space).as(user).query("q", q).expect(200)
	}
	search("#recipe").has("#", 1).has("0.title", "Pie")
	search("recipe").has("#", 3)
	search("#").has("#", 0)
}

func TestCollectionsKeepAManualOrder(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	e.join(member, space, "CONTRIBUTOR", owner)
	a := e.createCollection(owner, space, "A", "")
	b := e.createCollection(owner, space, "B", "")
	c := e.createCollection(owner, space, "C", "")
	names := func(path string) string {
		return e.req("GET", "/api/v1/spaces/%s/collections", space).as(member).expect(200).get(path).String()
	}
	move := func(id, body string) *request {
		return e.req("POST", "/api/v1/collections/%s/move", id).as(owner).json(body)
	}
	require.Equal(t, `["A","B","C"]`, names("#.name"))

	move(c, fmt.Sprintf(`{"parentId":null,"beforeId":%q}`, a)).expect(200)
	require.Equal(t, `["C","A","B"]`, names("#.name"))
	// Without beforeId it goes to the end
	move(a, `{"parentId":null}`).expect(200)
	require.Equal(t, `["C","B","A"]`, names("#.name"))

	// Into another collection and in front of a child there, in one go
	x := e.createCollection(owner, space, "X", c)
	move(b, fmt.Sprintf(`{"parentId":%q,"beforeId":%q}`, c, x)).expect(200)
	require.Equal(t, `["C","A"]`, names("#.name"))
	require.Equal(t, `["B","X"]`, names("0.children.#.name"))
	// New collections are appended
	e.createCollection(owner, space, "Y", c)
	require.Equal(t, `["B","X","Y"]`, names("0.children.#.name"))

	// beforeId has to be below the target parent
	move(a, fmt.Sprintf(`{"parentId":null,"beforeId":%q}`, x)).expect(400)
	move(a, fmt.Sprintf(`{"parentId":null,"beforeId":%q}`, a)).expect(400)
	// Ordering is editing
	e.req("POST", "/api/v1/collections/%s/move", a).as(e.register()).json(`{"parentId":null}`).expect(403)
}
