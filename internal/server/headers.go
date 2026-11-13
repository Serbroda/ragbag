package server

import (
	"strings"

	"github.com/labstack/echo/v4"
)

// The app loads its scripts and styles only from this server; the CSP keeps injected markup from running
// anything. Images come from anywhere: link previews and favicons are loaded by the browser from the linked
// sites (http as well, for homelab services). Inline styles stay allowed, the UI library sets style attributes.
const appCSP = "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob: https: http:; font-src 'self' data:; connect-src 'self' blob: data:; " +
	"manifest-src 'self'; object-src 'none'; base-uri 'none'; form-action 'self'; frame-src 'none'; " +
	"frame-ancestors 'none'"

// apiCSP: API answers are data, never pages.
const apiCSP = "default-src 'none'; frame-ancestors 'none'"

// hsts only goes out over HTTPS (also behind a proxy sending X-Forwarded-Proto), so instances reached by plain
// HTTP in a homelab keep working. No includeSubDomains: other services on the domain may still use HTTP.
const hsts = "max-age=31536000"

// securityHeaders sets the headers every response gets; handlers may override them (images use a sandbox).
func (s *Server) securityHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Response().Header()
		header.Set("X-Content-Type-Options", "nosniff")
		// Opening a saved link tells the site nothing about Ragbag (share links carry their token in the path)
		header.Set("Referrer-Policy", "no-referrer")
		header.Set("X-Frame-Options", "DENY")
		// Not for the window the bookmarklet opens: it closes itself after saving, which needs window.opener
		if c.Request().URL.Path != "/add" {
			header.Set("Cross-Origin-Opener-Policy", "same-origin")
		}
		header.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=(), "+
			"browsing-topics=()")
		if strings.HasPrefix(c.Request().URL.Path, "/api/") {
			header.Set("Content-Security-Policy", apiCSP)
			// Personal data: never kept by the browser or a proxy (images set their own caching)
			header.Set("Cache-Control", "no-store")
		} else {
			header.Set("Content-Security-Policy", appCSP)
		}
		if c.Scheme() == "https" {
			header.Set("Strict-Transport-Security", hsts)
		}
		return next(c)
	}
}
