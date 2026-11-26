package architecture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintDetectsBoundaryViolations(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"internal/modules/a/handler/handler.go": `package handler
import "minimal-dep/internal/modules/b/repository"
`,
		"internal/modules/a/usecase/usecase.go": `package usecase
import "minimal-dep/internal/modules/b/api"
`,
		"internal/modules/b/usecase/usecase.go": `package usecase
import "minimal-dep/internal/modules/a/api"
`,
		"internal/platform/http/http.go": `package http
import "minimal-dep/internal/modules/a/api"
`,
		"cmd/tool/main.go": `package main
import "example.org/unapproved/lib"
`,
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	violations, err := Lint(root)
	if err != nil {
		t.Fatalf("Lint() error = %v", err)
	}
	rules := make(map[string]bool)
	for _, v := range violations {
		rules[v.Rule] = true
	}
	for _, rule := range []string{"module-boundary", "module-cycle", "platform-isolation", "dependency-policy"} {
		if !rules[rule] {
			t.Errorf("rule %q was not detected in %+v", rule, violations)
		}
	}
}

func TestLintDetectsLayerViolation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "internal", "modules", "tasks", "handler", "handler.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `package handler
import "minimal-dep/internal/modules/tasks/repository"
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	violations, err := Lint(root)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range violations {
		if v.Rule == "layer-direction" {
			found = true
		}
	}
	if !found {
		t.Fatalf("layer-direction violation was not detected: %+v", violations)
	}
}

func TestLintDetectsSecurityViolations(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"internal/modules/a/repository/sql.go": `package repository
import (
	"context"
	"database/sql"
	"fmt"
)
func find(db *sql.DB, ctx context.Context, value string) {
	db.ExecContext(ctx, fmt.Sprintf("SELECT * FROM t WHERE x = %s", value))
}
`,
		"internal/modules/a/handler/handler.go": `package handler
import "text/template"
var _ = template.New
`,
		"internal/app/app.go": `package app
func New() {}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	violations, err := Lint(root)
	if err != nil {
		t.Fatal(err)
	}
	rules := make(map[string]bool)
	for _, v := range violations {
		rules[v.Rule] = true
	}
	for _, rule := range []string{"sql-parameterized", "html-template-only", "security-middleware"} {
		if !rules[rule] {
			t.Errorf("rule %q was not detected in %+v", rule, violations)
		}
	}
}

func TestLintPassesCleanTree(t *testing.T) {
	violations, err := Lint(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("repository has %d violations: %v", len(violations), violations)
	}
}
