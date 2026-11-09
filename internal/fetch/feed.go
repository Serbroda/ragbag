package fetch

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	maxFeedBytes = 2 << 20
	maxFeedItems = 50
)

var (
	ErrNotAFeed = errors.New("not an RSS or Atom feed")
	ErrNoFeed   = errors.New("no feed found")
)

type Feed struct {
	Title string
	Items []FeedItem
}

type FeedItem struct {
	Title      string
	URL        string
	Published  *time.Time
	Categories []string
}

// Feed loads an RSS (0.9x, 1.0, 2.0) or Atom feed. A feed cut off at the size limit keeps the items before.
func (c *Client) Feed(ctx context.Context, feedURL string) (*Feed, error) {
	res, err := c.GetStart(ctx, feedURL, maxFeedBytes)
	if err != nil {
		return nil, err
	}
	return ParseFeed(res.Body, res.URL)
}

// FindFeed takes a feed or a page linking to one (<link rel="alternate" type="application/rss+xml">) and
// returns the feed's address along with the loaded feed.
func (c *Client) FindFeed(ctx context.Context, address string) (string, *Feed, error) {
	res, err := c.GetStart(ctx, address, maxFeedBytes)
	if err != nil {
		return "", nil, err
	}
	if feed, err := ParseFeed(res.Body, res.URL); err == nil {
		return address, feed, nil
	}
	reader, err := charset.NewReader(bytes.NewReader(res.Body), res.ContentType)
	if err != nil {
		return "", nil, ErrNoFeed
	}
	doc, err := html.Parse(reader)
	if err != nil {
		return "", nil, ErrNoFeed
	}
	found := ParseMetadata(doc, res.URL).Feed
	if found == "" {
		return "", nil, ErrNoFeed
	}
	feed, err := c.Feed(ctx, found)
	if err != nil {
		return "", nil, err
	}
	return found, feed, nil
}

type rawLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Text string `xml:",chardata"`
}

type rawCategory struct {
	Term string `xml:"term,attr"`
	Text string `xml:",chardata"`
}

// rawItem is an RSS <item> or an Atom <entry>; tags without namespace match any (dc:date, atom:link)
type rawItem struct {
	Title      string        `xml:"title"`
	Links      []rawLink     `xml:"link"`
	GUID       string        `xml:"guid"`
	PubDate    string        `xml:"pubDate"`
	Date       string        `xml:"date"`
	Published  string        `xml:"published"`
	Updated    string        `xml:"updated"`
	Categories []rawCategory `xml:"category"`
}

// ParseFeed reads RSS or Atom; relative links are resolved against base. Anything else is ErrNotAFeed.
func ParseFeed(body []byte, base *url.URL) (*Feed, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = charset.NewReaderLabel
	// Feeds in the wild use HTML entities (no AutoClose: RSS has a <link> element with content)
	decoder.Strict = false
	decoder.Entity = xml.HTMLEntity

	feed := &Feed{}
	root := ""
	for len(feed.Items) < maxFeedItems {
		token, err := decoder.Token()
		if err != nil {
			// The end, or a feed cut off at the size limit: what was read so far counts
			break
		}
		switch element := token.(type) {
		case xml.StartElement:
			name := strings.ToLower(element.Name.Local)
			if root == "" {
				if name != "rss" && name != "feed" && name != "rdf" {
					return nil, ErrNotAFeed
				}
				root = name
				continue
			}
			switch {
			case name == "item" || name == "entry":
				var raw rawItem
				if err := decoder.DecodeElement(&raw, &element); err != nil {
					return feed, nil
				}
				if item, ok := feedItem(raw, base); ok {
					feed.Items = append(feed.Items, item)
				}
			case name == "title" && feed.Title == "" && len(feed.Items) == 0:
				var title string
				if err := decoder.DecodeElement(&title, &element); err == nil {
					feed.Title = plainText(title)
				}
			}
		}
	}
	if root == "" {
		return nil, ErrNotAFeed
	}
	// Most feeds are newest first already; the others are sorted when every item has a date
	dated := true
	for _, item := range feed.Items {
		dated = dated && item.Published != nil
	}
	if dated {
		sort.SliceStable(feed.Items, func(i, j int) bool { return feed.Items[i].Published.After(*feed.Items[j].Published) })
	}
	return feed, nil
}

func feedItem(raw rawItem, base *url.URL) (FeedItem, bool) {
	title := plainText(raw.Title)
	link := ""
	for _, candidate := range raw.Links {
		// Atom: the alternate (or rel-less) link's href; RSS: the element's text
		rel := strings.ToLower(strings.TrimSpace(candidate.Rel))
		if rel != "" && rel != "alternate" {
			continue
		}
		link = resolve(base, firstNonBlank(candidate.Href, candidate.Text))
		if link != "" {
			break
		}
	}
	if link == "" && strings.Contains(raw.GUID, "://") {
		link = resolve(base, raw.GUID)
	}
	if title == "" || link == "" {
		return FeedItem{}, false
	}
	item := FeedItem{Title: title, URL: link, Published: parseFeedTime(raw.PubDate, raw.Published, raw.Date, raw.Updated)}
	for _, category := range raw.Categories {
		if value := plainText(firstNonBlank(category.Term, category.Text)); value != "" {
			item.Categories = append(item.Categories, value)
		}
	}
	return item, true
}

var feedTimeLayouts = []string{
	time.RFC1123Z, time.RFC1123, time.RFC3339, time.RFC3339Nano,
	"Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST", "Mon, 2 Jan 2006 15:04 -0700",
	"Mon, 02 Jan 2006 15:04 -0700", "2 Jan 2006 15:04:05 -0700", "02 Jan 2006 15:04:05 -0700",
	"2006-01-02T15:04:05Z0700", "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02",
}

// parseFeedTime takes the first of the values that parses.
func parseFeedTime(values ...string) *time.Time {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		for _, layout := range feedTimeLayouts {
			if parsed, err := time.Parse(layout, value); err == nil {
				parsed = parsed.UTC()
				return &parsed
			}
		}
	}
	return nil
}

// plainText turns a title that may carry (escaped) HTML into text with collapsed whitespace.
func plainText(value string) string {
	if strings.ContainsAny(value, "<&") {
		var text strings.Builder
		tokenizer := html.NewTokenizer(strings.NewReader(value))
		for {
			kind := tokenizer.Next()
			if kind == html.ErrorToken {
				break
			}
			if kind == html.TextToken {
				text.Write(tokenizer.Text())
			}
		}
		value = text.String()
	}
	return strings.Join(strings.Fields(value), " ")
}
