package api

import (
	"net/http"

	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/Serbroda/ragbag/internal/utils"
	"github.com/labstack/echo/v4"
)

type apiServer struct {
	authService       services.AuthService
	spaceService      services.SpaceService
	collectionService services.CollectionService
}

func NewApiServer(
	authService services.AuthService,
	spaceService services.SpaceService,
	collectionService services.CollectionService,
) ServerInterface {
	return apiServer{
		authService:       authService,
		spaceService:      spaceService,
		collectionService: collectionService,
	}
}

func (a apiServer) GetSpaces(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	spaces, err := a.spaceService.GetSpaces(ctx.Request().Context(), auth.ID)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	return ctx.JSON(http.StatusOK, utils.MapSlice(spaces, func(item sqlc.FindSpacesByUserIdRow) SpaceDto {
		return SpaceDto{
			Id:   item.Space.ID,
			Name: item.Space.Name,
		}
	}))
}

func (a apiServer) GetSpace(ctx echo.Context, spaceId Id) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	space, err := a.spaceService.GetSpace(ctx.Request().Context(), auth.ID, spaceId)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Space with id "+spaceId+" not found")
	}

	return ctx.JSON(http.StatusOK, SpaceDto{
		Id:   space.ID,
		Name: space.Name,
	})
}

// Collections

func (a apiServer) GetCollections(ctx echo.Context, spaceId Id) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	space, err := a.spaceService.GetSpace(ctx.Request().Context(), auth.ID, spaceId)
	if err != nil {
		return ctx.String(http.StatusNotFound, "Space with id "+spaceId+" not found")
	}

	tree, err := a.collectionService.GetVisibleCollectionsTree(ctx.Request().Context(), auth.ID, space.ID)
	if err != nil {
		return ctx.String(http.StatusNotFound, err.Error())
	}

	return ctx.JSON(http.StatusOK, tree)
}

func (a apiServer) CreateCollection(ctx echo.Context, spaceId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) DeleteCollection(ctx echo.Context, spaceId Id, collectionId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) GetCollection(ctx echo.Context, spaceId Id, collectionId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) UpdateCollection(ctx echo.Context, spaceId Id, collectionId Id) error {
	//TODO implement me
	panic("implement me")
}

// Bookmarks

func (a apiServer) GetBookmarks(ctx echo.Context, spaceId Id, collectionId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) CreateBookmark(ctx echo.Context, spaceId Id, collectionId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) DeleteBookmark(ctx echo.Context, spaceId Id, collectionId Id, bookmarkId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) GetBookmark(ctx echo.Context, spaceId Id, collectionId Id, bookmarkId Id) error {
	//TODO implement me
	panic("implement me")
}

func (a apiServer) UpdateBookmark(ctx echo.Context, spaceId Id, collectionId Id, bookmarkId Id) error {
	//TODO implement me
	panic("implement me")
}
