package security

import (
	"context"
	"errors"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

var (
	ContextKeyAuthentication = "authentication"
)

type Authentication struct {
	ID   string
	Role string
}

func CreateJwtConfig() echojwt.Config {
	return echojwt.Config{
		SigningKey: []byte(JwtSecretKey),
		ContextKey: "token",
		SuccessHandler: func(c echo.Context) {
			token, ok := c.Get("token").(*jwt.Token)
			if !ok {
				return
			}
			auth, err := ParseToken(token)
			if err != nil {
				return
			}
			c.Set(ContextKeyAuthentication, auth)

			req := c.Request()
			ctx := req.Context()
			ctx = context.WithValue(ctx, ContextKeyAuthentication, auth)
			c.SetRequest(req.WithContext(ctx))
		},
	}
}

func ParseToken(token *jwt.Token) (Authentication, error) {
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Authentication{}, errors.New("failed to get claims of token")
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return Authentication{}, errors.New("failed to get sub from claims")
	}

	role, ok := claims["roles"].(string)
	if !ok {
		return Authentication{}, errors.New("failed to get role from claims")
	}

	return Authentication{
		ID:   sub,
		Role: role,
	}, nil
}

func GetAuthentication(ctx echo.Context) (Authentication, error) {
	auth, ok := ctx.Get(ContextKeyAuthentication).(Authentication)
	if !ok {
		return auth, echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}
	return auth, nil
}

// neue Funktion für context.Context (Strict handlers)
func GetAuthenticationFromContext(ctx context.Context) (Authentication, error) {
	auth, ok := ctx.Value(ContextKeyAuthentication).(Authentication)
	if !ok {
		// echo.NewHTTPError ist weiterhin ein geeigneter Fehler für HTTP-Antworten
		return Authentication{}, echo.NewHTTPError(http.StatusUnauthorized, "Unauthorized")
	}
	return auth, nil
}
