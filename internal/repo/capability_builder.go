package repo

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/routerforge/cli/pkg/models"
)

// BuildCapabilityGraph runs all analyzers against the Codebase and produces
// a connected CapabilityGraph that answers: what does it do, how, and who is responsible.
func BuildCapabilityGraph(cb *models.Codebase) *models.CapabilityGraph {
	g := &models.CapabilityGraph{}

	extractRoutes(g, cb)
	extractHandlers(g, cb)
	extractMiddleware(g, cb)
	extractServices(g, cb)
	extractRepositories(g, cb)
	extractDataModels(g, cb)
	extractInterfacesAndImplementations(g, cb)
	extractPackageNodes(g, cb)
	extractEntrypoints(g, cb)
	extractDatabases(g, cb)
	linkCallSites(g, cb)

	return g
}

// routerMethods are known HTTP router registration method names.
var routerMethods = map[string]bool{
	"Handle": true, "HandleFunc": true,
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
	"Any": true, "Route": true, "Group": true,
}

func extractRoutes(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			for _, cs := range fn.Calls {
				if !routerMethods[cs.Callee] {
					continue
				}
				if len(cs.Args) == 0 {
					continue
				}
		routePath := cs.Args[0]
			method := "ANY"
			switch cs.Callee {
			case "Handle", "HandleFunc":
				method, routePath = parseRoutePattern(routePath)
			default:
				method = cs.Callee
			}
				nodeID := "route:" + method + ":" + routePath
				handlerName := ""
				if len(cs.Args) > 1 {
					handlerName = cs.Args[1]
				}
				node := g.AddNode(nodeID, models.CapRoute,
					method+" "+routePath, pkg.Name,
					"HTTP route: "+method+" "+routePath, cs.Location,
					map[string]string{"method": method, "path": routePath})

				// link route to handler if we can identify one
				if handlerName != "" && handlerName != "func(...)" {
					handlerID := "handler:" + handlerName
					g.AddEdge(node.ID, handlerID, "routes_to", "routes to "+handlerName)
				}

				// link route to calling function (the registrar)
				registrarID := "fn:" + pkg.Name + "." + fn.Name
				if fn.Receiver != "" {
					registrarID = "fn:" + pkg.Name + "." + fn.Receiver + "." + fn.Name
				}
				g.AddEdge(registrarID, node.ID, "registers", "registers route")
			}
		}
	}
}

func extractHandlers(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			// ServeHTTP method = standard Go HTTP handler
			if fn.Name == "ServeHTTP" && fn.Receiver != "" {
				id := "handler:" + pkg.Name + "." + fn.Receiver + ".ServeHTTP"
				g.AddNode(id, models.CapHandler,
					fn.Receiver+".ServeHTTP", pkg.Name,
					"ServeHTTP handler on "+fn.Receiver, fn.Location,
					map[string]string{"receiver": fn.Receiver})

				// link handler to its type
				typeID := "type:" + pkg.Name + "." + fn.Receiver
				g.AddEdge(id, typeID, "method_of", "method of "+fn.Receiver)

				// link handler to its function node for call graph traceability
				fnID := "fn:" + pkg.Name + "." + fn.Receiver + ".ServeHTTP"
				g.AddEdge(id, fnID, "handles_as", "handles as "+fn.Receiver+".ServeHTTP")
			}

			// handler-like signatures: (w http.ResponseWriter, r *http.Request)
			if isHandlerSignature(fn) {
				id := "handler:" + pkg.Name + "." + fn.Name
				g.AddNode(id, models.CapHandler,
					fn.Name, pkg.Name,
					"HTTP handler function: "+fn.Name, fn.Location, nil)

				// link handler to its function node
				fnID := "fn:" + pkg.Name + "." + fn.Name
				if fn.Receiver != "" {
					fnID = "fn:" + pkg.Name + "." + fn.Receiver + "." + fn.Name
				}
				g.AddEdge(id, fnID, "handles_as", "handles as "+fn.Name)
			}
		}
	}
}

func extractMiddleware(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			// middleware by name
			if strings.Contains(fn.Name, "Middleware") || strings.Contains(fn.Name, "middleware") {
				id := "middleware:" + pkg.Name + "." + fn.Name
				g.AddNode(id, models.CapMiddleware,
					fn.Name, pkg.Name,
					"Middleware function: "+fn.Name, fn.Location, nil)
			}

			// middleware by call pattern: fn is passed as arg to Use, With, etc.
			for _, cs := range fn.Calls {
				if cs.Callee == "Use" || cs.Callee == "With" {
					for _, arg := range cs.Args {
						if arg != fn.Name && !strings.HasPrefix(arg, "\"") {
							mwID := "middleware:" + arg
							handlerID := "handler:" + pkg.Name + "." + fn.Name
							g.AddEdge(mwID, handlerID, "wraps", "wraps handler "+fn.Name)
						}
					}
				}
			}
		}
	}
}

func extractServices(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		path := filepath.ToSlash(pkg.Path)
		segments := strings.Split(strings.ToLower(path), "/")
		isService := false
		for _, seg := range segments {
			if seg == "service" || seg == "services" {
				isService = true
				break
			}
		}

		// also detect service by interface definition pattern
		hasServiceIface := false
		for _, iface := range pkg.Interfaces {
			for _, m := range iface.Methods {
				if strings.Contains(m, "Service") || strings.Contains(m, "service") {
					hasServiceIface = true
					break
				}
			}
			if hasServiceIface {
				break
			}
		}

		if isService || hasServiceIface {
			id := "service:" + pkg.Name
			g.AddNode(id, models.CapService,
				pkg.Name, pkg.Name,
				"Service layer: "+pkg.Path, pkg.Location, nil)

			// link service to its interfaces
			for _, iface := range pkg.Interfaces {
				ifaceID := "iface:" + pkg.Name + "." + iface.Name
				g.AddEdge(ifaceID, id, "contract_for", "interface contract for service")
			}

			// link service to any packages it imports (dependencies)
			for _, imp := range pkg.Imports {
				target := shortenImport(imp.Path)
				if target == "" || target == pkg.Name {
					continue
				}
				depID := "pkg:" + target
				g.AddEdge(id, depID, "depends_on", "imports "+target)
			}
		}
	}
}

// repositoryMethods are typical CRUD method names that indicate a repository/storage type.
var repositoryMethods = map[string]bool{
	"Create": true, "Find": true, "FindByID": true, "Get": true,
	"GetByID": true, "Update": true, "Delete": true, "Save": true,
	"Upsert": true, "Query": true, "List": true, "Count": true,
	"FindAll": true, "FindOne": true,
}

func extractRepositories(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		path := filepath.ToSlash(pkg.Path)
		segments := strings.Split(strings.ToLower(path), "/")
		isRepo := false
		for _, seg := range segments {
			if seg == "repo" || seg == "repository" || seg == "repositories" ||
				seg == "store" || seg == "stores" || seg == "db" || seg == "database" {
				isRepo = true
				break
			}
		}

		hasRepoType := false
		typeMethods := map[string]int{}
		for _, fn := range pkg.Functions {
			if fn.Receiver != "" {
				if repositoryMethods[fn.Name] {
					typeMethods[fn.Receiver]++
				}
			}
		}
		for typeName, count := range typeMethods {
			if count >= 2 {
				hasRepoType = true
				repoID := "repo:" + pkg.Name + "." + typeName
				g.AddNode(repoID, models.CapRepository,
					typeName, pkg.Name,
					"Repository/storage: "+typeName+" ("+itoa(count)+" CRUD methods)",
					findTypeLocation(pkg, typeName), nil)
			}
		}

		if isRepo && !hasRepoType {
			id := "repo:" + pkg.Name
			g.AddNode(id, models.CapRepository,
				pkg.Name, pkg.Name,
				"Repository package: "+pkg.Path, pkg.Location, nil)
		}
	}
}

func extractDataModels(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, typ := range pkg.Types {
			if typ.Kind != "struct" || len(typ.Fields) == 0 {
				continue
			}
			hasTag := false
			tagTypes := []string{}
			for _, f := range typ.Fields {
				if strings.Contains(f.Tag, "json:") {
					hasTag = true
					if !contains(tagTypes, "json") {
						tagTypes = append(tagTypes, "json")
					}
				}
				if strings.Contains(f.Tag, "xml:") {
					hasTag = true
					if !contains(tagTypes, "xml") {
						tagTypes = append(tagTypes, "xml")
					}
				}
				if strings.Contains(f.Tag, "gorm:") || strings.Contains(f.Tag, "db:") ||
					strings.Contains(f.Tag, "bson:") || strings.Contains(f.Tag, "yaml:") {
					hasTag = true
					if !contains(tagTypes, "db") {
						tagTypes = append(tagTypes, "db")
					}
				}
			}
			if hasTag {
				id := "model:" + pkg.Name + "." + typ.Name
				cat := strings.Join(tagTypes, ",")
				g.AddNode(id, models.CapDataModel,
					typ.Name, pkg.Name,
					"Data model: "+typ.Name+" ("+cat+" tags)",
					typ.Location, map[string]string{"tags": cat})
			}
		}
	}
}

func extractInterfacesAndImplementations(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, iface := range pkg.Interfaces {
			ifaceID := "iface:" + pkg.Name + "." + iface.Name
			g.AddNode(ifaceID, models.CapInterface,
				iface.Name, pkg.Name,
				"Interface: "+iface.Name+" ("+itoa(len(iface.Methods))+" methods)",
				iface.Location, map[string]string{"method_count": itoa(len(iface.Methods))})

			// link interface to its package
			pkgID := "pkg:" + pkg.Name
			g.AddEdge(ifaceID, pkgID, "defined_in", "defined in package "+pkg.Name)

			// link implementations
			for _, impl := range iface.Implementors {
				implID := "impl:" + impl
				g.AddNode(implID, models.CapImplementation,
					impl, pkg.Name,
					"Implements "+iface.Name+": "+impl, iface.Location, nil)
				g.AddEdge(implID, ifaceID, "implements", "implements "+iface.Name)
			}
		}
	}
}

func extractPackageNodes(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		id := "pkg:" + pkg.Name
		g.AddNode(id, models.CapPackage,
			pkg.Name+" ("+pkg.Path+")", pkg.Name,
			"Package: "+pkg.Path+" ("+itoa(len(pkg.Types))+" types, "+itoa(len(pkg.Functions))+" funcs)",
			pkg.Location, map[string]string{
				"types":     itoa(len(pkg.Types)),
				"functions": itoa(len(pkg.Functions)),
				"imports":   itoa(len(pkg.Imports)),
			})
	}
}

func extractEntrypoints(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			if fn.Name == "main" && fn.Receiver == "" {
				id := "entrypoint:" + pkg.Name + ".main"
				g.AddNode(id, models.CapEntrypoint,
					"main() in "+pkg.Name, pkg.Name,
					"Application entry point: main()", fn.Location, nil)
			}
		}
	}
}

func extractDatabases(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, imp := range pkg.Imports {
			if isDatabaseImport(imp.Path) {
				dbName := detectDatabaseName(imp.Path)
				id := "db:" + dbName
				g.AddNode(id, models.CapDatabase,
					dbName, pkg.Name,
					"Database: "+dbName+" (imported by "+pkg.Name+")",
					imp.Location, map[string]string{"driver": imp.Path})

				pkgID := "pkg:" + pkg.Name
				g.AddEdge(pkgID, id, "uses", "uses "+dbName)
			}
		}
	}
}

func linkCallSites(g *models.CapabilityGraph, cb *models.Codebase) {
	for _, pkg := range cb.Packages {
		for _, fn := range pkg.Functions {
			callerID := "fn:" + pkg.Name + "." + fn.Name
			if fn.Receiver != "" {
				callerID = "fn:" + pkg.Name + "." + fn.Receiver + "." + fn.Name
			}

			for _, cs := range fn.Calls {
				calleePkg := ""
				calleeName := cs.Callee
				if strings.Contains(cs.CallExpr, ".") {
					parts := strings.SplitN(cs.CallExpr, ".", 2)
					calleePkg = parts[0]
					calleeName = parts[1]
				}

				calleeID := "fn:" + calleePkg + "." + calleeName
				g.AddEdge(callerID, calleeID, "calls", cs.CallExpr)
			}
		}
	}
}

func isHandlerSignature(fn *models.Function) bool {
	if fn.Receiver != "" {
		return false
	}
	hasResponseWriter := false
	hasRequest := false
	for _, p := range fn.Params {
		if strings.Contains(p, "ResponseWriter") || strings.Contains(p, "http.ResponseWriter") {
			hasResponseWriter = true
		}
		if strings.Contains(p, "*http.Request") || strings.Contains(p, "http.Request") {
			hasRequest = true
		}
	}
	return hasResponseWriter && hasRequest
}

func isDatabaseImport(path string) bool {
	dbImports := []string{
		"database/sql", "github.com/lib/pq", "github.com/go-sql-driver/mysql",
		"github.com/mattn/go-sqlite3", "github.com/jackc/pgx", "github.com/jackc/pgconn",
		"go.mongodb.org/mongo-driver", "github.com/go-redis/redis", "github.com/gomodule/redigo",
		"github.com/elastic/go-elasticsearch", "github.com/couchbase/gocb",
		"github.com/aws/aws-sdk-go/service/dynamodb", "github.com/gocql/gocql",
		"gorm.io/gorm", "github.com/jmoiron/sqlx", "github.com/upper/db",
		"github.com/dgraph-io/badger", "github.com/boltdb/bolt", "go.etcd.io/bbolt",
		"github.com/syndtr/goleveldb",
	}
	for _, db := range dbImports {
		if path == db || strings.HasPrefix(path, db+"/") {
			return true
		}
	}
	return false
}

func detectDatabaseName(path string) string {
	switch {
	case strings.Contains(path, "postgres") || strings.Contains(path, "pq") ||
		strings.Contains(path, "pgx"):
		return "postgresql"
	case strings.Contains(path, "mysql"):
		return "mysql"
	case strings.Contains(path, "sqlite"):
		return "sqlite"
	case strings.Contains(path, "mongo"):
		return "mongodb"
	case strings.Contains(path, "redis"):
		return "redis"
	case strings.Contains(path, "elastic"):
		return "elasticsearch"
	case strings.Contains(path, "dynamo"):
		return "dynamodb"
	case strings.Contains(path, "cassandra") || strings.Contains(path, "gocql"):
		return "cassandra"
	case strings.Contains(path, "badger") || strings.Contains(path, "bbolt") ||
		strings.Contains(path, "bolt") || strings.Contains(path, "leveldb"):
		return "embedded_db"
	case strings.Contains(path, "database/sql") || strings.Contains(path, "sqlx"):
		return "sql_database"
	case strings.Contains(path, "gorm"):
		return "gorm"
	default:
		parts := strings.Split(path, "/")
		return parts[len(parts)-1]
	}
}

// parseRoutePattern extracts HTTP method and path from a Go 1.22+ pattern
// like "GET /api/users" or "POST /api/users/:id"
func parseRoutePattern(pattern string) (method, path string) {
	pattern = strings.Trim(pattern, "\"")
	parts := strings.SplitN(pattern, " ", 2)
	if len(parts) == 2 {
		upper := strings.ToUpper(parts[0])
		switch upper {
		case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS":
			return upper, parts[1]
		}
	}
	return "ANY", pattern
}

func findTypeLocation(pkg *models.Package, typeName string) models.Location {
	for _, typ := range pkg.Types {
		if typ.Name == typeName {
			return typ.Location
		}
	}
	return pkg.Location
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
