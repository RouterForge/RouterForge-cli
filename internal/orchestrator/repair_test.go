package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileSectionsStripsSeparator(t *testing.T) {
	dir := t.TempDir()
	files, err := writeFileSections(dir, "FILE: main.go\n---\npackage main\n\nfunc main() {}\n")
	if err != nil {
		t.Fatalf("writeFileSections: %v", err)
	}
	if files != 1 {
		t.Fatalf("expected 1 file, got %d", files)
	}
	data, err := os.ReadFile(filepath.Join(dir, "main.go"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "---") {
		t.Fatalf("separator leaked into generated source: %q", string(data))
	}
}

func TestWriteFileSectionsRejectsTraversal(t *testing.T) {
	dir := t.TempDir()
	_, err := writeFileSections(dir, "FILE: ../escape.txt\n---\nnope\n")
	if err == nil {
		t.Fatal("expected traversal path to be rejected")
	}
}

func TestValidateProjectStaticWebMissingAsset(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(`<script src="missing.js"></script>`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	result := ValidateProject(dir)
	if result.Success {
		t.Fatal("expected validation failure")
	}
	if len(result.Errors) == 0 || !strings.Contains(result.Errors[0], "missing asset") {
		t.Fatalf("expected missing asset error, got %#v", result.Errors)
	}
}

func TestValidateProjectGoRequiresModule(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	result := ValidateProject(dir)
	if result.Success {
		t.Fatal("expected validation failure")
	}
	if len(result.Errors) != 1 || result.Errors[0] != "go.mod is missing" {
		t.Fatalf("expected go.mod error, got %#v", result.Errors)
	}
}
