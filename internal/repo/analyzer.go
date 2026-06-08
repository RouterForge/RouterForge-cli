package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type RepoInfo struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	Branch     string `json:"branch"`
	LocalPath  string `json:"local_path"`
	CommitSHA  string `json:"commit_sha"`
}

type Manager struct {
	baseDir string
}

func NewManager(baseDir string) *Manager {
	os.MkdirAll(baseDir, 0755)
	return &Manager{baseDir: baseDir}
}

func (m *Manager) Clone(url, name string) (*RepoInfo, error) {
	path := filepath.Join(m.baseDir, name)
	if _, err := os.Stat(path); err == nil {
		return m.Info(name)
	}

	cmd := exec.Command("git", "clone", "--depth=1", url, path)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("clone %s: %w\n%s", url, err, string(out))
	}

	return m.Info(name)
}

func (m *Manager) Info(name string) (*RepoInfo, error) {
	path := filepath.Join(m.baseDir, name)
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rev-parse: %w", err)
	}

	cmd2 := exec.Command("git", "branch", "--show-current")
	cmd2.Dir = path
	branch, _ := cmd2.Output()

	return &RepoInfo{
		Name:      name,
		LocalPath: path,
		CommitSHA: strings.TrimSpace(string(out)),
		Branch:    strings.TrimSpace(string(branch)),
	}, nil
}

func (m *Manager) List() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

type SourceAnalyzer struct{}

func (a *SourceAnalyzer) DetectLanguage(path string) string {
	entries, _ := os.ReadDir(path)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch {
		case strings.HasSuffix(e.Name(), ".go"):
			return "go"
		case strings.HasSuffix(e.Name(), ".ts") || strings.HasSuffix(e.Name(), ".tsx"):
			return "typescript"
		case strings.HasSuffix(e.Name(), ".py"):
			return "python"
		case strings.HasSuffix(e.Name(), ".js") || strings.HasSuffix(e.Name(), ".jsx"):
			return "javascript"
		}
	}
	return "unknown"
}

type Pattern struct {
	Name        string   `json:"name"`
	Confidence  float64  `json:"confidence"`
	Indicators  []string `json:"indicators"`
}

type PatternDetector struct{}

func (d *PatternDetector) Detect(path string) []Pattern {
	var patterns []Pattern

	if hasFile(path, "go.mod") {
		patterns = append(patterns, Pattern{Name: "go_module", Confidence: 1.0, Indicators: []string{"go.mod"}})
	}
	if hasFile(path, "package.json") {
		patterns = append(patterns, Pattern{Name: "npm_package", Confidence: 1.0, Indicators: []string{"package.json"}})
	}
	if hasFile(path, "Cargo.toml") {
		patterns = append(patterns, Pattern{Name: "rust_crate", Confidence: 1.0, Indicators: []string{"Cargo.toml"}})
	}
	if hasDir(path, "internal") {
		patterns = append(patterns, Pattern{Name: "go_internal_layout", Confidence: 0.8, Indicators: []string{"internal/"}})
	}
	if hasFile(path, "Dockerfile") {
		patterns = append(patterns, Pattern{Name: "containerized", Confidence: 0.9, Indicators: []string{"Dockerfile"}})
	}
	if hasDir(path, ".github/workflows") {
		patterns = append(patterns, Pattern{Name: "github_ci", Confidence: 0.9, Indicators: []string{".github/workflows"}})
	}

	return patterns
}

type CapabilityNode struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Requires    []string `json:"requires"`
}

type CapabilityGraph struct {
	Nodes map[string]*CapabilityNode `json:"nodes"`
}

func NewCapabilityGraph() *CapabilityGraph {
	return &CapabilityGraph{Nodes: make(map[string]*CapabilityNode)}
}

func (g *CapabilityGraph) Add(name, desc string, requires []string) {
	g.Nodes[name] = &CapabilityNode{Name: name, Description: desc, Requires: requires}
}

type FeatureMatrix struct {
	Repos    []string          `json:"repos"`
	Features []string          `json:"features"`
	Data     map[string]map[string]bool `json:"data"`
}

func NewFeatureMatrix() *FeatureMatrix {
	return &FeatureMatrix{
		Data: make(map[string]map[string]bool),
	}
}

func (fm *FeatureMatrix) AddRepo(name string) {
	fm.Repos = append(fm.Repos, name)
	if fm.Data[name] == nil {
		fm.Data[name] = make(map[string]bool)
	}
}

func (fm *FeatureMatrix) SetFeature(repo, feature string, present bool) {
	if fm.Data[repo] == nil {
		fm.Data[repo] = make(map[string]bool)
	}
	fm.Data[repo][feature] = present
	fm.addFeature(feature)
}

func (fm *FeatureMatrix) addFeature(f string) {
	for _, existing := range fm.Features {
		if existing == f {
			return
		}
	}
	fm.Features = append(fm.Features, f)
}

func (fm *FeatureMatrix) Markdown() string {
	var b strings.Builder
	b.WriteString("| Feature |")
	for _, r := range fm.Repos {
		b.WriteString(" " + r + " |")
	}
	b.WriteString("\n|")
	for range fm.Repos {
		b.WriteString("---|")
	}
	b.WriteString("---|\n")

	for _, f := range fm.Features {
		b.WriteString("| " + f + " |")
		for _, r := range fm.Repos {
			if fm.Data[r][f] {
				b.WriteString(" ✅ |")
			} else {
				b.WriteString(" ❌ |")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

type Recommender struct {
	Graph *CapabilityGraph
}

func NewRecommender(g *CapabilityGraph) *Recommender {
	return &Recommender{Graph: g}
}

func (r *Recommender) Recommend(needs []string) []string {
	seen := map[string]bool{}
	var result []string
	var addAll func(names []string)
	addAll = func(names []string) {
		for _, n := range names {
			if seen[n] {
				continue
			}
			seen[n] = true
			if node, ok := r.Graph.Nodes[n]; ok {
				addAll(node.Requires)
			}
			result = append(result, n)
		}
	}
	addAll(needs)
	return result
}

func hasFile(path, name string) bool {
	_, err := os.Stat(filepath.Join(path, name))
	return err == nil
}

func hasDir(path, name string) bool {
	info, err := os.Stat(filepath.Join(path, name))
	return err == nil && info.IsDir()
}
