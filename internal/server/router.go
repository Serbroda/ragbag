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
	AuthService       services.AuthService
	SpaceService      services.SpaceService
	CollectionService services.CollectionService
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

	ssi := api.NewApiServer(conf.AuthService, conf.SpaceService, conf.CollectionService)

	// optional: middlewares für strict handler (logging, authz mapping, etc.)
	var strictMW []api.StrictMiddlewareFunc

	handler := api.NewStrictHandler(ssi, strictMW)

	// Wichtig: beim v1Group registrieren, damit die Routes unter /api/v1 liegen
	api.RegisterHandlers(v1Group, handler)

	printRoutes(e)
	return e
}

func printRoutes(e *echo.Echo) {
	log.Info("Registered following routes\n\n")
	for _, r := range e.Routes() {
		log.Infof(" - %v %v\n", r.Method, r.Path)
	}
}
