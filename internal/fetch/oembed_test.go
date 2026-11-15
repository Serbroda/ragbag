package fetch

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustURL(t *testing.T, value string) *url.URL {
	parsed, err := url.Parse(value)
	require.NoError(t, err)
	return parsed
}

func TestKnownSitesAreAskedViaOEmbed(t *testing.T) {
	for link, endpoint := range map[string]string{
		"https://www.youtube.com/watch?v=abc&t=1": "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fwww.youtube.com%2Fwatch%3Fv%3Dabc%26t%3D1",
		"https://youtu.be/abc":                    "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fyoutu.be%2Fabc",
		"https://music.youtube.com/watch?v=abc":   "https://www.youtube.com/oembed?format=json&url=https%3A%2F%2Fmusic.youtube.com%2Fwatch%3Fv%3Dabc",
		"https://x.com/jack/status/20":            "https://publish.twitter.com/oembed?omit_script=1&url=https%3A%2F%2Fx.com%2Fjack%2Fstatus%2F20",
	} {
		require.Equal(t, endpoint, oembedEndpoint(mustURL(t, link)), link)
	}
	for _, link := range []string{"https://notyoutube.com/watch?v=abc", "https://youtube.com.evil.example/", "https://example.org/"} {
		require.Empty(t, oembedEndpoint(mustURL(t, link)), link)
	}
}

func TestParseOEmbed(t *testing.T) {
	e := parseOEmbed([]byte(`{"title":" Never Gonna Give You Up ","author_name":"Rick Astley","provider_name":"YouTube",
		"thumbnail_url":"https://i.ytimg.com/vi/abc/hqdefault.jpg","html":"<iframe src=\"https://www.youtube.com/embed/abc\"></iframe>"}`))
	require.True(t, e.found())
	require.Equal(t, "Never Gonna Give You Up", e.Title)
	require.Equal(t, "https://i.ytimg.com/vi/abc/hqdefault.jpg", e.ThumbnailURL)

	require.False(t, parseOEmbed([]byte(`not json`)).found())
	require.Empty(t, parseOEmbed([]byte(`{"title":"x","thumbnail_url":"javascript:alert(1)"}`)).ThumbnailURL)
}

func TestTheEmbedReplacesConsentAndBotPages(t *testing.T) {
	link := mustURL(t, "https://www.youtube.com/watch?v=abc")
	video := embed{Title: "Never Gonna Give You Up", AuthorName: "Rick Astley", ProviderName: "YouTube",
		ThumbnailURL: "https://i.ytimg.com/vi/abc/hqdefault.jpg"}

	// A consent page, or YouTube's generic page
	for _, page := range []Metadata{
		{Title: "Bevor Sie zu YouTube weitergehen", Favicon: "https://consent.youtube.com/favicon.ico"},
		{Title: "YouTube", Description: "Enjoy the videos and music you love"},
		{},
	} {
		require.Equal(t, Metadata{
			Title:       "Never Gonna Give You Up",
			Description: "Rick Astley · YouTube",
			OgImage:     "https://i.ytimg.com/vi/abc/hqdefault.jpg",
			Favicon:     "https://www.youtube.com/favicon.ico",
		}, video.over(page, link), page.Title)
	}

	// The real page has more: its description, keywords and a larger image
	page := Metadata{Title: "Never Gonna Give You Up", Description: "The official video", Keywords: []string{"rick"},
		OgImage: "https://i.ytimg.com/vi/abc/maxresdefault.jpg"}
	require.Equal(t, page, video.over(page, link))
	page.OgImage = ""
	require.Equal(t, video.ThumbnailURL, video.over(page, link).OgImage)
}

func TestPostsAreDescribedByTheirText(t *testing.T) {
	post := parseOEmbed([]byte(`{"author_name":"jack","provider_name":"X","html":"<blockquote class=\"twitter-tweet\"><p lang=\"en\">just setting up<br>my <a href=\"#\">twttr</a></p>&mdash; jack (@jack) <a href=\"https://x.com/jack/status/20\">March 21, 2006</a></blockquote>"}`))
	link := mustURL(t, "https://x.com/jack/status/20")

	metadata := post.over(Metadata{Title: "X"}, link)
	require.Equal(t, "jack on X", metadata.Title)
	require.Equal(t, "just setting up my twttr", metadata.Description)

	// A page that shows the post itself stays
	page := Metadata{Title: `jack on X: "just setting up my twttr" / X`, Description: "just setting up my twttr"}
	require.Equal(t, page, post.over(page, link))
}
