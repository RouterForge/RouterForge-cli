package fusion

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Capability struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`
	Patterns    []string `json:"patterns"`
}

type FusionGraph struct {
	Capabilities []Capability `json:"capabilities"`
	Connections  []Connection `json:"connections"`
}

type Connection struct {
	From string `json:"from"`
	To   string `json:"to"`
	Why  string `json:"why"`
}

type Engine struct {
	graph FusionGraph
}

func NewEngine() *Engine {
	return &Engine{
		graph: FusionGraph{
			Capabilities: defaultCapabilities(),
			Connections:  defaultConnections(),
		},
	}
}

func defaultCapabilities() []Capability {
	return []Capability{
		{Name: "file_ops", Description: "Create, read, write, edit files", Source: "opencode", Patterns: []string{"read_file", "write_file", "edit_file"}},
		{Name: "shell_exec", Description: "Execute shell commands", Source: "opencode", Patterns: []string{"run_command", "bash"}},
		{Name: "code_search", Description: "Search codebase for patterns", Source: "opencode", Patterns: []string{"grep", "search"}},
		{Name: "web_fetch", Description: "Fetch and process web content", Source: "opencode", Patterns: []string{"web_fetch", "http_get"}},
		{Name: "browser_automation", Description: "Control headless browser", Source: "routerforge", Patterns: []string{"navigate", "click", "type", "screenshot"}},
		{Name: "agent_orchestration", Description: "Spawn and coordinate sub-agents", Source: "routerforge", Patterns: []string{"spawn", "delegate", "head_manager"}},
		{Name: "repo_intelligence", Description: "Analyze repository structure and code", Source: "routerforge", Patterns: []string{"analyze", "ast", "callgraph", "detect"}},
		{Name: "architecture_planning", Description: "Design system architecture and team structure", Source: "routerforge", Patterns: []string{"plan", "design", "team_manager"}},
		{Name: "governance", Description: "Enforce review gates and quality controls", Source: "routerforge", Patterns: []string{"gate", "review", "approve"}},
		{Name: "lifecycle_mgmt", Description: "Manage Demo/Prototype/MVP/Production lifecycle", Source: "routerforge", Patterns: []string{"lifecycle", "advance", "phase"}},
	}
}

func defaultConnections() []Connection {
	return []Connection{
		{From: "architecture_planning", To: "agent_orchestration", Why: "Plans drive agent creation"},
		{From: "agent_orchestration", To: "file_ops", Why: "Agents need to create files"},
		{From: "agent_orchestration", To: "shell_exec", Why: "Agents execute commands"},
		{From: "agent_orchestration", To: "code_search", Why: "Agents search code"},
		{From: "repo_intelligence", To: "architecture_planning", Why: "Analysis informs architecture"},
		{From: "governance", To: "lifecycle_mgmt", Why: "Gates control phase transitions"},
		{From: "browser_automation", To: "web_fetch", Why: "Browser can fetch content"},
		{From: "lifecycle_mgmt", To: "architecture_planning", Why: "Each phase re-evaluates architecture"},
	}
}

func (e *Engine) Graph() FusionGraph {
	return e.graph
}

func (e *Engine) StudyRepo(path string) ([]Capability, error) {
	var discovered []Capability

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read repo: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			switch entry.Name() {
			case "internal", "pkg", "lib", "src":
				discovered = append(discovered, Capability{
					Name:        fmt.Sprintf("go_package_%s", entry.Name()),
					Description: fmt.Sprintf("Contains %s directory — standard Go project layout", entry.Name()),
					Source:      filepath.Base(path),
					Patterns:    []string{fmt.Sprintf("%s/", entry.Name())},
				})
			case "cmd":
				discovered = append(discovered, Capability{
					Name:        "cli_entrypoint",
					Description: "Has CLI entrypoint in cmd/",
					Source:      filepath.Base(path),
					Patterns:    []string{"cmd/", "main.go"},
				})
			case "agent", "agents":
				discovered = append(discovered, Capability{
					Name:        "agent_system",
					Description: "Has agent management system",
					Source:      filepath.Base(path),
					Patterns:    []string{"agent/", "agents/"},
				})
			case "orchestrator":
				discovered = append(discovered, Capability{
					Name:        "orchestration",
					Description: "Has orchestration engine",
					Source:      filepath.Base(path),
					Patterns:    []string{"orchestrator/"},
				})
			}
		} else {
			switch entry.Name() {
			case "go.mod":
				discovered = append(discovered, Capability{
					Name:        "go_module",
					Description: "Go module with dependency management",
					Source:      filepath.Base(path),
					Patterns:    []string{"go.mod"},
				})
			case "Dockerfile":
				discovered = append(discovered, Capability{
					Name:        "containerized",
					Description: "Docker container support",
					Source:      filepath.Base(path),
					Patterns:    []string{"Dockerfile"},
				})
			case "Makefile":
				discovered = append(discovered, Capability{
					Name:        "build_system",
					Description: "Make-based build system",
					Source:      filepath.Base(path),
					Patterns:    []string{"Makefile"},
				})
			}
		}
	}

	return discovered, nil
}

func (e *Engine) StudyRemoteRepo(url string) ([]Capability, error) {
	tmpDir, err := os.MkdirTemp("", "fusion-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	name := filepath.Base(url)
	name = strings.TrimSuffix(name, ".git")

	cmd := exec.Command("git", "clone", "--depth=1", url, filepath.Join(tmpDir, name))
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("clone %s: %w\n%s", url, err, string(out))
	}

	return e.StudyRepo(filepath.Join(tmpDir, name))
}

func (e *Engine) ToJSON() string {
	b, _ := json.MarshalIndent(e.graph, "", "  ")
	return string(b)
}

func (e *Engine) Markdown() string {
	var b strings.Builder
	b.WriteString("# Capability Fusion Graph\n\n")
	b.WriteString("## Capabilities\n\n")
	b.WriteString("| Capability | Description | Source |\n")
	b.WriteString("|------------|-------------|--------|\n")
	for _, c := range e.graph.Capabilities {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.Name, c.Description, c.Source))
	}
	b.WriteString("\n## Connections\n\n")
	b.WriteString("| From | To | Rationale |\n")
	b.WriteString("|------|----|----------|\n")
	for _, c := range e.graph.Connections {
		b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", c.From, c.To, c.Why))
	}
	return b.String()
}
