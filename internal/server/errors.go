package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/Serbroda/ragbag/internal/store"
	"github.com/labstack/echo/v4"
)

// apiError is answered as RFC 9457 problem details; the frontend shows `detail`.
type apiError struct {
	status int
	title  string
	detail string
}

func (e *apiError) Error() string { return e.detail }

func badRequest(format string, args ...any) error {
	return &apiError{http.StatusBadRequest, "Invalid request", fmt.Sprintf(format, args...)}
}

func unauthorized(detail string) error {
	return &apiError{http.StatusUnauthorized, "Unauthorized", detail}
}

func forbidden(format string, args ...any) error {
	return &apiError{http.StatusForbidden, "Access denied", fmt.Sprintf(format, args...)}
}

func notFound(format string, args ...any) error {
	return &apiError{http.StatusNotFound, "Resource not found", fmt.Sprintf(format, args...)}
}

func conflict(format string, args ...any) error {
	return &apiError{http.StatusConflict, "Conflict", fmt.Sprintf(format, args...)}
}

// orNotFound turns a missing row into a 404 and passes other errors through.
func orNotFound(err error, format string, args ...any) error {
	if errors.Is(err, store.ErrNotFound) {
		return notFound(format, args...)
	}
	return err
}

type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance"`
}

func errorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}
	p := problem{Type: "about:blank", Instance: c.Request().URL.Path}
	var (
		apiErr  *apiError
		httpErr *echo.HTTPError
	)
	switch {
	case errors.As(err, &apiErr):
		p.Status, p.Title, p.Detail = apiErr.status, apiErr.title, apiErr.detail
	case errors.As(err, &httpErr):
		// Routing (404/405), request binding and body size errors
		p.Status, p.Title = httpErr.Code, http.StatusText(httpErr.Code)
		if message, ok := httpErr.Message.(string); ok && httpErr.Code != http.StatusInternalServerError {
			p.Detail = message
		}
	default:
		// Logged with its cause by logRequests
		p.Status, p.Title = http.StatusInternalServerError, "Internal Server Error"
	}
	c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
	if c.Request().Method == http.MethodHead {
		_ = c.NoContent(p.Status)
		return
	}
	_ = c.JSON(p.Status, p)
}

// bind reads a JSON body; malformed JSON is a 400 with a readable message.
func bind(c echo.Context, target any) error {
	if err := (&echo.DefaultBinder{}).BindBody(c, target); err != nil {
		return badRequest("Malformed request body")
	}
	return nil
}
