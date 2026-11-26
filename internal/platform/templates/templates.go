// Package templates renders HTML with html/template.
// A Renderer owns one parsed template set: the shared layout plus the view
// templates of a single module.
package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
)

// Renderer executes named templates from a parsed template set.
type Renderer struct {
	tmpl *template.Template
}

// New parses layoutFile from layout and viewPatterns from views.
func New(layout fs.FS, layoutFile string, views fs.FS, viewPatterns ...string) (*Renderer, error) {
	tmpl := template.New("views")
	if _, err := tmpl.ParseFS(layout, layoutFile); err != nil {
		return nil, fmt.Errorf("parse layout %s: %w", layoutFile, err)
	}
	for _, pattern := range viewPatterns {
		if _, err := tmpl.ParseFS(views, pattern); err != nil {
			return nil, fmt.Errorf("parse views %s: %w", pattern, err)
		}
	}
	if tmpl.Lookup("app/layout") == nil {
		return nil, fmt.Errorf("layout template app/layout is not defined")
	}
	return &Renderer{tmpl: tmpl}, nil
}

// Execute writes the named template to w.
func (r *Renderer) Execute(w io.Writer, name string, data any) error {
	t := r.tmpl.Lookup(name)
	if t == nil {
		return fmt.Errorf("template %q is not defined", name)
	}
	return t.Execute(w, data)
}
