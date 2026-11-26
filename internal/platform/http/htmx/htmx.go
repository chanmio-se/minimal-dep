// Package htmx centralizes HTMX request detection and HTML response selection.
package htmx

import (
	"net/http"
	"strings"

	"minimal-dep/internal/platform/templates"
)

// IsHTMXRequest reports whether the request was sent by HTMX.
func IsHTMXRequest(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

// Name selects the fragment for HTMX requests and the full page otherwise.
func Name(r *http.Request, page, fragment string) string {
	if IsHTMXRequest(r) {
		return fragment
	}
	return page
}

// Render writes HTML for either a full document or an HTMX fragment.
func Render(w http.ResponseWriter, r *http.Request, renderer *templates.Renderer, page, fragment string, data any) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return renderer.Execute(w, Name(r, page, fragment), data)
}
