package fetch

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func TestParseRSS(t *testing.T) {
	base, _ := url.Parse("https://news.example/feed")
	feed, err := ParseFeed([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>Example News</title>
<item><title>Older &amp; wiser</title><link>https://news.example/a</link>
	<pubDate>Mon, 09 Nov 2026 08:00:00 +0000</pubDate><category>Classic</category></item>
<item><title><![CDATA[Patch <b>1.2</b>&nbsp;notes]]></title><link>/b</link>
	<pubDate>Mon, 09 Nov 2026 10:30:00 GMT</pubDate><category>WoW: Forever</category></item>
<item><title>Without a link</title></item>
</channel></rss>`), base)
	if err != nil {
		t.Fatal(err)
	}
	if feed.Title != "Example News" || len(feed.Items) != 2 {
		t.Fatalf("unexpected feed: %+v", feed)
	}
	first := feed.Items[0]
	if first.Title != "Patch 1.2 notes" || first.URL != "https://news.example/b" || first.Published == nil ||
		first.Published.Hour() != 10 || first.Categories[0] != "WoW: Forever" {
		t.Errorf("newest item first, HTML stripped, link resolved: %+v", first)
	}
	if feed.Items[1].Title != "Older & wiser" {
		t.Errorf("entities decoded: %q", feed.Items[1].Title)
	}
}

func TestParseAtom(t *testing.T) {
	feed, err := ParseFeed([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><title type="text">Blog</title>
<entry><title type="html">Hello &lt;em&gt;world&lt;/em&gt;</title>
	<link rel="replies" href="https://blog.example/1#comments"/><link href="https://blog.example/1"/>
	<updated>2026-11-08T12:00:00Z</updated><category term="news"/></entry>
</feed>`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Items) != 1 {
		t.Fatalf("unexpected feed: %+v", feed)
	}
	item := feed.Items[0]
	if item.Title != "Hello world" || item.URL != "https://blog.example/1" || item.Published == nil ||
		item.Categories[0] != "news" {
		t.Errorf("unexpected item: %+v", item)
	}
}

func TestParseFeedRefusesOthersAndKeepsCutOffFeeds(t *testing.T) {
	if _, err := ParseFeed([]byte(`<!doctype html><html><head><title>Page</title></head></html>`), nil); err != ErrNotAFeed {
		t.Errorf("an HTML page is not a feed: %v", err)
	}
	if _, err := ParseFeed([]byte(`{"items": []}`), nil); err != ErrNotAFeed {
		t.Errorf("JSON is not a feed: %v", err)
	}
	cut := `<rss><channel><item><title>One</title><link>https://a.example/1</link></item><item><title>Tw`
	feed, err := ParseFeed([]byte(cut), nil)
	if err != nil || len(feed.Items) != 1 {
		t.Errorf("the items before the cut count: %+v %v", feed, err)
	}
	if feed, _ := ParseFeed([]byte(`<rss><channel><item><title>x</title><link>javascript:alert(1)</link></item></channel></rss>`), nil); len(feed.Items) != 0 {
		t.Errorf("only http(s) links: %+v", feed.Items)
	}
}

func TestFeedDiscovery(t *testing.T) {
	base, _ := url.Parse("https://www.example.org/news")
	doc, _ := html.Parse(strings.NewReader(`<html><head>
		<link rel="alternate" hreflang="en" href="/en/news">
		<link rel="alternate" type="application/rss+xml" title="News" href="/news/rss/all">
		<link rel="alternate" type="application/atom+xml" href="/comments.atom">
	</head></html>`))
	if feed := ParseMetadata(doc, base).Feed; feed != "https://www.example.org/news/rss/all" {
		t.Errorf("the first linked feed: %q", feed)
	}
}
