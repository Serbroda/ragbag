// Package server implements the REST API (internal/api, generated from openapi/openapi.yaml) and serves the
// frontend.
package server

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/Serbroda/ragbag/internal/api"
	"github.com/Serbroda/ragbag/internal/auth"
	"github.com/Serbroda/ragbag/internal/config"
	"github.com/Serbroda/ragbag/internal/fetch"
	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	refreshCookie     = "refresh_token"
	refreshCookiePath = "/api/auth/refresh"
	userIDKey         = "userID"
	usernameKey       = "username"
)

type Server struct {
	cfg    config.Config
	store  *store.Store
	tokens *auth.Tokens
	fetch  *fetch.Client
	admins map[string]bool
	// version is written into exports and shown in the frontend
	version string
}

var _ api.ServerInterface = (*Server)(nil)

func New(cfg config.Config, st *store.Store, version string) *Server {
	admins := map[string]bool{}
	for _, name := range cfg.AdminUsers {
		admins[strings.ToLower(name)] = true
	}
	return &Server{
		cfg:     cfg,
		store:   st,
		tokens:  auth.NewTokens(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		fetch:   fetch.NewClient(cfg.AllowPrivateHosts),
		admins:  admins,
		version: version,
	}
}

// Handler wires the API under /api and the frontend under /app.
func (s *Server) Handler(frontend fs.FS) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = errorHandler
	// The client IP (logs): X-Forwarded-For only counts when it comes from a proxy in the own network (nginx
	// proxy manager & co.), otherwise anybody could put a fake address into the logs, e.g. to fool fail2ban
	e.IPExtractor = echo.ExtractIPFromXFFHeader()
	e.Use(s.logRequests)
	e.Use(middleware.Recover())
	// Uploads are limited to 5 MB, imports can be larger
	e.Use(middleware.BodyLimit("25M"))
	if len(s.cfg.CORSAllowedOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: s.cfg.CORSAllowedOrigins,
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
			// No AllowHeaders: the requested headers are echoed; browsers ignore "*" for requests with credentials
			AllowCredentials: true,
		}))
	}

	group := e.Group("/api", s.authenticate)
	api.RegisterHandlersWithBaseURL(group, s, "")
	group.Any("/*", func(c echo.Context) error { return echo.ErrNotFound })

	s.serveFrontend(e, frontend)
	return e
}

// publicPaths can be used without being signed in.
var publicPaths = []string{
	"/api/app/config/features", "/api/auth/register", "/api/auth/login", "/api/auth/refresh", "/api/auth/logout",
	"/api/auth/password-reset/*", "/api/v1/images/*", "/api/v1/shares/*",
}

func isPublic(requestPath string) bool {
	for _, pattern := range publicPaths {
		if matched, _ := path.Match(pattern, requestPath); matched {
			return true
		}
	}
	return false
}

// authenticate accepts valid access tokens whose token version was not revoked meanwhile.
func (s *Server) authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if c.Request().Method == http.MethodOptions || isPublic(c.Request().URL.Path) {
			return next(c)
		}
		header := c.Request().Header.Get(echo.HeaderAuthorization)
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			return unauthorized("Authentication required")
		}
		claims, err := s.tokens.Parse(token, auth.TypeAccess)
		if err != nil {
			return unauthorized("Invalid or expired token")
		}
		user, err := s.store.UserByID(c.Request().Context(), claims.Subject)
		if err != nil {
			return unauthorized("Invalid or expired token")
		}
		if claims.TokenVersion < user.TokenVersion {
			return unauthorized("Token has been revoked")
		}
		c.Set(userIDKey, user.ID)
		c.Set(usernameKey, user.Username)
		s.withUnlock(c, user)
		return next(c)
	}
}

func currentUser(c echo.Context) string {
	id, _ := c.Get(userIDKey).(string)
	return id
}

func ctxOf(c echo.Context) context.Context { return c.Request().Context() }

// issueTokens answers a login: a short-lived access token in the body and the refresh token as an httpOnly
// cookie only sent to the refresh endpoint.
func (s *Server) issueTokens(c echo.Context, user *store.User) error {
	access, err := s.tokens.Access(user.ID, user.Username, user.TokenVersion)
	if err != nil {
		return err
	}
	refresh, err := s.tokens.Refresh(user.ID, user.TokenVersion)
	if err != nil {
		return err
	}
	c.SetCookie(s.cookie(c, refresh, s.tokens.RefreshTTL()))
	return c.JSON(http.StatusOK, api.LoginResponse{AccessToken: access})
}

func (s *Server) cookie(c echo.Context, value string, maxAge time.Duration) *http.Cookie {
	cookie := &http.Cookie{
		Name:     refreshCookie,
		Value:    value,
		Path:     refreshCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Forced by configuration, otherwise whenever the browser talks HTTPS (directly or via a reverse proxy)
		Secure: s.cfg.ForceSecureCookie || c.Scheme() == "https",
	}
	if maxAge > 0 {
		cookie.Expires = time.Now().Add(maxAge)
		cookie.MaxAge = int(maxAge.Seconds())
	} else {
		cookie.MaxAge = -1
	}
	return cookie
}

func (s *Server) isAdmin(user *store.User) bool {
	return s.admins[strings.ToLower(user.Username)]
}

// serveFrontend serves the single page app under /app; unknown paths get index.html for client side routing.
func (s *Server) serveFrontend(e *echo.Echo, frontend fs.FS) {
	e.GET("/", func(c echo.Context) error { return c.Redirect(http.StatusFound, "/app/") })
	fileServer := http.StripPrefix("/app", http.FileServer(http.FS(frontend)))
	e.GET("/app", func(c echo.Context) error { return c.Redirect(http.StatusFound, "/app/") })
	e.GET("/app/*", func(c echo.Context) error {
		name := strings.TrimPrefix(c.Request().URL.Path, "/app/")
		if strings.HasPrefix(name, "share/") {
			// Public links are meant for the people they are sent to, not for search engines
			c.Response().Header().Set("X-Robots-Tag", "noindex, nofollow")
		}
		if name != "" {
			if info, err := fs.Stat(frontend, name); err == nil && !info.IsDir() {
				if strings.HasPrefix(name, "assets/") {
					// Vite puts content hashes into asset names
					c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fileServer.ServeHTTP(c.Response(), c.Request())
				return nil
			}
			if path.Ext(name) != "" {
				return echo.ErrNotFound
			}
		}
		index, err := fs.ReadFile(frontend, "index.html")
		if errors.Is(err, fs.ErrNotExist) {
			return c.String(http.StatusNotFound, "Frontend not built: run `make build-ui` (or use the Vite dev server)")
		}
		if err != nil {
			return err
		}
		c.Response().Header().Set("Cache-Control", "no-cache")
		return c.HTMLBlob(http.StatusOK, s.withMeta(c, name, index))
	})
}
