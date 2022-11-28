package handlers

import (
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
	baseUrl            string = utils.MustGetEnv("SERVER_BASE_URL")
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
	if err != nil || payload.Username == "" || payload.Password == "" {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	user, err := si.Services.FindUserByUsername(ctx.Request().Context(), payload.Username)
	if err != nil || user.ID < 1 || !utils.CheckBcryptHash(payload.Password, user.Password) {
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
	if err != nil || payload.Username == "" || payload.Password == "" {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	if si.Services.ExistsUser(ctx.Request().Context(), payload.Username) {
		return ctx.String(http.StatusConflict, "user already exists")
	}

	hashedPassword, _ := utils.HashBcrypt(payload.Password)

	user, err := si.Services.CreateUser(ctx.Request().Context(), gen.CreateUserParams{
		Username:  strings.ToLower(payload.Username),
		Password:  hashedPassword,
		Email:     payload.Email,
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Active:    false,
	})
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	token, err := services.Service.CreateActivationToken(ctx.Request().Context(), user.ID)
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	err = utils.SendMailTemplate(utils.MailWithTemplate{
		Mail: utils.Mail{
			To:      []string{payload.Email},
			Subject: "Verify your email address",
		},
		Template: "resources/templates/email/email-verification.html",
		Data: struct {
			Name string
			Link string
		}{
			Name: payload.FirstName,
			Link: fmt.Sprintf("%s/api/v1/activate?code=%s", baseUrl, token),
		},
	})

	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusCreated, &public.UserDto{
		Id:       &user.ID,
		Username: &user.Username,
	})
}

func (si *PublicServerInterfaceImpl) RefreshToken(ctx echo.Context) error {
	var payload public.RefreshTokenJSONRequestBody
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
	err := si.Services.ActivateUser(ctx.Request().Context(), params.Code)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	return ctx.String(http.StatusOK, "user activated")
}

func (si *PublicServerInterfaceImpl) RequestPasswordReset(ctx echo.Context) error {
	var payload public.RequestPasswordResetJSONRequestBody
	err := ctx.Bind(&payload)
	if err != nil || payload.Email == "" {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	user, err := si.Queries.FindByEmail(ctx.Request().Context(), payload.Email)
	if err != nil {
		return ctx.String(http.StatusNoContent, "password reset mail sent")
	}

	token, err := si.Services.CreatePasswordResetToken(ctx.Request().Context(), user.ID)
	if err == nil {
		utils.SendMailTemplate(utils.MailWithTemplate{
			Mail: utils.Mail{
				To:      []string{payload.Email},
				Subject: "Password reset",
			},
			Template: "resources/templates/email/password-reset.html",
			Data: struct {
				Name     string
				Username string
				Link     string
			}{
				Name:     user.FirstName,
				Username: user.Username,
				Link:     fmt.Sprintf("%s/api/v1/password_reset?code=%s", baseUrl, token),
			},
		})
	}

	return ctx.String(http.StatusNoContent, "password reset mail sent")
}

func (si *PublicServerInterfaceImpl) StartPasswordReset(ctx echo.Context, params public.StartPasswordResetParams) error {
	ctx.Response().Header().Set("password_reset_code", params.Code)
	return ctx.Redirect(http.StatusFound, fmt.Sprintf("%s/password_reset", baseUrl))
}

func (si *PublicServerInterfaceImpl) ResetPassword(ctx echo.Context) error {
	var payload public.ResetPasswordJSONRequestBody
	err := ctx.Bind(&payload)
	if err != nil || payload.Email == "" {
		return ctx.String(http.StatusBadRequest, "bad request")
	}

	prt, err := si.Queries.FindPasswordResetCodeByEmailAndToken(ctx.Request().Context(), gen.FindPasswordResetCodeByEmailAndTokenParams{
		Email:     payload.Email,
		TokenHash: utils.HashSha3256(payload.Code),
	})

	if err != nil {
		return err
	}

	if prt.ExpiresAt.Before(time.Now()) || !prt.Useractive {
		return ctx.String(http.StatusBadRequest, "password reset code expired or user not active")
	}

	err = si.Services.ChangePassword(ctx.Request().Context(), prt.UserID, payload.Password)
	if err != nil {
		return err
	}
	return ctx.String(http.StatusNoContent, "password reset")
}
