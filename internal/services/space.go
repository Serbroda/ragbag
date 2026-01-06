package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
)

type SpaceService interface {
	Create(ctx context.Context, userId string, params sqlc.InsertSpaceParams) (sqlc.Space, error)
	GetSpaces(ctx context.Context, userId string) ([]sqlc.FindSpacesByUserIdRow, error)
	GetSpace(ctx context.Context, auth string, id string) (sqlc.Space, error)
	GetSpaceByUser(ctx context.Context, auth string, id string) (sqlc.FindSpaceByIdAndUserIdRow, error)
}

type spaceService struct {
	queries *sqlc.Queries
}

func NewSpaceService(queries *sqlc.Queries) SpaceService {
	return &spaceService{queries: queries}
}

func (s spaceService) Create(ctx context.Context, userId string, params sqlc.InsertSpaceParams) (sqlc.Space, error) {
	params.ID = db.NewDBID().String()
	space, err := s.queries.InsertSpace(ctx, params)
	if err != nil {
		return sqlc.Space{}, err
	}
	err = s.queries.InsertSpaceUser(ctx, sqlc.InsertSpaceUserParams{
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

func (s spaceService) GetSpace(ctx context.Context, auth string, id string) (sqlc.Space, error) {
	return s.queries.FindSpaceById(ctx, id)
}

func (s spaceService) GetSpaceByUser(ctx context.Context, auth string, id string) (sqlc.FindSpaceByIdAndUserIdRow, error) {
	return s.queries.FindSpaceByIdAndUserId(ctx, sqlc.FindSpaceByIdAndUserIdParams{
		SpaceID: id,
		UserID:  auth,
	})
}
