package repo

import (
	"github.com/routerforge/cli/pkg/models"
)

// AnalyzeOwnership determines which packages own which capabilities,
// providing a map of responsibility distribution across the codebase.
func AnalyzeOwnership(g *models.CapabilityGraph, cb *models.Codebase) []*models.OwnershipInfo {
	if g == nil {
		return nil
	}

	pkgMap := make(map[string]*models.OwnershipInfo)
	for _, pkg := range cb.Packages {
		pkgMap[pkg.Name] = &models.OwnershipInfo{
			Package: pkg.Name,
			Path:    pkg.Path,
		}
	}

	for _, node := range g.Nodes {
		info, ok := pkgMap[node.Package]
		if !ok {
			info = &models.OwnershipInfo{Package: node.Package}
			pkgMap[node.Package] = info
		}

		switch node.Type {
		case models.CapRoute:
			info.Routes++
		case models.CapHandler:
			info.Handlers++
		case models.CapMiddleware:
			info.Middleware++
		case models.CapService:
			info.Services++
		case models.CapRepository:
			info.Repositories++
		case models.CapDataModel:
			info.DataModels++
		case models.CapInterface:
			info.Interfaces++
		case models.CapEntrypoint:
			info.Entrypoints++
		}
	}

	var result []*models.OwnershipInfo
	for _, info := range pkgMap {
		info.Total = info.Routes + info.Handlers + info.Middleware +
			info.Services + info.Repositories + info.DataModels +
			info.Interfaces + info.Entrypoints
		result = append(result, info)
	}

	return result
}
