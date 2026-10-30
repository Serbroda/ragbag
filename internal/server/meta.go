package server

import (
	"bytes"
	"fmt"
	"html"
	"strings"

	"github.com/labstack/echo/v4"
)

// Link previews: WhatsApp, Signal, Discord & co. read Open Graph tags from the HTML without running JavaScript,
// so the server writes them into index.html for the page that is requested.

const (
	defaultDescription = "Collect, organize and share links with your people. Self-hosted, like Discord for bookmarks."
	// ui/v1/public/og-image.png, 1200×630
	previewImage = "/app/og-image.png"
)

type pageMeta struct {
	title, description string
}

// pageMeta describes the page at /app/<name>. Shared collections show their name and content; everything that
// would need a login stays generic, and a link that is unknown, revoked or private gives nothing away.
func (s *Server) pageMeta(c echo.Context, name string) pageMeta {
	meta := pageMeta{title: "Ragbag", description: defaultDescription}
	parts := strings.Split(strings.Trim(name, "/"), "/")
	switch {
	case len(parts) >= 2 && parts[0] == "share":
		var collectionID *string
		if len(parts) >= 3 && parts[2] != "" {
			collectionID = &parts[2]
		}
		shared, err := s.sharedCollection(c, parts[1], collectionID)
		if err != nil {
			return meta
		}
		names := make([]string, 0, len(shared.Path))
		for _, crumb := range shared.Path {
			names = append(names, crumb.Name)
		}
		meta.title = strings.Join(names, " › ")
		meta.description = fmt.Sprintf("%s · shared with Ragbag", counted(len(shared.Bookmarks), "link"))
		if len(shared.Children) > 0 {
			meta.description = fmt.Sprintf("%s · %s · shared with Ragbag", counted(len(shared.Bookmarks), "link"),
				counted(len(shared.Children), "collection"))
		}
		if shared.Description != nil && strings.TrimSpace(*shared.Description) != "" {
			meta.description = strings.TrimSpace(*shared.Description) + " (" + meta.description + ")"
		}
	case len(parts) >= 2 && parts[0] == "invite":
		meta.title = "You're invited to a space on Ragbag"
		meta.description = "Open the link to join and share links together."
	}
	return meta
}

func counted(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// withMeta puts the title and preview tags of the page into index.html.
func (s *Server) withMeta(c echo.Context, name string, index []byte) []byte {
	meta := s.pageMeta(c, name)
	base := c.Scheme() + "://" + c.Request().Host
	tags := []string{
		"<title>" + html.EscapeString(meta.title) + "</title>",
		`<meta name="description" content="` + html.EscapeString(meta.description) + `" />`,
		`<meta property="og:site_name" content="Ragbag" />`,
		`<meta property="og:type" content="website" />`,
		`<meta property="og:title" content="` + html.EscapeString(meta.title) + `" />`,
		`<meta property="og:description" content="` + html.EscapeString(meta.description) + `" />`,
		`<meta property="og:url" content="` + html.EscapeString(base+c.Request().URL.Path) + `" />`,
		`<meta property="og:image" content="` + html.EscapeString(base+previewImage) + `" />`,
		`<meta property="og:image:width" content="1200" />`,
		`<meta property="og:image:height" content="630" />`,
		`<meta property="og:image:alt" content="Ragbag" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
	}
	block := []byte(strings.Join(tags, "\n    "))
	if bytes.Contains(index, []byte("<title>Ragbag</title>")) {
		return bytes.Replace(index, []byte("<title>Ragbag</title>"), block, 1)
	}
	return bytes.Replace(index, []byte("</head>"), append(block, []byte("\n  </head>")...), 1)
}
