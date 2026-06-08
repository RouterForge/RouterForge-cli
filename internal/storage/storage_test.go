package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileStore(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	fs := NewFileStore(dir)

	err := fs.Write("test.txt", []byte("hello"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	data, err := fs.Read("test.txt")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("Read = %q, want %q", string(data), "hello")
	}
}

func TestConfigDefault(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Model == "" {
		t.Error("Default config should have a model")
	}
	if cfg.DataDir == "" {
		t.Error("Default config should have a data dir")
	}
}

func TestConfigSaveLoad(t *testing.T) {
	dir, _ := os.MkdirTemp("", "routerforge-test")
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "config.json")
	cfg := &Config{
		Model:      "zen/big-pickle",
		SmallModel: "zen/deepseek",
		ProjectDir: "/test",
	}

	err := cfg.Save(path)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if loaded.Model != "zen/big-pickle" {
		t.Errorf("Model = %q, want %q", loaded.Model, "zen/big-pickle")
	}
}
