package archlintplugin

import (
	"os"
	"regexp"
	"testing"

	"github.com/golangci/plugin-module-register/register"
)

func TestPluginRegistered(t *testing.T) {
	if _, err := register.GetPlugin(linterName); err != nil {
		t.Fatalf("plugin %q is not registered: %v", linterName, err)
	}
}

func TestAnalyzerRunsDespiteTypeErrors(t *testing.T) {
	analyzers, err := (plugin{}).BuildAnalyzers()
	if err != nil {
		t.Fatal(err)
	}
	if len(analyzers) != 1 {
		t.Fatalf("BuildAnalyzers() returned %d analyzers, want 1", len(analyzers))
	}
	if !analyzers[0].RunDespiteErrors {
		t.Fatal("analyzer must run even when another linter reports type errors")
	}
}

func TestGolangciLintVersionsMatch(t *testing.T) {
	mod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	modVersion := regexp.MustCompile(`(?m)^[ \t]*github\.com/golangci/golangci-lint/v2[ \t]+(v[^\s]+)`).FindSubmatch(mod)
	if modVersion == nil {
		t.Fatal("go.mod does not pin golangci-lint/v2")
	}

	custom, err := os.ReadFile("../../.custom-gcl.yml")
	if err != nil {
		t.Fatal(err)
	}
	customVersion := regexp.MustCompile(`(?m)^version:\s*(v[^\s]+)`).FindSubmatch(custom)
	if customVersion == nil {
		t.Fatal(".custom-gcl.yml does not pin a version")
	}

	if string(modVersion[1]) != string(customVersion[1]) {
		t.Fatalf("golangci-lint version mismatch: go.mod=%s .custom-gcl.yml=%s", modVersion[1], customVersion[1])
	}
}
