package htmx

import (
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"minimal-dep/internal/platform/templates"
)

func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"true", true},
		{"TRUE", true},
		{"false", false},
		{"", false},
	}
	for _, tc := range tests {
		r := httptest.NewRequest("GET", "/tasks", nil)
		if tc.header != "" {
			r.Header.Set("HX-Request", tc.header)
		}
		if got := IsHTMXRequest(r); got != tc.want {
			t.Errorf("IsHTMXRequest(%q) = %v, want %v", tc.header, got, tc.want)
		}
	}
}

func TestRenderSelectsTemplate(t *testing.T) {
	fsys := fstest.MapFS{
		"layouts/app.html": &fstest.MapFile{Data: []byte(`{{define "app/layout"}}<html>{{template "content" .}}</html>{{end}}`)},
		"views/page.html":  &fstest.MapFile{Data: []byte(`{{define "content"}}<main>page</main>{{end}}{{define "fragment"}}<div>fragment</div>{{end}}`)},
	}
	renderer, err := templates.New(fsys, "layouts/app.html", fsys, "views/*.html")
	if err != nil {
		t.Fatal(err)
	}
	data := struct{}{}

	r := httptest.NewRequest("GET", "/tasks", nil)
	w := httptest.NewRecorder()
	if err := Render(w, r, renderer, "content", "fragment", data); err != nil {
		t.Fatal(err)
	}
	if got := w.Body.String(); got != "<main>page</main>" {
		t.Fatalf("normal response = %q", got)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	r = httptest.NewRequest("GET", "/tasks", nil)
	r.Header.Set("HX-Request", "true")
	w = httptest.NewRecorder()
	if err := Render(w, r, renderer, "content", "fragment", data); err != nil {
		t.Fatal(err)
	}
	if got := w.Body.String(); got != "<div>fragment</div>" {
		t.Fatalf("HTMX response = %q", got)
	}
}

func TestName(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if got := Name(r, "page", "fragment"); got != "page" {
		t.Fatalf("Name() = %q, want page", got)
	}
	r.Header.Set("HX-Request", "true")
	if got := Name(r, "page", "fragment"); got != "fragment" {
		t.Fatalf("Name() = %q, want fragment", got)
	}
}
