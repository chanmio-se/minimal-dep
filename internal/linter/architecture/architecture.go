// Package architecture implements static fitness functions for the module
// boundaries, dependency directions and dependency policy.
package architecture

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const modulePath = "minimal-dep"

// approvedExternalDependencies records approved non-standard Go modules that
// production source may import. Tool dependencies such as golangci-lint are
// declared in go.mod but are not imports, so they are not listed here. The
// initial framework uses only the standard library in production.
var approvedExternalDependencies = map[string]bool{}

// Violation is one failed fitness function.
type Violation struct {
	Path    string
	Line    int
	Rule    string
	Message string
}

// String formats a violation for CLI output.
func (v Violation) String() string {
	if v.Path == "" {
		return fmt.Sprintf("[%s] %s", v.Rule, v.Message)
	}
	return fmt.Sprintf("%s:%d: [%s] %s", v.Path, v.Line, v.Rule, v.Message)
}

var allowedLayerImports = map[string]map[string]bool{
	"domain":     {},
	"api":        {"domain": true},
	"repository": {"domain": true},
	"usecase":    {"domain": true, "repository": true},
	"handler":    {"domain": true, "usecase": true, "api": true},
	"module":     {"domain": true, "api": true, "repository": true, "usecase": true, "handler": true},
}

// Lint checks all Go files below root and returns every violation found.
func Lint(root string) ([]Violation, error) {
	var violations []Violation
	graph := make(map[string]map[string]bool)
	modules := make(map[string]bool)

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		violations = append(violations, checkSecurity(rel, fset, file)...)
		module, layer := classify(rel)
		if module != "" {
			modules[module] = true
		}
		for _, spec := range file.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			line := fset.Position(spec.Pos()).Line
			violations = append(violations, checkImport(rel, line, module, layer, importPath, graph)...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	violations = append(violations, findCycles(graph, modules)...)
	violations = append(violations, checkSecurityComposition(root)...)
	return violations, nil
}

// checkSecurity implements the static security fitness functions: HTML must
// be rendered through html/template, and SQL calls must not concatenate query
// text from expressions other than literals and constants.
func checkSecurity(rel string, fset *token.FileSet, file *ast.File) []Violation {
	var violations []Violation
	add := func(node ast.Node, rule, message string) {
		violations = append(violations, Violation{
			Path: rel, Line: fset.Position(node.Pos()).Line, Rule: rule, Message: message,
		})
	}

	// The module generator renders source code, not HTTP responses.
	isGenerator := strings.HasPrefix(rel, "internal/generator/")
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err == nil && importPath == "text/template" && !isGenerator {
			add(spec, "html-template-only", "use html/template so output is escaped")
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		queryArg := 0
		switch sel.Sel.Name {
		case "Exec", "Query", "QueryRow":
			queryArg = 0
		case "ExecContext", "QueryContext", "QueryRowContext":
			queryArg = 1
		default:
			return true
		}
		if len(call.Args) > queryArg && !safeSQLExpr(call.Args[queryArg]) {
			add(call, "sql-parameterized", fmt.Sprintf("%s must use a constant query and parameters", sel.Sel.Name))
		}
		return true
	})
	return violations
}

func safeSQLExpr(expr ast.Expr) bool {
	switch x := expr.(type) {
	case *ast.BasicLit:
		return x.Kind == token.STRING
	case *ast.Ident:
		return true
	case *ast.ParenExpr:
		return safeSQLExpr(x.X)
	case *ast.BinaryExpr:
		return x.Op == token.ADD && safeSQLExpr(x.X) && safeSQLExpr(x.Y)
	default:
		return false
	}
}

// checkSecurityComposition verifies that the standard HTTP chain includes the
// security middleware required by the architecture.
func checkSecurityComposition(root string) []Violation {
	path := filepath.Join(root, "internal", "app", "app.go")
	src, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return []Violation{{Rule: "security-middleware", Message: "cannot read composition root: " + err.Error()}}
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return []Violation{{Rule: "security-middleware", Message: "cannot parse composition root: " + err.Error()}}
	}
	required := []string{"SecurityHeaders", "MaxBody", "CSRF", "Auth"}
	found := make(map[string]bool)
	ast.Inspect(file, func(node ast.Node) bool {
		sel, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "middleware" {
			found[sel.Sel.Name] = true
		}
		return true
	})

	var violations []Violation
	for _, name := range required {
		if !found[name] {
			violations = append(violations, Violation{
				Path: "internal/app/app.go", Rule: "security-middleware",
				Message: "standard HTTP chain is missing middleware." + name,
			})
		}
	}
	return violations
}

func classify(rel string) (module, layer string) {
	parts := strings.Split(rel, "/")
	if len(parts) < 3 || parts[0] != "internal" || parts[1] != "modules" {
		return "", ""
	}
	module = parts[2]
	if len(parts) == 3 || parts[3] == "module.go" {
		return module, "module"
	}
	return module, parts[3]
}

func checkImport(rel string, line int, module, layer, importPath string, graph map[string]map[string]bool) []Violation {
	var violations []Violation
	add := func(rule, message string) {
		violations = append(violations, Violation{Path: rel, Line: line, Rule: rule, Message: message})
	}

	if first := strings.Split(importPath, "/")[0]; strings.Contains(first, ".") && !approvedExternalDependencies[importPath] {
		add("dependency-policy", fmt.Sprintf("unapproved external dependency %q", importPath))
	}

	const modulesRoot = modulePath + "/internal/modules/"
	if strings.HasPrefix(rel, "internal/platform/") && strings.HasPrefix(importPath, modulesRoot) {
		add("platform-isolation", fmt.Sprintf("platform must not import %q", importPath))
	}
	// The composition root, commands and cross-module tests wire concrete
	// modules, so boundary and layer rules apply only inside module packages.
	if module == "" || !strings.HasPrefix(importPath, modulesRoot) {
		return violations
	}

	rest := strings.TrimPrefix(importPath, modulesRoot)
	parts := strings.Split(rest, "/")
	other := parts[0]
	if other == module {
		target := "module"
		if len(parts) > 1 {
			target = parts[1]
		}
		// Tests may construct collaborators from any layer of their own module.
		if !strings.HasSuffix(rel, "_test.go") && !allowedLayerImports[layer][target] {
			add("layer-direction", fmt.Sprintf("layer %q must not import %q", layer, target))
		}
		return violations
	}

	if len(parts) < 2 || parts[1] != "api" {
		add("module-boundary", fmt.Sprintf("module %q must depend on %q only through its api package", module, other))
		return violations
	}
	if graph[module] == nil {
		graph[module] = make(map[string]bool)
	}
	graph[module][other] = true
	return violations
}

func findCycles(graph map[string]map[string]bool, modules map[string]bool) []Violation {
	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)

	state := make(map[string]int)
	var stack []string
	var cycles []string

	var visit func(string)
	visit = func(name string) {
		state[name] = 1
		stack = append(stack, name)
		targets := make([]string, 0, len(graph[name]))
		for target := range graph[name] {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		for _, target := range targets {
			switch state[target] {
			case 0:
				visit(target)
			case 1:
				start := 0
				for i, n := range stack {
					if n == target {
						start = i
						break
					}
				}
				path := append(append([]string{}, stack[start:]...), target)
				cycles = append(cycles, strings.Join(path, " -> "))
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = 2
	}
	for _, name := range names {
		if state[name] == 0 {
			visit(name)
		}
	}

	sort.Strings(cycles)
	violations := make([]Violation, 0, len(cycles))
	for _, cycle := range cycles {
		violations = append(violations, Violation{Rule: "module-cycle", Message: "module dependency cycle: " + cycle})
	}
	return violations
}
