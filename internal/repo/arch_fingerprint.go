package repo

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ArchitectureProfile struct {
	Architecture string            `json:"architecture"`
	Confidence   float64           `json:"confidence"`
	Layers       []string          `json:"layers"`
	Patterns     []string          `json:"patterns"`
	Evidence     map[string]string `json:"evidence"`
}

func (a *ASTAnalyzer) FingerprintArchitecture(root string) *ArchitectureProfile {
	evidence := make(map[string]string)

	hasHandlers := hasPatternInFiles(root, "handler", "Handler")
	hasRepos := hasPatternInFiles(root, "repository", "Repository", "repo")
	hasModels := hasPatternInFiles(root, "model", "Model", "entity")
	hasServices := hasPatternInFiles(root, "service", "Service", "usecase")
	hasControllers := hasPatternInFiles(root, "controller", "Controller")
	hasPorts := hasPatternInFiles(root, "port", "Port")
	hasAdapters := hasPatternInFiles(root, "adapter", "Adapter")
	hasDomain := hasPatternInFiles(root, "domain", "Domain")
	hasInfra := hasPatternInFiles(root, "infrastructure", "Infrastructure", "infra")
	hasCmd := hasPatternInFiles(root, "cmd", "main")
	hasInternal := hasPatternInFiles(root, "internal")
	hasPkg := hasPatternInFiles(root, "pkg")

	if hasHandlers {
		evidence["handlers"] = "handler/Handler pattern detected"
	}
	if hasRepos {
		evidence["repositories"] = "repository/repo pattern detected"
	}
	if hasModels {
		evidence["models"] = "model/entity pattern detected"
	}
	if hasServices {
		evidence["services"] = "service/usecase pattern detected"
	}
	if hasControllers {
		evidence["controllers"] = "controller pattern detected"
	}
	if hasPorts {
		evidence["ports"] = "port pattern detected"
	}
	if hasAdapters {
		evidence["adapters"] = "adapter pattern detected"
	}
	if hasDomain {
		evidence["domain"] = "domain package detected"
	}
	if hasInfra {
		evidence["infrastructure"] = "infrastructure package detected"
	}
	if hasCmd && hasInternal && hasPkg {
		evidence["go_layout"] = "Standard Go project layout (cmd/internal/pkg)"
	}

	arch, confidence, layers, patterns := classifyArchitecture(evidence)

	return &ArchitectureProfile{
		Architecture: arch,
		Confidence:   confidence,
		Layers:       layers,
		Patterns:     patterns,
		Evidence:     evidence,
	}
}

func hasPatternInFiles(root string, patterns ...string) bool {
	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return err
		}
		base := strings.ToLower(fi.Name())
		for _, p := range patterns {
			if strings.Contains(base, strings.ToLower(p)) {
				return os.ErrExist
			}
		}
		return nil
	})
	return err == os.ErrExist
}

type ArchDetector struct{}

func (d *ArchDetector) DetectInterfaceUsage(root string) map[string][]string {
	result := make(map[string][]string)

	err := filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil
		}

		pkgName := f.Name.Name

		ast.Inspect(f, func(n ast.Node) bool {
			if imp, ok := n.(*ast.ImportSpec); ok && imp.Path != nil {
				impPath := strings.Trim(imp.Path.Value, "\"")
				if strings.HasPrefix(impPath, root) || strings.Contains(impPath, "/") {
					short := shortenImport(impPath)
					if !contains(result[pkgName], short) {
						result[pkgName] = append(result[pkgName], short)
					}
				}
			}
			return true
		})
		return nil
	})
	if err != nil {
		return result
	}
	return result
}

func classifyArchitecture(evidence map[string]string) (string, float64, []string, []string) {
	score := 0

	layers := []string{}
	patterns := []string{}

	if hasKeys(evidence, "handlers", "services", "models") {
		score += 3
		layers = append(layers, "handlers", "services", "models")
		patterns = append(patterns, "layered_architecture")
	}

	if hasKeys(evidence, "ports", "adapters", "domain") {
		score += 3
		layers = append(layers, "ports", "adapters", "domain")
		patterns = append(patterns, "hexagonal_architecture")
	}

	if hasKeys(evidence, "controllers", "services", "models") {
		score += 3
		layers = append(layers, "controllers", "services", "models")
		patterns = append(patterns, "mvc_pattern")
	}

	if hasKeys(evidence, "domain", "infrastructure", "services") {
		score += 3
		layers = append(layers, "domain", "infrastructure", "services")
		patterns = append(patterns, "ddd_pattern")
	}

	if _, ok := evidence["go_layout"]; ok {
		score += 1
		layers = append(layers, "cmd", "internal", "pkg")
		patterns = append(patterns, "standard_go_layout")
	}

	confidence := float64(score) / float64(maxScore(evidence))

	architecture := "unknown"
	if confidence >= 0.7 {
		if hasKeys(evidence, "ports") || hasKeys(evidence, "adapters") {
			architecture = "hexagonal"
		} else if hasKeys(evidence, "controllers") {
			architecture = "mvc"
		} else if hasKeys(evidence, "domain") {
			architecture = "domain_driven"
		} else {
			architecture = "layered"
		}
	} else if confidence >= 0.4 {
		architecture = "modular_monolith"
	} else {
		architecture = "flat"
	}

	return architecture, confidence, unique(layers), unique(patterns)
}

func hasKeys(m map[string]string, keys ...string) bool {
	for _, k := range keys {
		if _, ok := m[k]; !ok {
			return false
		}
	}
	return true
}

func maxScore(evidence map[string]string) int {
	n := len(evidence)
	if n < 3 {
		return 9
	}
	return n * 3
}

func unique(slice []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, s := range slice {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
