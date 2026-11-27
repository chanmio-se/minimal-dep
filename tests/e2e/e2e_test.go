//go:build e2e

package e2e

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"

	"minimal-dep/internal/app"
	"minimal-dep/internal/platform/config"
)

func TestHTMXSwapsInBrowser(t *testing.T) {
	cfg := config.Config{
		Addr:            ":0",
		Env:             "test",
		LogLevel:        "error",
		SessionSecret:   "e2e-integration-test-secret",
		SessionTTL:      8 * time.Hour,
		MaxBodyBytes:    1 << 20,
		RequestTimeout:  5 * time.Second,
		ShutdownTimeout: time.Second,
	}
	application, err := app.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(application.Handler())
	t.Cleanup(server.Close)

	// Prefer an installed Chrome/Chromium so CI does not download a browser;
	// rod falls back to its pinned Chromium when none is found.
	l := launcher.New()
	if bin, ok := launcher.LookPath(); ok {
		l = l.Bin(bin)
	}
	browser := rod.New().ControlURL(l.MustLaunch()).MustConnect()
	t.Cleanup(browser.MustClose)

	page := browser.MustPage(server.URL).Timeout(15 * time.Second)
	page.MustWaitLoad()
	if !page.MustEval("() => typeof window.htmx !== 'undefined'").Bool() {
		t.Fatal("htmx.min.js was not loaded")
	}

	// The login form is a plain POST and must keep working without HTMX.
	page.MustElement("#email").MustInput("user@example.com")
	page.MustElement("#name").MustInput("Alice")
	page.MustElement("form button[type=submit]").MustClick()
	page.MustElement("#task-list")

	// Count real client-side swaps so a full-page fallback cannot pass.
	page.MustEval(`() => {
		window.__swaps = 0;
		document.body.addEventListener("htmx:afterSwap", () => { window.__swaps += 1; });
	}`)

	page.MustElement("#title").MustInput("buy milk")
	page.MustElement("form.task-form button").MustClick()
	if got := page.MustElement("#task-list .task-title").MustText(); got != "buy milk" {
		t.Fatalf("task title = %q, want %q", got, "buy milk")
	}

	page.MustElement("#task-list form button").MustClick()
	page.MustElement("#task-list .task.done")
	if has, _, _ := page.Has("#task-list form"); has {
		t.Fatal("completed task still has a form")
	}

	if got := page.MustEval("() => window.__swaps").Int(); got != 2 {
		t.Fatalf("htmx swaps = %d, want 2", got)
	}
	if got := page.MustInfo().URL; got != server.URL+"/tasks" {
		t.Fatalf("URL after HTMX requests = %q, want %q", got, server.URL+"/tasks")
	}
}
