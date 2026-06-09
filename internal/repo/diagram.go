package repo

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ServiceMap struct {
	Services []Service `json:"services"`
}

type Service struct {
	Name        string   `json:"name"`
	Package     string   `json:"package"`
	Type        string   `json:"type"` // "cmd", "internal", "pkg"
	Description string   `json:"description"`
	Dependencies []string `json:"dependencies"`
	Exports     []string `json:"exports"`
	Entrypoints []string `json:"entrypoints"`
}

func BuildServiceMap(root string) (*ServiceMap, error) {
	sm := &ServiceMap{}

	aa := &ASTAnalyzer{}
	pkgs, depGraph, err := aa.AnalyzeGoRepo(root)
	if err != nil {
		return nil, fmt.Errorf("analyze repo: %w", err)
	}

	for _, pkg := range pkgs {
		s := Service{
			Name:    pkg.Name,
			Package: pkg.Path,
			Type:    classifyPackageType(pkg.Path),
		}

		for _, fn := range pkg.Functions {
			if fn.IsExported {
				s.Exports = append(s.Exports, fn.Name)
			}
		}

		for _, imp := range pkg.Imports {
			short := shortenImport(imp)
			if short != "" && short != pkg.Name {
				s.Dependencies = append(s.Dependencies, short)
			}
		}

		if containsCmd(pkg.Functions) || isMainLike(pkg.Name, pkg.Path) {
			s.Entrypoints = append(s.Entrypoints, "main")
		}
		if strings.Contains(pkg.Path, "cmd") || strings.HasPrefix(pkg.Name, "main") {
			s.Type = "cmd"
		}

		sm.Services = append(sm.Services, s)
	}

	if depGraph != nil {
		for _, edge := range depGraph.Edges {
			for i, s := range sm.Services {
				if s.Name == edge.Source {
					sm.Services[i].Dependencies = append(sm.Services[i].Dependencies, edge.Target)
				}
			}
		}
	}

	return sm, nil
}

func classifyPackageType(path string) string {
	if strings.Contains(path, "/cmd/") || strings.HasSuffix(path, "/cmd") {
		return "cmd"
	}
	if strings.Contains(path, "/internal/") {
		return "internal"
	}
	if strings.Contains(path, "/pkg/") {
		return "pkg"
	}
	return "other"
}

func containsCmd(fns []FunctionInfo) bool {
	for _, fn := range fns {
		if fn.Name == "main" {
			return true
		}
	}
	return false
}

func isMainLike(name, path string) bool {
	return name == "main" || strings.HasSuffix(path, "/cmd")
}

type ArchitectureDoc struct {
	Title        string          `json:"title"`
	Language     string          `json:"language"`
	Patterns     []Pattern       `json:"patterns"`
	ArchProfile  *ArchitectureProfile `json:"arch_profile,omitempty"`
	ServiceMap   *ServiceMap     `json:"service_map,omitempty"`
	DepMermaid   string          `json:"dep_mermaid"`
	CallMermaid  string          `json:"call_mermaid"`
	ArchMermaid  string          `json:"arch_mermaid"`
	Summary      string          `json:"summary"`
}

func GenerateArchitectureDoc(root string) (*ArchitectureDoc, error) {
	doc := &ArchitectureDoc{
		Title: filepath.Base(root),
	}

	sa := &SourceAnalyzer{}
	doc.Language = sa.DetectLanguage(root)

	pd := &PatternDetector{}
	doc.Patterns = pd.Detect(root)

	aa := &ASTAnalyzer{}

	if doc.Language == "go" {
		doc.ArchProfile = aa.FingerprintArchitecture(root)

		sm, err := BuildServiceMap(root)
		if err == nil {
			doc.ServiceMap = sm
		}

		_, depGraph, err := aa.AnalyzeGoRepo(root)
		if err == nil && depGraph != nil {
			doc.DepMermaid = depGraph.Markdown()
		}

		cg, err := aa.BuildCallGraph(root)
		if err == nil && cg != nil {
			doc.CallMermaid = cg.Markdown()
		}

		if doc.ArchProfile != nil {
			var b strings.Builder
			b.WriteString("graph TD\n")
			b.WriteString(fmt.Sprintf("  subgraph %s\n", doc.Title))
			for _, layer := range doc.ArchProfile.Layers {
				b.WriteString(fmt.Sprintf("    subgraph %s\n", layer))
				for _, s := range sm.Services {
					if strings.Contains(s.Package, "/"+layer+"/") || strings.HasPrefix(s.Package, layer) {
						b.WriteString(fmt.Sprintf("      %s[%s]\n", sanitizeNode(s.Name), s.Name))
					}
				}
				b.WriteString("    end\n")
			}
			b.WriteString("  end\n")
			for _, s := range sm.Services {
				for _, dep := range s.Dependencies {
					b.WriteString(fmt.Sprintf("  %s --> %s\n", sanitizeNode(s.Name), sanitizeNode(dep)))
				}
			}
			doc.ArchMermaid = b.String()
		}

		parts := []string{fmt.Sprintf("Language: %s", doc.Language)}
		parts = append(parts, fmt.Sprintf("Patterns: %d", len(doc.Patterns)))
		if doc.ArchProfile != nil {
			parts = append(parts, fmt.Sprintf("Architecture: %s", doc.ArchProfile.Architecture))
		}
		if doc.ServiceMap != nil {
			parts = append(parts, fmt.Sprintf("Services: %d", len(doc.ServiceMap.Services)))
		}
		doc.Summary = strings.Join(parts, " | ")
	}

	return doc, nil
}

func (doc *ArchitectureDoc) Markdown() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Architecture: %s\n\n", doc.Title))
	b.WriteString(fmt.Sprintf("**Summary:** %s\n\n", doc.Summary))

	if doc.ArchProfile != nil {
		b.WriteString(fmt.Sprintf("## Architecture Style\n\n**%s** (%.0f%% confidence)\n\n", doc.ArchProfile.Architecture, doc.ArchProfile.Confidence*100))
		if len(doc.ArchProfile.Layers) > 0 {
			b.WriteString("**Layers:** " + strings.Join(doc.ArchProfile.Layers, ", ") + "\n\n")
		}
		if len(doc.ArchProfile.Evidence) > 0 {
			b.WriteString("**Evidence:**\n")
			for k, v := range doc.ArchProfile.Evidence {
				b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
			}
			b.WriteString("\n")
		}
	}

	if doc.ServiceMap != nil && len(doc.ServiceMap.Services) > 0 {
		b.WriteString("## Service Map\n\n")
		b.WriteString("| Service | Type | Exports | Dependencies |\n")
		b.WriteString("|---------|------|---------|-------------|\n")
		for _, s := range doc.ServiceMap.Services {
			exports := strings.Join(s.Exports, ", ")
			if len(exports) > 40 {
				exports = exports[:37] + "..."
			}
			deps := strings.Join(s.Dependencies, ", ")
			if len(deps) > 40 {
				deps = deps[:37] + "..."
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", s.Name, s.Type, exports, deps))
		}
		b.WriteString("\n")
	}

	if doc.DepMermaid != "" {
		b.WriteString("## Dependency Graph\n\n```mermaid\n")
		b.WriteString(doc.DepMermaid)
		b.WriteString("```\n\n")
	}

	if doc.CallMermaid != "" {
		b.WriteString("## Call Graph\n\n```mermaid\n")
		b.WriteString(doc.CallMermaid)
		b.WriteString("```\n\n")
	}

	if doc.ArchMermaid != "" {
		b.WriteString("## Architecture Diagram\n\n```mermaid\n")
		b.WriteString(doc.ArchMermaid)
		b.WriteString("```\n\n")
	}

	b.WriteString("## Patterns\n\n")
	for _, p := range doc.Patterns {
		b.WriteString(fmt.Sprintf("- **%s** (%.0f%% confidence)\n", p.Name, p.Confidence*100))
	}

	return b.String()
}

func sanitizeNode(name string) string {
	r := strings.NewReplacer(".", "_", "/", "_", "-", "_", " ", "_")
	return r.Replace(name)
}

func SaveArchitectureDoc(root, outputPath string) error {
	doc, err := GenerateArchitectureDoc(root)
	if err != nil {
		return err
	}

	md := doc.Markdown()
	if outputPath == "" {
		fmt.Println(md)
		return nil
	}

	dir := filepath.Dir(outputPath)
	os.MkdirAll(dir, 0755)
	return os.WriteFile(outputPath, []byte(md), 0644)
}

func GenerateServiceMap(root string) (string, error) {
	sm, err := BuildServiceMap(root)
	if err != nil {
		return "", err
	}
	sort.Slice(sm.Services, func(i, j int) bool {
		return sm.Services[i].Name < sm.Services[j].Name
	})
	var b strings.Builder
	b.WriteString("# Service Map\n\n")
	b.WriteString("```mermaid\ngraph LR\n")
	for _, s := range sm.Services {
		shape := "[" + s.Name + "]"
		if s.Type == "cmd" {
			shape = "(" + s.Name + ")"
		}
		b.WriteString(fmt.Sprintf("  %s%s\n", sanitizeNode(s.Name), shape))
	}
	for _, s := range sm.Services {
		for _, dep := range s.Dependencies {
			b.WriteString(fmt.Sprintf("  %s --> %s\n", sanitizeNode(s.Name), sanitizeNode(dep)))
		}
	}
	b.WriteString("```\n")
	return b.String(), nil
}
