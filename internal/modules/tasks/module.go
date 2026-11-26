// Package tasks wires the tasks module and registers its routes.
package tasks

import (
	"embed"
	"errors"
	"log/slog"
	"net/http"

	"minimal-dep/internal/modules/tasks/api"
	"minimal-dep/internal/modules/tasks/handler"
	"minimal-dep/internal/modules/tasks/repository"
	"minimal-dep/internal/modules/tasks/usecase"
	"minimal-dep/internal/platform/templates"
)

//go:embed templates
var templateFS embed.FS

// Config holds the dependencies supplied by the composition root.
type Config struct {
	Repo   repository.Repository
	Logger *slog.Logger
}

// Module is the tasks module boundary.
type Module struct {
	handler *handler.Handler
	usecase *usecase.Usecase
}

// New assembles the tasks module.
func New(cfg Config) (*Module, error) {
	if cfg.Repo == nil {
		return nil, errors.New("tasks repository is required")
	}
	if cfg.Logger == nil {
		return nil, errors.New("tasks logger is required")
	}
	renderer, err := templates.New(templates.LayoutFS(), "layouts/app.html", templateFS,
		"templates/pages/*.html", "templates/fragments/*.html")
	if err != nil {
		return nil, err
	}
	uc := usecase.New(cfg.Repo)
	return &Module{
		handler: handler.New(uc, renderer, cfg.Logger),
		usecase: uc,
	}, nil
}

// Register adds task routes to mux.
func (m *Module) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /tasks", m.handler.Index)
	mux.HandleFunc("POST /tasks", m.handler.Create)
	mux.HandleFunc("POST /tasks/{id}/complete", m.handler.Complete)
}

// API returns the public tasks contract.
func (m *Module) API() api.Service { return m.usecase }
