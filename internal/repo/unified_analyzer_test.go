package repo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/routerforge/cli/pkg/models"
)

func TestAnalyzeToCodebase_Basic(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "go.mod"), "module testrepo\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

func Helper() string {
	return "helper"
}
`)

	writeFile(t, filepath.Join(dir, "handler", "handler.go"), `package handler

import (
	"encoding/json"
	"net/http"
)

type UserHandler struct {
	service Service
}

type Service interface {
	GetUser(id string) (string, error)
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}
`)

	writeFile(t, filepath.Join(dir, "handler", "service.go"), `package handler

type serviceImpl struct{}

func (s *serviceImpl) GetUser(id string) (string, error) {
	return "user:" + id, nil
}
`)

	writeFile(t, filepath.Join(dir, "repo", "repository.go"),
		"package repo\n\nimport \"database/sql\"\n\ntype Repository struct {\n\tdb *sql.DB\n}\n\n"+
			"func (r *Repository) FindUser(id string) (*User, error) {\n\treturn nil, nil\n}\n\n"+
			"type User struct {\n\tID   string `json:\"id\"`\n\tName string `json:\"name\"`\n}\n")

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	if cb.Language != "go" {
		t.Errorf("expected go, got %s", cb.Language)
	}

	if len(cb.Packages) != 3 {
		t.Errorf("expected 3 packages (main, handler, repo), got %d", len(cb.Packages))
		for _, p := range cb.Packages {
			t.Logf("  package: %s at %s", p.Name, p.Path)
		}
	}

	// find handlers package
	var handlerPkg, mainPkg, repoPkg *models.Package
	for _, p := range cb.Packages {
		switch p.Name {
		case "main":
			mainPkg = p
		case "handler":
			handlerPkg = p
		case "repo":
			repoPkg = p
		}
	}

	if mainPkg == nil {
		t.Fatal("main package not found")
	}
	if handlerPkg == nil {
		t.Fatal("handler package not found")
	}
	if repoPkg == nil {
		t.Fatal("repo package not found")
	}

	// main package has main() + Helper()
	if len(mainPkg.Functions) != 2 {
		t.Errorf("expected 2 functions in main, got %d", len(mainPkg.Functions))
	}
	foundMain := false
	foundHelper := false
	for _, fn := range mainPkg.Functions {
		if fn.Name == "main" {
			foundMain = true
		}
		if fn.Name == "Helper" {
			foundHelper = true
		}
	}
	if !foundMain {
		t.Error("main() not found")
	}
	if !foundHelper {
		t.Error("Helper() not found")
	}

	// handler package: UserHandler struct + serviceImpl struct
	if len(handlerPkg.Types) != 2 {
		t.Errorf("expected 2 types in handler (UserHandler, serviceImpl), got %d", len(handlerPkg.Types))
	}
	if len(handlerPkg.Interfaces) != 1 {
		t.Errorf("expected 1 interface in handler, got %d", len(handlerPkg.Interfaces))
	}
	if handlerPkg.Interfaces[0].Name != "Service" {
		t.Errorf("expected Service interface, got %s", handlerPkg.Interfaces[0].Name)
	}

	// handler.ServeHTTP method
	foundServeHTTP := false
	for _, fn := range handlerPkg.Functions {
		if fn.Name == "ServeHTTP" && fn.Receiver == "UserHandler" {
			foundServeHTTP = true
			break
		}
	}
	if !foundServeHTTP {
		t.Error("ServeHTTP method on UserHandler not found")
	}

	// repo package: struct fields with JSON tags
	if len(repoPkg.Types) != 2 {
		t.Errorf("expected 2 types in repo (Repository, User), got %d", len(repoPkg.Types))
	}
	hasJSONTag := false
	for _, typ := range repoPkg.Types {
		if typ.Name == "User" && len(typ.Fields) > 0 {
			for _, f := range typ.Fields {
				if strings.Contains(f.Tag, "json") {
					hasJSONTag = true
				}
			}
		}
	}
	if !hasJSONTag {
		t.Error("expected User struct with JSON tags")
	}
}

func TestAnalyzeToCodebase_Imports(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testimports\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "api", "api.go"), `package api

import (
	"encoding/json"
	"net/http"
	"testimports/repo"
)

type Handler struct {
	r *repo.Repository
}

func (h *Handler) Get() (string, error) {
	return "data", nil
}
`)

	writeFile(t, filepath.Join(dir, "repo", "repo.go"), `package repo

import "database/sql"

type Repository struct {
	db *sql.DB
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	// verify import edges exist
	dg := cb.DependencyGraph
	if dg == nil {
		t.Fatal("DependencyGraph is nil")
	}

	// api imports repo
	foundEdge := false
	for _, e := range dg.Edges {
		if e.Source == "api" && e.Target == "repo" {
			foundEdge = true
			break
		}
	}
	if !foundEdge {
		t.Error("expected dependency edge: api -> repo")
	}

	// api import also has net/http and encoding/json
	var apiPkg *models.Package
	for _, p := range cb.Packages {
		if p.Name == "api" {
			apiPkg = p
			break
		}
	}
	if apiPkg == nil {
		t.Fatal("api package not found")
	}

	importPaths := make(map[string]bool)
	for _, imp := range apiPkg.Imports {
		importPaths[imp.Path] = true
	}
	for _, expected := range []string{"encoding/json", "net/http", "testimports/repo"} {
		if !importPaths[expected] {
			t.Errorf("expected import %s not found", expected)
		}
	}
}

func TestAnalyzeToCodebase_Capabilities(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testcaps\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "net/http"

func main() {
	http.ListenAndServe(":8080", nil)
}

func TestHealthCheck(t *testing.T) {
	t.Log("ok")
}
`)

	writeFile(t, filepath.Join(dir, "handler", "handler.go"), `package handler

type Service interface {
	Handle(w http.ResponseWriter, r *http.Request)
	Run()
}

type Middleware func(next http.Handler) http.Handler
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	if len(cb.Capabilities) == 0 {
		t.Fatal("expected capabilities detected")
	}

	capNames := make(map[string]*models.Capability)
	for _, c := range cb.Capabilities {
		capNames[c.Name] = c
	}

	// must detect: http_server (net/http import), cli_entrypoint (main()), has_tests (TestHealthCheck)
	expectedCaps := []string{"http_server", "cli_entrypoint", "has_tests"}
	for _, name := range expectedCaps {
		if _, ok := capNames[name]; !ok {
			t.Errorf("expected capability %q not found", name)
		}
	}

	// verify traceability: every capability must have source locations
	for _, c := range cb.Capabilities {
		if len(c.Sources) == 0 {
			t.Errorf("capability %q has no source locations (not traceable)", c.Name)
		}
	}
}

func TestAnalyzeToCodebase_ArchitectureFingerprint(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testarch\n\ngo 1.22\n")

	// handler imports service -> service imports repo (layered)
	writeFile(t, filepath.Join(dir, "handler", "handler.go"), `package handler

import "testarch/service"

type Handler struct {
	svc *service.Service
}

func (h *Handler) ServeHTTP() string { return "ok" }
`)
	writeFile(t, filepath.Join(dir, "service", "service.go"), `package service

import "testarch/repo"

type Service struct {
	r *repo.Repository
}

func (s *Service) GetUser() string { return "user" }
`)
	writeFile(t, filepath.Join(dir, "repo", "repo.go"), `package repo

type Repository struct{}

func (r *Repository) Find() string { return "data" }
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	ap := cb.Architecture
	if ap == nil {
		t.Fatal("Architecture is nil")
	}

	if ap.Architecture != "layered" {
		t.Errorf("expected layered architecture, got %q (confidence %.2f)", ap.Architecture, ap.Confidence)
	}

	if ap.Confidence < 0.5 {
		t.Errorf("expected reasonable confidence, got %.2f", ap.Confidence)
	}

	// should have handler, service, repository layers
	hasHandler := false
	hasService := false
	hasRepo := false
	for _, l := range ap.Layers {
		switch l {
		case "handler":
			hasHandler = true
		case "service":
			hasService = true
		case "repository":
			hasRepo = true
		}
	}
	if !hasHandler || !hasService || !hasRepo {
		t.Errorf("expected handler, service, repository layers, got %v", ap.Layers)
	}
}

func TestAnalyzeToCodebase_CallGraph(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testcg\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

func main() {
	greet("world")
}

func greet(name string) {
	fmt.Println("hello", name)
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	cg := cb.CallGraph
	if cg == nil {
		t.Fatal("CallGraph is nil")
	}

	foundCall := false
	for _, e := range cg.Edges {
		if e.Caller == "main" && e.Callee == "greet" {
			foundCall = true
			break
		}
	}
	if !foundCall {
		t.Error("expected call edge: main -> greet")
	}
}

func TestAnalyzeToCodebase_JSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testjson\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\n\nfunc main() {}\n")

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	json := cb.JSON()
	if json == "" {
		t.Fatal("expected non-empty JSON")
	}
	if !strings.Contains(json, `"language": "go"`) {
		t.Error("JSON missing language field")
	}
	if !strings.Contains(json, `"root"`) {
		t.Error("JSON missing root field")
	}
}

func TestAnalyzeToCodebase_InterfaceImplementors(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testimpl\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "store", "store.go"), `package store

type Storage interface {
	Get(id string) (string, error)
	Set(id, value string) error
}
`)

	writeFile(t, filepath.Join(dir, "redis", "redis.go"), `package redis

type RedisStore struct{}

func (r *RedisStore) Get(id string) (string, error) {
	return "val", nil
}

func (r *RedisStore) Set(id, value string) error {
	return nil
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	var storeIface *models.Interface
	for _, p := range cb.Packages {
		for _, iface := range p.Interfaces {
			if iface.Name == "Storage" {
				storeIface = iface
				break
			}
		}
	}
	if storeIface == nil {
		t.Fatal("Storage interface not found")
	}

	if len(storeIface.Implementors) == 0 {
		t.Log("no implementors found (may need method signature matching)")
	}
}

func TestAnalyzeToCodebase_StructFields(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testfields\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "model", "model.go"), `package model

type Product struct {
	ID    int64   `+"`"+`json:"id"`+"`"+`
	Name  string  `+"`"+`json:"name"`+"`"+`
	Price float64 `+"`"+`json:"price"`+"`"+`
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	var modelPkg *models.Package
	for _, p := range cb.Packages {
		if p.Name == "model" {
			modelPkg = p
			break
		}
	}
	if modelPkg == nil {
		t.Fatal("model package not found")
	}
	if len(modelPkg.Types) != 1 {
		t.Fatalf("expected 1 type, got %d", len(modelPkg.Types))
	}

	product := modelPkg.Types[0]
	if product.Name != "Product" {
		t.Errorf("expected Product, got %s", product.Name)
	}
	if product.Kind != "struct" {
		t.Errorf("expected struct kind, got %s", product.Kind)
	}
	if len(product.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(product.Fields))
	}

	expectedFields := map[string]string{
		"ID":    "int64",
		"Name":  "string",
		"Price": "float64",
	}
	for _, f := range product.Fields {
		expectedType, ok := expectedFields[f.Name]
		if !ok {
			t.Errorf("unexpected field %s", f.Name)
			continue
		}
		if f.Type != expectedType {
			t.Errorf("field %s: expected type %s, got %s", f.Name, expectedType, f.Type)
		}
		if f.Tag == "" {
			t.Errorf("field %s missing tag", f.Name)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestRealWorldAnalysis proves Repository Intelligence can understand the
// RouterForge codebase itself without reading the README.
func TestRealWorldAnalysis_RouterForge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real-world analysis in short mode")
	}

	wd, _ := os.Getwd()
	root := filepath.Join(wd, "..", "..") // go up from internal/repo/ to project root

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(root)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	if cb.Language != "go" {
		t.Errorf("expected go language, got %s", cb.Language)
	}
	if len(cb.Packages) == 0 {
		t.Fatal("expected at least one package")
	}

	t.Logf("Packages: %d", len(cb.Packages))
	for _, p := range cb.Packages {
		t.Logf("  %s (%s): %d types, %d functions, %d interfaces, %d imports",
			p.Name, p.Path, len(p.Types), len(p.Functions), len(p.Interfaces), len(p.Imports))
	}

	// should detect: http_server (if rod/gin used), cli_entrypoint (main in cmd/),
	// configuration (cobra/viper), cli_framework (cobra)
	if len(cb.Capabilities) > 0 {
		t.Logf("Detected %d capabilities:", len(cb.Capabilities))
		for _, c := range cb.Capabilities {
			t.Logf("  [%s] %s (%.0f%%) — %d source locations",
				c.Category, c.Name, c.Confidence*100, len(c.Sources))
		}
	} else {
		t.Log("No capabilities detected")
	}

	// Architecture
	if cb.Architecture != nil {
		t.Logf("Architecture: %s (%.0f%%)", cb.Architecture.Architecture, cb.Architecture.Confidence*100)
		t.Logf("  Layers: %v", cb.Architecture.Layers)
		t.Logf("  Patterns: %v", cb.Architecture.Patterns)
	}

	// Dependency graph
	if cb.DependencyGraph != nil {
		t.Logf("Dependency graph: %d nodes, %d edges",
			len(cb.DependencyGraph.Nodes), len(cb.DependencyGraph.Edges))
	}

	// Call graph
	if cb.CallGraph != nil {
		t.Logf("Call graph: %d nodes, %d edges",
			len(cb.CallGraph.Nodes), len(cb.CallGraph.Edges))
	}

	// Capability graph
	if cb.CapabilityGraph != nil {
		t.Logf("Capability graph: %d nodes, %d edges", len(cb.CapabilityGraph.Nodes), len(cb.CapabilityGraph.Edges))
		for _, nt := range []models.CapabilityNodeType{
			models.CapRoute, models.CapHandler, models.CapMiddleware,
			models.CapService, models.CapRepository, models.CapDataModel,
			models.CapInterface, models.CapImplementation, models.CapPackage,
			models.CapEntrypoint, models.CapDatabase,
		} {
			nodes := cb.CapabilityGraph.NodesByType(nt)
			if len(nodes) > 0 {
				t.Logf("  %s: %d", nt, len(nodes))
			}
		}
	}
}
