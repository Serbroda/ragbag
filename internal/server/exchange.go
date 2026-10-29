package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/fetch"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// Exchange format of a collection export: the collection with its links and sub-collections, importable into
// any collection of any installation (also files exported by the former Java version).
//
// format_version is raised on every incompatible change; parseExchange converts older files step by step and
// rejects newer ones. Member lists of private collections are installation specific and not exported: a private
// collection is imported as private to the importing user.
const (
	exchangeFormat        = "ragbag-collection"
	exchangeFormatVersion = 1

	maxImportCollections = 1_000
	maxImportLinks       = 20_000
	maxImportDepth       = 32
)

type exchangeFile struct {
	Format        string        `json:"format"`
	FormatVersion int           `json:"format_version"`
	AppVersion    string        `json:"app_version"`
	ExportedAt    time.Time     `json:"exported_at"`
	Collection    *exchangeNode `json:"collection"`
}

type exchangeNode struct {
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Private     bool           `json:"private"`
	Links       []exchangeLink `json:"links"`
	Collections []exchangeNode `json:"collections"`
}

type exchangeLink struct {
	URL         string     `json:"url"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Tags        []string   `json:"tags"`
	OgImage     *string    `json:"og_image"`
	Favicon     *string    `json:"favicon"`
	Canonical   *string    `json:"canonical"`
	CreatedAt   *time.Time `json:"created_at"`
}

// ExportCollection returns the collection as the user sees it: private sub-collections of others are left out.
func (s *Server) ExportCollection(c echo.Context, collectionID api.ID) error {
	collection, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermImportExport)
	if err != nil {
		return err
	}
	collections, err := s.store.CollectionsBySpace(ctxOf(c), collection.SpaceID)
	if err != nil {
		return err
	}
	hidden, err := s.hiddenCollections(ctxOf(c), s.store, collection.SpaceID, currentUser(c))
	if err != nil {
		return err
	}
	children := map[string][]store.Collection{}
	for _, child := range collections {
		if child.ParentID != nil && !hidden[child.ID] {
			children[*child.ParentID] = append(children[*child.ParentID], child)
		}
	}
	root, err := s.exportNode(c, collection, children)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, exchangeFile{
		Format: exchangeFormat, FormatVersion: exchangeFormatVersion, AppVersion: s.version,
		ExportedAt: time.Now().UTC(), Collection: root,
	})
}

func (s *Server) exportNode(c echo.Context, collection *store.Collection, children map[string][]store.Collection) (*exchangeNode, error) {
	bookmarks, err := s.store.BookmarksInCollection(ctxOf(c), collection.ID)
	if err != nil {
		return nil, err
	}
	node := &exchangeNode{Name: collection.Name, Description: collection.Description, Private: collection.Restricted,
		Links: []exchangeLink{}, Collections: []exchangeNode{}}
	// Oldest first: the import creates links in file order, so the newest one stays the newest
	for i := len(bookmarks) - 1; i >= 0; i-- {
		b := bookmarks[i]
		createdAt := b.CreatedAt
		node.Links = append(node.Links, exchangeLink{URL: b.URL, Title: b.Title, Description: b.Description,
			Tags: b.Tags, OgImage: b.OgImage, Favicon: b.Favicon, Canonical: b.Canonical, CreatedAt: &createdAt})
	}
	list := children[collection.ID]
	sort.Slice(list, func(i, j int) bool { return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name) })
	for i := range list {
		child, err := s.exportNode(c, &list[i], children)
		if err != nil {
			return nil, err
		}
		node.Collections = append(node.Collections, *child)
	}
	return node, nil
}

// parseExchange reads a file of any known format version and converts it to the current one, like migrations.
func parseExchange(raw []byte) (*exchangeFile, error) {
	var header struct {
		Format        string `json:"format"`
		FormatVersion *int   `json:"format_version"`
	}
	if err := json.Unmarshal(raw, &header); err != nil || header.Format != exchangeFormat {
		return nil, badRequest("Not a Ragbag collection export")
	}
	if header.FormatVersion == nil || *header.FormatVersion < 1 {
		return nil, badRequest("The export file has no valid format_version")
	}
	if *header.FormatVersion > exchangeFormatVersion {
		return nil, badRequest("The file was exported by a newer version of Ragbag (format %d), please update first",
			*header.FormatVersion)
	}
	// Version 1 is the current one; converters for later versions go here (v1 → v2, …)
	var file exchangeFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, badRequest("The export file is invalid: %s", err.Error())
	}
	return &file, nil
}

type importCounter struct {
	collections, links                          int
	created, merged, imported, skipped, invalid int
	start                                       time.Time
}

// ImportCollection adds the file's links to the collection and merges its sub-collections with existing ones of
// the same name. As preview everything runs and is rolled back, so the result tells exactly what would happen.
func (s *Server) ImportCollection(c echo.Context, collectionID api.ID, params api.ImportCollectionParams) error {
	target, _, err := s.requireCollection(ctxOf(c), s.store, collectionID, currentUser(c), PermImportExport)
	if err != nil {
		return err
	}
	var body json.RawMessage
	if err := bind(c, &body); err != nil {
		return err
	}
	file, err := parseExchange(body)
	if err != nil {
		return err
	}
	preview := params.Preview != nil && *params.Preview
	counter := &importCounter{start: time.Now().UTC()}
	if file.Collection != nil {
		if err := checkImportLimits(file.Collection, 0, counter); err != nil {
			return err
		}
	}
	run := s.store.InTx
	if preview {
		run = s.store.DryRun
	}
	err = run(ctxOf(c), func(tx *store.Store) error {
		if file.Collection == nil {
			return nil
		}
		hidden, err := s.hiddenCollections(ctxOf(c), tx, target.SpaceID, currentUser(c))
		if err != nil {
			return err
		}
		return s.importNode(c, tx, target, file.Collection, hidden, counter)
	})
	if err != nil {
		return err
	}
	if !preview {
		audit(c, slog.LevelInfo, "collection imported", "collection", collectionID, "links", counter.imported,
			"skipped", counter.skipped, "invalid", counter.invalid)
	}
	appVersion := file.AppVersion
	return c.JSON(http.StatusOK, api.ImportResultDto{
		Preview: preview, FormatVersion: file.FormatVersion, AppVersion: &appVersion,
		CollectionsCreated: counter.created, CollectionsMerged: counter.merged,
		BookmarksImported: counter.imported, BookmarksSkipped: counter.skipped, BookmarksInvalid: counter.invalid,
	})
}

func checkImportLimits(node *exchangeNode, depth int, counter *importCounter) error {
	if depth > maxImportDepth {
		return badRequest("Collections are nested deeper than %d levels", maxImportDepth)
	}
	counter.collections++
	counter.links += len(node.Links)
	if counter.collections > maxImportCollections || counter.links > maxImportLinks {
		return badRequest("An import can hold at most %d collections and %d links", maxImportCollections, maxImportLinks)
	}
	for i := range node.Collections {
		if err := checkImportLimits(&node.Collections[i], depth+1, counter); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) importNode(c echo.Context, tx *store.Store, target *store.Collection, node *exchangeNode,
	hidden map[string]bool, counter *importCounter) error {
	existing, err := tx.URLsInCollection(ctxOf(c), target.ID)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, url := range existing {
		known[url] = true
	}
	for _, link := range node.Links {
		url, err := fetch.NormalizeURL(link.URL)
		if err != nil {
			counter.invalid++
			continue
		}
		if known[url] {
			counter.skipped++
			continue
		}
		known[url] = true
		bookmark := &store.Bookmark{
			CollectionID: target.ID,
			URL:          url,
			Title:        ptr(truncate(firstNonEmpty(strings.TrimSpace(deref(link.Title)), url), maxTitleLength)),
			Description:  blankToNil(link.Description),
			// Rendered as images and links for every member, so only http(s) URLs are kept
			OgImage:   httpURLOrNil(link.OgImage),
			Favicon:   httpURLOrNil(link.Favicon),
			Canonical: httpURLOrNil(link.Canonical),
			CreatedBy: currentUser(c),
			Tags:      normalizeTags(&link.Tags),
		}
		// Original date if the file has one, otherwise strictly increasing in file order (oldest first)
		if link.CreatedAt != nil {
			bookmark.CreatedAt = *link.CreatedAt
		} else {
			bookmark.CreatedAt = counter.start.Add(time.Duration(counter.imported) * time.Millisecond)
		}
		if err := tx.CreateBookmark(ctxOf(c), bookmark); err != nil {
			return err
		}
		counter.imported++
	}

	collections, err := tx.CollectionsBySpace(ctxOf(c), target.SpaceID)
	if err != nil {
		return err
	}
	for i := range node.Collections {
		childNode := &node.Collections[i]
		name := strings.TrimSpace(childNode.Name)
		if name == "" {
			name = "Imported"
		}
		name = string([]rune(name)[:min(len([]rune(name)), 80)])

		// Only collections the user can see are merged into; hidden ones must stay untouched
		var child *store.Collection
		for j := range collections {
			candidate := &collections[j]
			if candidate.ParentID != nil && *candidate.ParentID == target.ID && !hidden[candidate.ID] &&
				strings.EqualFold(candidate.Name, name) {
				child = candidate
				break
			}
		}
		if child != nil {
			counter.merged++
		} else {
			child = &store.Collection{SpaceID: target.SpaceID, ParentID: &target.ID, Name: name,
				Description: blankToNil(childNode.Description), CreatedBy: currentUser(c), Restricted: childNode.Private}
			var members []string
			if child.Restricted {
				members = []string{currentUser(c)}
			}
			if err := tx.CreateCollection(ctxOf(c), child, members); err != nil {
				return err
			}
			collections = append(collections, *child)
			counter.created++
		}
		if err := s.importNode(c, tx, child, childNode, hidden, counter); err != nil {
			return err
		}
	}
	return nil
}

func httpURLOrNil(value *string) *string {
	if value == nil || !strings.Contains(*value, "://") {
		return nil
	}
	url, err := fetch.NormalizeURL(*value)
	if err != nil {
		return nil
	}
	return &url
}
