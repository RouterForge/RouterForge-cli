package fusion

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/routerforge/cli/internal/repo"
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

type DeepStudyResult struct {
	Language    string                    `json:"language"`
	Patterns    []repo.Pattern            `json:"patterns"`
	ASTPackages []*repo.PackageInfo       `json:"ast_packages,omitempty"`
	DepGraph    *repo.DepGraph            `json:"dep_graph,omitempty"`
	CallGraph   *repo.CallGraph           `json:"call_graph,omitempty"`
	ImportGraph *repo.ImportGraph         `json:"import_graph,omitempty"`
	ArchProfile *repo.ArchitectureProfile `json:"arch_profile,omitempty"`
	CapGraph    *repo.CapabilityGraph     `json:"cap_graph,omitempty"`
	FeatureMtx  *repo.FeatureMatrix       `json:"feature_matrix,omitempty"`
	CapDiscover []Capability              `json:"cap_discover,omitempty"`
	Summary     string                    `json:"summary"`
}

type Engine struct {
	graph     FusionGraph
	ast       *repo.ASTAnalyzer
	detector  *repo.PatternDetector
	src       *repo.SourceAnalyzer
}

func NewEngine() *Engine {
	return &Engine{
		graph: FusionGraph{
			Capabilities: defaultCapabilities(),
			Connections:  defaultConnections(),
		},
		ast:      &repo.ASTAnalyzer{},
		detector: &repo.PatternDetector{},
		src:      &repo.SourceAnalyzer{},
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

func (e *Engine) DeepStudy(path string) (*DeepStudyResult, error) {
	result := &DeepStudyResult{}

	lang := e.src.DetectLanguage(path)
	result.Language = lang
	result.Summary = fmt.Sprintf("Language: %s", lang)

	patterns := e.detector.Detect(path)
	result.Patterns = patterns

	surfaceCaps, err := e.StudyRepo(path)
	if err != nil {
		surfaceCaps = nil
	}
	result.CapDiscover = surfaceCaps

	if lang == "go" {
		astPkgs, depGraph, astErr := e.ast.AnalyzeGoRepo(path)
		if astErr == nil {
			result.ASTPackages = astPkgs
			result.DepGraph = depGraph

			astPats := e.ast.DetectCapabilitiesFromAST(astPkgs)
			for _, p := range astPats {
				patterns = append(patterns, p)
			}
			result.Patterns = patterns

			cgGraph := repo.NewCapabilityGraph()
			for _, pkg := range astPkgs {
				cgGraph.Add(pkg.Name, "AST detected package", nil)
				for _, fn := range pkg.Functions {
					if fn.IsExported {
						cgGraph.Add(pkg.Name+"."+fn.Name, "Exported function", []string{pkg.Name})
					}
				}
			}
			result.CapGraph = cgGraph
		}

		callGraph, cgErr := e.ast.BuildCallGraph(path)
		if cgErr == nil && len(callGraph.Nodes) > 0 {
			result.CallGraph = callGraph
		}

		impGraph, igErr := e.ast.BuildImportGraph(path)
		if igErr == nil && len(impGraph.Nodes) > 0 {
			result.ImportGraph = impGraph
		}

		archProfile := e.ast.FingerprintArchitecture(path)
		if archProfile != nil {
			result.ArchProfile = archProfile
		}

		fm := repo.NewFeatureMatrix()
		fm.AddRepo(filepath.Base(path))
		for _, p := range patterns {
			fm.SetFeature(filepath.Base(path), p.Name, true)
		}
		if astPkgs != nil {
			for _, pkg := range astPkgs {
				fm.SetFeature(filepath.Base(path), "pkg_"+pkg.Name, true)
				if len(pkg.Interfaces) > 0 {
					fm.SetFeature(filepath.Base(path), "has_interfaces", true)
				}
				if len(pkg.Functions) > 0 {
					fm.SetFeature(filepath.Base(path), "has_functions", true)
				}
			}
		}
		result.FeatureMtx = fm

		parts := []string{fmt.Sprintf("Language: %s", lang)}
		parts = append(parts, fmt.Sprintf("Patterns: %d", len(patterns)))
		if astPkgs != nil {
			parts = append(parts, fmt.Sprintf("Packages: %d", len(astPkgs)))
		}
		if callGraph != nil {
			parts = append(parts, fmt.Sprintf("Call Graph: %d nodes, %d edges", len(callGraph.Nodes), len(callGraph.Edges)))
		}
		if archProfile != nil {
			parts = append(parts, fmt.Sprintf("Architecture: %s (%.0f%%)", archProfile.Architecture, archProfile.Confidence*100))
		}
		result.Summary = strings.Join(parts, " | ")
	}

	return result, nil
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

func (r *DeepStudyResult) JSON() string {
	b, _ := json.MarshalIndent(r, "", "  ")
	return string(b)
}

func (r *DeepStudyResult) Markdown() string {
	var b strings.Builder
	b.WriteString("# Deep Repository Study\n\n")
	b.WriteString(fmt.Sprintf("**Summary:** %s\n\n", r.Summary))
	b.WriteString(fmt.Sprintf("**Language:** %s\n\n", r.Language))

	if len(r.Patterns) > 0 {
		b.WriteString("## Patterns\n\n| Pattern | Confidence |\n|---------|----------|\n")
		for _, p := range r.Patterns {
			b.WriteString(fmt.Sprintf("| %s | %.0f%% |\n", p.Name, p.Confidence*100))
		}
		b.WriteString("\n")
	}

	if r.ArchProfile != nil {
		b.WriteString(fmt.Sprintf("## Architecture: %s (%.0f%% confidence)\n\n", r.ArchProfile.Architecture, r.ArchProfile.Confidence*100))
		if len(r.ArchProfile.Layers) > 0 {
			b.WriteString(fmt.Sprintf("**Layers:** %s\n\n", strings.Join(r.ArchProfile.Layers, ", ")))
		}
		if len(r.ArchProfile.Patterns) > 0 {
			b.WriteString(fmt.Sprintf("**Patterns:** %s\n\n", strings.Join(r.ArchProfile.Patterns, ", ")))
		}
		if len(r.ArchProfile.Evidence) > 0 {
			b.WriteString("**Evidence:**\n")
			for k, v := range r.ArchProfile.Evidence {
				b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
			b.WriteString("\n")
		}
	}

	if r.ImportGraph != nil && len(r.ImportGraph.Nodes) > 0 {
		b.WriteString("## Import Graph\n\n")
		b.WriteString(fmt.Sprintf("%d packages, %d dependencies\n\n", len(r.ImportGraph.Nodes), len(r.ImportGraph.Edges)))
		for _, e := range r.ImportGraph.Edges {
			b.WriteString(fmt.Sprintf("- %s → %s\n", e.Source, e.Target))
		}
		b.WriteString("\n")
	}

	if r.CallGraph != nil && len(r.CallGraph.Nodes) > 0 {
		b.WriteString("## Call Graph\n\n")
		b.WriteString(fmt.Sprintf("%d functions, %d calls\n\n", len(r.CallGraph.Nodes), len(r.CallGraph.Edges)))
		b.WriteString("```mermaid\n")
		b.WriteString("graph TD\n")
		for _, e := range r.CallGraph.Edges {
			b.WriteString(fmt.Sprintf("  %s --> %s\n", sanitizeNode(e.Caller), sanitizeNode(e.Callee)))
		}
		b.WriteString("```\n\n")
	}

	if len(r.ASTPackages) > 0 {
		b.WriteString("## Packages\n\n| Package | Path | Functions | Types | Interfaces |\n|---------|------|-----------|-------|------------|\n")
		for _, pkg := range r.ASTPackages {
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d |\n", pkg.Name, pkg.Path, len(pkg.Functions), len(pkg.Types), len(pkg.Interfaces)))
		}
		b.WriteString("\n")
	}

	if r.FeatureMtx != nil {
		b.WriteString("## Feature Matrix\n\n")
		b.WriteString(r.FeatureMtx.Markdown())
		b.WriteString("\n")
	}

	return b.String()
}

func sanitizeNode(name string) string {
	return strings.NewReplacer(".", "_", "/", "_", "-", "_", " ", "_").Replace(name)
}
