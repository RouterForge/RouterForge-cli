package repo

import (
	"fmt"
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// ExplainReport is a human-readable synthesis of what the analysis understands
// about a codebase. It is derived entirely from the Semantic Model — no README.
type ExplainReport struct {
	Name         string                     `json:"name"`
	Summary      string                     `json:"summary"`
	Architecture string                     `json:"architecture"`
	Stats        ExplainStats               `json:"stats"`
	Routes       []ExplainRoute             `json:"routes"`
	Flows        []*models.RequestFlow      `json:"flows,omitempty"`
	Violations   []*models.LayerViolation   `json:"violations,omitempty"`
	Ownership    []*models.OwnershipInfo    `json:"ownership,omitempty"`
	Capabilities []*models.Capability       `json:"capabilities,omitempty"`
}

type ExplainStats struct {
	Packages       int `json:"packages"`
	Functions      int `json:"functions"`
	Types          int `json:"types"`
	Interfaces     int `json:"interfaces"`
	CapGraphNodes  int `json:"cap_graph_nodes"`
	CapGraphEdges  int `json:"cap_graph_edges"`
	CallGraphNodes int `json:"call_graph_nodes"`
	CallGraphEdges int `json:"call_graph_edges"`
}

type ExplainRoute struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Handler   string `json:"handler,omitempty"`
	Package   string `json:"package"`
}

// BuildExplainReport synthesizes everything the Semantic Model knows about a codebase
// into a structured, human-readable report.
func BuildExplainReport(cb *models.Codebase) *ExplainReport {
	r := &ExplainReport{}

	// derive a meaningful name from the root path
	parts := strings.Split(strings.TrimRight(cb.Root, "/"), "/")
	r.Name = parts[len(parts)-1]

	// aggregate stats
	pkgNames := map[string]bool{}
	funcCount := 0
	typeCount := 0
	ifaceCount := 0
	for _, pkg := range cb.Packages {
		pkgNames[pkg.Name] = true
		funcCount += len(pkg.Functions)
		typeCount += len(pkg.Types)
		ifaceCount += len(pkg.Interfaces)
	}
	r.Stats.Packages = len(pkgNames)
	r.Stats.Functions = funcCount
	r.Stats.Types = typeCount
	r.Stats.Interfaces = ifaceCount

	if cb.CapabilityGraph != nil {
		r.Stats.CapGraphNodes = len(cb.CapabilityGraph.Nodes)
		r.Stats.CapGraphEdges = len(cb.CapabilityGraph.Edges)
	}
	if cb.CallGraph != nil {
		r.Stats.CallGraphNodes = len(cb.CallGraph.Nodes)
		r.Stats.CallGraphEdges = len(cb.CallGraph.Edges)
	}

	// architecture
	if cb.Architecture != nil {
		r.Architecture = fmt.Sprintf("%s (%.0f%% confidence, %d layers: %s)",
			cb.Architecture.Architecture,
			cb.Architecture.Confidence*100,
			len(cb.Architecture.Layers),
			strings.Join(cb.Architecture.Layers, ", "))
	} else {
		r.Architecture = "unknown"
	}

	// routes
	r.Routes = extractRouteSummary(cb.CapabilityGraph)

	// flows, violations, ownership
	r.Flows = cb.RequestFlows
	r.Violations = cb.LayerViolations
	r.Ownership = cb.Ownership
	r.Capabilities = cb.Capabilities

	// build one-line summary
	parts2 := []string{
		fmt.Sprintf("%d packages", r.Stats.Packages),
		fmt.Sprintf("%d functions", r.Stats.Functions),
		fmt.Sprintf("%d types", r.Stats.Types),
	}
	if len(r.Routes) > 0 {
		parts2 = append(parts2, fmt.Sprintf("%d routes", len(r.Routes)))
	}
	if len(r.Violations) > 0 {
		parts2 = append(parts2, fmt.Sprintf("%d violations", len(r.Violations)))
	}
	r.Summary = strings.Join(parts2, ", ")

	return r
}

func extractRouteSummary(g *models.CapabilityGraph) []ExplainRoute {
	if g == nil {
		return nil
	}
	var routes []ExplainRoute
	for _, node := range g.NodesByType(models.CapRoute) {
		r := ExplainRoute{
			Method:  node.Properties["method"],
			Path:    node.Properties["path"],
			Package: node.Package,
		}
		// find the handler this route routes to
		for _, e := range g.EdgesFrom(node.ID) {
			if e.Type == "routes_to" {
				handler := g.NodeByID(e.TargetID)
				if handler != nil {
					r.Handler = handler.Name
				}
				break
			}
		}
		routes = append(routes, r)
	}
	return routes
}

// Markdown renders the ExplainReport as human-readable Markdown.
func (r *ExplainReport) Markdown() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Explain: %s\n\n", r.Name))
	b.WriteString(fmt.Sprintf("**Summary:** %s\n\n", r.Summary))

	// Architecture
	b.WriteString("## Architecture\n\n")
	b.WriteString(fmt.Sprintf("%s\n\n", r.Architecture))

	// Stats
	b.WriteString("## Stats\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n|---|---|\n"))
	b.WriteString(fmt.Sprintf("| Packages | %d |\n", r.Stats.Packages))
	b.WriteString(fmt.Sprintf("| Functions | %d |\n", r.Stats.Functions))
	b.WriteString(fmt.Sprintf("| Types | %d |\n", r.Stats.Types))
	b.WriteString(fmt.Sprintf("| Interfaces | %d |\n", r.Stats.Interfaces))
	if r.Stats.CapGraphNodes > 0 {
		b.WriteString(fmt.Sprintf("| Capability Graph | %d nodes, %d edges |\n", r.Stats.CapGraphNodes, r.Stats.CapGraphEdges))
	}
	if r.Stats.CallGraphNodes > 0 {
		b.WriteString(fmt.Sprintf("| Call Graph | %d nodes, %d edges |\n", r.Stats.CallGraphNodes, r.Stats.CallGraphEdges))
	}
	b.WriteString("\n")

	// Routes
	if len(r.Routes) > 0 {
		b.WriteString("## Routes\n\n")
		b.WriteString("| Method | Path | Handler | Package |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, route := range r.Routes {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
				route.Method, route.Path, route.Handler, route.Package))
		}
		b.WriteString("\n")
	}

	// Request Flows
	if len(r.Flows) > 0 {
		b.WriteString("## Request Flows\n\n")
		for _, f := range r.Flows {
			b.WriteString(fmt.Sprintf("### %s %s\n\n", f.Method, f.Path))
			b.WriteString(fmt.Sprintf("- **Handler:** %s (in %s)\n", f.Handler.Name, f.Handler.Package))
			if len(f.Middleware) > 0 {
				ms := []string{}
				for _, m := range f.Middleware {
					ms = append(ms, m.Name)
				}
				b.WriteString(fmt.Sprintf("- **Middleware:** %s\n", strings.Join(ms, " → ")))
			}
			if len(f.Services) > 0 {
				ss := []string{}
				for _, s := range f.Services {
					ss = append(ss, fmt.Sprintf("%s (%s)", s.Name, s.Package))
				}
				b.WriteString(fmt.Sprintf("- **Services:** %s\n", strings.Join(ss, ", ")))
			}
			if len(f.Repositories) > 0 {
				rs := []string{}
				for _, r := range f.Repositories {
					rs = append(rs, fmt.Sprintf("%s (%s)", r.Name, r.Package))
				}
				b.WriteString(fmt.Sprintf("- **Repositories:** %s\n", strings.Join(rs, ", ")))
			}
			if len(f.Models) > 0 {
				b.WriteString(fmt.Sprintf("- **Data Models:** %s\n", strings.Join(f.Models, ", ")))
			}
			if f.Database != "" {
				b.WriteString(fmt.Sprintf("- **Database:** %s\n", f.Database))
			}
			b.WriteString("\n")
		}
	}

	// Layer Violations
	if len(r.Violations) > 0 {
		b.WriteString("## Layer Violations\n\n")
		b.WriteString("| Severity | Source | Target | Description |\n")
		b.WriteString("|---|---|---|---|\n")
		for _, v := range r.Violations {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", v.Severity, v.Source, v.Target, v.Description))
		}
		b.WriteString("\n")
	} else {
		b.WriteString("## Layer Violations\n\nNone detected — clean architecture.\n\n")
	}

	// Ownership
	if len(r.Ownership) > 0 {
		b.WriteString("## Ownership\n\n")
		b.WriteString("| Package | Path | Routes | Handlers | Middleware | Services | Repos | Models | Interfaces | Entry | Total |\n")
		b.WriteString("|---|---|---|---|---|---|---|---|---|---|---|\n")
		for _, o := range r.Ownership {
			path := o.Path
			if len(path) > 40 {
				path = "..." + path[len(path)-37:]
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d | %d |\n",
				o.Package, path, o.Routes, o.Handlers, o.Middleware,
				o.Services, o.Repositories, o.DataModels, o.Interfaces,
				o.Entrypoints, o.Total))
		}
		b.WriteString("\n")
	}

	// Capabilities
	if len(r.Capabilities) > 0 {
		b.WriteString("## Capabilities\n\n")
		for _, c := range r.Capabilities {
			b.WriteString(fmt.Sprintf("- **%s** (%s, %.0f%%): %s\n", c.Name, c.Category, c.Confidence*100, c.Description))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// Short returns a single-line summary of the report.
func (r *ExplainReport) Short() string {
	parts := []string{r.Summary, "Arch: " + r.Architecture}
	if len(r.Routes) > 0 {
		parts = append(parts, fmt.Sprintf("Routes: %d", len(r.Routes)))
	}
	return strings.Join(parts, " | ")
}
