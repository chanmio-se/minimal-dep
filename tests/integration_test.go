package tests

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"minimal-dep/internal/app"
	"minimal-dep/internal/platform/config"
)

var (
	csrfPattern   = regexp.MustCompile(`name="csrf_token" value="([^"]+)"`)
	taskIDPattern = regexp.MustCompile(`action="/tasks/([a-f0-9]+)/complete"`)
)

func newServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	cfg := config.Config{
		Addr:            ":0",
		Env:             "test",
		LogLevel:        "error",
		SessionSecret:   "integration-test-secret",
		SessionTTL:      8 * time.Hour,
		MaxBodyBytes:    1 << 20,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: time.Second,
	}
	application, err := app.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(application.Handler())
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(server.Close)
	return server, &http.Client{Jar: jar}
}

func request(t *testing.T, client *http.Client, method, target string, form url.Values, htmx bool) (int, string, string) {
	t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(method, target, body)
	if err != nil {
		t.Fatal(err)
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, string(data), resp.Request.URL.Path
}

func csrfToken(t *testing.T, body string) string {
	t.Helper()
	match := csrfPattern.FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("CSRF token not found in body: %q", body)
	}
	return match[1]
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("cookie %q not found in %+v", name, cookies)
	return nil
}

func TestSecurityDefaultsInStandardChain(t *testing.T) {
	server, _ := newServer(t)
	handler := server.Config.Handler

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	wantHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for name, want := range wantHeaders {
		if got := rec.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if got := rec.Header().Get("Content-Security-Policy"); !strings.Contains(got, "default-src 'self'") {
		t.Errorf("Content-Security-Policy = %q", got)
	}

	oversized := httptest.NewRequest("POST", "/login", strings.NewReader(strings.Repeat("x", (1<<20)+1)))
	oversized.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, oversized)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("oversized body status = %d, want 413", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/login", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /login status = %d", rec.Code)
	}
	csrfCookie := findCookie(t, rec.Result().Cookies(), "csrf_token")
	if !csrfCookie.HttpOnly || csrfCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("CSRF cookie attributes = %+v", csrfCookie)
	}
	form := url.Values{
		"email": {"user@example.com"}, "name": {"Alice"},
		"csrf_token": {csrfToken(t, rec.Body.String())},
	}
	login := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
	login.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	login.AddCookie(csrfCookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, login)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/tasks" {
		t.Fatalf("login status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
	sessionCookie := findCookie(t, rec.Result().Cookies(), "session")
	if !sessionCookie.HttpOnly || sessionCookie.Secure ||
		sessionCookie.SameSite != http.SameSiteLaxMode || sessionCookie.MaxAge != int((8*time.Hour)/time.Second) {
		t.Fatalf("session cookie attributes = %+v", sessionCookie)
	}
}

func TestFullFlowWithHTMXAndFallback(t *testing.T) {
	server, client := newServer(t)

	status, body, _ := request(t, client, "GET", server.URL+"/healthz", nil, false)
	if status != http.StatusOK || !strings.Contains(body, `"status":"ok"`) {
		t.Fatalf("health status = %d, body = %q", status, body)
	}

	status, body, path := request(t, client, "GET", server.URL+"/tasks", nil, false)
	if status != http.StatusOK || path != "/login" {
		t.Fatalf("unauthenticated /tasks status = %d, path = %q", status, path)
	}
	token := csrfToken(t, body)

	login := url.Values{"email": {"user@example.com"}, "name": {"Alice"}, "csrf_token": {token}}
	status, body, path = request(t, client, "POST", server.URL+"/login", login, false)
	if status != http.StatusOK || path != "/tasks" || !strings.Contains(body, "タスク") {
		t.Fatalf("login status = %d, path = %q, body = %q", status, path, body)
	}
	token = csrfToken(t, body)

	create := url.Values{"title": {"buy milk"}, "csrf_token": {token}}
	status, body, _ = request(t, client, "POST", server.URL+"/tasks", create, false)
	if status != http.StatusOK || !strings.Contains(body, "buy milk") {
		t.Fatalf("create status = %d, body = %q", status, body)
	}

	match := taskIDPattern.FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("task id not found in body: %q", body)
	}
	token = csrfToken(t, body)

	status, body, _ = request(t, client, "GET", server.URL+"/tasks", nil, true)
	if status != http.StatusOK || !strings.Contains(body, `id="task-list"`) || strings.Contains(body, "<!doctype html>") {
		t.Fatalf("HTMX GET status = %d, body = %q", status, body)
	}

	status, body, _ = request(t, client, "GET", server.URL+"/tasks", nil, false)
	if status != http.StatusOK || !strings.Contains(body, "<!doctype html>") {
		t.Fatalf("normal GET status = %d, body = %q", status, body)
	}

	token = csrfToken(t, body)
	status, body, _ = request(t, client, "POST", fmt.Sprintf("%s/tasks/%s/complete", server.URL, match[1]),
		url.Values{"csrf_token": {token}}, true)
	if status != http.StatusOK || !strings.Contains(body, `class="task done"`) {
		t.Fatalf("HTMX complete status = %d, body = %q", status, body)
	}

	status, _, _ = request(t, client, "POST", server.URL+"/tasks", url.Values{"title": {"no csrf"}}, false)
	if status != http.StatusForbidden {
		t.Fatalf("missing CSRF token status = %d, want 403", status)
	}

	status, body, _ = request(t, client, "GET", server.URL+"/static/css/app.css", nil, false)
	if status != http.StatusOK || !strings.Contains(body, ".task-form") {
		t.Fatalf("static asset status = %d", status)
	}
}
