package services

import (
	"context"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/teris-io/shortid"
)

type SpaceService interface {
	Create(ctx context.Context, userId int64, params sqlc.InsertSpaceParams) (sqlc.Space, error)
	GetSpaces(ctx context.Context, userId int64) ([]sqlc.FindSpacesByUserIdRow, error)
	GetSpace(ctx context.Context, auth int64, id string) (sqlc.Space, error)
	GetSpaceByUser(ctx context.Context, auth int64, id string) (sqlc.FindSpaceBySidAndUserIdRow, error)
}

type spaceService struct {
	queries *sqlc.Queries
}

func NewSpaceService(queries *sqlc.Queries) SpaceService {
	return &spaceService{queries: queries}
}

func (s spaceService) Create(ctx context.Context, userId int64, params sqlc.InsertSpaceParams) (sqlc.Space, error) {
	params.Sid = shortid.MustGenerate()
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

func (s spaceService) GetSpaces(ctx context.Context, userId int64) ([]sqlc.FindSpacesByUserIdRow, error) {
	return s.queries.FindSpacesByUserId(ctx, userId)
}

func (s spaceService) GetSpace(ctx context.Context, auth int64, id string) (sqlc.Space, error) {
	return s.queries.FindSpaceBySid(ctx, id)
}

func (s spaceService) GetSpaceByUser(ctx context.Context, auth int64, id string) (sqlc.FindSpaceBySidAndUserIdRow, error) {
	return s.queries.FindSpaceBySidAndUserId(ctx, sqlc.FindSpaceBySidAndUserIdParams{
		SpaceID: id,
		UserID:  auth,
	})
}
