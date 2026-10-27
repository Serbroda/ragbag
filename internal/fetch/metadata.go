package fetch

import (
	"bytes"
	"context"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/charset"
)

const (
	maxPageBytes = 2 << 20
	maxKeywords  = 8
)

type Metadata struct {
	Title       string
	Description string
	OgImage     string
	Favicon     string
	Canonical   string
	Keywords    []string
}

// Found tells whether the page had anything worth showing.
func (m Metadata) Found() bool { return m.Title != "" || m.Description != "" }

// Metadata loads a page and extracts title, description, preview image, favicon and keywords. Failures
// return empty metadata: links are saved anyway.
func (c *Client) Metadata(ctx context.Context, pageURL string) Metadata {
	res, err := c.Get(ctx, pageURL, maxPageBytes)
	if err != nil {
		return Metadata{}
	}
	reader, err := charset.NewReader(bytes.NewReader(res.Body), res.ContentType)
	if err != nil {
		return Metadata{}
	}
	doc, err := html.Parse(reader)
	if err != nil {
		return Metadata{}
	}
	return ParseMetadata(doc, res.URL)
}

// ParseMetadata reads the head of a parsed page; relative URLs are resolved against base.
func ParseMetadata(doc *html.Node, base *url.URL) Metadata {
	var (
		title, ogTitle, description, ogDescription, ogImage, canonical, icon, keywords string
	)
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if title == "" && n.FirstChild != nil {
					title = n.FirstChild.Data
				}
			case "meta":
				name, property, content := strings.ToLower(attr(n, "name")), strings.ToLower(attr(n, "property")), attr(n, "content")
				switch {
				case property == "og:title" && ogTitle == "":
					ogTitle = content
				case property == "og:description" && ogDescription == "":
					ogDescription = content
				case property == "og:image" && ogImage == "":
					ogImage = content
				case name == "description" && description == "":
					description = content
				case name == "keywords" && keywords == "":
					keywords = content
				}
			case "link":
				rel := " " + strings.ToLower(attr(n, "rel")) + " "
				switch {
				case strings.Contains(rel, " canonical ") && canonical == "":
					canonical = attr(n, "href")
				case strings.Contains(rel, " icon ") && icon == "":
					icon = attr(n, "href")
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	metadata := Metadata{
		// Open Graph values are usually cleaner than <title> (no " | Site name" suffix)
		Title:       firstNonBlank(ogTitle, title),
		Description: firstNonBlank(description, ogDescription),
		OgImage:     resolve(base, ogImage),
		Canonical:   resolve(base, canonical),
		Favicon:     resolve(base, icon),
		Keywords:    parseKeywords(keywords),
	}
	if metadata.Favicon == "" {
		metadata.Favicon = resolve(base, "/favicon.ico")
	}
	return metadata
}

func parseKeywords(content string) []string {
	var result []string
	seen := map[string]bool{}
	for _, part := range strings.Split(content, ",") {
		keyword := strings.ToLower(strings.TrimSpace(part))
		if keyword == "" || len(keyword) > 40 || seen[keyword] {
			continue
		}
		seen[keyword] = true
		result = append(result, keyword)
		if len(result) == maxKeywords {
			break
		}
	}
	return result
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// resolve makes value absolute; only http(s) results are kept.
func resolve(base *url.URL, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	ref, err := url.Parse(value)
	if err != nil {
		return ""
	}
	if !ref.IsAbs() {
		if base == nil {
			return ""
		}
		ref = base.ResolveReference(ref)
	}
	if ref.Scheme != "http" && ref.Scheme != "https" {
		return ""
	}
	return ref.String()
}
