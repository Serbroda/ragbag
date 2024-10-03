package http

import (
	"backend/internal/dto"
	"backend/internal/security"
	"backend/internal/services"
	"backend/internal/sqlc"
	"github.com/labstack/echo/v4"
	"net/http"
)

type CreateSpaceRequest struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description"`
	Visibility  string  `json:"visibility"`
}

type SpaceHandler struct {
	SpaceService *services.SpaceService
}

func RegisterSpaceHandlers(e *echo.Group, h SpaceHandler, baseUrl string, middlewares ...echo.MiddlewareFunc) {
	e.POST(baseUrl+"/spaces", h.CreateSpace, middlewares...)
	e.GET(baseUrl+"/spaces", h.GetSpaces, middlewares...)
	//e.GET(baseUrl+"/spaces/:id", h.GetSpaceById, middlewares...)
	//e.DELETE(baseUrl+"/spaces/:id", h.DeleteSpace, middlewares...)
	//e.GET(baseUrl+"/spaces/:id/pages/tree", h.GetPagesTreeBySpaceId, middlewares...)
}

func (h *SpaceHandler) CreateSpace(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	var payload CreateSpaceRequest
	if err := BindAndValidate(ctx, &payload); err != nil {
		return err
	}

	params := sqlc.CreateSpaceParams{
		Name:        payload.Name,
		Description: payload.Description,
		Visibility:  payload.Visibility,
	}

	space, err := h.SpaceService.CreateSpace(auth, params)

	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, space)
}

func (h *SpaceHandler) GetSpaces(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return err
	}

	spaces, err := h.SpaceService.GetSpacesByUser(auth)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return ctx.JSON(http.StatusOK, dto.SpaceDtosFromSpaces(spaces))
}

/*func (h *SpaceHandler) GetSpaceById(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return ctx.String(http.StatusUnauthorized, err.Error())
	}

	id, err := bson.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}

	space, err := h.ContentService.GetSpaceById(context.TODO(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ctx.String(http.StatusNotFound, err.Error())
		}
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if err := h.checkPermission(&space, auth, false); err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, space)
}

func (h *SpaceHandler) DeleteSpace(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return ctx.String(http.StatusUnauthorized, err.Error())
	}

	id, err := bson.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}

	space, err := h.ContentService.GetSpaceById(context.TODO(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ctx.String(http.StatusNotFound, err.Error())
		}
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if err := h.checkPermission(&space, auth, true); err != nil {
		return err
	}

	err = h.ContentService.DeleteSpace(context.TODO(), id)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.String(http.StatusOK, "ok")
}

func (h *SpaceHandler) GetPagesTreeBySpaceId(ctx echo.Context) error {
	auth, err := security.GetAuthentication(ctx)
	if err != nil {
		return ctx.String(http.StatusUnauthorized, err.Error())
	}

	id, err := bson.ObjectIDFromHex(ctx.Param("id"))
	if err != nil {
		return ctx.String(http.StatusBadRequest, err.Error())
	}

	space, err := h.ContentService.GetSpaceById(context.TODO(), id)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return ctx.String(http.StatusNotFound, err.Error())
		}
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	if err := h.checkPermission(&space, auth, false); err != nil {
		return err
	}

	pages, err := h.ContentService.GetPagesBySpaceId(context.TODO(), space.ID)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
	pagesPointers := slice.ConvertToPointerSlice(pages)

	pagesTree := h.ContentService.BuildPageTree(pagesPointers)
	return ctx.JSON(http.StatusOK, pagesTree)
}

func (h *SpaceHandler) checkPermission(space *internal.Space, auth security.Authentication, requireAdmin bool) error {
	if space.OwnerID != auth.UserId && !slice.ContainsFiltered(space.Shared, func(s internal.UserIdWithRole) bool {
		return s.UserID == auth.UserId && (!requireAdmin || s.Role == "ADMIN")
	}) {
		return echo.NewHTTPError(http.StatusForbidden, "Forbidden")
	}
	return nil
}*/
