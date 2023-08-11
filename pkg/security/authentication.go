package security

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Serbroda/ragbag/pkg/user"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
)

var (
	ContextKeyAuthentication = "authentication"
)

type Authentication struct {
	Subject int64
	Roles   []string
}

func CreateJwtConfig(userService user.UserService) echojwt.Config {
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

			}
			c.Set(ContextKeyAuthentication, auth)
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
	userId, err := strconv.ParseInt(sub, 10, 64)
	if err != nil {
		return Authentication{}, errors.New("failed parse int of sub")
	}
	roleInterfaces, ok := claims["roles"].([]interface{})
	if !ok {
		return Authentication{}, errors.New("failed to get roles from claims")
	}
	roles := []string{}
	for _, ri := range roleInterfaces {
		roles = append(roles, ri.(string))
	}
	return Authentication{
		Subject: userId,
		Roles:   roles,
	}, nil
}

func HasAnyRoleMiddleware(roles ...string) func(next echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			u := c.Get(ContextKeyAuthentication)
			authentication, ok := u.(Authentication)
			if !ok {
				return c.String(http.StatusUnauthorized, "Unauthorized")
			}
			if !IncludesAnyRole(authentication.Roles, roles...) {
				return c.String(http.StatusForbidden, "Forbidden")
			}
			return next(c)
		}
	}
}

func IncludesAnyRole(roles []string, role ...string) bool {
	for _, ur := range roles {
		for _, r := range roles {
			if strings.EqualFold(ur, r) {
				return true
			}
		}
	}
	return false
}
