package server_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSharedCollectionsGetALinkPreview(t *testing.T) {
	f := newShareFixture(t)
	f.req("PUT", "/api/v1/collections/%s", f.shared).as(f.owner).
		json(`{"name":"WoW <Forever> & \"Friends\"","description":"Alles für unseren Server"}`).expect(200)
	token := f.share()

	page := f.req("GET", "/app/share/%s", token).expect(200).Body
	require.Contains(t, page, `<title>WoW &lt;Forever&gt; &amp; &#34;Friends&#34;</title>`)
	require.Contains(t, page, `<meta property="og:title" content="WoW &lt;Forever&gt; &amp; &#34;Friends&#34;" />`)
	require.Contains(t, page, `content="Alles für unseren Server (1 link · 1 collection · shared with Ragbag)"`)
	require.Contains(t, page, `<meta property="og:image" content="http://`)
	require.Contains(t, page, `/app/og-image.png" />`)
	require.Contains(t, page, fmt.Sprintf(`<meta property="og:url" content="%s/app/share/%s" />`, f.url, token))
	require.NotContains(t, page, "Raid notes")

	sub := f.req("GET", "/app/share/%s/%s", token, f.sub).expect(200).Body
	require.Contains(t, sub, `<meta property="og:title" content="WoW &lt;Forever&gt; &amp; &#34;Friends&#34; › Addons" />`)
	require.Contains(t, sub, `content="1 link · 1 collection · shared with Ragbag"`)

	// Private, revoked and unknown links give nothing away
	for _, path := range []string{"/app/share/" + token + "/" + f.secret, "/app/share/unknown"} {
		generic := f.req("GET", "%s", path).expect(200).Body
		require.Contains(t, generic, `<meta property="og:title" content="Ragbag" />`, path)
		require.NotContains(t, generic, "Raid notes")
		require.NotContains(t, generic, "Forever")
	}
	f.req("DELETE", "/api/v1/collections/%s/share", f.shared).as(f.owner).expect(204)
	require.NotContains(t, f.req("GET", "/app/share/%s", token).expect(200).Body, "Forever")
}

func TestOtherPagesGetAGenericPreview(t *testing.T) {
	e := newEnv(t)
	page := e.req("GET", "/app/").expect(200).Body
	require.Contains(t, page, `<meta property="og:title" content="Ragbag" />`)
	require.Contains(t, page, `<meta name="twitter:card" content="summary_large_image" />`)
	// Behind a proxy that terminates HTTPS the image URL uses https
	behind := e.req("GET", "/app/login").header("X-Forwarded-Proto", "https").expect(200).Body
	require.Contains(t, behind, `<meta property="og:image" content="https://`)

	invite := e.req("GET", "/app/invite/some-token").expect(200).Body
	require.Contains(t, invite, "You&#39;re invited to a space on Ragbag")
}
