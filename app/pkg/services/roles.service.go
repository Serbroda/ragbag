package services

import (
	"context"
	"github.com/Serbroda/ragbag/app/pkg/sqlc"
)

func (s *Services) FindRolesByNamesIn(ctx context.Context, roles []string) []sqlc.Role {
	var result []sqlc.Role
	for _, r := range roles {
		res, err := s.Queries.FindRoleByName(ctx, r)
		if err == nil && res.ID > 0 {
			result = append(result, res)
		}
	}
	return result
}
