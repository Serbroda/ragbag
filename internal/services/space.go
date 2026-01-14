package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type SpaceService interface {
	Create(ctx context.Context, userId string, params sqlc.InsertSpaceParams) (sqlc.Space, error)
	GetSpaces(ctx context.Context, userId string) ([]sqlc.FindSpacesByUserIdRow, error)
	GetSpace(ctx context.Context, userId, spaceId string) (sqlc.FindSpaceByIdAndUserIdRow, error)
}

type spaceService struct {
	queries *sqlc.Queries
}

func NewSpaceService(queries *sqlc.Queries) SpaceService {
	return &spaceService{queries: queries}
}

func (s spaceService) Create(ctx context.Context, userId string, params sqlc.InsertSpaceParams) (sqlc.Space, error) {
	params.ID = db.NewDBID().String()
	space, err := s.queries.InsertSpace(ctx, sqlc.InsertSpaceParams{
		ID:        params.ID,
		Name:      params.Name,
		CreatedBy: userId,
	})
	if err != nil {
		return sqlc.Space{}, err
	}
	err = s.queries.InsertSpaceMember(ctx, sqlc.InsertSpaceMemberParams{
		SpaceID: space.ID,
		UserID:  userId,
		Role:    "OWNER",
	})
	if err != nil {
		return sqlc.Space{}, err
	}
	return space, nil
}

func (s spaceService) GetSpaces(ctx context.Context, userId string) ([]sqlc.FindSpacesByUserIdRow, error) {
	return s.queries.FindSpacesByUserId(ctx, userId)
}

func (s spaceService) GetSpace(ctx context.Context, userId, spaceId string) (sqlc.FindSpaceByIdAndUserIdRow, error) {
	return s.queries.FindSpaceByIdAndUserId(ctx, sqlc.FindSpaceByIdAndUserIdParams{
		UserID:  userId,
		SpaceID: spaceId,
	})
}
