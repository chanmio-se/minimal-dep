// Package middleware implements HTTP cross-cutting concerns.
package middleware

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middlewares in order: the first one is the outermost.
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		h := final
		for i := len(middlewares) - 1; i >= 0; i-- {
			h = middlewares[i](h)
		}
		return h
	}
}

type contextKey string

const (
	requestIDKey contextKey = "request-id"
	csrfTokenKey contextKey = "csrf-token"
	principalKey contextKey = "principal"
)

// RequestID accepts an incoming X-Request-ID or generates a random one.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = randomHex(16)
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequestIDFrom returns the request ID stored in ctx.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// AccessLog writes one structured log line per request.
func AccessLog(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(sw, r)
			logger.Info("access",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", RequestIDFrom(r.Context()),
			)
		})
	}
}

// Recover turns a panic into a 500 response and logs the panic value.
func Recover(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					logger.Error("panic recovered",
						"panic", v,
						"method", r.Method,
						"path", r.URL.Path,
						"request_id", RequestIDFrom(r.Context()),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeaders sets conservative default response headers.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

// CSRF implements the double-submit cookie pattern. Safe requests obtain a
// token in their cookie and context; unsafe requests must echo it in the
// csrf_token form field or X-CSRF-Token header.
func CSRF(secure bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("csrf_token")
			token := ""
			if err == nil && len(cookie.Value) == 64 {
				token = cookie.Value
			}

			if isSafe(r.Method) {
				if token == "" {
					token = randomHex(32)
					http.SetCookie(w, &http.Cookie{
						Name: "csrf_token", Value: token, Path: "/",
						HttpOnly: true, Secure: secure, SameSite: http.SameSiteLaxMode,
					})
				}
				ctx := context.WithValue(r.Context(), csrfTokenKey, token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			submitted := r.Header.Get("X-CSRF-Token")
			if submitted == "" {
				submitted = r.FormValue("csrf_token")
			}
			if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(submitted)) != 1 {
				http.Error(w, "invalid CSRF token", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// CSRFToken returns the token to embed in forms.
func CSRFToken(ctx context.Context) string {
	token, _ := ctx.Value(csrfTokenKey).(string)
	return token
}

// CSRFTokenFor returns the token for a request, including unsafe requests
// whose context does not carry it (for example when rendering an HTMX
// fragment in response to a POST).
func CSRFTokenFor(r *http.Request) string {
	if token := CSRFToken(r.Context()); token != "" {
		return token
	}
	if cookie, err := r.Cookie("csrf_token"); err == nil {
		return cookie.Value
	}
	return ""
}

// MaxBody limits the request body size. Oversized requests fail immediately
// when Content-Length is known and with http.MaxBytesError while streaming.
func MaxBody(limit int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limit {
				http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IsBodyTooLarge reports whether err came from the request body size limit.
func IsBodyTooLarge(err error) bool {
	var maxErr *http.MaxBytesError
	return errors.As(err, &maxErr)
}

// Timeout bounds the total time spent handling a request.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, d, "request timeout\n")
	}
}

// Principal is the authenticated caller, provided by an application module.
type Principal struct {
	ID   string
	Name string
}

// Auth resolves a principal and rejects unauthenticated requests to non-public
// paths. GET and HEAD requests are redirected to /login.
func Auth(resolve func(*http.Request) (Principal, bool), public func(*http.Request) bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolve != nil {
				if p, ok := resolve(r); ok {
					ctx := context.WithValue(r.Context(), principalKey, p)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			if public != nil && public(r) {
				next.ServeHTTP(w, r)
				return
			}
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}

// PrincipalFrom returns the authenticated caller stored in ctx.
func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

func isSafe(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func randomHex(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(buf)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}
