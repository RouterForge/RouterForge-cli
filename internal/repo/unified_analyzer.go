package repo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// AnalyzeToCodebase performs a single-pass analysis of a Go codebase
// and returns the complete semantic Codebase model with full traceability.
func (a *ASTAnalyzer) AnalyzeToCodebase(root string) (*models.Codebase, error) {
	cb := &models.Codebase{
		Root:     root,
		Language: "go",
	}

	pkgMap := make(map[string]*models.Package)
	importEdges := make(map[string]map[string]int)
	funcMap := make(map[string]map[string]*models.Function)
	allCallSites := []*models.CallSite{}

	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return err
		}
		if isVendorOrHidden(path) {
			return filepath.SkipDir
		}

		pkg, callSites, err := parseGoDir(path, root, a)
		if err != nil || pkg == nil {
			return nil
		}

		// deduplicate packages by path
		if _, exists := pkgMap[pkg.Path]; exists {
			return nil
		}
		pkgMap[pkg.Path] = pkg
		cb.Packages = append(cb.Packages, pkg)

		// collect imports as edges
		for _, imp := range pkg.Imports {
			target := shortenImport(imp.Path)
			if target == "" || target == pkg.Name {
				continue
			}
			if importEdges[pkg.Name] == nil {
				importEdges[pkg.Name] = make(map[string]int)
			}
			importEdges[pkg.Name][target]++
		}

		// index functions by name
		if funcMap[pkg.Name] == nil {
			funcMap[pkg.Name] = make(map[string]*models.Function)
		}
		for i := range pkg.Functions {
			fn := pkg.Functions[i]
			key := fn.Name
			if fn.Receiver != "" {
				key = fn.Receiver + "." + fn.Name
			}
			funcMap[pkg.Name][key] = fn
		}

		allCallSites = append(allCallSites, callSites...)

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk error: %w", err)
	}

	// assign call sites to their calling functions
	for _, cs := range allCallSites {
		for _, pkg := range cb.Packages {
			for _, fn := range pkg.Functions {
				if fn.Name == cs.Caller && fn.Package == cs.Caller {
					fn.Calls = append(fn.Calls, cs)
					break
				}
			}
		}
	}

	// build dependency graph
	cb.DependencyGraph = buildDepGraph(importEdges)

	// build call graph
	callNodes := make(map[string]bool)
	callEdges := []*models.CallSite{}
	for _, cs := range allCallSites {
		callNodes[cs.Caller] = true
		callNodes[cs.Callee] = true
		callEdges = append(callEdges, cs)
	}
	cb.CallGraph = &models.CallGraph{Nodes: mapKeys(callNodes), Edges: callEdges}

	// resolve interface implementors
	resolveImplementors(cb)

	// detect capabilities from the semantic model
	cb.Capabilities = detectCapabilities(cb)

	// fingerprint architecture from code relationships
	cb.Architecture = fingerprintArchitecture(cb, importEdges)

	return cb, nil
}

func parseGoDir(dir, root string, a *ASTAnalyzer) (*models.Package, []*models.CallSite, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}

	var result *models.Package
	var callSites []*models.CallSite

	for pkgName, pkg := range pkgs {
		if strings.HasSuffix(pkgName, "_test") {
			continue
		}

		relPath, _ := filepath.Rel(root, dir)
		p := &models.Package{
			Name: pkgName,
			Path: relPath,
			Location: models.Location{
				File: filepath.Join(relPath, "*.go"),
				Line: 1,
			},
		}

		importSeen := make(map[string]bool)

		for fileName, f := range pkg.Files {
			p.Files = append(p.Files, fileName)

			for _, imp := range f.Imports {
				if imp.Path == nil {
					continue
				}
				path := strings.Trim(imp.Path.Value, "\"")
				if importSeen[path] {
					continue
				}
				importSeen[path] = true

				alias := ""
				if imp.Name != nil {
					alias = imp.Name.Name
				}
				p.Imports = append(p.Imports, &models.Import{
					Path:  path,
					Alias: alias,
					Location: models.Location{
						File: fileName,
						Line: fset.Position(imp.Pos()).Line,
					},
				})
			}

			ast.Inspect(f, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.FuncDecl:
					fn := extractFunction(x, pkgName, fset)
					if fn != nil {
						p.Functions = append(p.Functions, fn)
					}

				case *ast.GenDecl:
					for _, spec := range x.Specs {
						switch ts := spec.(type) {
						case *ast.TypeSpec:
							loc := models.Location{
								File: fileName,
								Line: fset.Position(ts.Pos()).Line,
							}
							switch t := ts.Type.(type) {
							case *ast.InterfaceType:
								iface := extractInterface(ts, t, pkgName, fset, fileName)
								if iface != nil {
									p.Interfaces = append(p.Interfaces, iface)
								}
							case *ast.StructType:
								typ := &models.Type{
									Name:     ts.Name.Name,
									Kind:     "struct",
									Package:  pkgName,
									Exported: ts.Name.IsExported(),
									Location: loc,
								}
								for _, field := range t.Fields.List {
									f := extractField(field, pkgName, fset, fileName)
									if f != nil {
										typ.Fields = append(typ.Fields, f)
									}
								}
								p.Types = append(p.Types, typ)
							default:
								kind := fmt.Sprintf("%T", ts.Type)
								kind = strings.TrimPrefix(kind, "*ast.")
								p.Types = append(p.Types, &models.Type{
									Name:     ts.Name.Name,
									Kind:     kind,
									Package:  pkgName,
									Exported: ts.Name.IsExported(),
									Location: loc,
								})
							}
						}
					}

				case *ast.CallExpr:
					cs := extractCallSite(x, pkgName, fset, fileName)
					if cs != nil {
						callSites = append(callSites, cs)
					}
				}
				return true
			})
		}

		result = p
		break
	}

	return result, callSites, nil
}

func extractFunction(x *ast.FuncDecl, pkgName string, fset *token.FileSet) *models.Function {
	fn := &models.Function{
		Name:     x.Name.Name,
		Package:  pkgName,
		Exported: x.Name.IsExported(),
		Location: models.Location{
			File: fset.Position(x.Pos()).Filename,
			Line: fset.Position(x.Pos()).Line,
		},
	}

	if x.Recv != nil && len(x.Recv.List) > 0 {
		recv := x.Recv.List[0]
		recvType := typeExprToString(recv.Type)
		recvType = strings.TrimPrefix(recvType, "*")
		recvType = strings.TrimPrefix(recvType, "&")
		fn.Receiver = recvType
	}

	if x.Type.Params != nil {
		for _, p := range x.Type.Params.List {
			fn.Params = append(fn.Params, typeExprToString(p.Type))
		}
	}
	if x.Type.Results != nil {
		for _, r := range x.Type.Results.List {
			fn.Results = append(fn.Results, typeExprToString(r.Type))
		}
	}

	return fn
}

func extractInterface(ts *ast.TypeSpec, t *ast.InterfaceType, pkgName string, fset *token.FileSet, fileName string) *models.Interface {
	iface := &models.Interface{
		Name:    ts.Name.Name,
		Package: pkgName,
		Location: models.Location{
			File: fileName,
			Line: fset.Position(ts.Pos()).Line,
		},
	}
	for _, m := range t.Methods.List {
		if len(m.Names) > 0 {
			iface.Methods = append(iface.Methods, m.Names[0].Name)
		} else if se, ok := m.Type.(*ast.SelectorExpr); ok {
			if ident, ok := se.X.(*ast.Ident); ok {
				iface.Methods = append(iface.Methods, ident.Name+"."+se.Sel.Name)
			}
		}
	}
	return iface
}

func extractField(field *ast.Field, pkgName string, fset *token.FileSet, fileName string) *models.Field {
	if len(field.Names) == 0 {
		return nil
	}
	f := &models.Field{
		Name: field.Names[0].Name,
		Type: typeExprToString(field.Type),
		Location: models.Location{
			File: fileName,
			Line: fset.Position(field.Pos()).Line,
		},
		Exported: field.Names[0].IsExported(),
	}
	if field.Tag != nil {
		f.Tag = field.Tag.Value
	}
	return f
}

func extractCallSite(x *ast.CallExpr, pkgName string, fset *token.FileSet, fileName string) *models.CallSite {
	switch fun := x.Fun.(type) {
	case *ast.SelectorExpr:
		pkg := ""
		if ident, ok := fun.X.(*ast.Ident); ok {
			pkg = ident.Name
		}
		cs := &models.CallSite{
			Caller:   pkgName,
			Callee:   fun.Sel.Name,
			CallExpr: pkg + "." + fun.Sel.Name,
			Location: models.Location{
				File: fileName,
				Line: fset.Position(x.Pos()).Line,
			},
		}
		return cs
	case *ast.Ident:
		cs := &models.CallSite{
			Caller:   pkgName,
			Callee:   fun.Name,
			CallExpr: fun.Name,
			Location: models.Location{
				File: fileName,
				Line: fset.Position(x.Pos()).Line,
			},
		}
		return cs
	}
	return nil
}

func typeExprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeExprToString(t.X)
	case *ast.SelectorExpr:
		return typeExprToString(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeExprToString(t.Elt)
	case *ast.MapType:
		return "map[" + typeExprToString(t.Key) + "]" + typeExprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.Ellipsis:
		return "..." + typeExprToString(t.Elt)
	case *ast.FuncType:
		return "func(...)"
	case *ast.ChanType:
		return "chan " + typeExprToString(t.Value)
	default:
		return fmt.Sprintf("%s", expr)
	}
}

func buildDepGraph(edges map[string]map[string]int) *models.DepGraph {
	nodes := make(map[string]bool)
	var depEdges []*models.DepEdge

	for src, targets := range edges {
		nodes[src] = true
		for tgt, w := range targets {
			nodes[tgt] = true
			depEdges = append(depEdges, &models.DepEdge{
				Source: src,
				Target: tgt,
				Weight: w,
			})
		}
	}

	return &models.DepGraph{
		Nodes: mapKeys(nodes),
		Edges: depEdges,
	}
}

func resolveImplementors(cb *models.Codebase) {
	type methodSig struct {
		Name   string
		Params string
	}
	ifaceMethods := make(map[*models.Interface]map[methodSig]bool)
	for _, p := range cb.Packages {
		for _, iface := range p.Interfaces {
			ifaceMethods[iface] = make(map[methodSig]bool)
			for _, m := range iface.Methods {
				ifaceMethods[iface][methodSig{Name: m}] = true
			}
		}
	}

	typeMethods := make(map[string]map[methodSig]bool)
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			if fn.Receiver != "" {
				if typeMethods[fn.Receiver] == nil {
					typeMethods[fn.Receiver] = make(map[methodSig]bool)
				}
				typeMethods[fn.Receiver][methodSig{Name: fn.Name}] = true
			}
		}
	}

	for iface, reqMethods := range ifaceMethods {
		for typeName, typeMeths := range typeMethods {
			implements := true
			for m := range reqMethods {
				if !typeMeths[m] {
					implements = false
					break
				}
			}
			if implements && len(reqMethods) > 0 {
				for _, pkg := range cb.Packages {
					for _, ifc := range pkg.Interfaces {
						if ifc.Name == iface.Name {
							ifc.Implementors = append(ifc.Implementors, typeName)
						}
					}
				}
			}
		}
	}
}

func detectCapabilities(cb *models.Codebase) []*models.Capability {
	var caps []*models.Capability

	for _, pkg := range cb.Packages {
		for _, iface := range pkg.Interfaces {
			for _, m := range iface.Methods {
				if m == "ServeHTTP" {
					caps = addCap(caps, "http_handler", "Handles HTTP requests via ServeHTTP interface", "pattern", iface.Location)
				}
				if m == "Run" || m == "Start" {
					caps = addCap(caps, "runnable_service", "Has a runnable entry point (Run/Start)", "pattern", iface.Location)
				}
			}
		}

		for _, fn := range pkg.Functions {
			if fn.Name == "main" && fn.Receiver == "" {
				caps = addCap(caps, "cli_entrypoint", "Has a main() entry point", "feature", fn.Location)
			}
			if fn.Name == "ServeHTTP" {
				caps = addCap(caps, "http_handler", "Has ServeHTTP handler method", "pattern", fn.Location)
			}
			if strings.HasPrefix(fn.Name, "Test") {
				caps = addCap(caps, "has_tests", "Contains test functions", "feature", fn.Location)
			}
			if strings.Contains(fn.Name, "Handler") || strings.Contains(fn.Name, "handler") {
				caps = addCap(caps, "request_handler", "Defines request handler functions", "pattern", fn.Location)
			}
			if strings.Contains(fn.Name, "Migrate") || strings.Contains(fn.Name, "migrate") {
				caps = addCap(caps, "database_migrations", "Has database migration functions", "feature", fn.Location)
			}
			if strings.Contains(fn.Name, "Middleware") || strings.Contains(fn.Name, "middleware") {
				caps = addCap(caps, "middleware", "Defines HTTP middleware", "pattern", fn.Location)
			}
		}

		for _, imp := range pkg.Imports {
			loc := imp.Location
			switch {
			case containsAny(imp.Path, "net/http", "gin", "echo", "fiber", "mux", "chi"):
				caps = addCap(caps, "http_server", "Uses an HTTP server framework", "framework", loc)
			case containsAny(imp.Path, "database/sql", "sqlx", "gorm", "ent", "pg", "mongo", "redis", "dynamo"):
				caps = addCap(caps, "database", "Uses a database driver or ORM", "framework", loc)
			case containsAny(imp.Path, "encoding/json", "jsoniter", "protobuf", "grpc"):
				caps = addCap(caps, "serialization", "Uses structured serialization", "framework", loc)
			case containsAny(imp.Path, "kafka", "amqp", "rabbitmq", "nats", "mqtt"):
				caps = addCap(caps, "message_queue", "Uses message queue or event bus", "integration", loc)
			case containsAny(imp.Path, "prometheus", "otel", "opentelemetry", "sentry", "datadog"):
				caps = addCap(caps, "observability", "Uses observability/monitoring tools", "integration", loc)
			case containsAny(imp.Path, "test", "mock", "suite", "assert"):
				caps = addCap(caps, "testing_framework", "Uses testing/mocking libraries", "framework", loc)
			case containsAny(imp.Path, "oauth", "jwt", "crypto", "auth", "token"):
				caps = addCap(caps, "authentication", "Uses authentication/authorization", "feature", loc)
			case containsAny(imp.Path, "yaml", "toml", "viper", "env", "config"):
				caps = addCap(caps, "configuration", "Uses configuration management", "feature", loc)
			case containsAny(imp.Path, "cli", "cobra", "flag", "pflag", "urfave"):
				caps = addCap(caps, "cli_framework", "Uses a CLI framework", "framework", loc)
			case containsAny(imp.Path, "graphql", "gqlgen", "gopherql"):
				caps = addCap(caps, "graphql", "Has GraphQL support", "feature", loc)
			}
		}

		for _, typ := range pkg.Types {
			if typ.Kind == "struct" && len(typ.Fields) > 0 {
				for _, f := range typ.Fields {
					if strings.Contains(f.Tag, `json:`) {
						caps = addCap(caps, "json_models", "Has JSON-serializable data models", "feature", typ.Location)
						break
					}
				}
			}
		}
	}

	return dedupCaps(caps)
}

func addCap(caps []*models.Capability, name, desc, category string, loc models.Location) []*models.Capability {
	for _, c := range caps {
		if c.Name == name {
			c.Sources = append(c.Sources, loc)
			c.Evidence = append(c.Evidence, desc)
			return caps
		}
	}
	return append(caps, &models.Capability{
		Name:        name,
		Description: desc,
		Confidence:  0.9,
		Category:    category,
		Sources:     []models.Location{loc},
		Evidence:    []string{desc},
	})
}

func dedupCaps(caps []*models.Capability) []*models.Capability {
	seen := make(map[string]bool)
	var result []*models.Capability
	for _, c := range caps {
		if !seen[c.Name] {
			seen[c.Name] = true
			result = append(result, c)
		}
	}
	return result
}

func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func mapKeys(m map[string]bool) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
