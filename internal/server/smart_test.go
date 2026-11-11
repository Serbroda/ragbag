package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type smartFixture struct {
	*env
	owner        *session
	space, links string
}

func newSmartFixture(t *testing.T) *smartFixture {
	e := newEnv(t)
	f := &smartFixture{env: e, owner: e.register()}
	f.space = e.firstSpace(f.owner)
	f.links = e.createCollection(f.owner, f.space, "Links", "")
	f.add(f.owner, f.links, "Pie", "recipe", "sweet")
	f.add(f.owner, f.links, "Soup", "recipe")
	f.add(f.owner, f.links, "Recipe for disaster")
	return f
}

func (f *smartFixture) add(s *session, collectionID, title string, tags ...string) string {
	list := ""
	for i, tag := range tags {
		if i > 0 {
			list += ","
		}
		list += fmt.Sprintf("%q", tag)
	}
	return f.req("POST", "/api/v1/collections/%s/bookmarks", collectionID).as(s).
		json(fmt.Sprintf(`{"url":"http://127.0.0.1/%s","title":%q,"tags":[%s]}`, title[:3], title, list)).expect(200).str("id")
}

func (f *smartFixture) smart(s *session, name, query, parentID string) *request {
	body := fmt.Sprintf(`{"name":%q,"query":%q}`, name, query)
	if parentID != "" {
		body = fmt.Sprintf(`{"name":%q,"query":%q,"parentId":%q}`, name, query, parentID)
	}
	return f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(s).json(body)
}

func (f *smartFixture) list(s *session, collectionID string) *response {
	return f.req("GET", "/api/v1/collections/%s/bookmarks", collectionID).as(s).expect(200)
}

func TestSmartCollectionsListWhatTheirSearchFinds(t *testing.T) {
	f := newSmartFixture(t)
	created := f.smart(f.owner, "Recipes", " #recipe ", "").expect(200).has("query", "#recipe")
	smart := created.str("id")
	f.list(f.owner, smart).has("#", 2).has("0.title", "Soup").has("1.title", "Pie").has("0.collectionId", f.links)
	f.req("GET", "/api/v1/collections/%s", smart).as(f.owner).expect(200).has("query", "#recipe")
	f.req("GET", "/api/v1/spaces/%s/collections", f.space).as(f.owner).expect(200).
		has(`#(name=="Recipes").query`, "#recipe").has(`#(name=="Recipes").bookmarkCount`, 0)

	// Every term has to match: tags exactly, words anywhere, quotes keep a phrase together
	f.req("PUT", "/api/v1/collections/%s", smart).as(f.owner).json(`{"name":"Recipes","query":"#recipe #sweet"}`).
		expect(200).has("query", "#recipe #sweet")
	f.list(f.owner, smart).has("#", 1).has("0.title", "Pie")
	f.req("PUT", "/api/v1/collections/%s", smart).as(f.owner).json(`{"name":"Recipes","query":"recipe disaster"}`).expect(200)
	f.list(f.owner, smart).has("#", 1).has("0.title", "Recipe for disaster")
	f.req("PUT", "/api/v1/collections/%s", smart).as(f.owner).json(`{"name":"Recipes","query":"\"disaster recipe\""}`).expect(200)
	f.list(f.owner, smart).has("#", 0)
	// Renaming keeps the search
	f.req("PUT", "/api/v1/collections/%s", smart).as(f.owner).json(`{"name":"Food"}`).expect(200).
		has("name", "Food").has("query", `"disaster recipe"`)

	f.req("PUT", "/api/v1/collections/%s", smart).as(f.owner).json(`{"name":"Food","query":" "}`).expect(400)
	f.smart(f.owner, "Too long", fmt.Sprintf("%0201d", 0), "").expect(400)
	// An ordinary collection never becomes smart: its links would vanish
	f.req("PUT", "/api/v1/collections/%s", f.links).as(f.owner).json(`{"name":"Links","query":"x"}`).expect(400)
	// The search itself got the same terms
	f.req("GET", "/api/v1/spaces/%s/bookmarks/search", f.space).as(f.owner).query("q", "#recipe pie").expect(200).has("#", 1)
}

func TestSmartCollectionsHoldNothingAndAreNeverPublic(t *testing.T) {
	f := newSmartFixture(t)
	smart := f.smart(f.owner, "Recipes", "#recipe", f.links).expect(200).str("id")

	res := f.req("POST", "/api/v1/collections/%s/bookmarks", smart).as(f.owner).json(`{"url":"http://127.0.0.1/x"}`).expect(400)
	require.Contains(t, res.str("detail"), "smart collection")
	link := f.list(f.owner, f.links).str("0.id")
	f.req("PUT", "/api/v1/bookmarks/%s", link).as(f.owner).
		json(fmt.Sprintf(`{"url":"http://127.0.0.1/Rec","title":"x","collectionId":%q}`, smart)).expect(400)
	f.smart(f.owner, "Nested", "x", smart).expect(400)
	f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(fmt.Sprintf(`{"name":"Child","parentId":%q}`, smart)).expect(400)
	other := f.createCollection(f.owner, f.space, "Other", "")
	f.req("POST", "/api/v1/collections/%s/move", other).as(f.owner).json(fmt.Sprintf(`{"parentId":%q}`, smart)).expect(400)
	f.req("POST", "/api/v1/collections/%s/import", smart).as(f.owner).
		json(`{"format":"ragbag-collection","format_version":1}`).expect(400)

	f.req("PUT", "/api/v1/collections/%s/share", smart).as(f.owner).expect(400)
	// Below a shared collection it is left out, and its address answers like any unknown one
	token := f.req("PUT", "/api/v1/collections/%s/share", f.links).as(f.owner).expect(200).str("token")
	f.req("GET", "/api/v1/shares/%s", token).expect(200).has("children.#", 0).has("bookmarks.#", 3)
	f.req("GET", "/api/v1/shares/%s", token).query("collectionId", smart).expect(404)
}

func TestSmartCollectionsShowEveryMemberOnlyWhatTheyMaySee(t *testing.T) {
	f := newSmartFixture(t)
	member := f.register()
	f.join(member, f.space, "VIEWER", f.owner)
	secret := f.req("POST", "/api/v1/spaces/%s/collections", f.space).as(f.owner).
		json(`{"name":"Secret","restricted":true}`).expect(200).str("id")
	f.add(f.owner, secret, "Hidden cake", "recipe")
	smart := f.smart(f.owner, "Recipes", "#recipe", "").expect(200).str("id")

	f.list(f.owner, smart).has("#", 3)
	f.list(member, smart).has("#", 2)
	// Links in the trash are gone from it as well
	f.req("DELETE", "/api/v1/bookmarks/%s", f.list(member, smart).str("0.id")).as(f.owner).expect(204)
	f.list(member, smart).has("#", 1)
	// Viewers cannot change the search
	f.req("PUT", "/api/v1/collections/%s", smart).as(member).json(`{"name":"Recipes","query":"x"}`).expect(403)
}

func TestSmartCollectionsAreExportedAndImportedAsSearches(t *testing.T) {
	f := newSmartFixture(t)
	f.smart(f.owner, "Recipes", "#recipe", f.links).expect(200)
	file := f.req("GET", "/api/v1/collections/%s/export", f.links).as(f.owner).expect(200)
	file.has("collection.collections.0.query", "#recipe").has("collection.collections.0.links.#", 0)

	target := f.createCollection(f.owner, f.space, "Copy", "")
	f.req("POST", "/api/v1/collections/%s/import", target).as(f.owner).json(file.Body).expect(200).
		has("collectionsCreated", 1).has("bookmarksImported", 3)
	f.req("GET", "/api/v1/spaces/%s/collections", f.space).as(f.owner).expect(200).
		has(`#(name=="Copy").children.0.query`, "#recipe")
}
