package fusion

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/routerforge/cli/internal/repo"
	"github.com/routerforge/cli/pkg/models"
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

			result.CapGraph = AutoBuildCapabilityGraph(result)
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

// Explain produces a human-readable synthesis of a codebase — architecture,
// routes, request flows, layer violations, and ownership — derived entirely
// from the Semantic Model with no README required.
func (e *Engine) Explain(path string) (*repo.ExplainReport, error) {
	cb, err := e.DeepStudyCodebase(path)
	if err != nil {
		return nil, fmt.Errorf("deep study: %w", err)
	}
	return repo.BuildExplainReport(cb), nil
}

// DeepStudyCodebase performs a single-pass analysis that returns the complete
// semantic Codebase model. This is the V2 entry point for Repository Intelligence.
func (e *Engine) DeepStudyCodebase(path string) (*models.Codebase, error) {
	cb, err := e.ast.AnalyzeToCodebase(path)
	if err != nil {
		return nil, fmt.Errorf("analyze to codebase: %w", err)
	}

	// build capability graph from semantic model
	cb.CapabilityGraph = repo.BuildCapabilityGraph(cb)

	// extract request flows (route → handler → service → repo → model)
	cb.RequestFlows = repo.ExtractRequestFlows(cb.CapabilityGraph, cb)

	// detect layer violations
	cb.LayerViolations = repo.DetectLayerViolations(cb, cb.CapabilityGraph)

	// analyze ownership distribution
	cb.Ownership = repo.AnalyzeOwnership(cb.CapabilityGraph, cb)

	// surface-level capabilities supplement the AST-derived ones
	surfaceCaps, _ := e.StudyRepo(path)
	for _, sc := range surfaceCaps {
		cb.Capabilities = append(cb.Capabilities, &models.Capability{
			Name:        sc.Name,
			Description: sc.Description,
			Confidence:  0.6,
			Category:    "surface",
			Sources:     []models.Location{{File: path, Line: 1}},
			Evidence:    []string{sc.Description},
		})
	}

	return cb, nil
}

func (e *Engine) ToJSON() string {
	b, _ := json.MarshalIndent(e.graph, "", "  ")
	return string(b)
}

func AutoBuildCapabilityGraph(result *DeepStudyResult) *repo.CapabilityGraph {
	g := repo.NewCapabilityGraph()

	if result == nil {
		return g
	}

	for _, p := range result.Patterns {
		name := p.Name
		desc := fmt.Sprintf("Detected pattern: %s (%.0f%% confidence)", p.Name, p.Confidence*100)
		g.Add(name, desc, nil)
	}

	if result.ArchProfile != nil {
		g.Add("arch_"+result.ArchProfile.Architecture,
			fmt.Sprintf("Architecture: %s (%d layers)", result.ArchProfile.Architecture, len(result.ArchProfile.Layers)),
			nil)
		for _, layer := range result.ArchProfile.Layers {
			g.Add("layer_"+layer, fmt.Sprintf("Architecture layer: %s", layer), []string{"arch_" + result.ArchProfile.Architecture})
		}
	}

	for _, pkg := range result.ASTPackages {
		g.Add("pkg_"+pkg.Name, fmt.Sprintf("Package: %s (%d functions, %d types, %d interfaces)",
			pkg.Name, len(pkg.Functions), len(pkg.Types), len(pkg.Interfaces)),
			detectPackageDeps(pkg, result.ASTPackages))

		for _, iface := range pkg.Interfaces {
			g.Add("iface_"+pkg.Name+"."+iface.Name,
				fmt.Sprintf("Interface %s with %d methods", iface.Name, len(iface.Methods)),
				[]string{"pkg_" + pkg.Name})
		}

		capName := detectPackageCapability(pkg.Name, pkg.Imports)
		if capName != "" {
			g.Add(pkg.Name+"_"+capName, fmt.Sprintf("Package %s provides %s", pkg.Name, capName),
				[]string{"pkg_" + pkg.Name})
		}

		if containsHTTP(pkg.Imports) {
			g.Add("http_handler_"+pkg.Name, fmt.Sprintf("HTTP handler in %s", pkg.Name),
				[]string{"pkg_" + pkg.Name, "pattern_http_server"})
		}
		if containsDatabase(pkg.Imports) {
			g.Add("db_access_"+pkg.Name, fmt.Sprintf("Database access in %s", pkg.Name),
				[]string{"pkg_" + pkg.Name})
		}
	}

	if result.ImportGraph != nil {
		for _, edge := range result.ImportGraph.Edges {
			src := findPackageNode(edge.Source, g)
			tgt := findPackageNode(edge.Target, g)
			if src != "" && tgt != "" {
				if g.Nodes[src] != nil && !contains(g.Nodes[src].Requires, tgt) {
					g.Nodes[src].Requires = append(g.Nodes[src].Requires, tgt)
				}
			}
		}
	}

	if result.CallGraph != nil && len(result.CallGraph.Edges) > 0 {
		g.Add("has_callgraph", fmt.Sprintf("Call graph: %d nodes, %d edges", len(result.CallGraph.Nodes), len(result.CallGraph.Edges)),
			nil)
	}

	return g
}

func detectPackageDeps(pkg *repo.PackageInfo, allPkgs []*repo.PackageInfo) []string {
	var deps []string
	for _, imp := range pkg.Imports {
		short := shortenImport(imp)
		if short != "" && short != pkg.Name {
			for _, other := range allPkgs {
				if other.Name == short {
					deps = append(deps, "pkg_"+short)
					break
				}
			}
		}
	}
	return deps
}

func detectPackageCapability(name string, imports []string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "auth") || strings.Contains(lower, "login") || strings.Contains(lower, "user") {
		return "auth_system"
	}
	if strings.Contains(lower, "api") || strings.Contains(lower, "handler") || strings.Contains(lower, "route") {
		return "api_layer"
	}
	if strings.Contains(lower, "db") || strings.Contains(lower, "database") || strings.Contains(lower, "store") || strings.Contains(lower, "repo") || strings.Contains(lower, "model") {
		return "data_layer"
	}
	if strings.Contains(lower, "cache") || strings.Contains(lower, "redis") || strings.Contains(lower, "mem") {
		return "caching"
	}
	if strings.Contains(lower, "cli") || strings.Contains(lower, "cmd") || strings.Contains(lower, "command") {
		return "cli_interface"
	}
	if strings.Contains(lower, "config") || strings.Contains(lower, "setting") {
		return "configuration"
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "spec") || strings.Contains(lower, "mock") {
		return "testing"
	}
	if strings.Contains(lower, "event") || strings.Contains(lower, "bus") || strings.Contains(lower, "pub") || strings.Contains(lower, "sub") {
		return "event_system"
	}
	if strings.Contains(lower, "plugin") || strings.Contains(lower, "extens") {
		return "plugin_system"
	}
	if strings.Contains(lower, "agent") || strings.Contains(lower, "orchestrat") {
		return "agent_system"
	}
	if strings.Contains(lower, "monitor") || strings.Contains(lower, "log") || strings.Contains(lower, "metric") || strings.Contains(lower, "trace") {
		return "observability"
	}
	if strings.Contains(lower, "middleware") || strings.Contains(lower, "filter") || strings.Contains(lower, "interceptor") {
		return "middleware"
	}
	return ""
}

func containsHTTP(imports []string) bool {
	for _, imp := range imports {
		lower := strings.ToLower(imp)
		if strings.Contains(lower, "http") || strings.Contains(lower, "gin") || strings.Contains(lower, "echo") || strings.Contains(lower, "fiber") || strings.Contains(lower, "chi") || strings.Contains(lower, "mux") {
			return true
		}
	}
	return false
}

func containsDatabase(imports []string) bool {
	for _, imp := range imports {
		lower := strings.ToLower(imp)
		if strings.Contains(lower, "sql") || strings.Contains(lower, "db") || strings.Contains(lower, "database") || strings.Contains(lower, "bolt") || strings.Contains(lower, "redis") || strings.Contains(lower, "mongo") || strings.Contains(lower, "postgres") || strings.Contains(lower, "mysql") {
			return true
		}
	}
	return false
}

func findPackageNode(name string, g *repo.CapabilityGraph) string {
	if g.Nodes["pkg_"+name] != nil {
		return "pkg_" + name
	}
	return ""
}

func shortenImport(imp string) string {
	parts := strings.Split(imp, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return imp
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
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
