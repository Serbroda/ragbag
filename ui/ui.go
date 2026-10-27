// Package ui embeds the frontend. The Svelte app lives in v1 (`make build-ui` builds it into v1/dist); a new UI
// generation could be developed next to it as v2.
package ui

import (
	"embed"
	"io/fs"
)

// v1/dist/.gitkeep is committed so the package compiles before the first frontend build.
//
//go:embed all:v1/dist
var dist embed.FS

// Frontend is the built app, served under /app.
func Frontend() fs.FS {
	frontend, err := fs.Sub(dist, "v1/dist")
	if err != nil {
		panic(err) // the embed pattern above guarantees the directory
	}
	return frontend
}
