package repo

import (
	"path/filepath"
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// fingerprintArchitecture analyzes code relationships to determine the
// architectural style, rather than relying on directory name heuristics.
func fingerprintArchitecture(cb *models.Codebase, importEdges map[string]map[string]int) *models.ArchProfile {
	var evidence []*models.ArchEvidence

	layerClass := classifyPackages(cb)
	deps := buildLayerDepMatrix(layerClass, importEdges)

	scores := map[string]float64{
		"layered":   0,
		"hexagonal": 0,
		"mvc":       0,
		"ddd":       0,
	}
	maxPossible := map[string]float64{
		"layered":   2,
		"hexagonal": 3,
		"mvc":       2,
		"ddd":       3,
	}

	// --- Layered architecture checks ---

	// Check 1: Handler → Service import
	if h := deps["handler"]["service"]; h > 0 {
		scores["layered"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Handlers import services (layered)",
			Source:      "handler",
			Target:      "service",
			Positive:    true,
		})
	}
	if h := deps["handler"]["service"]; h > 0 {
		scores["mvc"] += 1
	}

	// Check 2: Service → Repository import
	if deps["service"]["repository"] > 0 {
		scores["layered"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Services import repositories (layered)",
			Source:      "service",
			Target:      "repository",
			Positive:    true,
		})
	}
	if deps["service"]["repository"] > 0 {
		scores["mvc"] += 1
	}

	// Layer violation: Handler → Repository (skipping service layer)
	if deps["handler"]["repository"] > 0 && deps["handler"]["service"] == 0 {
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "layer_violation",
			Description: "Handlers import repositories directly, bypassing services",
			Source:      "handler",
			Target:      "repository",
			Positive:    false,
		})
		scores["layered"] -= 0.5
	}

	// --- Hexagonal architecture checks ---

	// Check 1: Port interfaces exist
	if _, ok := layerClass["port"]; ok {
		scores["hexagonal"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Has port layer (interfaces defined in domain)",
			Source:      "port",
			Positive:    true,
		})
	}

	// Check 2: Adapter → Port import (adapter implements port)
	if deps["adapter"]["port"] > 0 {
		scores["hexagonal"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Adapters import ports (dependency inversion)",
			Source:      "adapter",
			Target:      "port",
			Positive:    true,
		})
	}

	// Check 3: Port → Domain or Port → Model (port defines interface used by domain)
	if deps["port"]["domain"] > 0 || deps["port"]["model"] > 0 {
		scores["hexagonal"] += 0.5
	}
	if deps["handler"]["port"] > 0 || deps["handler"]["adapter"] > 0 {
		scores["hexagonal"] += 0.5
	}

	// --- DDD checks ---

	// Check 1: Domain layer has zero external infra deps
	if _, ok := layerClass["domain"]; ok {
		scores["ddd"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Has domain layer",
			Source:      "domain",
			Positive:    true,
		})
	}

	// Domain imports infrastructure = DDD violation
	if deps["domain"]["infrastructure"] > 0 {
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "layer_violation",
			Description: "Domain imports infrastructure (DDD violation)",
			Source:      "domain",
			Target:      "infrastructure",
			Positive:    false,
		})
		scores["ddd"] -= 1
	}

	// Infrastructure → Domain imports are fine (DIP)
	if deps["infrastructure"]["domain"] > 0 {
		scores["ddd"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "import_direction",
			Description: "Infrastructure depends on domain (dependency inversion)",
			Source:      "infrastructure",
			Target:      "domain",
			Positive:    true,
		})
	}

	// Domain defined interfaces implemented by infrastructure
	domainIfaceImpls := countCrossLayerImplementors(cb, "domain")
	infraIfaceImpls := countCrossLayerImplementors(cb, "infrastructure")
	if domainIfaceImpls > 0 || infraIfaceImpls > 0 {
		scores["ddd"] += 1
		evidence = append(evidence, &models.ArchEvidence{
			Kind:        "interface_impl",
			Description: "Domain interfaces implemented by infrastructure layer",
			Positive:    true,
		})
	}

	// --- Determine winner ---
	winner := "flat"
	bestScore := 0.0
	archNames := []string{"layered", "hexagonal", "mvc", "ddd"}
	for _, name := range archNames {
		maxVal := maxPossible[name]
		if maxVal > 0 {
			normalized := scores[name] / maxVal
			if normalized > bestScore {
				bestScore = normalized
				winner = name
			}
		}
	}

	// collect detected layers
	var layers []string
	for name := range layerClass {
		layers = append(layers, name)
	}

	var archPatterns []string
	for _, name := range archNames {
		if scores[name] > 0 {
			archPatterns = append(archPatterns, name+"_architecture")
		}
	}

	confidence := bestScore
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.1 {
		winner = "flat"
	}

	return &models.ArchProfile{
		Architecture: winner,
		Confidence:   confidence,
		Layers:       layers,
		Patterns:     archPatterns,
		Evidence:     evidence,
	}
}

// classifyPackages maps each package to one or more layer categories based on its path.
func classifyPackages(cb *models.Codebase) map[string]bool {
	classes := make(map[string]bool)
	for _, pkg := range cb.Packages {
		dirs := pathSegments(pkg.Path)
		for _, d := range dirs {
			switch d {
			case "handler", "handlers":
				classes["handler"] = true
			case "service", "services":
				classes["service"] = true
			case "repository", "repositories", "repo", "repos", "store", "stores", "db", "database":
				classes["repository"] = true
			case "model", "models", "entity", "entities":
				classes["model"] = true
			case "domain":
				classes["domain"] = true
			case "port", "ports":
				classes["port"] = true
			case "adapter", "adapters":
				classes["adapter"] = true
			case "infrastructure", "infra":
				classes["infrastructure"] = true
			case "controller", "controllers":
				classes["controller"] = true
			case "cmd":
				classes["cmd"] = true
			case "internal":
				classes["internal"] = true
			case "pkg":
				classes["pkg"] = true
			case "api":
				classes["api"] = true
			case "middleware", "middlewares":
				classes["middleware"] = true
			case "config", "configuration":
				classes["config"] = true
			}
		}
	}
	return classes
}

// buildLayerDepMatrix computes import counts between layer categories.
func buildLayerDepMatrix(layerClass map[string]bool, edges map[string]map[string]int) map[string]map[string]int {
	matrix := make(map[string]map[string]int)
	for src := range layerClass {
		matrix[src] = make(map[string]int)
		for tgt := range layerClass {
			matrix[src][tgt] = 0
		}
	}

	// If no layers classified, return empty matrix
	if len(layerClass) == 0 {
		return matrix
	}

	for srcPkg, targets := range edges {
		srcClass := classifySingle(srcPkg, layerClass)
		if srcClass == "" {
			continue
		}
		for tgtPkg, weight := range targets {
			tgtClass := classifySingle(tgtPkg, layerClass)
			if tgtClass == "" {
				continue
			}
			matrix[srcClass][tgtClass] += weight
		}
	}

	return matrix
}

func classifySingle(pkgName string, layerClass map[string]bool) string {
	lower := strings.ToLower(pkgName)
	for layer := range layerClass {
		// check if pkgName or any path segment matches the layer
		segments := pathSegments(lower)
		for _, s := range segments {
			if s == layer {
				return layer
			}
		}
	}
	return ""
}

func pathSegments(p string) []string {
	var segs []string
	for _, part := range strings.Split(filepath.ToSlash(p), "/") {
		part = strings.TrimSpace(part)
		if part != "" {
			segs = append(segs, strings.ToLower(part))
		}
	}
	return segs
}

func countCrossLayerImplementors(cb *models.Codebase, layer string) int {
	count := 0
	for _, pkg := range cb.Packages {
		if !pathContainsLayer(pkg.Path, layer) {
			continue
		}
		for _, iface := range pkg.Interfaces {
			if len(iface.Implementors) > 0 {
				count++
			}
		}
	}
	return count
}

func pathContainsLayer(path, layer string) bool {
	for _, seg := range pathSegments(path) {
		if seg == layer {
			return true
		}
	}
	return false
}
