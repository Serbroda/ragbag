package services

import (
	"context"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/teris-io/shortid"
)

type SpaceService interface {
	Create(ctx context.Context, params sqlc.InsertSpaceParams) (sqlc.Space, error)
	GetSpaces(ctx context.Context, ownerId int64) ([]sqlc.Space, error)
	GetSpace(ctx context.Context, auth int64, id string) (sqlc.Space, error)
}

type spaceService struct {
	queries *sqlc.Queries
}

func NewSpaceService(queries *sqlc.Queries) SpaceService {
	return &spaceService{queries: queries}
}

func (s spaceService) Create(ctx context.Context, params sqlc.InsertSpaceParams) (sqlc.Space, error) {
	params.Sid = shortid.MustGenerate()
	return s.queries.InsertSpace(ctx, params)
}

func (s spaceService) GetSpaces(ctx context.Context, ownerId int64) ([]sqlc.Space, error) {
	return s.queries.FindSpacesByOwnerId(ctx, ownerId)
}

func (s spaceService) GetSpace(ctx context.Context, auth int64, id string) (sqlc.Space, error) {
	return s.queries.FindSpaceBySid(ctx, sqlc.FindSpaceBySidParams{
		ID:     id,
		UserID: auth,
	})
}
