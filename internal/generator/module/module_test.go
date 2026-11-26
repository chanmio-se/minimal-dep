package module

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerate(t *testing.T) {
	root := t.TempDir()
	guidance, err := Generate(root, "widgets")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	base := filepath.Join(root, "internal", "modules", "widgets")
	for _, rel := range []string{
		"api/api.go",
		"domain/item.go",
		"repository/repository.go",
		"repository/memory.go",
		"usecase/usecase.go",
		"usecase/usecase_test.go",
		"handler/handler.go",
		"handler/handler_test.go",
		"templates/pages/index.html",
		"templates/fragments/list.html",
		"module.go",
	} {
		if _, err := os.Stat(filepath.Join(base, filepath.FromSlash(rel))); err != nil {
			t.Errorf("generated file %s is missing: %v", rel, err)
		}
	}
	body, err := os.ReadFile(filepath.Join(base, "module.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "package widgets") || !strings.Contains(string(body), "GET /widgets") {
		t.Fatalf("unexpected module.go:\n%s", body)
	}
	if !strings.Contains(guidance, "widgetsModule.Register(mux)") {
		t.Fatalf("guidance = %q", guidance)
	}

	if _, err := Generate(root, "widgets"); err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("second Generate() error = %v, want overwrite protection", err)
	}
}

func TestGenerateRejectsInvalidNames(t *testing.T) {
	for _, name := range []string{"", "Widgets", "1widgets", "my-widgets", "../escape"} {
		if _, err := Generate(t.TempDir(), name); err == nil {
			t.Errorf("Generate(%q) must fail", name)
		}
	}
}
