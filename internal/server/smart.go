package server

import (
	"strings"

	"github.com/Serbroda/ragbag/internal/store"
)

// Smart collections (collections.query) have no links of their own: they list what their search finds in the
// whole space, for every member what that member may see (GetBookmarks). They take neither links nor
// sub-collections and cannot be shared publicly, which would show links nobody decided to publish. Whether a
// collection is smart is decided when it is created.

const maxSmartQueryLength = 200

// smartQuery checks the search of a smart collection; nil or blank is none.
func smartQuery(value *string) (*string, error) {
	query := blankToNil(value)
	if query == nil {
		return nil, nil
	}
	if len([]rune(*query)) > maxSmartQueryLength {
		return nil, badRequest("The search of a smart collection must be at most %d characters", maxSmartQueryLength)
	}
	if len(store.SearchTerms(*query)) == 0 {
		return nil, badRequest("The search of a smart collection must not be empty")
	}
	return query, nil
}

// holdsLinks refuses smart collections where a collection is about to get links or sub-collections.
func holdsLinks(collection *store.Collection) error {
	if collection.Query != nil {
		return badRequest("“%s” is a smart collection: it shows the results of its search and cannot hold anything itself",
			strings.TrimSpace(collection.Name))
	}
	return nil
}
