// Package identity wires the identity module and registers its routes.
package identity

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"minimal-dep/internal/modules/identity/api"
	"minimal-dep/internal/modules/identity/handler"
	"minimal-dep/internal/modules/identity/repository"
	"minimal-dep/internal/modules/identity/usecase"
	"minimal-dep/internal/platform/http/cookie"
	"minimal-dep/internal/platform/session"
	"minimal-dep/internal/platform/templates"
)

//go:embed templates
var templateFS embed.FS

// Config holds the dependencies supplied by the composition root.
type Config struct {
	Repo          repository.Repository
	Logger        *slog.Logger
	SessionTTL    time.Duration
	SecureCookies bool
}

// Module is the identity module boundary.
type Module struct {
	handler *handler.Handler
	usecase *usecase.Usecase
}

// New assembles the identity module.
func New(cfg Config) (*Module, error) {
	if cfg.Repo == nil {
		return nil, errors.New("identity repository is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("identity logger is required")
	}
	if cfg.SessionTTL <= 0 {
		return nil, errors.New("identity session TTL must be positive")
	}
	renderer, err := templates.New(templates.LayoutFS(), "layouts/app.html", templateFS,
		"templates/pages/*.html")
	if err != nil {
		return nil, err
	}
	uc := usecase.New(cfg.Repo, session.New(cfg.SessionTTL))
	cookieCfg := cookie.Session{
		Name:     api.SessionCookieName,
		Secure:   cfg.SecureCookies,
		MaxAge:   int(cfg.SessionTTL / time.Second),
		SameSite: http.SameSiteLaxMode,
	}
	return &Module{
		handler: handler.New(uc, renderer, cfg.Logger, cookieCfg),
		usecase: uc,
	}, nil
}

// Register adds identity routes to mux.
func (m *Module) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /login", m.handler.LoginPage)
	mux.HandleFunc("POST /login", m.handler.Login)
	mux.HandleFunc("POST /logout", m.handler.Logout)
}

// API returns the public identity contract.
func (m *Module) API() api.Service { return m.usecase }
