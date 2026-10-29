package server

import (
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// secretPath matches the path segments that are secrets themselves: public share links, invites and password
// reset links. They never go into a log.
var secretPath = regexp.MustCompile(`(/(?:shares?|invites?|password-reset|reset-password)/)[^/]+`)

// logPath is the request path for logs: without the query (search terms stay private) and with secrets masked.
func logPath(path string) string {
	return secretPath.ReplaceAllString(path, "${1}***")
}

// logRequests writes one line per request: API calls at INFO (server errors at ERROR, with the cause), the
// frontend's files and images at DEBUG.
func (s *Server) logRequests(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		if err != nil {
			// Answers the request now, so the status below is the one the client gets
			c.Error(err)
		}
		req, status := c.Request(), c.Response().Status
		path := req.URL.Path
		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case req.Method == http.MethodOptions, !strings.HasPrefix(path, "/api/"), strings.HasPrefix(path, "/api/v1/images/"):
			level = slog.LevelDebug
		}
		attrs := []slog.Attr{
			slog.String("method", req.Method),
			slog.String("path", logPath(path)),
			slog.Int("status", status),
			slog.Float64("ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("ip", c.RealIP()),
		}
		if user, _ := c.Get(usernameKey).(string); user != "" {
			attrs = append(attrs, slog.String("user", user))
		}
		if err != nil && status >= http.StatusInternalServerError {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
		slog.LogAttrs(req.Context(), level, "request", attrs...)
		return nil
	}
}

// audit logs something users did that matters when troubleshooting (logins, memberships, deletions).
func audit(c echo.Context, level slog.Level, msg string, args ...any) {
	if user, _ := c.Get(usernameKey).(string); user != "" {
		args = append([]any{"user", user}, args...)
	}
	slog.Log(c.Request().Context(), level, msg, append(args, "ip", c.RealIP())...)
}
