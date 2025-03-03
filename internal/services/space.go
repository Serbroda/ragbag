package services

import (
	"context"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/teris-io/shortid"
)

type SpaceService interface {
	Create(ctx context.Context, params sqlc.InsertSpaceParams) (sqlc.Space, error)
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
