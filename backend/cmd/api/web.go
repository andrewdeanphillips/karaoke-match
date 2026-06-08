package main

import (
	"embed"
	"io/fs"
	"net/http"
)

// The container build overwrites web/dist with the frontend's compiled
// assets before this is compiled, so the embed bundles the real app in one
// binary. The committed placeholder at web/dist/index.html exists only so
// this directive has something to embed for local builds.
//
//go:embed web/dist
var embeddedFrontend embed.FS

// frontendHandler serves the embedded frontend build, rebased so that
// index.html and the asset directory sit at the filesystem root rather than
// under web/dist.
func frontendHandler() http.Handler {
	dist, err := fs.Sub(embeddedFrontend, "web/dist")
	if err != nil {
		panic(err) // unreachable: web/dist is guaranteed to exist by the embed directive above
	}
	return http.FileServerFS(dist)
}
