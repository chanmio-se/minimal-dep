// Package cookie centralizes secure cookie attributes for sessions.
package cookie

import (
	"net/http"
	"time"
)

// Session describes the attributes applied to session cookies.
type Session struct {
	Name     string
	Secure   bool
	MaxAge   int
	SameSite http.SameSite
}

// Set writes a session cookie with secure defaults.
func Set(w http.ResponseWriter, cfg Session, token string) {
	http.SetCookie(w, newCookie(cfg, token))
}

// Clear expires the session cookie.
func Clear(w http.ResponseWriter, cfg Session) {
	c := newCookie(cfg, "")
	c.MaxAge = -1
	c.Expires = epoch()
	http.SetCookie(w, c)
}

// Read returns a cookie value by name.
func Read(r *http.Request, name string) (string, bool) {
	c, err := r.Cookie(name)
	if err != nil {
		return "", false
	}
	return c.Value, true
}

func epoch() time.Time { return time.Unix(0, 0) }

func newCookie(cfg Session, value string) *http.Cookie {
	sameSite := cfg.SameSite
	if sameSite == 0 {
		sameSite = http.SameSiteLaxMode
	}
	return &http.Cookie{
		Name: cfg.Name, Value: value, Path: "/",
		HttpOnly: true, Secure: cfg.Secure, SameSite: sameSite,
		MaxAge: cfg.MaxAge,
	}
}
