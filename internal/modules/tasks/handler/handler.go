// Package handler handles task HTTP requests.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"minimal-dep/internal/modules/tasks/api"
	"minimal-dep/internal/modules/tasks/usecase"
	"minimal-dep/internal/platform/http/htmx"
	"minimal-dep/internal/platform/http/middleware"
	"minimal-dep/internal/platform/http/respond"
	"minimal-dep/internal/platform/templates"
)

// Handler exposes task endpoints.
type Handler struct {
	usecase  *usecase.Usecase
	renderer *templates.Renderer
	logger   *slog.Logger
}

// New builds a task handler.
func New(uc *usecase.Usecase, renderer *templates.Renderer, logger *slog.Logger) *Handler {
	return &Handler{usecase: uc, renderer: renderer, logger: logger}
}

type tasksView struct {
	Tasks     []api.Task
	CSRFToken string
}

// Index renders the task page, or the list fragment for HTMX requests.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	owner, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	tasks, err := h.usecase.List(r.Context(), owner.ID)
	if err != nil {
		respond.Error(w, r, h.logger, http.StatusInternalServerError, "could not load tasks", err)
		return
	}
	h.render(w, r, tasksView{Tasks: tasks, CSRFToken: middleware.CSRFTokenFor(r)})
}

// Create adds a task and returns either a redirect or the updated fragment.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	owner, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := r.ParseForm(); err != nil {
		status := http.StatusBadRequest
		if middleware.IsBodyTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
		}
		respond.Error(w, r, h.logger, status, "invalid form", err)
		return
	}
	title := strings.TrimSpace(r.PostFormValue("title"))
	if title == "" || len(title) > 200 {
		respond.Error(w, r, h.logger, http.StatusBadRequest, "title must be 1-200 characters", nil)
		return
	}
	if _, err := h.usecase.Create(r.Context(), owner.ID, title); err != nil {
		h.usecaseError(w, r, err)
		return
	}
	h.afterMutation(w, r)
}

// Complete completes a task and returns either a redirect or the fragment.
func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	owner, ok := middleware.PrincipalFrom(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.usecase.Complete(r.Context(), owner.ID, r.PathValue("id")); err != nil {
		h.usecaseError(w, r, err)
		return
	}
	h.afterMutation(w, r)
}

func (h *Handler) afterMutation(w http.ResponseWriter, r *http.Request) {
	if !htmx.IsHTMXRequest(r) {
		http.Redirect(w, r, "/tasks", http.StatusSeeOther)
		return
	}
	owner, _ := middleware.PrincipalFrom(r.Context())
	tasks, err := h.usecase.List(r.Context(), owner.ID)
	if err != nil {
		respond.Error(w, r, h.logger, http.StatusInternalServerError, "could not load tasks", err)
		return
	}
	h.render(w, r, tasksView{Tasks: tasks, CSRFToken: middleware.CSRFTokenFor(r)})
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, view tasksView) {
	if err := htmx.Render(w, r, h.renderer,
		"app/layout", "tasks/list", view); err != nil {
		respond.Error(w, r, h.logger, http.StatusInternalServerError, "render failed", err)
	}
}

func (h *Handler) usecaseError(w http.ResponseWriter, r *http.Request, err error) {
	status, message := http.StatusInternalServerError, "could not update tasks"
	switch {
	case errors.Is(err, usecase.ErrInvalidInput):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, usecase.ErrForbidden):
		status, message = http.StatusForbidden, "task belongs to another owner"
	case errors.Is(err, usecase.ErrNotFound):
		status, message = http.StatusNotFound, "task not found"
	}
	respond.Error(w, r, h.logger, status, message, err)
}
