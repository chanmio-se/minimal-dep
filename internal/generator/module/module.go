// Package module generates a new feature module skeleton.
package module

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

type spec struct {
	Name   string
	Pascal string
	LBrace string
	RBrace string
}

type target struct {
	path     string
	template string
}

// Generate writes a module skeleton under <root>/internal/modules/<name>.
// Existing files are never overwritten. It returns registration guidance for
// the composition root.
func Generate(root, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("invalid module name %q: use lowercase letters, digits and underscores, starting with a letter", name)
	}
	s := spec{
		Name:   name,
		Pascal: pascal(name),
		LBrace: "{{",
		RBrace: "}}",
	}

	targets := []target{
		{"api/api.go", "api.go.tmpl"},
		{"domain/item.go", "domain.go.tmpl"},
		{"repository/repository.go", "repository.go.tmpl"},
		{"repository/memory.go", "memory.go.tmpl"},
		{"usecase/usecase.go", "usecase.go.tmpl"},
		{"usecase/usecase_test.go", "usecase_test.go.tmpl"},
		{"handler/handler.go", "handler.go.tmpl"},
		{"handler/handler_test.go", "handler_test.go.tmpl"},
		{"templates/pages/index.html", "page.html.tmpl"},
		{"templates/fragments/list.html", "fragment.html.tmpl"},
		{"module.go", "module.go.tmpl"},
	}

	base := filepath.Join(root, "internal", "modules", name)
	paths := make([]string, len(targets))
	for i, t := range targets {
		paths[i] = filepath.Join(base, filepath.FromSlash(t.path))
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return "", fmt.Errorf("refusing to overwrite existing file %s", path)
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("check %s: %w", path, err)
		}
	}

	for i, t := range targets {
		tmpl, err := template.New(t.template).ParseFS(templateFS, "templates/"+t.template)
		if err != nil {
			return "", fmt.Errorf("parse %s: %w", t.template, err)
		}
		if err := os.MkdirAll(filepath.Dir(paths[i]), 0o755); err != nil {
			return "", fmt.Errorf("create directory for %s: %w", paths[i], err)
		}
		file, err := os.OpenFile(paths[i], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return "", fmt.Errorf("create %s: %w", paths[i], err)
		}
		if err := tmpl.Execute(file, s); err != nil {
			_ = file.Close()
			return "", fmt.Errorf("write %s: %w", paths[i], err)
		}
		if err := file.Close(); err != nil {
			return "", fmt.Errorf("close %s: %w", paths[i], err)
		}
	}

	return fmt.Sprintf(`Module %q created at %s.

Register it in internal/app/app.go (Composition Root):

  %sModule, err := %s.New(%s.Config{
      Repo:   %srepo.NewMemory(),
      Logger: logger,
  })
  if err != nil {
      return nil, err
  }
  %sModule.Register(mux)
  _ = %sModule.API()

Authentication, CSRF and the request body limit come from the standard middleware
chain in internal/app/app.go. The generated module includes handler security tests
for unauthenticated access, missing CSRF tokens and cross-owner authorization.
Replace the memory repository with a SQL repository when a database is configured.
`, name, base, name, name, name, name, name, name), nil
}

func pascal(name string) string {
	parts := strings.Split(name, "_")
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		b.WriteString(part[1:])
	}
	return b.String()
}
