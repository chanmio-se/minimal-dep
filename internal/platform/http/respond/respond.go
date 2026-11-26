// Package respond converts layer errors into HTTP responses.
package respond

import (
	"log/slog"
	"net/http"

	"minimal-dep/internal/platform/http/htmx"
)

// Error logs the internal error and writes a safe public message. HTMX
// requests receive the same body as normal requests so form fallbacks work.
func Error(w http.ResponseWriter, r *http.Request, logger *slog.Logger, status int, public string, err error) {
	if err != nil {
		logger.Error("request failed",
			"status", status,
			"method", r.Method,
			"path", r.URL.Path,
			"htmx", htmx.IsHTMXRequest(r),
			"error", err,
		)
	}
	if public == "" {
		public = http.StatusText(status)
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(public + "\n"))
}
