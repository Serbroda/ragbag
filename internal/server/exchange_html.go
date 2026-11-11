package server

import (
	"bytes"
	"io"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Browsers and most bookmark services (Chrome, Firefox, Safari, Edge, Pocket, Raindrop, Pinboard, Linkding)
// export the "Netscape bookmark file": HTML nobody ever specified, with folders as <DT><H3>name</H3><DL>…</DL>
// and links as <DT><A HREF ADD_DATE TAGS>title</A>, optionally followed by <DD>description. None of its tags are
// closed reliably, so the file is read token by token instead of as a document.

// bookmarkFolder is a folder while reading; exchangeNode holds its children by value, which pointers into a
// growing tree do not survive.
type bookmarkFolder struct {
	name        string
	description *string
	links       []exchangeLink
	folders     []*bookmarkFolder
}

func (f *bookmarkFolder) node() exchangeNode {
	node := exchangeNode{Name: f.name, Description: f.description, Links: f.links, Collections: []exchangeNode{}}
	for _, child := range f.folders {
		node.Collections = append(node.Collections, child.node())
	}
	return node
}

// parseBookmarksHTML converts a browser's bookmark export into an exchange file: folders become sub-collections
// of the import target, links outside of any folder go into the target itself.
func parseBookmarksHTML(raw []byte) (*exchangeFile, error) {
	root := &bookmarkFolder{}
	// One entry per open <DL>; a list that follows a folder's heading holds that folder's content
	open := []*bookmarkFolder{root}
	var pending *bookmarkFolder
	// What a following <DD> describes
	var describe func(text string)
	found := false

	tokens := html.NewTokenizer(bytes.NewReader(raw))
	for {
		kind := tokens.Next()
		if kind == html.ErrorToken {
			if tokens.Err() != io.EOF {
				return nil, badRequest("The bookmarks file cannot be read")
			}
			break
		}
		current := open[len(open)-1]
		switch kind {
		case html.StartTagToken:
			token := tokens.Token()
			switch token.Data {
			case "dl":
				if pending != nil {
					current = pending
					pending = nil
				}
				if len(open) > maxImportDepth+1 {
					return nil, badRequest("Collections are nested deeper than %d levels", maxImportDepth)
				}
				open = append(open, current)
			case "h3":
				folder := &bookmarkFolder{name: textUntil(tokens, "h3")}
				current.folders = append(current.folders, folder)
				pending = folder
				describe = func(text string) { folder.description = &text }
				found = true
			case "a":
				link := exchangeLink{URL: attribute(token, "href"), CreatedAt: bookmarkDate(token)}
				if tags := attribute(token, "tags"); tags != "" {
					link.Tags = strings.Split(tags, ",")
				}
				if title := textUntil(tokens, "a"); title != "" {
					link.Title = &title
				}
				current.links = append(current.links, link)
				index := len(current.links) - 1
				describe = func(text string) { current.links[index].Description = &text }
				found = true
			case "dd":
				if describe != nil {
					if text := textUntil(tokens, ""); text != "" {
						describe(text)
					}
					describe = nil
				}
			}
		case html.EndTagToken:
			if token := tokens.Token(); token.Data == "dl" && len(open) > 1 {
				open = open[:len(open)-1]
				pending = nil
			}
		}
	}
	if !found {
		return nil, badRequest("Not a bookmarks file: it contains no links")
	}
	node := root.node()
	return &exchangeFile{Format: exchangeFormat, FormatVersion: exchangeFormatVersion, Collection: &node}, nil
}

// textUntil collects the text up to the closing tag, or up to the next tag of any kind if none is named (<DD> is
// never closed).
func textUntil(tokens *html.Tokenizer, closing string) string {
	var text strings.Builder
	for {
		switch tokens.Next() {
		case html.ErrorToken:
			return strings.TrimSpace(text.String())
		case html.TextToken:
			text.Write(tokens.Text())
			if closing == "" {
				return strings.TrimSpace(text.String())
			}
		case html.EndTagToken:
			if name, _ := tokens.TagName(); closing == "" || string(name) == closing {
				return strings.TrimSpace(text.String())
			}
		default:
			if closing == "" {
				return strings.TrimSpace(text.String())
			}
		}
	}
}

func attribute(token html.Token, name string) string {
	for _, attr := range token.Attr {
		if attr.Key == name {
			return strings.TrimSpace(attr.Val)
		}
	}
	return ""
}

// bookmarkDate reads when the link was saved: Unix seconds in ADD_DATE (Pocket: TIME_ADDED), which some
// services write in milli- or microseconds. Dates that make no sense are left out, the import then counts on.
func bookmarkDate(token html.Token) *time.Time {
	value := attribute(token, "add_date")
	if value == "" {
		value = attribute(token, "time_added")
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds <= 0 {
		return nil
	}
	for seconds > 100_000_000_000 {
		seconds /= 1000
	}
	date := time.Unix(seconds, 0).UTC()
	if date.After(time.Now()) {
		return nil
	}
	return &date
}
