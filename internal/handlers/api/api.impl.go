package api

import (
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/Serbroda/ragbag/internal/utils"
	"github.com/labstack/echo/v4"
	"net/http"
)

type apiServer struct {
	authService  services.AuthService
	spaceService services.SpaceService
}

func NewApiServer(authService services.AuthService, spaceService services.SpaceService) ServerInterface {
	return apiServer{
		authService:  authService,
		spaceService: spaceService,
	}
}

func (a apiServer) GetSpaces(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	spaces, err := a.spaceService.GetSpaces(ctx.Request().Context(), auth.UserId)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	return ctx.JSON(http.StatusOK, utils.MapSlice(spaces, func(item sqlc.Space) SpaceDto {
		return SpaceDto{
			Id:   item.Sid,
			Name: item.Name,
		}
	}))
}

func (a apiServer) GetSpace(ctx echo.Context, spaceId Id) error {
	space, err := a.spaceService.GetSpace(ctx.Request().Context(), spaceId)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Space with id "+spaceId+" not found")
	}

	return ctx.JSON(http.StatusOK, SpaceDto{
		Id:   space.Sid,
		Name: space.Name,
	})
}
