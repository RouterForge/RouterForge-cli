package engine

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SearchMatch struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Content string `json:"content"`
}

type SearchEngine struct {
	workdir string
}

func NewSearchEngine(workdir string) *SearchEngine {
	return &SearchEngine{workdir: workdir}
}

func (se *SearchEngine) Grep(pattern string, include string) ([]SearchMatch, error) {
	var matches []SearchMatch

	err := filepath.Walk(se.workdir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if include != "" {
			match, err := filepath.Match(include, info.Name())
			if err != nil || !match {
				return nil
			}
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		relPath, _ := filepath.Rel(se.workdir, path)
		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if strings.Contains(line, pattern) {
				matches = append(matches, SearchMatch{
					File:    relPath,
					Line:    lineNum,
					Content: strings.TrimSpace(line),
				})
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	return matches, nil
}
