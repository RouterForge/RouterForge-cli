package engine

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileOps struct {
	workdir string
}

func NewFileOps(workdir string) *FileOps {
	return &FileOps{workdir: workdir}
}

func (fo *FileOps) Read(path string) (string, error) {
	fullPath := fo.resolve(path)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(data), nil
}

func (fo *FileOps) Write(path, content string) error {
	fullPath := fo.resolve(path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory: %w", err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}
	return nil
}

func (fo *FileOps) Delete(path string) error {
	return os.Remove(fo.resolve(path))
}

func (fo *FileOps) Exists(path string) bool {
	_, err := os.Stat(fo.resolve(path))
	return err == nil
}

func (fo *FileOps) Mkdir(path string) error {
	return os.MkdirAll(fo.resolve(path), 0755)
}

func (fo *FileOps) List(dir string) ([]string, error) {
	entries, err := os.ReadDir(fo.resolve(dir))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (fo *FileOps) resolve(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(fo.workdir, path)
}

func (fo *FileOps) Glob(pattern string) ([]string, error) {
	matches, err := filepath.Glob(fo.resolve(pattern))
	if err != nil {
		return nil, err
	}
	return matches, nil
}
