package repo

import (
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// ExtractRequestFlows traces every detected HTTP route through the system,
// following call chains through handlers, services, repositories, and data models.
// Each flow is derived entirely from code analysis — no README required.
func ExtractRequestFlows(g *models.CapabilityGraph, cb *models.Codebase) []*models.RequestFlow {
	if g == nil {
		return nil
	}

	pkgMap := make(map[string]*models.Package)
	for _, pkg := range cb.Packages {
		pkgMap[pkg.Name] = pkg
	}

	classify := func(pkgName string) (isSvc, isRepo, isModel bool) {
		pkg, ok := pkgMap[pkgName]
		if !ok {
			return false, false, false
		}
		for _, seg := range pathSegments(pkg.Path) {
			switch seg {
			case "service", "services":
				isSvc = true
			case "repo", "repository", "repositories", "store", "stores", "db", "database":
				isRepo = true
			case "model", "models", "entity", "entities":
				isModel = true
			}
		}
		return
	}

	// find entrypoint nodes
	entrypoints := g.NodesByType(models.CapEntrypoint)
	entrypointName := ""
	if len(entrypoints) > 0 {
		entrypointName = entrypoints[0].Name
	}

	// build package→database mapping
	pkgDB := map[string]string{}
	for _, db := range g.NodesByType(models.CapDatabase) {
		pkgDB[db.Package] = db.Name
	}

	var flows []*models.RequestFlow
	routes := g.NodesByType(models.CapRoute)
	for _, route := range routes {
		flow := &models.RequestFlow{
			Method:     route.Properties["method"],
			Path:       route.Properties["path"],
			Package:    route.Package,
			Entrypoint: entrypointName,
		}

		// follow routes_to → handler
		handlerNode := findRouteHandler(g, route)
		if handlerNode == nil {
			flows = append(flows, flow)
			continue
		}
		flow.Handler = models.FlowStep{
			Name:     handlerNode.Name,
			Package:  handlerNode.Package,
			Location: handlerNode.Location,
		}

		// construct the handler's function node ID to trace calls
		fnID := handlerToFnID(handlerNode)

		// trace the call chain
		seen := map[string]bool{}
		traceHandlerCalls(g, fnID, classify, pkgMap, flow, seen)

		// attach database if handler's package or any touched service uses one
		if flow.Database == "" {
			if db, ok := pkgDB[handlerNode.Package]; ok {
				flow.Database = db
			}
		}
		if flow.Database == "" {
			for _, svc := range flow.Services {
				if db, ok := pkgDB[svc.Package]; ok {
					flow.Database = db
					break
				}
			}
		}

		flows = append(flows, flow)
	}
	return flows
}

// findRouteHandler follows the "routes_to" edge from a route node to its handler.
func findRouteHandler(g *models.CapabilityGraph, route *models.CapabilityNode) *models.CapabilityNode {
	for _, e := range g.EdgesFrom(route.ID) {
		if e.Type == "routes_to" {
			return g.NodeByID(e.TargetID)
		}
	}
	return nil
}

// handlerToFnID converts a handler node's ID to its corresponding function node ID.
func handlerToFnID(handler *models.CapabilityNode) string {
	// handler IDs are "handler:pkg.Name" or "handler:pkg.Receiver.ServeHTTP"
	// fn IDs are "fn:pkg.Name" or "fn:pkg.Receiver.ServeHTTP"
	if !strings.HasPrefix(handler.ID, "handler:") {
		return ""
	}
	return "fn:" + strings.TrimPrefix(handler.ID, "handler:")
}

// traceHandlerCalls walks the call graph from a handler's function node,
// classifying callees as services, repositories, or data model references.
func traceHandlerCalls(g *models.CapabilityGraph, fnID string,
	classify func(string) (bool, bool, bool),
	pkgMap map[string]*models.Package,
	flow *models.RequestFlow,
	seen map[string]bool) {

	if fnID == "" || seen[fnID] {
		return
	}
	seen[fnID] = true

	for _, e := range g.EdgesFrom(fnID) {
		if e.Type != "calls" {
			continue
		}
		calleeID := e.TargetID
		if seen[calleeID] {
			continue
		}
		seen[calleeID] = true

		// parse callee package and function name
		calleePkg, calleeFn := parseFnID(calleeID)
		if calleePkg == "" {
			continue
		}
		isSvc, isRepo, isModel := classify(calleePkg)

		step := models.FlowStep{
			Name:    calleeFn,
			Package: calleePkg,
			Location: lookupFnLocation(pkgMap, calleePkg, calleeFn),
		}

		switch {
		case isSvc:
			if !containsFlowStep(flow.Services, step) {
				flow.Services = append(flow.Services, step)
			}
			// recurse into service calls
			traceHandlerCalls(g, calleeID, classify, pkgMap, flow, seen)

		case isRepo:
			if !containsFlowStep(flow.Repositories, step) {
				flow.Repositories = append(flow.Repositories, step)
			}
			// recurse into repo calls to find model references
			traceHandlerCalls(g, calleeID, classify, pkgMap, flow, seen)

		case isModel:
			if !contains(flow.Models, calleePkg+"."+calleeFn) {
				flow.Models = append(flow.Models, calleePkg+"."+calleeFn)
			}

		default:
			// recurse into unknown callees to find deeper calls
			traceHandlerCalls(g, calleeID, classify, pkgMap, flow, seen)
		}
	}
}

// parseFnID extracts package and function name from an fn: ID.
func parseFnID(id string) (pkg, fn string) {
	if !strings.HasPrefix(id, "fn:") {
		return "", ""
	}
	rest := strings.TrimPrefix(id, "fn:")
	parts := strings.SplitN(rest, ".", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return rest, rest
}

// lookupFnLocation finds the Location of a function in a given package.
func lookupFnLocation(pkgMap map[string]*models.Package, pkgName, fnName string) models.Location {
	pkg, ok := pkgMap[pkgName]
	if !ok {
		return models.Location{}
	}
	for _, fn := range pkg.Functions {
		fullName := fn.Name
		if fn.Receiver != "" {
			fullName = fn.Receiver + "." + fn.Name
		}
		if fullName == fnName {
			return fn.Location
		}
	}
	return models.Location{}
}

func containsFlowStep(steps []models.FlowStep, s models.FlowStep) bool {
	for _, st := range steps {
		if st.Package == s.Package && st.Name == s.Name {
			return true
		}
	}
	return false
}


