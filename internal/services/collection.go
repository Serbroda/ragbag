package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
)

type CollectionService interface {
	CreateCollection(ctx context.Context, userId security.AuthenticationId, spaceId string, name string) (sqlc.Collection, error)
	GetAllCollectionsBySpaceId(ctx context.Context, userId security.AuthenticationId, spaceId string) ([]sqlc.Collection, error)
	GetCollectionById(ctx context.Context, userId security.AuthenticationId, collectionId string) (sqlc.FindCollectionByIdAndUserIdRow, error)
}

type collectionService struct {
	queries *sqlc.Queries
}

func NewCollectionService(queries *sqlc.Queries) CollectionService {
	return &collectionService{queries: queries}
}

func (s *collectionService) GetAllCollectionsBySpaceId(ctx context.Context, userId security.AuthenticationId, spaceId string) ([]sqlc.Collection, error) {
	// Lade sichtbare Collections für den Benutzer
	visibleRows, err := s.queries.FindCollectionsBySpaceIdAndUserId(ctx, sqlc.FindCollectionsBySpaceIdAndUserIdParams{
		UserID:  userId.String(),
		SpaceID: spaceId,
	})
	if err != nil {
		return nil, err
	}

	return visibleRows, nil
}

func (s *collectionService) CreateCollection(ctx context.Context, userId security.AuthenticationId, spaceId string, name string) (sqlc.Collection, error) {
	collection, err := s.queries.InsertCollection(ctx, sqlc.InsertCollectionParams{
		ID:        db.NewDBID().String(),
		SpaceID:   spaceId,
		Name:      name,
		CreatedBy: userId.String(),
	})
	if err != nil {
		return sqlc.Collection{}, err
	}
	return collection, nil
}

func (s *collectionService) GetCollectionById(ctx context.Context, userId security.AuthenticationId, collectionId string) (sqlc.FindCollectionByIdAndUserIdRow, error) {
	collection, err := s.queries.FindCollectionByIdAndUserId(ctx, sqlc.FindCollectionByIdAndUserIdParams{
		ID:     db.NewDBID().String(),
		UserID: userId.String(),
	})
	if err != nil {
		return sqlc.FindCollectionByIdAndUserIdRow{}, err
	}
	return collection, nil
}
