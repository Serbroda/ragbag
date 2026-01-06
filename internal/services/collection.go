package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type CollectionService interface {
	GetVisibleCollectionsTree(ctx context.Context, userId string, spaceId string) ([]sqlc.GetCollectionsByUserAndSpaceRow, error)
	CreateCollection(ctx context.Context, userId string, spaceId string, name string) (sqlc.Collection, error)
	GetCollection(ctx context.Context, auth string, id string) (sqlc.FindCollectionByIdAndUserIdRow, error)
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

func (s *collectionService) CreateCollection(ctx context.Context, userId string, spaceId string, name string) (sqlc.Collection, error) {
	collection, err := s.queries.InsertCollection(ctx, sqlc.InsertCollectionParams{
		ID:      db.NewDBID().String(),
		SpaceID: spaceId,
		Name:    name,
	})
	if err != nil {
		return sqlc.Collection{}, err
	}
	err = s.queries.InsertCollectionAndUser(ctx, sqlc.InsertCollectionAndUserParams{
		CollectionID: collection.ID,
		UserID:       userId,
		Role:         "OWNER",
	})
	if err != nil {
		return sqlc.Collection{}, err
	}
	return collection, nil
}

func (s *collectionService) GetCollection(ctx context.Context, auth string, id string) (sqlc.FindCollectionByIdAndUserIdRow, error) {
	return s.queries.FindCollectionByIdAndUserId(ctx, sqlc.FindCollectionByIdAndUserIdParams{
		CollectionID: id,
		UserID:       auth,
	})
}
