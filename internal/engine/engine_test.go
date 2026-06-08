package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileOpsReadWrite(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	fo := NewFileOps(dir)

	err := fo.Write("test.txt", "hello world")
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := fo.Read("test.txt")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if content != "hello world" {
		t.Errorf("Read content = %q, want %q", content, "hello world")
	}
}

func TestFileOpsExists(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	fo := NewFileOps(dir)

	if fo.Exists("nonexistent.txt") {
		t.Error("Exists should return false for nonexistent file")
	}

	fo.Write("exists.txt", "test")
	if !fo.Exists("exists.txt") {
		t.Error("Exists should return true for existing file")
	}
}

func TestFileOpsMkdir(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	fo := NewFileOps(dir)

	err := fo.Mkdir("subdir")
	if err != nil {
		t.Fatalf("Mkdir failed: %v", err)
	}

	info, err := os.Stat(filepath.Join(dir, "subdir"))
	if err != nil || !info.IsDir() {
		t.Error("Mkdir did not create directory")
	}
}

func TestShellRunner(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	runner := NewShellRunner(dir)
	result, err := runner.Run(context.Background(), "echo hello")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result.Stdout != "hello" {
		t.Errorf("Stdout = %q, want %q", result.Stdout, "hello")
	}
	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestShellRunnerError(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	runner := NewShellRunner(dir)
	result, err := runner.Run(context.Background(), "exit 42")
	if err != nil {
		t.Fatalf("Run should not return error for non-zero exit: %v", err)
	}
	if result.ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestSearchEngine(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	fo := NewFileOps(dir)
	fo.Write("file1.go", "package main\nfunc hello() {}")
	fo.Write("file2.go", "package main\nfunc world() {}")

	se := NewSearchEngine(dir)
	matches, err := se.Grep("func", "*.go")
	if err != nil {
		t.Fatalf("Grep failed: %v", err)
	}
	if len(matches) != 2 {
		t.Errorf("Expected 2 matches, got %d", len(matches))
	}
}
