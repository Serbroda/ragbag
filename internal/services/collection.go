package services

import (
	"context"

	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type CollectionService interface {
	GetVisibleCollectionsTree(ctx context.Context, userId string, spaceId string) ([]sqlc.GetCollectionsByUserAndSpaceRow, error)
}

type collectionService struct {
	queries *sqlc.Queries
}

func NewCollectionService(queries *sqlc.Queries) CollectionService {
	return &collectionService{queries: queries}
}

func (s *collectionService) GetVisibleCollectionsTree(ctx context.Context, userId string, spaceId string) ([]sqlc.GetCollectionsByUserAndSpaceRow, error) {
	// Lade sichtbare Collections für den Benutzer
	visibleRows, err := s.queries.GetCollectionsByUserAndSpace(ctx, sqlc.GetCollectionsByUserAndSpaceParams{
		UserID:  userId,
		SpaceID: spaceId,
	})
	if err != nil {
		return nil, err
	}

	return visibleRows, nil
}
