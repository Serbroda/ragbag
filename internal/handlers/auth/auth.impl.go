package auth

import (
	"errors"
	"net/http"

	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/handlers"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/labstack/echo/v4"
)

type authServer struct {
	authService services.AuthService
}

func NewAuthServer(authService services.AuthService) ServerInterface {
	return authServer{
		authService: authService,
	}
}

func (a authServer) SignUp(ctx echo.Context) error {
	var payload RegistrationRequest
	if err := handlers.BindAndValidate(ctx, &payload); err != nil {
		return err
	}

	entity, err := a.authService.Register(ctx.Request().Context(), sqlc.InsertUserParams{
		Email:    string(payload.Email),
		Password: payload.Password,
	})

	if err != nil {
		if errors.Is(err, services.ErrUserAlreadyExists) {
			return ctx.String(http.StatusConflict, err.Error())
		}
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, UserDto{
		Email: entity.Email,
		Id:    entity.ID,
	})
}

func (a authServer) Login(ctx echo.Context) error {
	var payload LoginRequest
	if err := handlers.BindAndValidate(ctx, &payload); err != nil {
		return err
	}

	tokenPair, err := a.authService.Login(ctx.Request().Context(), string(payload.Email), payload.Password)
	if err != nil {
		return handleAuthError(ctx, err)
	}

	a.setRefreshTokenCookie(ctx, tokenPair.RefreshToken)

	return ctx.JSON(http.StatusOK, LoginResponse{
		AccessToken: *tokenPair.AccessToken.Token,
	})
}

func (a authServer) RefreshToken(ctx echo.Context) error {
	refreshTokenCookie, err := ctx.Cookie("refreshToken")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "Refresh token not found in cookies")
	}

	refreshToken := refreshTokenCookie.Value

	tokenPair, err := a.authService.RefreshToken(ctx.Request().Context(), refreshToken)
	if err != nil {
		return handleAuthError(ctx, err)
	}

	a.setRefreshTokenCookie(ctx, tokenPair.RefreshToken)

	return ctx.JSON(http.StatusOK, LoginResponse{
		AccessToken: *tokenPair.AccessToken.Token,
	})
}

func (a authServer) setRefreshTokenCookie(ctx echo.Context, refreshToken security.JWTInfo) {
	ctx.SetCookie(&http.Cookie{
		Name:     "refreshToken",
		Value:    *refreshToken.Token,
		Expires:  refreshToken.Expiration,
		HttpOnly: true,
		Secure:   true, // Set to true in production (HTTPS only)
		SameSite: http.SameSiteLaxMode,
		//SameSite: http.SameSiteNoneMode,
		Path: "/",
	})
}

func handleAuthError(ctx echo.Context, err error) error {
	switch {
	case errors.Is(err, services.ErrUserNotFound), errors.Is(err, services.ErrWrongPassword), errors.Is(err, services.ErrInvalidToken):
		return ctx.String(http.StatusUnauthorized, "bad credentials")
	default:
		return ctx.String(http.StatusInternalServerError, err.Error())
	}
}
