package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func writeFileSections(projectDir, result string) (int, error) {
	filesWritten := 0
	sections := strings.Split(result, "FILE:")
	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		parts := strings.SplitN(section, "\n", 2)
		fileName := strings.TrimSpace(parts[0])
		fileName = strings.TrimLeft(fileName, "- ")
		fileName = strings.TrimSpace(fileName)
		if fileName == "" {
			continue
		}

		content := ""
		if len(parts) > 1 {
			content = strings.TrimSpace(parts[1])
			content = strings.TrimPrefix(content, "---\r\n")
			content = strings.TrimPrefix(content, "---\n")
			content = strings.TrimSpace(content)
		}

		fullPath, err := safeProjectPath(projectDir, fileName)
		if err != nil {
			return filesWritten, err
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return filesWritten, err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return filesWritten, fmt.Errorf("write %s: %w", fileName, err)
		}
		filesWritten++
	}
	return filesWritten, nil
}

func safeProjectPath(projectDir, fileName string) (string, error) {
	clean := filepath.Clean(fileName)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid generated path %q", fileName)
	}
	return filepath.Join(projectDir, clean), nil
}
