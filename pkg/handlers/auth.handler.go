package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/gen"
	"github.com/Serbroda/ragbag/gen/public"
	"github.com/Serbroda/ragbag/pkg/services"
	"github.com/Serbroda/ragbag/pkg/utils"
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var (
	jwtSecretKey       string = utils.MustGetEnv("JWT_SECRET_KEY")
	jwtAccessTokenExp  int64  = utils.MustParseInt64(utils.GetEnvFallback("JWT_ACCESS_EXPIRE_MINUTES", "15"))
	jwtRefreshTokenExp int64  = utils.MustParseInt64(utils.GetEnvFallback("JWT_REFRESH_EXPIRE_MINUTES", "10080"))
)

type PublicServerInterfaceImpl struct {
	Services *services.Services
	Queries  *gen.Queries
}

type JwtCustomClaims struct {
	Name  string `json:"name,omitempty"`
	Roles string `json:"roles,omitempty"`
	jwt.StandardClaims
}

func generateTokenPair(user *gen.User) (public.TokenPairDto, error) {
	userIdStr := strconv.FormatInt(user.ID, 10)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &JwtCustomClaims{
		Name: user.Username,
		StandardClaims: jwt.StandardClaims{
			Subject:   userIdStr,
			ExpiresAt: time.Now().Add(time.Minute * time.Duration(jwtAccessTokenExp)).Unix(),
		},
	})
	t, err := token.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return public.TokenPairDto{}, err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, &JwtCustomClaims{
		StandardClaims: jwt.StandardClaims{
			Subject:   userIdStr,
			ExpiresAt: time.Now().Add(time.Minute * time.Duration(jwtRefreshTokenExp)).Unix(),
		},
	})
	rt, err := refreshToken.SignedString([]byte(jwtSecretKey))
	if err != nil {
		return public.TokenPairDto{}, err
	}

	return public.TokenPairDto{
		AccessToken:  &t,
		RefreshToken: &rt,
	}, nil
}

func (si *PublicServerInterfaceImpl) Login(ctx echo.Context) error {
	var payload public.LoginDto
	err := ctx.Bind(&payload)
	if err != nil || payload.Username == nil || payload.Password == nil {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	user, err := si.Services.FindUserByUsername(ctx.Request().Context(), *payload.Username)
	if err != nil || user.ID < 1 || !utils.CheckPasswordHash(*payload.Password, user.Password) {
		return ctx.String(http.StatusNotFound, "invalid login")
	}

	tokenPair, err := generateTokenPair(&user)

	if err != nil {
		return ctx.String(http.StatusInternalServerError, "failed to generate token")
	}

	return ctx.JSON(http.StatusOK, tokenPair)
}

func (si *PublicServerInterfaceImpl) Register(ctx echo.Context) error {
	var payload public.RegistrationDto
	err := ctx.Bind(&payload)
	if err != nil || payload.Username == nil || payload.Password == nil {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	if si.Services.ExistsUser(ctx.Request().Context(), *payload.Password) {
		return ctx.String(http.StatusConflict, "user already exists")
	}

	hashedPassword, _ := utils.HashPassword(*payload.Password)

	activationCode := utils.RandomString(32)

	user, err := si.Services.CreateUser(ctx.Request().Context(), gen.CreateUserParams{
		Username:                strings.ToLower(*payload.Username),
		Password:                hashedPassword,
		Email:                   *payload.Email,
		ActivationCode:          sql.NullString{String: activationCode, Valid: true},
		ActivationSentAt:        sql.NullTime{Time: time.Now(), Valid: true},
		ActivationCodeExpiresAt: sql.NullTime{Time: time.Now().Add(time.Hour * 48), Valid: true},
		Active:                  false,
	})

	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	fmt.Printf("http://localhost:8080/api/v1/activate?code=%s\n", activationCode)

	return ctx.JSON(http.StatusCreated, &public.UserDto{
		Id:       &user.ID,
		Username: &user.Username,
	})
}

func (si *PublicServerInterfaceImpl) RefreshToken(ctx echo.Context) error {
	var payload public.RefreshTokenJSONBody
	err := ctx.Bind(&payload)
	if err != nil || payload.RefreshToken == "" {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	token, err := jwt.Parse(payload.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(jwtSecretKey), nil
	})

	if err != nil {
		return middleware.ErrJWTInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		return middleware.ErrJWTInvalid
	}

	sub := claims["sub"].(string)
	id := utils.MustParseInt64(sub)
	user, err := si.Queries.FindUser(ctx.Request().Context(), id)

	if err != nil || user.ID < 1 || !user.Active {
		return echo.ErrUnauthorized
	}

	newTokenPair, err := generateTokenPair(&user)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, newTokenPair)
}

func (si *PublicServerInterfaceImpl) Activate(ctx echo.Context, params public.ActivateParams) error {
	user, err := si.Services.FindUserByActivationCode(ctx.Request().Context(), params.Code)
	if err != nil {
		return ctx.String(http.StatusNotFound, "user not found")
	}

	if user.Active {
		return ctx.String(http.StatusAccepted, "user already activated")
	}

	if !user.ActivationCodeExpiresAt.Valid || user.ActivationCodeExpiresAt.Time.Before(time.Now()) {
		return ctx.String(http.StatusBadRequest, "activation expired")
	}

	si.Queries.UpdateUser(ctx.Request().Context(), gen.UpdateUserParams{
		ID:                    user.ID,
		ActivationConfirmedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Active:                true,

		Password:                user.Password,
		Name:                    user.Name,
		Email:                   user.Email,
		ActivationCode:          user.ActivationCode,
		ActivationSentAt:        user.ActivationSentAt,
		ActivationCodeExpiresAt: user.ActivationCodeExpiresAt,
	})
	return ctx.String(http.StatusOK, "user activated")
}
