package services

import (
	"context"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type CollectionTreeNode struct {
	Collection sqlc.Collection       `json:"collection"`
	Children   []*CollectionTreeNode `json:"children"`
}

type CollectionService interface {
	GetVisibleCollectionsTree(ctx context.Context, userId int64, spaceId int64) ([]*CollectionTreeNode, error)
}

type collectionService struct {
	queries *sqlc.Queries
}

func NewCollectionService(queries *sqlc.Queries) CollectionService {
	return &collectionService{queries: queries}
}

func (s *collectionService) GetVisibleCollectionsTree(ctx context.Context, userId int64, spaceId int64) ([]*CollectionTreeNode, error) {
	// Lade alle Collections (für Pfad-Auflösung)
	allCollections, err := s.queries.GetAllCollections(ctx)
	if err != nil {
		return nil, err
	}

	// Lade sichtbare Collections für den Benutzer
	visibleRows, err := s.queries.GetCollectionsByUserAndSpace(ctx, sqlc.GetCollectionsByUserAndSpaceParams{
		UserID:  userId,
		SpaceID: spaceId,
	})
	if err != nil {
		return nil, err
	}

	visibleMap := make(map[int64]bool)
	for _, row := range visibleRows {
		visibleMap[row.Collection.ID] = true
	}

	// Rekursiv Eltern hinzufügen
	collectionByID := make(map[int64]sqlc.Collection)
	for _, c := range allCollections {
		collectionByID[c.ID] = c
	}

	added := true
	for added {
		added = false
		for id := range visibleMap {
			parentId := collectionByID[id].ParentID
			if parentId != nil && !visibleMap[*parentId] {
				visibleMap[*parentId] = true
				added = true
			}
		}
	}

	// Baumstruktur aufbauen
	nodes := make(map[int64]*CollectionTreeNode)
	var roots []*CollectionTreeNode

	for _, c := range allCollections {
		if !visibleMap[c.ID] {
			continue
		}
		nodes[c.ID] = &CollectionTreeNode{
			Collection: c,
			Children:   []*CollectionTreeNode{},
		}
	}

	for _, c := range allCollections {
		if !visibleMap[c.ID] {
			continue
		}
		node := nodes[c.ID]
		if c.ParentID != nil && visibleMap[*c.ParentID] {
			nodes[*c.ParentID].Children = append(nodes[*c.ParentID].Children, node)
		} else {
			roots = append(roots, node)
		}
	}

	return roots, nil
}
