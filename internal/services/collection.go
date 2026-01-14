package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type CollectionService interface {
	CreateCollection(ctx context.Context, userId string, spaceId string, name string) (sqlc.Collection, error)
	GetCollections(ctx context.Context, userId string, spaceId string) ([]sqlc.Collection, error)
}

type collectionService struct {
	queries *sqlc.Queries
}

func NewCollectionService(queries *sqlc.Queries) CollectionService {
	return &collectionService{queries: queries}
}

func (s *collectionService) GetCollections(ctx context.Context, userId string, spaceId string) ([]sqlc.Collection, error) {
	// Lade sichtbare Collections für den Benutzer
	visibleRows, err := s.queries.FindCollectionsBySpaceIdAndUserId(ctx, sqlc.FindCollectionsBySpaceIdAndUserIdParams{
		UserID:  userId,
		SpaceID: spaceId,
	})
	if err != nil {
		return nil, err
	}

	return visibleRows, nil
}

func (s *collectionService) CreateCollection(ctx context.Context, userId string, spaceId string, name string) (sqlc.Collection, error) {
	collection, err := s.queries.InsertCollection(ctx, sqlc.InsertCollectionParams{
		ID:        db.NewDBID().String(),
		SpaceID:   spaceId,
		Name:      name,
		CreatedBy: userId,
	})
	if err != nil {
		return sqlc.Collection{}, err
	}
	return collection, nil
}
