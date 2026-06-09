package repo

import (
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// DetectLayerViolations analyzes the codebase for architectural constraint violations,
// including layer bypass, circular dependencies, wrong-direction imports, and DDD violations.
func DetectLayerViolations(cb *models.Codebase, g *models.CapabilityGraph) []*models.LayerViolation {
	if cb == nil {
		return nil
	}

	var violations []*models.LayerViolation

	// classify packages by layer
	layerClass := classifyPackages(cb)

	// build layer import matrix
	importEdges := buildImportEdgeMap(cb)
	layerMatrix := buildLayerDepMatrix(layerClass, importEdges)

	// 1. Handler → Repository bypass (layered architecture violation)
	if layerMatrix["handler"]["repository"] > 0 {
		// find the specific handler→repo import edges
		for _, pkg := range cb.Packages {
			if !hasLayerClass(layerClass, pkg, "handler", "controller", "api") {
				continue
			}
			for _, imp := range pkg.Imports {
				targetPkg := shortenImport(imp.Path)
				if hasLayerClass(layerClass, cb.PackageByName(targetPkg), "repository", "repo", "store", "db", "database") {
					violations = append(violations, &models.LayerViolation{
						Source:      pkg.Name,
						Target:      targetPkg,
						Description: "Handler bypasses service layer — imports repository directly",
						Severity:    "high",
						Location:    imp.Location,
					})
				}
			}
		}
	}

	// 2. Service → Infrastructure/Database (service should only go through repository)
	if layerMatrix["service"]["database"] > 0 || layerMatrix["service"]["infrastructure"] > 0 {
		for _, pkg := range cb.Packages {
			if !hasLayerClass(layerClass, pkg, "service", "services") {
				continue
			}
			for _, imp := range pkg.Imports {
				targetPkg := shortenImport(imp.Path)
				if hasLayerClass(layerClass, cb.PackageByName(targetPkg), "database", "db", "infrastructure", "infra") {
					violations = append(violations, &models.LayerViolation{
						Source:      pkg.Name,
						Target:      targetPkg,
						Description: "Service imports infrastructure/database directly — should use repository layer",
						Severity:    "medium",
						Location:    imp.Location,
					})
				}
			}
		}
	}

	// 3. Domain → Infrastructure (DDD violation)
	if layerMatrix["domain"]["infrastructure"] > 0 {
		for _, pkg := range cb.Packages {
			if !hasLayerClass(layerClass, pkg, "domain") {
				continue
			}
			for _, imp := range pkg.Imports {
				targetPkg := shortenImport(imp.Path)
				if hasLayerClass(layerClass, cb.PackageByName(targetPkg), "infrastructure", "infra") {
					violations = append(violations, &models.LayerViolation{
						Source:      pkg.Name,
						Target:      targetPkg,
						Description: "Domain layer imports infrastructure — DDD violation (domain must be dependency-free)",
						Severity:    "high",
						Location:    imp.Location,
					})
				}
			}
		}
	}

	// 4. Circular dependency detection
	cycles := detectCircularDeps(cb)
	for _, cycle := range cycles {
		violations = append(violations, &models.LayerViolation{
			Source:      cycle[0],
			Target:      cycle[len(cycle)-1],
			Description: "Circular dependency: " + strings.Join(cycle, " → "),
			Severity:    "high",
		})
	}

	// 5. Handler → Model (handler should not know about data models directly)
	if layerMatrix["handler"]["model"] > 0 {
		for _, pkg := range cb.Packages {
			if !hasLayerClass(layerClass, pkg, "handler", "handlers", "controller", "controllers") {
				continue
			}
			for _, imp := range pkg.Imports {
				targetPkg := shortenImport(imp.Path)
				if hasLayerClass(layerClass, cb.PackageByName(targetPkg), "model", "models", "entity", "entities") {
					violations = append(violations, &models.LayerViolation{
						Source:      pkg.Name,
						Target:      targetPkg,
						Description: "Handler imports model directly — should use DTOs or service layer",
						Severity:    "low",
						Location:    imp.Location,
					})
				}
			}
		}
	}

	return violations
}

// hasLayerClass checks if a package belongs to any of the given layer classes.
func hasLayerClass(classes map[string]bool, pkg *models.Package, layers ...string) bool {
	if pkg == nil {
		return false
	}
	for _, seg := range pathSegments(pkg.Path) {
		for _, layer := range layers {
			if seg == layer {
				return true
			}
		}
	}
	// also check class map by package name
	for _, layer := range layers {
		if classes[layer] {
			// check if this pkg is classified as that layer
			for _, seg := range pathSegments(pkg.Path) {
				if seg == layer {
					return true
				}
			}
		}
	}
	return false
}

// buildImportEdgeMap builds a map of package name → (target name → weight).
func buildImportEdgeMap(cb *models.Codebase) map[string]map[string]int {
	edges := make(map[string]map[string]int)
	for _, pkg := range cb.Packages {
		for _, imp := range pkg.Imports {
			target := shortenImport(imp.Path)
			if target == "" || target == pkg.Name {
				continue
			}
			if edges[pkg.Name] == nil {
				edges[pkg.Name] = make(map[string]int)
			}
			edges[pkg.Name][target]++
		}
	}
	return edges
}

// detectCircularDeps finds circular dependencies between packages using DFS.
func detectCircularDeps(cb *models.Codebase) [][]string {
	edges := make(map[string][]string)
	for _, pkg := range cb.Packages {
		for _, imp := range pkg.Imports {
			target := shortenImport(imp.Path)
			if target == "" || target == pkg.Name {
				continue
			}
			edges[pkg.Name] = append(edges[pkg.Name], target)
		}
	}

	var cycles [][]string
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	var path []string
	var dfs func(node string)
	dfs = func(node string) {
		visited[node] = true
		recStack[node] = true
		path = append(path, node)

		for _, neighbor := range edges[node] {
			if !visited[neighbor] {
				dfs(neighbor)
			} else if recStack[neighbor] {
				// found a cycle
				var cycle []string
				start := -1
				for i, n := range path {
					if n == neighbor {
						start = i
						break
					}
				}
				if start >= 0 {
					cycle = append(cycle, path[start:]...)
					cycle = append(cycle, neighbor)
				}
				if len(cycle) > 2 {
					cycles = append(cycles, cycle)
				}
			}
		}

		path = path[:len(path)-1]
		recStack[node] = false
	}

	for _, pkg := range cb.Packages {
		if !visited[pkg.Name] {
			dfs(pkg.Name)
		}
	}

	return cycles
}


