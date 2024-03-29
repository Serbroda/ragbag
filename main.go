package main

import (
	"embed"
	"fmt"
	"log"
	"net/http"

	"github.com/Serbroda/ragbag/pkg/database"
	"github.com/Serbroda/ragbag/pkg/handlers"
	"github.com/Serbroda/ragbag/pkg/middlewares"
	"github.com/Serbroda/ragbag/pkg/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/teris-io/shortid"
)

//go:embed frontend/dist
var reactApp embed.FS

var (
	version string
)

func main() {
	fmt.Println("version=", version)

	var dbName = utils.GetEnv("DB_NAME", "ragbag.db")
	var serverAddress = utils.GetEnv("SERVER_URL", "0.0.0.0:8080")

	sid, _ := shortid.New(1, shortid.DefaultABC, 2342)
	shortid.SetDefault(sid)

	database.Connect(database.ConnectionOptions{Name: dbName})

	app := fiber.New(fiber.Config{
		DisableKeepalive: true,
	})

	app.Use(cors.New())

	serveStatic(app)

	app.Get("/version", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).SendString(version)
	})
	app.Post("/login", handlers.Login)
	app.Post("/register", handlers.Register)
	setupApiV1(app)

	log.Fatal(app.Listen(serverAddress))
}

func serveStatic(app *fiber.App) {
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(reactApp),
		PathPrefix: "frontend/dist",
	}))
}

func setupApiV1(app *fiber.App) {
	api := app.Group("/api/v1")

	usersGroup := api.Group("/users")
	usersGroup.Get("/me", middlewares.JWTProtected(), handlers.GetMe)
	usersGroup.Patch("/change_password", middlewares.JWTProtected(), handlers.ChangePassword)

	groupsGroup := api.Group("/groups")
	groupsGroup.Get("/", middlewares.JWTProtected(), handlers.GetGroups)
	groupsGroup.Get("/latest", middlewares.JWTProtected(), handlers.GetLatestGroups)

	groupsGroup.Get("/subscriptions", middlewares.JWTProtected(), handlers.GetGroupSubscriptions)
	groupsGroup.Post("/subscriptions/:groupId", middlewares.JWTProtected(), handlers.CreateGroupSubscription)
	groupsGroup.Delete("/subscriptions/:groupId", middlewares.JWTProtected(), handlers.DeleteGroupSubscription)

	groupsGroup.Get("/public/:groupId", handlers.GetGroupPublic)
	groupsGroup.Get("/public/:groupId/links", handlers.GetLinksPublic)

	groupsGroup.Get("/:groupId", middlewares.JWTProtected(), handlers.GetGroup)
	groupsGroup.Post("/", middlewares.JWTProtected(), handlers.CreateGroup)
	groupsGroup.Patch("/:groupId", middlewares.JWTProtected(), handlers.UpdateGroup)
	groupsGroup.Delete("/:groupId", middlewares.JWTProtected(), handlers.DeleteGroup)
	groupsGroup.Get("/:groupId/links", middlewares.JWTProtected(), handlers.GetLinks)
	groupsGroup.Post("/:groupId/links", middlewares.JWTProtected(), handlers.CreateLink)
	groupsGroup.Put("/:groupId/visibility", middlewares.JWTProtected(), handlers.ChangeGroupVisibility)

	linksGroup := api.Group("/links")
	linksGroup.Get("/meta/:url", handlers.GetMetaInfo)
	linksGroup.Get("/", middlewares.JWTProtected(), handlers.GetLatestLinks)
	linksGroup.Get("/:linkId", middlewares.JWTProtected(), handlers.GetLink)
	linksGroup.Patch("/:linkId", middlewares.JWTProtected(), handlers.UpdateLink)
	linksGroup.Delete("/:linkId", middlewares.JWTProtected(), handlers.DeleteLink)
}
