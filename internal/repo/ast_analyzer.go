package repo

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ASTAnalyzer struct{}

type PackageInfo struct {
	Name      string          `json:"name"`
	Path      string          `json:"path"`
	Functions []FunctionInfo  `json:"functions"`
	Types     []TypeInfo      `json:"types"`
	Imports   []string        `json:"imports"`
	Interfaces []InterfaceInfo `json:"interfaces"`
}

type FunctionInfo struct {
	Name      string   `json:"name"`
	Params    string   `json:"params"`
	Results   string   `json:"results"`
	IsExported bool    `json:"is_exported"`
	Line      int      `json:"line"`
}

type TypeInfo struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	IsExported bool   `json:"is_exported"`
	Line      int    `json:"line"`
}

type InterfaceInfo struct {
	Name    string   `json:"name"`
	Methods []string `json:"methods"`
	Line    int      `json:"line"`
}

type DepEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

type DepGraph struct {
	Nodes []string   `json:"nodes"`
	Edges []DepEdge  `json:"edges"`
}

func (a *ASTAnalyzer) AnalyzeGoPackage(path string) (*PackageInfo, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	info := &PackageInfo{
		Path: path,
		Imports: []string{},
	}

	for pkgName, pkg := range pkgs {
		info.Name = pkgName
		for _, f := range pkg.Files {
			for _, imp := range f.Imports {
				if imp.Path != nil {
					path := strings.Trim(imp.Path.Value, "\"")
					if !contains(info.Imports, path) {
						info.Imports = append(info.Imports, path)
					}
						}
			}

			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					params := ""
					if x.Type.Params != nil {
						var ps []string
						for _, p := range x.Type.Params.List {
							ps = append(ps, fmt.Sprintf("%s", p.Type))
						}
						params = strings.Join(ps, ", ")
					}
					results := ""
					if x.Type.Results != nil {
						var rs []string
						for _, r := range x.Type.Results.List {
							rs = append(rs, fmt.Sprintf("%s", r.Type))
						}
						results = strings.Join(rs, ", ")
					}
					info.Functions = append(info.Functions, FunctionInfo{
						Name:       x.Name.Name,
						Params:     params,
						Results:    results,
						IsExported: x.Name.IsExported(),
						Line:       fset.Position(x.Pos()).Line,
					})

				case *ast.GenDecl:
					for _, spec := range x.Specs {
						switch ts := spec.(type) {
						case *ast.TypeSpec:
							switch ts.Type.(type) {
							case *ast.InterfaceType:
								iface := ts.Type.(*ast.InterfaceType)
								methods := []string{}
								for _, m := range iface.Methods.List {
									if len(m.Names) > 0 {
										methods = append(methods, m.Names[0].Name)
									}
								}
								info.Interfaces = append(info.Interfaces, InterfaceInfo{
									Name:    ts.Name.Name,
									Methods: methods,
									Line:    fset.Position(ts.Pos()).Line,
								})
							default:
								kind := fmt.Sprintf("%T", ts.Type)
								kind = strings.TrimPrefix(kind, "*ast.")
								info.Types = append(info.Types, TypeInfo{
									Name:       ts.Name.Name,
									Kind:       kind,
									IsExported: ts.Name.IsExported(),
									Line:       fset.Position(ts.Pos()).Line,
								})
							}
						}
					}
				}
				return true
			})
		}
	}

	return info, nil
}

func (a *ASTAnalyzer) AnalyzeGoRepo(root string) ([]*PackageInfo, *DepGraph, error) {
	var pkgs []*PackageInfo
	graph := &DepGraph{}

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

		pkg, err := a.AnalyzeGoPackage(path)
		if err != nil {
			return nil
		}
		pkgs = append(pkgs, pkg)
		graph.Nodes = append(graph.Nodes, pkg.Name)

		for _, imp := range pkg.Imports {
			short := shortenImport(imp)
			if short != "" {
				graph.Edges = append(graph.Edges, DepEdge{
					Source: pkg.Name,
					Target: short,
					Weight: 1,
				})
			}
		}
		return nil
	})

	return pkgs, graph, err
}

func (a *ASTAnalyzer) DetectCapabilitiesFromAST(pkgs []*PackageInfo) []Pattern {
	var patterns []Pattern

	for _, pkg := range pkgs {
		for _, iface := range pkg.Interfaces {
			if containsMethod(iface, "ServeHTTP") {
				patterns = addPattern(patterns, "http_handler", "Handles HTTP requests", []string{iface.Name})
			}
			if containsMethod(iface, "Run") || containsMethod(iface, "Start") {
				patterns = addPattern(patterns, "runnable_service", "Has a runnable entry point", []string{iface.Name})
			}
		}
		for _, fn := range pkg.Functions {
			if fn.Name == "main" {
				patterns = addPattern(patterns, "cli_entrypoint", "Has a main entry point", []string{pkg.Path})
			}
			if strings.Contains(fn.Name, "Handler") || strings.Contains(fn.Name, "handler") {
				patterns = addPattern(patterns, "request_handler", "Has request handler functions", []string{fn.Name})
			}
			if strings.HasPrefix(fn.Name, "Test") {
				patterns = addPattern(patterns, "has_tests", "Has test functions", []string{fn.Name})
			}
		}
		for _, imp := range pkg.Imports {
			if strings.Contains(imp, "net/http") {
				patterns = addPattern(patterns, "http_server", "Uses net/http", []string{pkg.Name})
			}
			if strings.Contains(imp, "database/sql") || strings.Contains(imp, "sql") {
				patterns = addPattern(patterns, "database", "Uses database", []string{pkg.Name})
			}
			if strings.Contains(imp, "encoding/json") {
				patterns = addPattern(patterns, "json_api", "Uses JSON encoding", []string{pkg.Name})
			}
		}
	}

	return patterns
}

func (g *DepGraph) ToJSON() string {
	b, _ := json.MarshalIndent(g, "", "  ")
	return string(b)
}

func (g *DepGraph) Markdown() string {
	var b strings.Builder
	b.WriteString("## Dependency Graph\n\n")
	b.WriteString("```mermaid\ngraph TD\n")
	for _, e := range g.Edges {
		b.WriteString(fmt.Sprintf("  %s --> %s\n", e.Source, e.Target))
	}
	b.WriteString("```\n")
	return b.String()
}

func isVendorOrHidden(path string) bool {
	base := filepath.Base(path)
	return base == "vendor" || base == "node_modules" || strings.HasPrefix(base, ".")
}

func shortenImport(imp string) string {
	parts := strings.Split(imp, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return imp
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func containsMethod(iface InterfaceInfo, name string) bool {
	for _, m := range iface.Methods {
		if m == name {
			return true
		}
	}
	return false
}

func addPattern(patterns []Pattern, name, desc string, indicators []string) []Pattern {
	for i := range patterns {
		if patterns[i].Name == name {
			patterns[i].Indicators = append(patterns[i].Indicators, indicators...)
			return patterns
		}
	}
	return append(patterns, Pattern{
		Name:       name,
		Confidence: 0.9,
		Indicators: indicators,
	})
}
