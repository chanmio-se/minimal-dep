package quint

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"minimal-dep/internal/platform/http/middleware"
)

// TestCSRFTrace replays token issue/submit/verify traces through the real
// CSRF middleware. Model token names are aliases for the middleware's
// randomly generated cookie token.
func TestCSRFTrace(t *testing.T) {
	tr := loadTrace(t, "csrf")
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			fmt.Fprint(w, middleware.CSRFToken(r.Context()))
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	h := middleware.CSRF(false)(final)

	tokens := make(map[string]string) // model token name -> real token value
	cookie := ""
	submitted := ""
	tokenFor := func(alias string) string {
		if token, ok := tokens[alias]; ok {
			return token
		}
		return alias // c1/c2/bad/"" keep their literal value when unissued
	}

	for step, s := range tr.States {
		got := ""
		switch s.Action {
		case "init":
			got = "init"
		case "issue":
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/tasks", nil))
			cookie = cookieToken(t, step, rec)
			if rec.Body.String() != cookie || cookie == "" {
				t.Fatalf("step %d: issued token = %q, body = %q", step, cookie, rec.Body.String())
			}
			tokens[pickString(t, step, s, "token")] = cookie
			got = "issued"
		case "issueAgain":
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/tasks", nil)
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookie})
			h.ServeHTTP(rec, req)
			if body := rec.Body.String(); body != cookie {
				t.Fatalf("step %d: kept token = %q, body = %q", step, cookie, body)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Fatalf("step %d: safe request rotated the cookie", step)
			}
			got = "kept"
		case "verifyForm", "verifyHeader", "verifyBeforeIssue":
			submitted = tokenFor(pickString(t, step, s, "token"))
			form := url.Values{"csrf_token": {submitted}}
			req := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if s.Action == "verifyHeader" {
				req.Header.Set("X-CSRF-Token", submitted)
			}
			if cookie != "" {
				req.AddCookie(&http.Cookie{Name: "csrf_token", Value: cookie})
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			got = csrfCode(t, step, rec.Code)
			if len(rec.Result().Cookies()) != 0 {
				t.Fatalf("step %d: unsafe request changed the cookie", step)
			}
		default:
			t.Fatalf("step %d: unexpected action %q", step, s.Action)
		}
		assertCode(t, step, s.Last.Code, got)

		if cookie != tokenFor(s.Cookie) || submitted != tokenFor(s.Submitted) || cookie != tokenFor(s.Last.Token) {
			t.Fatalf("step %d: state = (cookie %q, submitted %q, last %q), want (%q, %q, %q)",
				step, cookie, submitted, s.Last.Token,
				tokenFor(s.Cookie), tokenFor(s.Submitted), tokenFor(s.Last.Token))
		}
	}
}

func cookieToken(t *testing.T, step int, rec *httptest.ResponseRecorder) string {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "csrf_token" {
			return c.Value
		}
	}
	t.Fatalf("step %d: no csrf_token cookie was issued", step)
	return ""
}

func csrfCode(t *testing.T, step int, status int) string {
	t.Helper()
	switch status {
	case http.StatusNoContent:
		return "allow"
	case http.StatusForbidden:
		return "deny"
	default:
		t.Fatalf("step %d: unexpected status %d", step, status)
		return ""
	}
}
