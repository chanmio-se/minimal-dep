package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"minimal-dep/internal/modules/tasks/repository"
	"minimal-dep/internal/modules/tasks/usecase"
	"minimal-dep/internal/platform/http/middleware"
	"minimal-dep/internal/platform/templates"
)

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	renderer, err := templates.New(templates.LayoutFS(), "layouts/app.html", os.DirFS("../templates"),
		"pages/*.html", "fragments/*.html")
	if err != nil {
		t.Fatal(err)
	}
	uc := usecase.New(repository.NewMemory(), usecase.WithIDFunc(func() string { return "t1" }))
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(uc, renderer, logger)
	withPrincipal := middleware.Auth(func(*http.Request) (middleware.Principal, bool) {
		return middleware.Principal{ID: "u1"}, true
	}, nil)
	return withPrincipal(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tasks":
			h.Index(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			h.Create(w, r)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/tasks/"):
			h.Complete(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

func formRequest(method, path, key, value string) *http.Request {
	form := url.Values{key: {value}}
	req := httptest.NewRequest(method, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req
}

func TestIndexFullPageAndFragment(t *testing.T) {
	h := newTestHandler(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("full page status = %d, body = %q", rec.Code, rec.Body.String())
	}

	req := httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("HX-Request", "true")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="task-list"`) ||
		strings.Contains(rec.Body.String(), "<!doctype html>") {
		t.Fatalf("fragment status = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestCreateAndComplete(t *testing.T) {
	h := newTestHandler(t)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, formRequest("POST", "/tasks", "title", "buy milk"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/tasks" {
		t.Fatalf("create status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	if !strings.Contains(rec.Body.String(), "buy milk") {
		t.Fatalf("created task is missing: %q", rec.Body.String())
	}

	mux := http.NewServeMux()
	mux.Handle("POST /tasks/{id}/complete", h)
	req := httptest.NewRequest("POST", "/tasks/t1/complete", nil)
	req.Header.Set("HX-Request", "true")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `id="task-list"`) {
		t.Fatalf("complete fragment status = %d, body = %q", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "done") {
		t.Fatalf("completed task is not marked done: %q", rec.Body.String())
	}
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	h := newTestHandler(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, formRequest("POST", "/tasks", "title", " "))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestIndexWithoutPrincipal(t *testing.T) {
	renderer, err := templates.New(templates.LayoutFS(), "layouts/app.html", os.DirFS("../templates"),
		"pages/*.html", "fragments/*.html")
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := New(usecase.New(repository.NewMemory()), renderer, logger)
	rec := httptest.NewRecorder()
	h.Index(rec, httptest.NewRequest("GET", "/tasks", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
}
