package repo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type CallGraphEdge struct {
	Caller      string `json:"caller"`
	Callee      string `json:"callee"`
	CallerPkg   string `json:"caller_pkg"`
	CalleePkg   string `json:"callee_pkg"`
	Line        int    `json:"line"`
}

type CallGraph struct {
	Nodes []string        `json:"nodes"`
	Edges []CallGraphEdge `json:"edges"`
}

type ImportGraph struct {
	Nodes []string   `json:"nodes"`
	Edges []DepEdge  `json:"edges"`
}

func (a *ASTAnalyzer) BuildCallGraph(root string) (*CallGraph, error) {
	cg := &CallGraph{}
	nodeSet := make(map[string]bool)

	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if isVendorOrHidden(filepath.Dir(path)) {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		pkgName := f.Name.Name

		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			callerName := fmt.Sprintf("%s.%s", pkgName, sel.Sel.Name)
			if !nodeSet[callerName] {
				cg.Nodes = append(cg.Nodes, callerName)
				nodeSet[callerName] = true
			}

			var calleePkg string
			switch x := sel.X.(type) {
			case *ast.Ident:
				calleePkg = x.Name
			case *ast.SelectorExpr:
				calleePkg = fmt.Sprintf("%s.%s", x.X, x.Sel.Name)
			}

			calleeName := fmt.Sprintf("%s.%s", calleePkg, sel.Sel.Name)
			if !nodeSet[calleeName] {
				cg.Nodes = append(cg.Nodes, calleeName)
				nodeSet[calleeName] = true
			}

			cg.Edges = append(cg.Edges, CallGraphEdge{
				Caller:    callerName,
				Callee:    calleeName,
				CallerPkg: pkgName,
				CalleePkg: calleePkg,
				Line:      fset.Position(call.Pos()).Line,
			})
			return true
		})
		return nil
	})

	return cg, err
}

func (a *ASTAnalyzer) BuildImportGraph(root string) (*ImportGraph, error) {
	ig := &ImportGraph{}
	nodeSet := make(map[string]bool)
	edgeSet := make(map[string]bool)

	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return err
		}
		if isVendorOrHidden(path) {
			return filepath.SkipDir
		}

		hasGo := false
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				hasGo = true
				break
			}
		}
		if !hasGo {
			return nil
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		for pkgName, pkg := range pkgs {
			if !nodeSet[pkgName] {
				ig.Nodes = append(ig.Nodes, pkgName)
				nodeSet[pkgName] = true
			}
			for _, f := range pkg.Files {
				for _, imp := range f.Imports {
					if imp.Path == nil {
						continue
					}
					impPath := strings.Trim(imp.Path.Value, "\"")
					short := shortenImport(impPath)
					if short == "" {
						continue
					}
					edgeKey := pkgName + "->" + short
					if !edgeSet[edgeKey] {
						ig.Edges = append(ig.Edges, DepEdge{
							Source: pkgName,
							Target: short,
							Weight: 1,
						})
						edgeSet[edgeKey] = true
					}
					if !nodeSet[short] {
						ig.Nodes = append(ig.Nodes, short)
						nodeSet[short] = true
					}
				}
			}
		}
		return nil
	})

	return ig, err
}

func (cg *CallGraph) Markdown() string {
	var b strings.Builder
	b.WriteString("## Call Graph\n\n")
	b.WriteString("```mermaid\ngraph LR\n")
	seen := make(map[string]bool)
	for _, e := range cg.Edges {
		key := e.Caller + "->" + e.Callee
		if seen[key] {
			continue
		}
		seen[key] = true
		callerLabel := strings.ReplaceAll(e.Caller, ".", "_")
		calleeLabel := strings.ReplaceAll(e.Callee, ".", "_")
		b.WriteString(fmt.Sprintf("  %s[%s] --> %s[%s]\n", callerLabel, e.Caller, calleeLabel, e.Callee))
	}
	b.WriteString("```\n")
	return b.String()
}
