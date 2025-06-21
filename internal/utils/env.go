package utils

import (
	"github.com/joho/godotenv"
	"os"
	"sync"
)

var (
	once sync.Once
)

func LoadEnv() {
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

func getEnv(key string) (string, bool) {
	once.Do(LoadEnv)
	return os.LookupEnv(key)
}

func GetEnvFallback(key, fallback string) string {
	if value, ok := getEnv(key); ok {
		return value
	}
	return fallback
}

func MustGetEnv(key string) string {
	if value, ok := getEnv(key); ok {
		return value
	}
	panic("mandatory env '" + key + "' not found")
}
