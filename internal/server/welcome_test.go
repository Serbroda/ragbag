package server_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheFirstSpaceOfANewAccountWelcomesItsOwner(t *testing.T) {
	e := newEnv(t)
	alice := e.register()
	space := e.firstSpace(alice)
	e.req("GET", "/api/v1/spaces").as(alice).expect(200).has("0.welcomed", false)
	e.req("GET", "/api/v1/spaces/%s", space).as(alice).expect(200).has("welcomed", false).has("role", "OWNER")

	e.req("POST", "/api/v1/spaces/%s/welcomed", space).as(alice).expect(204)
	e.req("POST", "/api/v1/spaces/%s/welcomed", space).as(alice).expect(204)
	e.req("GET", "/api/v1/spaces/%s", space).as(alice).expect(200).has("welcomed", true)

	// Spaces one creates oneself need no introduction
	e.req("POST", "/api/v1/spaces").as(alice).json(`{"name":"Second"}`).expect(201).has("welcomed", true)
	for _, s := range e.req("GET", "/api/v1/spaces").as(alice).expect(200).get("#.welcomed").Array() {
		require.True(t, s.Bool())
	}

	// Only members can mark it
	bob := e.register()
	e.req("POST", "/api/v1/spaces/%s/welcomed", space).as(bob).expect(403)
}

func TestNewMembersSeeTheWelcomeMessageOnce(t *testing.T) {
	e := newEnv(t)
	owner, member := e.register(), e.register()
	space := e.firstSpace(owner)
	update := func(s *session, body string) *request {
		return e.req("PUT", "/api/v1/spaces/%s", space).as(s).json(body)
	}

	update(owner, `{"name":"Guild","welcomeMessage":"# Hi\n\nRead the **rules**: [here](https://example.com)"}`).expect(200).
		has("welcomeMessage", "# Hi\n\nRead the **rules**: [here](https://example.com)")
	e.join(member, space, "CONTRIBUTOR", owner)
	res := e.req("GET", "/api/v1/spaces/%s", space).as(member).expect(200).has("welcomed", false)
	require.Contains(t, res.str("welcomeHtml"), "<strong>rules</strong>")
	require.Contains(t, res.str("welcomeHtml"), `<a href="https://example.com">here</a>`)
	e.req("POST", "/api/v1/spaces/%s/welcomed", space).as(member).expect(204)
	e.req("GET", "/api/v1/spaces/%s", space).as(member).expect(200).has("welcomed", true)

	// Contributors cannot change it; leaving it out keeps it, empty removes it
	update(member, `{"name":"Guild","welcomeMessage":"mine"}`).expect(403)
	update(owner, `{"name":"Guild renamed"}`).expect(200).has("welcomeMessage", "# Hi\n\nRead the **rules**: [here](https://example.com)")
	update(owner, fmt.Sprintf(`{"name":"Guild","welcomeMessage":%q}`, strings.Repeat("x", 10001))).expect(400)
	res = update(owner, `{"name":"Guild","welcomeMessage":"  "}`).expect(200)
	require.False(t, res.get("welcomeMessage").Exists(), res.Body)
	require.False(t, res.get("welcomeHtml").Exists(), res.Body)
}

// Members write the message, so it must not smuggle HTML or scripts into the page of others.
func TestTheWelcomeMessageCarriesNoHTML(t *testing.T) {
	e := newEnv(t)
	owner := e.register()
	space := e.firstSpace(owner)
	html := e.req("PUT", "/api/v1/spaces/%s", space).as(owner).
		json(`{"name":"S","welcomeMessage":"<script>alert(1)</script>\n\n<img src=x onerror=alert(1)>\n\n[click](javascript:alert(1))"}`).
		expect(200).str("welcomeHtml")
	require.NotContains(t, html, "<script")
	require.NotContains(t, html, "<img")
	require.NotContains(t, html, "javascript:")
}
