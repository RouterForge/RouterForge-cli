# Capability Graph Benchmark Suite

## Objective

Prove that RouterForge's Semantic Model derives trustworthy understanding of real Go codebases without reading README files. This is the gate between "interesting prototype" and "production-ready intelligence layer."

## Methodology

For each repository in the benchmark set:

1. Clone the repo
2. Manually create `ground_truth.json` — the expected capabilities (what a human expert finds by reading the code)
3. Run `routerforge explain <path>` 
4. Compare detected vs expected
5. Score each capability category

## Ground Truth Schema

```jsonc
{
  "repository": "owner/repo",
  "commit": "sha",
  "detected_at": "2026-06-09",
  "detection_boundary": {
    // Capabilities the system CAN detect given current language support.
    // This is not a limitation of the system — it documents what's in scope.
    "routes": true,          // HTTP route registrations (HandleFunc, GET, POST, etc.)
    "handlers": true,        // ServeHTTP methods + (w, r) function signatures
    "middleware": true,      // Functions with "Middleware" in name + Use()/With() args
    "services": true,        // Packages with service/ path segment + CRUD interface
    "repositories": true,    // Types with 2+ CRUD methods in repo/store/db packages
    "data_models": true,     // Structs with JSON/XML/BSON/GORM/YAML tags
    "interfaces": true,      // Named interface types
    "implementations": true, // Types matching interface method sets
    "entrypoints": true,     // main() functions
    "databases": true,       // Known database driver imports
    "architecture": true,    // Import-direction-based architecture classification
    "call_graph": true,      // Function-level call edges
    "dep_graph": true        // Package-level dependency edges
  },
  "detection_exclusions": [
    // Capabilities the system does NOT currently detect
    "middleware_chain_order",    // Ordering of middleware application
    "dependency_injection",      // DI container wiring
    "schema_definitions",        // Database schema from migrations
    "query_patterns",            // SQL queries embedded in code
    "event_handlers",            // Pub/sub handler registrations
    "config_sources"             // Config file loading (env, yaml, toml)
  ],
  "capabilities": {
    "routes": [
      {"method": "GET", "path": "/api/users"}
    ],
    "handlers": [
      {"name": "ListUsers", "package": "handler", "type": "function"}
    ],
    "middleware": [
      {"name": "AuthMiddleware", "package": "middleware"}
    ],
    "services": [
      {"name": "UserService", "package": "service", "methods": 5}
    ],
    "repositories": [
      {"name": "UserRepository", "package": "repo", "crud_methods": 4}
    ],
    "data_models": [
      {"name": "User", "package": "model", "fields": 6, "tags": ["json"]}
    ],
    "interfaces": [
      {"name": "UserStore", "package": "port", "methods": 4}
    ],
    "implementations": [
      {"interface": "UserStore", "implementation": "PostgresUserStore", "package": "infra"}
    ],
    "entrypoints": [
      {"package": "main", "file": "cmd/server/main.go"}
    ],
    "databases": [
      {"driver": "postgresql", "package": "repo"}
    ],
    "architecture": {
      "type": "layered",
      "confidence": 0.85,
      "layers": ["handler", "service", "repository", "model"],
      "evidence_count": 2
    }
  }
}
```

## Detection Boundaries

Not every Go codebase registers routes via `HandleFunc` or uses struct tags for data models. The following boundaries document what the current Go parser can and cannot detect:

| Capability | Detectable Patterns | Non-Detectable Patterns |
|---|---|---|
| Routes | `Handle`, `HandleFunc`, `GET`, `POST`, `PUT`, `DELETE`, `PATCH` calls with string literal paths | Routes registered via config files, route annotations, external router config |
| Handlers | `ServeHTTP` methods, functions with `(http.ResponseWriter, *http.Request)` params | Handlers injected via DI framework, handlers from third-party libraries |
| Middleware | Functions containing "Middleware" in name, args to `Use()`/`With()` calls | Middleware composed via chaining functions, middleware from middleware-specific frameworks |
| Services | Packages with `service`/`services` path, interfaces with CRUD-like methods | Services defined purely as structs without service-path or service-named interfaces |
| Repositories | Types with 2+ CRUD methods in `repo`/`store`/`db`/`database` packages | Repos using code generation (sqlc, ent), repos without CRUD-named methods |
| Data Models | Structs with `json`, `xml`, `bson`, `gorm`, `yaml`, `db` tags | Pure Go structs without tags (detected but not classified as data models) |
| Architecture | Import direction between handler/service/repo/domain layers | Architecture patterns expressed via package comments or build tags |

## Scoring Formula

### Per-Category Score

```
category_score = matched / max(expected_matched, detected_total)
```

Where:
- `matched` = items correctly detected (intersection of expected and detected)
- `expected_matched` = items in ground truth that are within `detection_boundary`
- `detected_total` = total items detected (including false positives)

### Category Weights

| Category | Weight | Rationale |
|---|---|---|
| routes | 0.15 | Core surface — users care about endpoints |
| handlers | 0.10 | Maps to route → handler linking |
| middleware | 0.05 | Nice to have, detection boundary is narrow |
| services | 0.10 | Core architecture layer |
| repositories | 0.10 | Core data access layer |
| data_models | 0.10 | Data schema understanding |
| interfaces | 0.05 | Contract detection |
| implementations | 0.05 | Implementation binding |
| entrypoints | 0.05 | Startup flow |
| databases | 0.05 | Storage understanding |
| architecture | 0.15 | High-level structure — users ask about this |
| call_graph | 0.025 | Useful but noisy |
| dep_graph | 0.025 | Useful but noisy |
| **total** | **1.00** | |

### Overall Score

```
overall = Σ(category_weight × category_score) × recall_penalty
```

Where `recall_penalty = detected_categories / expected_categories` (penalizes missing entire categories).

### Thresholds

| Score | Label | Meaning |
|---|---|---|
| ≥ 0.90 | Trustworthy | System can reliably explain this codebase |
| ≥ 0.75 | Informative | Most capabilities detected, some gaps |
| ≥ 0.50 | Partial | Major capability categories missing |
| < 0.50 | Exploratory | System not yet useful for this codebase |

## Benchmark Repositories (Target Set)

### Phase 1 — First 5 Repos (Go monorepos, diverse architectures)

1. **RouterForge** (self) — layered architecture, http.ServeMux, service/repo pattern
2. **OpenCode** — CLI tool, no HTTP server, Cobra commands
3. **chi** (go-chi/chi) — HTTP router library, middleware-heavy
4. **Grafana** (grafana/grafana) — large monorepo, multiple service types, complex architecture
5. **Caddy** (caddyserver/caddy) — HTTP server with module system, non-standard architecture

### Phase 2 — Next 5 Repos (Proving generalization)

6. **Kubernetes** (kubernetes/kubernetes) — massive monorepo, controller pattern
7. **Temporal** (temporalio/temporal) — distributed systems, workflow engine
8. **Hugo** (gohugoio/hugo) — static site generator, CLI + API
9. **Prometheus** (prometheus/prometheus) — monitoring system, service discovery
10. **Cobra** (spf13/cobra) — CLI framework, command registration pattern

### Phase 3 — 10 More (Stress test)

11–20: Mix of popular Go projects across domains

## Running the Benchmark

```bash
# Generate explain report for a repo
routerforge explain /path/to/repo --json > results/repo_name.json

# Compare against ground truth
routerforge benchmark compare results/repo_name.json ground_truth/repo_name.json

# Generate benchmark summary
routerforge benchmark summary results/
```

## Publishing Results

For each Phase, publish:
- `results/PHASE_N.md` — Summary table of per-repo scores
- `results/PHASE_N.json` — Machine-readable scores
- Key findings and detection gaps discovered
