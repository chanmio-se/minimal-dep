package templates

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderPageAndFragment(t *testing.T) {
	fsys := fstest.MapFS{
		"layouts/app.html":    &fstest.MapFile{Data: []byte(`{{define "app/layout"}}<html>{{template "content" .}}</html>{{end}}`)},
		"pages/index.html":    &fstest.MapFile{Data: []byte(`{{define "content"}}<h1>{{.Title}}</h1>{{template "list" .}}{{end}}`)},
		"fragments/list.html": &fstest.MapFile{Data: []byte(`{{define "list"}}<ul>{{range .Items}}<li>{{.}}</li>{{end}}</ul>{{end}}`)},
	}
	renderer, err := New(fsys, "layouts/app.html", fsys, "pages/*.html", "fragments/*.html")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	data := struct {
		Title string
		Items []string
	}{Title: "Tasks", Items: []string{"one", "<script>"}}

	var page bytes.Buffer
	if err := renderer.Execute(&page, "app/layout", data); err != nil {
		t.Fatalf("Execute(page) error = %v", err)
	}
	if !strings.HasPrefix(page.String(), "<html>") || !strings.Contains(page.String(), "<h1>Tasks</h1>") {
		t.Fatalf("page = %q", page.String())
	}
	if !strings.Contains(page.String(), "&lt;script&gt;") {
		t.Fatalf("html/template escaping failed: %q", page.String())
	}

	var fragment bytes.Buffer
	if err := renderer.Execute(&fragment, "list", data); err != nil {
		t.Fatalf("Execute(fragment) error = %v", err)
	}
	if strings.Contains(fragment.String(), "<html>") || !strings.HasPrefix(fragment.String(), "<ul>") {
		t.Fatalf("fragment = %q", fragment.String())
	}
}

func TestNewRequiresLayout(t *testing.T) {
	fsys := fstest.MapFS{"other.html": &fstest.MapFile{Data: []byte("x")}}
	if _, err := New(fsys, "missing.html", fsys); err == nil {
		t.Fatal("New() must fail when the layout is missing")
	}
}
