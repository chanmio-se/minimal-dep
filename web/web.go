// Package web exposes static assets embedded into the single binary.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var staticFS embed.FS

// StaticFS returns the static assets rooted at /static.
func StaticFS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic("static assets are missing: " + err.Error())
	}
	return sub
}
