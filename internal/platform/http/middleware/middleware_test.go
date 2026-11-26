package middleware

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestChainOrder(t *testing.T) {
	var order []string
	mw := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+"-before")
				next.ServeHTTP(w, r)
				order = append(order, name+"-after")
			})
		}
	}
	final := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "final") })
	Chain(mw("outer"), mw("inner"))(final).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	want := "outer-before,inner-before,final,inner-after,outer-after"
	if got := strings.Join(order, ","); got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestRequestID(t *testing.T) {
	var fromContext string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromContext = RequestIDFrom(r.Context())
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if fromContext == "" || rec.Header().Get("X-Request-ID") != fromContext {
		t.Fatalf("generated id = %q, header = %q", fromContext, rec.Header().Get("X-Request-ID"))
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "incoming-id")
	h.ServeHTTP(rec, req)
	if got := rec.Header().Get("X-Request-ID"); got != "incoming-id" {
		t.Fatalf("X-Request-ID = %q", got)
	}
}

func TestSecurityHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	SecurityHeaders(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy is missing")
	}
}

func TestRecover(t *testing.T) {
	h := Recover(testLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCSRF(t *testing.T) {
	var seen string
	h := CSRF(false)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = CSRFToken(r.Context())
		fmt.Fprint(w, seen)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	token := rec.Body.String()
	var cookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "csrf_token" {
			cookie = c
		}
	}
	if cookie == nil || token == "" || cookie.Value != token || seen != token {
		t.Fatalf("token = %q, cookie = %+v, seen = %q", token, cookie, seen)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/tasks", nil)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("CSRF without token status = %d, want 403", rec.Code)
	}

	form := url.Values{"csrf_token": {token}}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("CSRF with token status = %d, want 200", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/tasks", nil)
	req.Header.Set("X-CSRF-Token", token)
	req.AddCookie(cookie)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("CSRF header status = %d, want 200", rec.Code)
	}
}

func TestMaxBody(t *testing.T) {
	h := MaxBody(8)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			if IsBodyTooLarge(err) {
				http.Error(w, "too large", http.StatusRequestEntityTooLarge)
				return
			}
			t.Errorf("unexpected body error: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/tasks", strings.NewReader("123456789"))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("ContentLength limit status = %d", rec.Code)
	}

	req = httptest.NewRequest("POST", "/tasks", strings.NewReader("123456789"))
	req.ContentLength = -1 // force streaming path through MaxBytesReader
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("streaming limit status = %d", rec.Code)
	}
}

func TestAuth(t *testing.T) {
	resolve := func(r *http.Request) (Principal, bool) {
		if r.Header.Get("X-User") == "" {
			return Principal{}, false
		}
		return Principal{ID: "u1", Name: "Alice"}, true
	}
	public := func(r *http.Request) bool { return r.URL.Path == "/login" }
	h := Auth(resolve, public)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		fmt.Fprint(w, p.ID)
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/login" {
		t.Fatalf("unauthenticated GET status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/tasks", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated POST status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/login", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("public path status = %d", rec.Code)
	}

	req := httptest.NewRequest("GET", "/tasks", nil)
	req.Header.Set("X-User", "u1")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Body.String() != "u1" {
		t.Fatalf("principal body = %q", rec.Body.String())
	}
}

func TestAccessLog(t *testing.T) {
	h := AccessLog(testLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/tasks", nil))
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d", rec.Code)
	}
}
