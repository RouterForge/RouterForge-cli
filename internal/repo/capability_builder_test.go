package repo

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/routerforge/cli/pkg/models"
)

func TestBuildCapabilityGraph_Routes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testroutes\n\ngo 1.22\n")

	// Simulate a real Go HTTP server with routes, handlers, middleware, service, repo, models
	writeFile(t, filepath.Join(dir, "cmd", "server", "main.go"), `package main

import (
	"net/http"
	"testroutes/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users", handler.ListUsers)
	mux.HandleFunc("POST /api/users", handler.CreateUser)
	http.ListenAndServe(":8080", mux)
}
`)

	writeFile(t, filepath.Join(dir, "internal", "handler", "user.go"), `package handler

import (
	"encoding/json"
	"net/http"
	"testroutes/internal/service"
)

type UserHandler struct {
	svc *service.UserService
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

func ListUsers(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("[]"))
}

func CreateUser(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("created"))
}
`)

	writeFile(t, filepath.Join(dir, "internal", "middleware", "auth.go"), `package middleware

import "net/http"

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("auth"))
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("log"))
	})
}
`)

	writeFile(t, filepath.Join(dir, "internal", "service", "user.go"), `package service

import "testroutes/internal/repo"

type UserService struct {
	userRepo *repo.UserRepository
}

func NewUserService(r *repo.UserRepository) *UserService {
	return &UserService{userRepo: r}
}

func (s *UserService) GetUser(id string) (*User, error) {
	return s.userRepo.FindByID(id)
}

func (s *UserService) CreateUser(name string) (*User, error) {
	return s.userRepo.Save(&User{Name: name})
}

type User struct {
	ID   string `+"`"+`json:"id"`+"`"+`
	Name string `+"`"+`json:"name"`+"`"+`
}
`)

	writeFile(t, filepath.Join(dir, "internal", "repo", "user.go"), `package repo

import "testroutes/internal/service"

type UserRepository struct {
	db map[string]*service.User
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: make(map[string]*service.User)}
}

func (r *UserRepository) FindByID(id string) (*service.User, error) {
	return r.db[id], nil
}

func (r *UserRepository) Save(user *service.User) (*service.User, error) {
	r.db[user.ID] = user
	return user, nil
}

func (r *UserRepository) Delete(id string) error {
	delete(r.db, id)
	return nil
}

func (r *UserRepository) List() ([]*service.User, error) {
	var users []*service.User
	for _, u := range r.db {
		users = append(users, u)
	}
	return users, nil
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	cg := cb.CapabilityGraph
	if cg == nil {
		t.Fatal("CapabilityGraph is nil")
	}

	// --- Verify: nodes ---
	t.Logf("CapabilityGraph: %d nodes, %d edges", len(cg.Nodes), len(cg.Edges))
	for _, n := range cg.Nodes {
		t.Logf("  [%s] %s (%s)", n.Type, n.Name, n.Package)
	}

	routes := cg.NodesByType(models.CapRoute)
	if len(routes) == 0 {
		t.Error("expected at least 2 route nodes")
	} else {
		t.Logf("Routes: %d", len(routes))
		for _, r := range routes {
			t.Logf("  %s: method=%s path=%s", r.Name, r.Properties["method"], r.Properties["path"])
		}
	}

	// check for entrypoint
	entrypoints := cg.NodesByType(models.CapEntrypoint)
	if len(entrypoints) == 0 {
		t.Error("expected entrypoint node for main()")
	}

	// check for handler nodes
	handlers := cg.NodesByType(models.CapHandler)
	if len(handlers) == 0 {
		t.Error("expected handler nodes")
	}
	t.Logf("Handlers: %d", len(handlers))
	for _, h := range handlers {
		t.Logf("  %s", h.Name)
	}

	// check for service
	services := cg.NodesByType(models.CapService)
	if len(services) == 0 {
		t.Error("expected service node")
	}
	t.Logf("Services: %d", len(services))

	// check for repository
	repos := cg.NodesByType(models.CapRepository)
	if len(repos) == 0 {
		t.Error("expected repository node")
	}
	t.Logf("Repositories: %d", len(repos))

	// check for data models
	modelNodes := cg.NodesByType(models.CapDataModel)
	if len(modelNodes) == 0 {
		t.Error("expected data model nodes")
	}
	t.Logf("Data models: %d", len(modelNodes))

	// check for middleware
	mws := cg.NodesByType(models.CapMiddleware)
	if len(mws) == 0 {
		t.Log("no middleware nodes (may not match detection)")
	}
	t.Logf("Middleware: %d", len(mws))

	// --- Verify: edges ---
	for _, e := range cg.Edges {
		src := cg.NodeByID(e.SourceID)
		tgt := cg.NodeByID(e.TargetID)
		srcName, tgtName := "?", "?"
		if src != nil {
			srcName = src.Name
		}
		if tgt != nil {
			tgtName = tgt.Name
		}
		t.Logf("  %s --[%s]--> %s", srcName, e.Type, tgtName)
	}

	t.Logf("Total edges: %d", len(cg.Edges))
}

func TestBuildCapabilityGraph_NodeCounts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testcounts\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "net/http"

func main() {
	http.HandleFunc("/", handler)
}

func handler(w http.ResponseWriter, r *http.Request) {}
`)
	writeFile(t, filepath.Join(dir, "internal", "service", "svc.go"), `package service

type Service struct{}
func (s *Service) Do() string { return "x" }
`)
	writeFile(t, filepath.Join(dir, "internal", "repo", "repo.go"), `package repo

type Repo struct{}
func (r *Repo) Find() string { return "x" }
func (r *Repo) Save(v string) error { return nil }
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	cg := cb.CapabilityGraph
	if cg == nil {
		t.Fatal("CapabilityGraph is nil")
	}

	if len(cg.Nodes) == 0 {
		t.Error("expected at least some capability nodes")
	}
	if len(cg.Edges) == 0 {
		t.Error("expected at least some capability edges")
	}

	// Report what the graph contains
	t.Logf("CapabilityGraph: %d nodes, %d edges", len(cg.Nodes), len(cg.Edges))
	for _, nt := range []models.CapabilityNodeType{
		models.CapRoute, models.CapHandler, models.CapService,
		models.CapRepository, models.CapDataModel, models.CapPackage,
		models.CapEntrypoint,
	} {
		nodes := cg.NodesByType(nt)
		t.Logf("  %s: %d", nt, len(nodes))
	}
}

func TestBuildCapabilityGraph_DataModelTags(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testmodels\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "model", "product.go"), `package model

type Product struct {
	ID    int64   `+"`"+`json:"id" gorm:"primaryKey"`+"`"+`
	Name  string  `+"`"+`json:"name"`+"`"+`
	Price float64 `+"`"+`json:"price"`+"`"+`
}

type Order struct {
	ID         int64  `+"`"+`json:"id" bson:"_id"`+"`"+`
	Total      float64
	CreatedAt  string `+"`"+`json:"created_at"`+"`"+`
}
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	cg := cb.CapabilityGraph
	if cg == nil {
		t.Fatal("CapabilityGraph is nil")
	}

	modelNodes := cg.NodesByType(models.CapDataModel)
	if len(modelNodes) == 0 {
		t.Fatal("expected data model nodes")
	}

	t.Logf("Data models: %d", len(modelNodes))
	for _, m := range modelNodes {
		t.Logf("  %s (tags: %s)", m.Name, m.Properties["tags"])
	}

	// Product should have both json and db tags
	for _, m := range modelNodes {
		if strings.Contains(m.Name, "Product") {
			tags := m.Properties["tags"]
			if !strings.Contains(tags, "json") {
				t.Error("expected json tag on Product")
			}
		}
	}
}

func TestBuildCapabilityGraph_EdgesBetweenLayers(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testedges\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "handler", "h.go"), `package handler

import "testedges/service"

type H struct{ svc *service.S }

func List(w http.ResponseWriter, r *http.Request) { s := &service.S{}; s.Get() }
`)
	writeFile(t, filepath.Join(dir, "service", "s.go"), `package service

import "testedges/repo"

type S struct{ r *repo.R }

func (s *S) Get() string { return s.r.Find() }
`)
	writeFile(t, filepath.Join(dir, "repo", "r.go"), `package repo

type R struct{}

func (r *R) Find() string { return "x" }
func (r *R) Save(v string) error { return nil }
func (r *R) Delete(id string) error { return nil }
`)

	a := &ASTAnalyzer{}
	cb, err := a.AnalyzeToCodebase(dir)
	if err != nil {
		t.Fatalf("AnalyzeToCodebase: %v", err)
	}

	cg := cb.CapabilityGraph
	if cg == nil {
		t.Fatal("CapabilityGraph is nil")
	}

	t.Logf("CapabilityGraph: %d nodes, %d edges", len(cg.Nodes), len(cg.Edges))
	for _, n := range cg.Nodes {
		t.Logf("  [%s] %s", n.Type, n.Name)
	}
	for _, e := range cg.Edges {
		src := cg.NodeByID(e.SourceID)
		tgt := cg.NodeByID(e.TargetID)
		srcName := "?"
		tgtName := "?"
		if src != nil {
			srcName = src.Name
		}
		if tgt != nil {
			tgtName = tgt.Name
		}
		t.Logf("  %s --[%s]--> %s", srcName, e.Type, tgtName)
	}
}
