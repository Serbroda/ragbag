package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const maxOEmbedBytes = 256 << 10

// Some sites do not show a server what a link is about: YouTube answers data center addresses with a consent
// or bot check page, X and TikTok build their pages with scripts. They do answer oEmbed (https://oembed.com),
// a small JSON description of a link meant for previews, so for them it is asked as well.
var oembedProviders = []struct {
	hosts    []string // with their subdomains
	endpoint string   // %s is the escaped link
}{
	{[]string{"youtube.com", "youtu.be"}, "https://www.youtube.com/oembed?format=json&url=%s"},
	{[]string{"vimeo.com"}, "https://vimeo.com/api/oembed.json?url=%s"},
	{[]string{"open.spotify.com"}, "https://open.spotify.com/oembed?url=%s"},
	{[]string{"soundcloud.com"}, "https://soundcloud.com/oembed?format=json&url=%s"},
	{[]string{"tiktok.com"}, "https://www.tiktok.com/oembed?url=%s"},
	{[]string{"dailymotion.com", "dai.ly"}, "https://www.dailymotion.com/services/oembed?format=json&url=%s"},
	{[]string{"flickr.com", "flic.kr"}, "https://www.flickr.com/services/oembed?format=json&url=%s"},
	{[]string{"reddit.com"}, "https://www.reddit.com/oembed?url=%s"},
	{[]string{"x.com", "twitter.com"}, "https://publish.twitter.com/oembed?omit_script=1&url=%s"},
	{[]string{"bsky.app"}, "https://embed.bsky.app/oembed?url=%s"},
}

// oembedEndpoint is where to ask about the link, "" for sites without a known provider.
func oembedEndpoint(link *url.URL) string {
	host := strings.ToLower(strings.TrimSuffix(link.Hostname(), "."))
	for _, provider := range oembedProviders {
		for _, known := range provider.hosts {
			if host == known || strings.HasSuffix(host, "."+known) {
				return fmt.Sprintf(provider.endpoint, url.QueryEscape(link.String()))
			}
		}
	}
	return ""
}

// embed is what a provider tells about a link.
type embed struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ProviderName string `json:"provider_name"`
	ThumbnailURL string `json:"thumbnail_url"`
	HTML         string `json:"html"`
}

// oembed asks the provider; failures (unknown links, private videos) return an empty embed.
func (c *Client) oembed(ctx context.Context, endpoint string) embed {
	res, err := c.Get(ctx, endpoint, maxOEmbedBytes)
	if err != nil {
		return embed{}
	}
	return parseOEmbed(res.Body)
}

func parseOEmbed(body []byte) embed {
	var e embed
	if err := json.Unmarshal(body, &e); err != nil {
		return embed{}
	}
	e.Title, e.AuthorName, e.ProviderName = strings.TrimSpace(e.Title), strings.TrimSpace(e.AuthorName), strings.TrimSpace(e.ProviderName)
	e.ThumbnailURL = resolve(nil, e.ThumbnailURL)
	return e
}

func (e embed) found() bool { return e.Title != "" || e.AuthorName != "" }

// describes tells whether the page is about the embedded thing, and not a consent or bot check page or the
// site's generic page.
func (e embed) describes(page Metadata) bool {
	if !page.Found() || strings.EqualFold(page.Title, e.ProviderName) {
		return false
	}
	return e.Title == "" || strings.Contains(strings.ToLower(page.Title), strings.ToLower(e.Title))
}

// over combines the embed with the page loaded from the link: a page about the same thing has more (the
// description, keywords, a larger image), otherwise the embed replaces it.
func (e embed) over(page Metadata, link *url.URL) Metadata {
	if e.describes(page) {
		if page.OgImage == "" {
			page.OgImage = e.ThumbnailURL
		}
		return page
	}
	text := postText(e.HTML)
	title := e.Title
	if title == "" {
		// Posts (X, Bluesky) have no title, only their text
		title = strings.TrimSpace(e.AuthorName + " on " + e.ProviderName)
	}
	description := text
	if description == "" && e.AuthorName != "" {
		description = e.AuthorName
		if e.ProviderName != "" {
			description += " · " + e.ProviderName
		}
	}
	return Metadata{
		Title:       title,
		Description: description,
		OgImage:     e.ThumbnailURL,
		Favicon:     resolve(link, "/favicon.ico"),
	}
}

// postText is the text of the first paragraph of the embed's HTML: the post itself for X and Bluesky.
func postText(fragment string) string {
	if fragment == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(fragment))
	if err != nil {
		return ""
	}
	var paragraph *html.Node
	var find func(*html.Node)
	find = func(n *html.Node) {
		if paragraph != nil {
			return
		}
		if n.Type == html.ElementNode && n.Data == "p" {
			paragraph = n
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			find(child)
		}
	}
	find(doc)
	if paragraph == nil {
		return ""
	}
	var text strings.Builder
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			text.WriteString(n.Data)
		case n.Type == html.ElementNode && n.Data == "br":
			text.WriteString(" ")
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(paragraph)
	return strings.Join(strings.Fields(text.String()), " ")
}
