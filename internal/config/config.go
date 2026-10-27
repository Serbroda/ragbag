// Package config reads the server configuration from environment variables.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Dev enables the demo account (admin/test123) and a generated JWT secret; never use it in production.
	Dev  bool
	Addr string
	// DatabaseURL is "sqlite:<path>" or a "postgres://" URL.
	DatabaseURL string

	JWTSecret          []byte
	JWTIssuer          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	ForceSecureCookie  bool
	RegistrationOpen   bool
	AllowPrivateHosts  bool
	AdminUsers         []string
	CORSAllowedOrigins []string
}

// Load reads RAGBAG_* and the documented short variables (see .env.example).
func Load() (Config, error) {
	cfg := Config{
		Dev:                envBool("RAGBAG_DEV", false),
		Addr:               env("RAGBAG_ADDR", ":8080"),
		DatabaseURL:        env("DATABASE_URL", "sqlite:data/ragbag.db"),
		JWTIssuer:          "ragbag",
		AccessTokenTTL:     time.Duration(envInt("ACCESS_TOKEN_MINUTES", 15)) * time.Minute,
		RefreshTokenTTL:    time.Duration(envInt("REFRESH_TOKEN_MINUTES", 20160)) * time.Minute,
		ForceSecureCookie:  envBool("SECURE_COOKIE", false),
		RegistrationOpen:   envBool("REGISTRATION_ENABLED", true),
		AllowPrivateHosts:  envBool("ALLOW_PRIVATE_HOSTS", false),
		AdminUsers:         envList("ADMIN_USERS"),
		CORSAllowedOrigins: envList("CORS_ALLOWED_ORIGINS"),
	}

	secret := os.Getenv("JWT_SECRET")
	switch {
	case secret != "":
		if len(secret) < 32 {
			return cfg, errors.New("JWT_SECRET must be at least 32 characters (openssl rand -base64 32)")
		}
		cfg.JWTSecret = []byte(secret)
	case cfg.Dev:
		// Sessions do not survive a restart in dev, which is fine
		random := make([]byte, 32)
		if _, err := rand.Read(random); err != nil {
			return cfg, err
		}
		cfg.JWTSecret = []byte(base64.StdEncoding.EncodeToString(random))
	default:
		return cfg, errors.New("JWT_SECRET is required (or set RAGBAG_DEV=true for local development)")
	}

	if cfg.Dev {
		if len(cfg.AdminUsers) == 0 {
			cfg.AdminUsers = []string{"admin"}
		}
		if len(cfg.CORSAllowedOrigins) == 0 {
			// The Vite dev server
			cfg.CORSAllowedOrigins = []string{"http://localhost:5173"}
		}
	}
	return cfg, nil
}

func env(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	// Some podman-compose versions pass an empty ${VAR:-} default through literally
	if strings.HasPrefix(value, "${") {
		slog.Warn(fmt.Sprintf("%s=%q was not interpolated by compose, ignoring it", key, value))
		return fallback
	}
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		slog.Warn(fmt.Sprintf("%s=%q is not a boolean, using %v", key, value, fallback))
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := env(key, "")
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		slog.Warn(fmt.Sprintf("%s=%q is not a positive number, using %d", key, value, fallback))
		return fallback
	}
	return parsed
}

func envList(key string) []string {
	var values []string
	for _, part := range strings.Split(env(key, ""), ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
