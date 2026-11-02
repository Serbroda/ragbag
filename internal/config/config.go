// Package config reads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/logging"
	"github.com/Serbroda/ragbag/internal/mail"
)

type Config struct {
	Addr string
	// DatabaseURL is "sqlite:<path>" or a "postgres://" URL.
	DatabaseURL string
	// DataDir holds files next to the database: the initial password of the setup admin
	DataDir string

	JWTSecret          []byte
	JWTIssuer          string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	ForceSecureCookie  bool
	RegistrationOpen   bool
	AllowPrivateHosts  bool
	AdminUsers         []string
	CORSAllowedOrigins []string
	// RateLimits slow down password guessing, bot sign-ups and floods (per IP, user and account)
	RateLimits bool
	Quotas     Quotas
	// Mail is used when Mail.Host is set: confirming new accounts and resetting passwords by email
	Mail mail.Config
	// PublicURL is the address the instance is reached at, for links in emails (never taken from a request)
	PublicURL string
	Log       logging.Config
}

// Quotas keep single accounts from filling the disk on public instances; 0 means unlimited.
type Quotas struct {
	// LinksPerUser counts the links a user added, trashed ones included
	LinksPerUser int
	// SpacesPerUser counts the spaces a user owns; the first one of a new account is always created
	SpacesPerUser       int
	CollectionsPerSpace int
	// UploadMBPerUser is the size of the uploaded images (avatar, space icons, preview images)
	UploadMBPerUser int
}

// Load reads RAGBAG_* and the documented short variables (see .env.example).
func Load() (Config, error) {
	cfg := Config{
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
		RateLimits:         envBool("RATE_LIMIT_ENABLED", true),
	}

	cfg.DataDir = DataDir(cfg.DatabaseURL)

	cfg.Quotas = Quotas{
		LinksPerUser:        envInt("QUOTA_LINKS_PER_USER", 0),
		SpacesPerUser:       envInt("QUOTA_SPACES_PER_USER", 0),
		CollectionsPerSpace: envInt("QUOTA_COLLECTIONS_PER_SPACE", 0),
		UploadMBPerUser:     envInt("QUOTA_UPLOAD_MB_PER_USER", 0),
	}

	cfg.Mail = mail.Config{
		Host:     env("SMTP_HOST", ""),
		Port:     envInt("SMTP_PORT", 587),
		Username: env("SMTP_USERNAME", ""),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     env("SMTP_FROM", ""),
	}
	cfg.Mail.TLS = mail.TLSStartTLS
	if cfg.Mail.Port == 465 {
		cfg.Mail.TLS = mail.TLSImplicit
	}
	cfg.Mail.TLS = strings.ToLower(env("SMTP_TLS", cfg.Mail.TLS))
	cfg.PublicURL = strings.TrimRight(env("PUBLIC_URL", ""), "/")
	if cfg.Mail.Host != "" {
		if cfg.Mail.From == "" {
			return cfg, errors.New("SMTP_FROM is required with SMTP_HOST, e.g. Ragbag <ragbag@example.com>")
		}
		if !strings.HasPrefix(cfg.PublicURL, "https://") && !strings.HasPrefix(cfg.PublicURL, "http://") {
			return cfg, errors.New("PUBLIC_URL is required with SMTP_HOST, e.g. https://ragbag.example.com (links in emails)")
		}
	}

	level, err := logging.ParseLevel(env("LOG_LEVEL", "info"))
	if err != nil {
		return cfg, err
	}
	format := strings.ToLower(strings.TrimSpace(env("LOG_FORMAT", "text")))
	if format != "text" && format != "json" {
		return cfg, fmt.Errorf("LOG_FORMAT must be text or json, not %q", format)
	}
	cfg.Log = logging.Config{
		Level:     level,
		JSON:      format == "json",
		File:      strings.TrimSpace(env("LOG_FILE", "")),
		MaxSizeMB: envInt("LOG_MAX_SIZE_MB", 10),
		MaxFiles:  envInt("LOG_MAX_FILES", 5),
	}

	secret := os.Getenv("JWT_SECRET")
	switch {
	case secret != "":
		if len(secret) < 32 {
			return cfg, errors.New("JWT_SECRET must be at least 32 characters (openssl rand -base64 32)")
		}
		cfg.JWTSecret = []byte(secret)
	default:
		return cfg, errors.New("JWT_SECRET is required (openssl rand -base64 32)")
	}
	return cfg, nil
}

// DataDir: DATA_DIR, else systemd's StateDirectory, else the SQLite database's directory, else ./data.
func DataDir(databaseURL string) string {
	if dir := env("DATA_DIR", ""); dir != "" {
		return dir
	}
	if dir, _, _ := strings.Cut(os.Getenv("STATE_DIRECTORY"), ":"); dir != "" {
		return dir
	}
	if path, ok := strings.CutPrefix(databaseURL, "sqlite:"); ok {
		return filepath.Dir(path)
	}
	return "data"
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
