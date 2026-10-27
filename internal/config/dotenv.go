package config

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/joho/godotenv"
)

// dotenvFiles in order of precedence: a value from .env.local wins over the same value in .env.
var dotenvFiles = []string{".env.local", ".env"}

// LoadDotenv reads .env.local and .env from the working directory if they exist. Variables that are already set
// in the environment always win, so Docker and CI configuration is never overridden by a stray file.
func LoadDotenv() error {
	for _, file := range dotenvFiles {
		// godotenv.Load never overrides variables that are set already, which gives the order above
		err := godotenv.Load(file)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", file, err)
		}
		slog.Info("loaded environment file", "file", file)
	}
	return nil
}
