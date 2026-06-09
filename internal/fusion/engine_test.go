package fusion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeepStudyCodebase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testv2\n\ngo 1.22\n")
	writeFile(t, filepath.Join(dir, "main.go"), `package main

import "fmt"

func main() {
	fmt.Println("hello")
}
`)

	e := NewEngine()
	cb, err := e.DeepStudyCodebase(dir)
	if err != nil {
		t.Fatalf("DeepStudyCodebase: %v", err)
	}
	if cb == nil {
		t.Fatal("expected non-nil codebase")
	}
	if cb.Language != "go" {
		t.Errorf("expected go, got %s", cb.Language)
	}
	if len(cb.Packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(cb.Packages))
	}
	if !strings.Contains(cb.JSON(), `"language": "go"`) {
		t.Error("JSON output missing language field")
	}
}

func TestDeepStudyCodebase_CapabilitiesAndArchitecture(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module testarchv2\n\ngo 1.22\n")

	writeFile(t, filepath.Join(dir, "handler", "handler.go"), `package handler

import "testarchv2/service"

type HttpHandler struct {
	svc *service.Service
}

func (h *HttpHandler) ServeHTTP() string { return "ok" }

func handlerFunc() { println("handle") }
`)
	writeFile(t, filepath.Join(dir, "service", "service.go"), `package service

import "testarchv2/repo"

type Service struct {
	r *repo.Repository
}

func (s *Service) GetUser() string { return "data" }
`)
	writeFile(t, filepath.Join(dir, "repo", "repo.go"), `package repo

type Repository struct{}

func (r *Repository) Find() string { return "found" }

type User struct {
	ID   string `+"`"+`json:"id"`+"`"+`
	Name string `+"`"+`json:"name"`+"`"+`
}
`)

	e := NewEngine()
	cb, err := e.DeepStudyCodebase(dir)
	if err != nil {
		t.Fatalf("DeepStudyCodebase: %v", err)
	}

	// Verify architecture detection (handler -> service -> repo = layered)
	if cb.Architecture == nil {
		t.Fatal("Architecture is nil")
	}
	if cb.Architecture.Architecture != "layered" {
		t.Errorf("expected layered, got %q", cb.Architecture.Architecture)
	}

	// Verify capabilities
	capNames := make(map[string]bool)
	for _, c := range cb.Capabilities {
		capNames[c.Name] = true
	}

	expectedCaps := []string{"http_handler", "json_models"}
	for _, name := range expectedCaps {
		if !capNames[name] {
			t.Errorf("expected capability %q not found", name)
		}
	}

	// Verify traceability — every capability has source locations
	for _, c := range cb.Capabilities {
		if len(c.Sources) == 0 {
			t.Errorf("capability %q has no source traces", c.Name)
		}
	}

	// Verify the dependency graph has handler -> service -> repo
	dg := cb.DependencyGraph
	if dg == nil {
		t.Fatal("DependencyGraph is nil")
	}
	foundHandlerService := false
	foundServiceRepo := false
	for _, e := range dg.Edges {
		if e.Source == "handler" && e.Target == "service" {
			foundHandlerService = true
		}
		if e.Source == "service" && e.Target == "repo" {
			foundServiceRepo = true
		}
	}
	if !foundHandlerService {
		t.Error("expected handler -> service dep edge")
	}
	if !foundServiceRepo {
		t.Error("expected service -> repo dep edge")
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
