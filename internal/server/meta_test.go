package server_test

import (
	"fmt"
	"testing"

	"github.com/Serbroda/ragbag/internal/config"
	"github.com/stretchr/testify/require"
)

func TestSharedCollectionsGetALinkPreview(t *testing.T) {
	f := newShareFixture(t)
	f.req("PUT", "/api/v1/collections/%s", f.shared).as(f.owner).
		json(`{"name":"WoW <Forever> & \"Friends\"","description":"Alles für unseren Server"}`).expect(200)
	token := f.share()

	page := f.req("GET", "/share/%s", token).expect(200).Body
	require.Contains(t, page, `<title>WoW &lt;Forever&gt; &amp; &#34;Friends&#34;</title>`)
	require.Contains(t, page, `<meta property="og:title" content="WoW &lt;Forever&gt; &amp; &#34;Friends&#34;" />`)
	require.Contains(t, page, `content="Alles für unseren Server (1 link · 1 collection · shared with Ragbag)"`)
	require.Contains(t, page, `<meta property="og:image" content="http://`)
	require.Contains(t, page, `/og-image.png" />`)
	require.Contains(t, page, fmt.Sprintf(`<meta property="og:url" content="%s/share/%s" />`, f.url, token))
	require.NotContains(t, page, "Raid notes")

	sub := f.req("GET", "/share/%s/%s", token, f.sub).expect(200).Body
	require.Contains(t, sub, `<meta property="og:title" content="WoW &lt;Forever&gt; &amp; &#34;Friends&#34; › Addons" />`)
	require.Contains(t, sub, `content="1 link · 1 collection · shared with Ragbag"`)

	// Private, revoked and unknown links give nothing away
	for _, path := range []string{"/share/" + token + "/" + f.secret, "/share/unknown"} {
		generic := f.req("GET", "%s", path).expect(200).Body
		require.Contains(t, generic, `<meta property="og:title" content="Ragbag" />`, path)
		require.NotContains(t, generic, "Raid notes")
		require.NotContains(t, generic, "Forever")
	}
	f.req("DELETE", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(204)
	require.NotContains(t, f.req("GET", "/share/%s", token).expect(200).Body, "Forever")
}

func TestOtherPagesGetAGenericPreview(t *testing.T) {
	e := newEnv(t)
	page := e.req("GET", "/").expect(200).Body
	require.Contains(t, page, `<meta property="og:title" content="Ragbag" />`)
	require.Contains(t, page, `<meta name="twitter:card" content="summary_large_image" />`)
	// Behind a proxy that terminates HTTPS the image URL uses https
	behind := e.req("GET", "/login").header("X-Forwarded-Proto", "https").expect(200).Body
	require.Contains(t, behind, `<meta property="og:image" content="https://`)

	invite := e.req("GET", "/invite/some-token").expect(200).Body
	require.Contains(t, invite, "You&#39;re invited to a space on Ragbag")
}

func TestInvitesNameTheInviterAndTheSpaceInTheirPreview(t *testing.T) {
	e := newEnv(t, func(cfg *config.Config) { cfg.PublicURL = "https://ragbag.example.com" })
	owner := e.register()
	space := e.firstSpace(owner)
	e.req("PUT", "/api/v1/spaces/%s", space).as(owner).json(`{"name":"WoW <Forever>","description":"Guides & Tools"}`).expect(200)
	token := e.req("POST", "/api/v1/invites").as(owner).
		json(fmt.Sprintf(`{"targetType":"SPACE","targetId":%q,"role":"VIEWER","maxUses":1}`, space)).expect(201).str("token")

	page := e.req("GET", "/invite/%s", token).expect(200).Body
	require.Contains(t, page, fmt.Sprintf(`<meta property="og:title" content="%s invited you to WoW &lt;Forever&gt;" />`, owner.Username))
	require.Contains(t, page, `<meta property="og:description" content="Guides &amp; Tools · Ragbag is a collaborative link collection`)
	// Addresses from PUBLIC_URL, not from the request's host (internal behind a reverse proxy)
	require.Contains(t, page, fmt.Sprintf(`<meta property="og:url" content="https://ragbag.example.com/invite/%s" />`, token))
	require.Contains(t, page, `<meta property="og:image" content="https://ragbag.example.com/og-image.png" />`)

	// Used up: generic again
	e.req("POST", "/api/v1/invites/%s", token).as(e.register()).expect(200)
	used := e.req("GET", "/invite/%s", token).expect(200).Body
	require.Contains(t, used, "You&#39;re invited to a space on Ragbag")
	require.NotContains(t, used, "Forever")
}
