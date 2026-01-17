package services

import (
	"context"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
)

type SpaceService interface {
	CreateSpace(ctx context.Context, userId security.AuthenticationId, params sqlc.InsertSpaceParams) (sqlc.Space, error)
	GetAllSpaces(ctx context.Context, userId security.AuthenticationId) ([]sqlc.FindSpacesByUserIdRow, error)
	GetSpaceById(ctx context.Context, userId security.AuthenticationId, spaceId string) (sqlc.FindSpaceByIdAndUserIdRow, error)
}

type spaceService struct {
	queries *sqlc.Queries
}

func NewSpaceService(queries *sqlc.Queries) SpaceService {
	return &spaceService{queries: queries}
}

func (s spaceService) CreateSpace(ctx context.Context, userId security.AuthenticationId, params sqlc.InsertSpaceParams) (sqlc.Space, error) {
	params.ID = db.NewDBID().String()
	space, err := s.queries.InsertSpace(ctx, sqlc.InsertSpaceParams{
		ID:        params.ID,
		Name:      params.Name,
		CreatedBy: userId.String(),
	})
	if err != nil {
		return sqlc.Space{}, err
	}
	err = s.queries.InsertSpaceMember(ctx, sqlc.InsertSpaceMemberParams{
		SpaceID: space.ID,
		UserID:  userId.String(),
		Role:    "OWNER",
	})
	if err != nil {
		return sqlc.Space{}, err
	}
	return space, nil
}

func (s spaceService) GetAllSpaces(ctx context.Context, userId security.AuthenticationId) ([]sqlc.FindSpacesByUserIdRow, error) {
	return s.queries.FindSpacesByUserId(ctx, userId.String())
}

func (s spaceService) GetSpaceById(ctx context.Context, userId security.AuthenticationId, spaceId string) (sqlc.FindSpaceByIdAndUserIdRow, error) {
	return s.queries.FindSpaceByIdAndUserId(ctx, sqlc.FindSpaceByIdAndUserIdParams{
		UserID:  userId.String(),
		SpaceID: spaceId,
	})
}
