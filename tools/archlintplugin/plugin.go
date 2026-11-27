// Package archlintplugin adapts the architecture fitness functions to a
// golangci-lint module plugin. The rules themselves live in
// internal/linter/architecture; this package only forwards them to the
// go/analysis driver.
package archlintplugin

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"minimal-dep/internal/linter/architecture"
)

const linterName = "archlint"

var (
	rootOnce sync.Once
	rootPath string
	rootErr  error

	state              sync.Mutex
	graph              = architecture.NewGraph()
	modules            = map[string]bool{}
	cycles             = map[string]bool{}
	compositionChecked bool
)

type plugin struct{}

func init() {
	register.Plugin(linterName, func(conf any) (register.LinterPlugin, error) {
		return plugin{}, nil
	})
}

// GetLoadMode returns syntax-only: the fitness functions need ASTs, not types.
func (plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

// BuildAnalyzers exposes the fitness functions as one go/analysis analyzer.
func (plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{{
		Name:             linterName,
		Doc:              "checks module boundaries, dependency policy and static security rules",
		Run:              run,
		RunDespiteErrors: true,
	}}, nil
}

func run(pass *analysis.Pass) (any, error) {
	root, err := moduleRoot()
	if err != nil {
		return nil, err
	}

	for _, file := range pass.Files {
		filename := pass.Fset.Position(file.Pos()).Filename
		rel, err := filepath.Rel(root, filename)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		rel = filepath.ToSlash(rel)

		report(pass, pass.Fset, file, architecture.CheckSecurity(rel, pass.Fset, file))

		module, layer := architecture.Classify(rel)
		if module != "" {
			state.Lock()
			modules[module] = true
			state.Unlock()
		}

		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			line := pass.Fset.Position(spec.Pos()).Line
			state.Lock()
			violations := architecture.CheckImport(rel, line, module, layer, importPath, graph)
			state.Unlock()
			report(pass, pass.Fset, file, violations)
		}

		report(pass, pass.Fset, file, pendingCycles())
	}

	if len(pass.Files) > 0 {
		report(pass, pass.Fset, pass.Files[0], checkComposition(root))
	}
	return nil, nil
}

// pendingCycles returns cycles that have become visible after the imports of
// the current package were added. Reporting once per cycle keeps the result
// stable regardless of the order in which packages are analyzed.
func pendingCycles() []architecture.Violation {
	state.Lock()
	defer state.Unlock()

	var pending []architecture.Violation
	for _, violation := range architecture.FindCycles(graph, modules) {
		if cycles[violation.Message] {
			continue
		}
		cycles[violation.Message] = true
		pending = append(pending, violation)
	}
	return pending
}

func checkComposition(root string) []architecture.Violation {
	state.Lock()
	defer state.Unlock()

	if compositionChecked {
		return nil
	}
	compositionChecked = true
	return architecture.CheckSecurityComposition(root)
}

func report(pass *analysis.Pass, fset *token.FileSet, file *ast.File, violations []architecture.Violation) {
	for _, violation := range violations {
		pass.Reportf(position(fset, file, violation), "[%s] %s", violation.Rule, violation.Message)
	}
}

func position(fset *token.FileSet, file *ast.File, violation architecture.Violation) token.Pos {
	if violation.Path != "" && violation.Line > 0 {
		if tf := fset.File(file.Pos()); tf != nil && violation.Line <= tf.LineCount() {
			return tf.LineStart(violation.Line)
		}
	}
	return file.Pos()
}

func moduleRoot() (string, error) {
	rootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			rootErr = err
			return
		}
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				rootPath = dir
				return
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				rootPath = "."
				return
			}
			dir = parent
		}
	})
	return rootPath, rootErr
}
