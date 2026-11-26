package templates

import (
	"embed"
	"io/fs"
)

//go:embed layouts/app.html
var layoutFS embed.FS

// LayoutFS returns the shared layout templates.
func LayoutFS() fs.FS { return layoutFS }
