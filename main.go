package main

import (
	"embed"
	"fmt"

	"github.com/Serbroda/ragbag/gen/public"
	"github.com/Serbroda/ragbag/gen/restricted"
	"github.com/Serbroda/ragbag/pkg/database"
	"github.com/Serbroda/ragbag/pkg/handlers"
	"github.com/Serbroda/ragbag/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/teris-io/shortid"
)

var (
	//go:embed resources/db/migrations/*.sql
	migrations embed.FS

	//go:embed all:frontend/dist
	dist embed.FS
	//go:embed frontend/dist/index.html
	indexHTML     embed.FS
	distDirFS     = echo.MustSubFS(dist, "frontend/dist")
	distIndexHtml = echo.MustSubFS(indexHTML, "frontend/dist")
)

var (
	version       string
	serverAddress string = utils.GetEnvFallback("SERVER_URL", "0.0.0.0:8080")
	dbAddress     string = utils.MustGetEnv("DB_ADDRESS")
	dbName        string = utils.GetEnvFallback("DB_NAME", "ragbag")
	dbUser        string = utils.GetEnvFallback("DB_USER", "ragbag")
	dbPassword    string = utils.MustGetEnv("DB_PASSWORD")
)

func main() {
	fmt.Println("version=", version)

	database.OpenAndConfigure("mysql", getDsn(dbUser, dbPassword, dbAddress, dbName), migrations, "resources/db/migrations")

	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	shortid.SetDefault(sid)

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	registerHandlers(e)

	e.Logger.Fatal(e.Start(serverAddress))
}

func registerHandlers(e *echo.Echo) {
	registerStaticHandlers(e)
	registerApiHandlers(e)
}

func registerStaticHandlers(e *echo.Echo) {
	e.StaticFS("/", distDirFS)
	e.FileFS("/", "index.html", distIndexHtml)
}

func registerApiHandlers(e *echo.Echo) {
	baseUrl := "/api/v1"

	var publicApi handlers.PublicServerInterfaceImpl
	public.RegisterHandlersWithBaseURL(e, &publicApi, baseUrl)

	api := e.Group(baseUrl)
	config := middleware.JWTConfig{
		Claims:     &handlers.JwtCustomClaims{},
		SigningKey: []byte("secret"),
	}
	api.Use(middleware.JWTWithConfig(config))

	var restrictedApi handlers.RestrictedServerInterfaceImpl
	restricted.RegisterHandlers(api, &restrictedApi)
}

func getDsn(user, password, address, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", user, password, address, database)
}
