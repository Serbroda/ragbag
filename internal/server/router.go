package server

import (
	"github.com/Serbroda/ragbag/internal/handlers/api"
	"github.com/Serbroda/ragbag/internal/handlers/auth"
	"github.com/Serbroda/ragbag/internal/security"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/go-playground/validator/v10"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type Config struct {
	AuthService  services.AuthService
	SpaceService services.SpaceService
}

func NewServer(conf Config) *echo.Echo {
	e := echo.New()
	e.Validator = &CustomValidator{validator: validator.New()}
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	apiGroup := e.Group("/api")

	// register auth handlers
	authServer := auth.NewAuthServer(conf.AuthService)
	auth.RegisterHandlers(apiGroup, authServer)

	// register v1 server
	v1Group := apiGroup.Group("/v1")
	v1Group.Use(echojwt.WithConfig(security.CreateJwtConfig()))
	apiServer := api.NewApiServer(conf.AuthService, conf.SpaceService)
	api.RegisterHandlers(v1Group, apiServer)

	printRoutes(e)
	return e
}

func printRoutes(e *echo.Echo) {
	log.Debug("Registered following routes\n\n")
	for _, r := range e.Routes() {
		log.Debugf(" - %v %v\n", r.Method, r.Path)
	}
}
