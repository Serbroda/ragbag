// Package web embeds the built frontend (ragbag-frontend → web/dist, see the makefile).
package web

import "embed"

//go:embed all:dist
var Dist embed.FS
