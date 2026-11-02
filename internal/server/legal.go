package server

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/labstack/echo/v4"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// Legal pages (imprint, privacy policy, terms of use) are Markdown files the operator puts into DATA_DIR/legal,
// or addresses elsewhere (IMPRINT_URL & co.). Templates are in docs/legal. Files are read on every request, so
// changes need no restart; raw HTML in them is not passed through.

var legalTitles = map[string]string{"imprint": "Imprint", "privacy": "Privacy policy", "terms": "Terms of use"}

const maxLegalBytes = 1 << 20

var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

func (s *Server) legalFile(page string) string {
	return filepath.Join(s.cfg.DataDir, "legal", page+".md")
}

// legalLink is where a page is shown: its configured address, the app's page for a file, or "" without one.
func (s *Server) legalLink(page string) string {
	if address := s.cfg.LegalURLs[page]; address != "" {
		return address
	}
	if info, err := os.Stat(s.legalFile(page)); err == nil && info.Mode().IsRegular() {
		return "/app/legal/" + page
	}
	return ""
}

func (s *Server) GetLegalPage(c echo.Context, page api.GetLegalPageParamsPage) error {
	title, ok := legalTitles[string(page)]
	if !ok {
		return notFound("Unknown page: %s", page)
	}
	file, err := os.Open(s.legalFile(string(page)))
	if errors.Is(err, fs.ErrNotExist) {
		return notFound("This Ragbag has no %s", strings.ToLower(title))
	}
	if err != nil {
		return err
	}
	defer file.Close()
	source, err := io.ReadAll(io.LimitReader(file, maxLegalBytes))
	if err != nil {
		return err
	}
	// The first heading is the title, if the file starts with one
	if first, rest, _ := bytes.Cut(source, []byte("\n")); bytes.HasPrefix(first, []byte("# ")) {
		title = strings.TrimSpace(string(first[2:]))
		source = rest
	}
	var html bytes.Buffer
	if err := markdown.Convert(source, &html); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, api.LegalPageDto{Page: string(page), Title: title, Html: html.String()})
}
