package main

import (
	"flag"
	"fmt"
	"github.com/Serbroda/ragbag/internal/db"
	"github.com/Serbroda/ragbag/internal/db/migrations"
	sqlc "github.com/Serbroda/ragbag/internal/db/sqlc/gen"
	"github.com/Serbroda/ragbag/internal/server"
	"github.com/Serbroda/ragbag/internal/services"
	"github.com/Serbroda/ragbag/internal/utils"
	"github.com/joho/godotenv"
	"github.com/teris-io/shortid"
	"os"
)

const defaultServerPort = "8080"

func init() {
	sid, err := shortid.New(1, shortid.DefaultABC, 2342)
	if err != nil {
		panic(err)
	}
	shortid.SetDefault(sid)

	environment := os.Getenv("ENV")
	if environment == "" {
		environment = "development"
	}

	_ = godotenv.Load(".env." + environment + ".local")
	if environment != "test" {
		_ = godotenv.Load(".env.local")
	}
	_ = godotenv.Load(".env." + environment)
	_ = godotenv.Load()
}

var Version = "dev" // Default-Wert

func main() {
	versionFlag := flag.Bool("version", false, "show program version")
	flag.BoolVar(versionFlag, "v", false, "shorthand for --version")
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	dialect := utils.GetEnvFallback("DB_DIALECT", "sqlite")

	// Datenbankverbindung öffnen
	con := db.OpenConnection(
		dialect,
		utils.MustGetEnv("DB_PATH"),
	)
	defer con.Close() // Ensure the database connection is closed when the program exits

	// Run database migrations
	migrations.Migrate(
		con,
		utils.GetEnvFallback("DB_DIALECT", "sqlite3"),
		migrations.MigrationsCommon,
		migrations.MigrationsCommonDir,
	)

	// Initialize SQLC queries
	queries := sqlc.New(con)

	spaceService := services.NewSpaceService(queries)
	authService := services.NewAuthService(queries, spaceService)

	// Setup and configure the HTTP server
	e := server.NewServer(server.Config{
		AuthService:  authService,
		SpaceService: spaceService,
	})

	// Determine the server port (use default if not set)
	port := utils.GetEnvFallback("SERVER_PORT", defaultServerPort)

	// Start the server and log any fatal errors
	e.Logger.Fatal(e.Start(":" + port))
}
