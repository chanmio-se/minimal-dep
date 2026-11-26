// Package handler handles identity HTTP requests.
package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"minimal-dep/internal/modules/identity/usecase"
	"minimal-dep/internal/platform/http/cookie"
	"minimal-dep/internal/platform/http/htmx"
	"minimal-dep/internal/platform/http/middleware"
	"minimal-dep/internal/platform/http/respond"
	"minimal-dep/internal/platform/templates"
)

// Handler exposes identity endpoints.
type Handler struct {
	usecase   *usecase.Usecase
	renderer  *templates.Renderer
	logger    *slog.Logger
	cookieCfg cookie.Session
}

// New builds an identity handler.
func New(uc *usecase.Usecase, renderer *templates.Renderer, logger *slog.Logger, cookieCfg cookie.Session) *Handler {
	return &Handler{usecase: uc, renderer: renderer, logger: logger, cookieCfg: cookieCfg}
}

type loginView struct {
	CSRFToken string
	Error     string
}

// LoginPage renders the login form.
func (h *Handler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, r, loginView{CSRFToken: middleware.CSRFTokenFor(r)})
}

// Login authenticates the user and sets a session cookie.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		status := http.StatusBadRequest
		if middleware.IsBodyTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
		}
		respond.Error(w, r, h.logger, status, "invalid form", err)
		return
	}
	email := strings.TrimSpace(r.PostFormValue("email"))
	name := strings.TrimSpace(r.PostFormValue("name"))
	if !validLoginInput(email, name) {
		respond.Error(w, r, h.logger, http.StatusBadRequest,
			"email must be 1-254 characters and name 1-100 characters", nil)
		return
	}

	previous, _ := cookie.Read(r, h.cookieCfg.Name)
	token, err := h.usecase.LoginWithSession(r.Context(), email, name, previous)
	if err != nil {
		status := http.StatusInternalServerError
		message := "login failed"
		if errors.Is(err, usecase.ErrInvalidInput) {
			status, message = http.StatusBadRequest, "email and name are required"
		}
		respond.Error(w, r, h.logger, status, message, err)
		return
	}
	cookie.Set(w, h.cookieCfg, token)
	http.Redirect(w, r, "/tasks", http.StatusSeeOther)
}

// Logout revokes the session and removes the cookie.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if token, ok := cookie.Read(r, h.cookieCfg.Name); ok {
		if err := h.usecase.Logout(r.Context(), token); err != nil {
			respond.Error(w, r, h.logger, http.StatusInternalServerError, "logout failed", err)
			return
		}
	}
	cookie.Clear(w, h.cookieCfg)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (h *Handler) render(w http.ResponseWriter, r *http.Request, view loginView) {
	if err := htmx.Render(w, r, h.renderer, "app/layout", "identity/page/login", view); err != nil {
		respond.Error(w, r, h.logger, http.StatusInternalServerError, "render failed", err)
	}
}

func validLoginInput(email, name string) bool {
	return email != "" && len(email) <= 254 && strings.Contains(email, "@") &&
		name != "" && len(name) <= 100
}
