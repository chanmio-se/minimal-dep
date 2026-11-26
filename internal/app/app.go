// Package app is the composition root: it builds modules, injects their
// dependencies, registers routes and runs the HTTP server.
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"minimal-dep/internal/modules/identity"
	identityapi "minimal-dep/internal/modules/identity/api"
	identityrepo "minimal-dep/internal/modules/identity/repository"
	"minimal-dep/internal/modules/tasks"
	tasksapi "minimal-dep/internal/modules/tasks/api"
	tasksrepo "minimal-dep/internal/modules/tasks/repository"
	"minimal-dep/internal/platform/config"
	"minimal-dep/internal/platform/database"
	"minimal-dep/internal/platform/http/cookie"
	"minimal-dep/internal/platform/http/middleware"
	"minimal-dep/internal/platform/logging"
	"minimal-dep/web"
)

// App is a fully wired application instance.
type App struct {
	cfg      config.Config
	logger   *slog.Logger
	db       *sql.DB
	server   *http.Server
	identity identityapi.Service
	tasks    tasksapi.Service
}

// New assembles the application. It does not start listening.
func New(cfg config.Config) (*App, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	logger := logging.New(cfg.Env, cfg.LogLevel)
	db, err := database.Open(cfg.DatabaseDriver, cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	var identityRepo identityrepo.Repository = identityrepo.NewMemory()
	var tasksRepo tasksrepo.Repository = tasksrepo.NewMemory()
	if db != nil {
		identityRepo = identityrepo.NewSQL(db)
		tasksRepo = tasksrepo.NewSQL(db)
	}

	identityModule, err := identity.New(identity.Config{
		Repo:          identityRepo,
		Logger:        logger,
		SessionTTL:    cfg.SessionTTL,
		SecureCookies: cfg.SecureCookies,
	})
	if err != nil {
		return nil, fmt.Errorf("build identity module: %w", err)
	}
	tasksModule, err := tasks.New(tasks.Config{Repo: tasksRepo, Logger: logger})
	if err != nil {
		return nil, fmt.Errorf("build tasks module: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))
	mux.HandleFunc("GET /healthz", healthHandler(logger, db))
	identityModule.Register(mux)
	tasksModule.Register(mux)

	auth := middleware.Auth(
		func(r *http.Request) (middleware.Principal, bool) {
			token, ok := cookie.Read(r, identityapi.SessionCookieName)
			if !ok {
				return middleware.Principal{}, false
			}
			principal, err := identityModule.API().Authenticate(r.Context(), token)
			if err != nil {
				return middleware.Principal{}, false
			}
			return middleware.Principal{ID: principal.ID, Name: principal.Name}, true
		},
		func(r *http.Request) bool {
			path := r.URL.Path
			return path == "/login" || path == "/healthz" || strings.HasPrefix(path, "/static/")
		},
	)

	handler := middleware.Chain(
		middleware.Recover(logger),
		middleware.RequestID,
		middleware.AccessLog(logger),
		middleware.SecurityHeaders,
		middleware.MaxBody(cfg.MaxBodyBytes),
		middleware.CSRF(cfg.SecureCookies),
		middleware.Timeout(cfg.RequestTimeout),
		auth,
	)(mux)

	return &App{
		cfg:      cfg,
		logger:   logger,
		db:       db,
		identity: identityModule.API(),
		tasks:    tasksModule.API(),
		server: &http.Server{
			Addr:              cfg.Addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		},
	}, nil
}

// Handler returns the fully wrapped HTTP handler for tests.
func (a *App) Handler() http.Handler { return a.server.Handler }

// Logger returns the application logger.
func (a *App) Logger() *slog.Logger { return a.logger }

// Services returns module APIs for cross-module use in tests and tooling.
func (a *App) Services() (identityapi.Service, tasksapi.Service) {
	return a.identity, a.tasks
}

// Migrate applies embedded migrations when a database is configured.
func (a *App) Migrate(ctx context.Context, fsys fs.FS, pattern string) error {
	return database.Migrate(ctx, a.db, fsys, pattern)
}

// Run starts the server and shuts it down when ctx is cancelled or a signal
// arrives.
func (a *App) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("server listening", "addr", a.cfg.Addr)
		err := a.server.ListenAndServe()
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			errCh <- nil
			return
		}
		errCh <- fmt.Errorf("serve: %w", err)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout)
		defer cancel()
		a.logger.Info("server shutting down")
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		if a.db != nil {
			if err := a.db.Close(); err != nil {
				return fmt.Errorf("close database: %w", err)
			}
		}
		return <-errCh
	}
}

func healthHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := "ok"
		httpStatus := http.StatusOK
		if err := database.Ping(ctx, db); err != nil {
			status = "unavailable"
			httpStatus = http.StatusServiceUnavailable
			logger.Warn("health check failed", "error", err)
		}
		storage := "memory"
		if db != nil {
			storage = "sql"
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  status,
			"storage": storage,
		})
	}
}
