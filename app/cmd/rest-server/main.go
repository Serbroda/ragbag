package main

import (
	"context"
	"fmt"
	"github.com/Serbroda/ragbag"
	db2 "github.com/Serbroda/ragbag/app/pkg/db"
	handlers2 "github.com/Serbroda/ragbag/app/pkg/handlers"
	"github.com/Serbroda/ragbag/app/pkg/services"
	"github.com/Serbroda/ragbag/app/pkg/utils"
	echoSwagger "github.com/swaggo/echo-swagger"

	_ "github.com/Serbroda/ragbag/app/docs"
	"github.com/Serbroda/ragbag/app/gen"
	"github.com/Serbroda/ragbag/app/gen/public"
	"github.com/Serbroda/ragbag/app/gen/restricted"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/teris-io/shortid"
)

var (
	distDirFS     = echo.MustSubFS(ragbag.FrontendDist, "frontend/dist")
	distIndexHtml = echo.MustSubFS(ragbag.IndexHTML, "frontend/dist")
)

var (
	version       string
	serverAddress = utils.GetEnvFallback("SERVER_URL", "0.0.0.0:8080")
	dbName        = utils.GetEnvFallback("DB_NAME", "ragbag")
	jwtSecretKey  = utils.MustGetEnv("JWT_SECRET_KEY")
)

// @title Echo Swagger Example API
// @version 1.0
// @description This is a sample server server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:3000
// @BasePath /
// @schemes http
func main() {
	fmt.Println("version=", version)

	db2.OpenAndConfigure("sqlite", dbName, ragbag.Migrations, "app/resources/db/migrations/sqlite")

	services := services.New(db2.Queries)
	db2.InitializeAdmin(context.Background(), services)

	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	shortid.SetDefault(sid)

	e := echo.New()
	//e.Use(middleware.Logger())
	//e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(50)))
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	registerHandlers(e, db2.Queries, services)

	e.Logger.Fatal(e.Start(serverAddress))
}

func registerHandlers(e *echo.Echo, queries *gen.Queries, services *services.Services) {
	registerStaticHandlers(e)
	registerApiHandlers(e, queries, services)
}

func registerStaticHandlers(e *echo.Echo) {
	e.StaticFS("/", distDirFS)
	e.FileFS("/", "index.html", distIndexHtml)
}

func registerApiHandlers(e *echo.Echo, queries *gen.Queries, services *services.Services) {
	api := e.Group("/api")

	// public api
	public.RegisterHandlers(api, &handlers2.PublicServerInterfaceImpl{
		Services: services,
		Queries:  queries,
	})

	// restricted api
	restr := api.Group("")
	restr.Use(middleware.JWTWithConfig(middleware.JWTConfig{
		Claims:     &handlers2.JwtCustomClaims{},
		SigningKey: []byte(jwtSecretKey),
	}))
	restricted.RegisterHandlers(restr, &handlers2.RestrictedServerInterfaceImpl{})
}

func getDsn(user, password, address, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true", user, password, address, database)
}