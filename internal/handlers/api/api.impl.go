package api

import (
	"context"
	"fmt"

	"github.com/Serbroda/ragbag/internal/db"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/Serbroda/ragbag/internal/utils"
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
) StrictServerInterface {
	return apiServer{
		authService:       authService,
		spaceService:      spaceService,
		collectionService: collectionService,
	}
}

// GetSpaces implements StrictServerInterface
func (a apiServer) GetSpaces(ctx context.Context, request GetSpacesRequestObject) (GetSpacesResponseObject, error) {
	auth, err := security.GetAuthenticationFromContext(ctx)
	if err != nil {
		return nil, err
	}

	spaces, err := a.spaceService.GetSpaces(ctx, auth.ID)
	if err != nil {
		return nil, err
	}

	dtos := utils.MapSlice(spaces, func(item sqlc.FindSpacesByUserIdRow) SpaceDto {
		return SpaceDto{
			Id:   item.Space.ID,
			Name: item.Space.Name,
		}
	})

	return GetSpaces200JSONResponse(dtos), nil
}

// GetSpace implements StrictServerInterface
func (a apiServer) GetSpace(ctx context.Context, request GetSpaceRequestObject) (GetSpaceResponseObject, error) {
	return nil, fmt.Errorf("GetBookmarks not implemented")
}

// GetCollections implements StrictServerInterface
func (a apiServer) GetCollections(ctx context.Context, request GetCollectionsRequestObject) (GetCollectionsResponseObject, error) {
	auth, err := security.GetAuthenticationFromContext(ctx)
	if err != nil {
		return nil, err
	}

	tree, err := a.collectionService.GetCollections(ctx, auth.ID, request.SpaceId)
	if err != nil {
		return nil, err
	}

	dtos := utils.MapSlice(tree, func(item sqlc.Collection) CollectionDto {
		return CollectionDto{
			Id:   item.ID,
			Name: item.Name,
		}
	})
	return GetCollections200JSONResponse(dtos), nil
}

// CreateCollection implements StrictServerInterface
func (a apiServer) CreateCollection(ctx context.Context, request CreateCollectionRequestObject) (CreateCollectionResponseObject, error) {
	auth, err := security.GetAuthenticationFromContext(ctx)
	if err != nil {
		return nil, err
	}

	space, role, err := a.getSpaceById(ctx, auth.ID, request.SpaceId)
	if err != nil {
		msg := "Space with id " + request.SpaceId + " not found"
		return CreateCollection404JSONResponse{NotFoundJSONResponse{Message: &msg}}, nil
	}
	if role != "OWNER" && role != "CONTRIBUTOR" {
		return CreateCollection403Response{}, nil
	}

	collection, err := a.collectionService.CreateCollection(ctx, auth.ID, space.ID, request.Body.Name)
	if err != nil {
		return nil, err
	}

	// Implementation pending: create collection using request.Body
	// Return Not Implemented as error for now
	return CreateCollection200JSONResponse(CollectionDto{
		Id:   collection.ID,
		Name: collection.Name,
	}), nil
}

func (a apiServer) DeleteCollection(ctx context.Context, request DeleteCollectionRequestObject) (DeleteCollectionResponseObject, error) {
	return nil, fmt.Errorf("DeleteCollection not implemented")
}

func (a apiServer) GetCollection(ctx context.Context, request GetCollectionRequestObject) (GetCollectionResponseObject, error) {
	return nil, fmt.Errorf("DeleteCollection not implemented")
}

func (a apiServer) UpdateCollection(ctx context.Context, request UpdateCollectionRequestObject) (UpdateCollectionResponseObject, error) {
	return nil, fmt.Errorf("UpdateCollection not implemented")
}

func (a apiServer) GetBookmarks(ctx context.Context, request GetBookmarksRequestObject) (GetBookmarksResponseObject, error) {
	return nil, fmt.Errorf("GetBookmarks not implemented")
}

func (a apiServer) CreateBookmark(ctx context.Context, request CreateBookmarkRequestObject) (CreateBookmarkResponseObject, error) {
	return nil, fmt.Errorf("CreateBookmark not implemented")
}

func (a apiServer) DeleteBookmark(ctx context.Context, request DeleteBookmarkRequestObject) (DeleteBookmarkResponseObject, error) {
	return nil, fmt.Errorf("DeleteBookmark not implemented")
}

func (a apiServer) GetBookmark(ctx context.Context, request GetBookmarkRequestObject) (GetBookmarkResponseObject, error) {
	return nil, fmt.Errorf("GetBookmark not implemented")
}

func (a apiServer) UpdateBookmark(ctx context.Context, request UpdateBookmarkRequestObject) (UpdateBookmarkResponseObject, error) {
	return nil, fmt.Errorf("UpdateBookmark not implemented")
}

func (a apiServer) getSpaceById(ctx context.Context, authId string, spaceId string) (sqlc.Space, string, error) {
	id, err := db.ParseDBID(spaceId)
	if err != nil {
		return sqlc.Space{}, "", fmt.Errorf("Space with id " + spaceId + " not found")
	}

	space, err := a.spaceService.GetSpace(ctx, authId, id.String())
	if err != nil {
		return sqlc.Space{}, "", fmt.Errorf("Space with id " + spaceId + " not found")
	}

	return space.Space, space.UserRole, nil
}
