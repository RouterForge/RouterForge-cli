# Changelog

## [1.8.0] — 2026-06-10

### Added (Repair Engine — Closed Execution Loop)

#### RepairUntilValid (internal/orchestrator/repair.go)
- Post-generation validation loop that detects build/runtime failures and calls the configured model to repair
- `ValidateProject()` detects project type (go, node, static-web, empty) and runs project-appropriate validation:
  - Go: checks `go.mod` exists, runs `go test ./...`
  - Static web: validates `index.html` asset references, runs `node --check` on JS files
  - Node: runs `npm run build` / `npm test` when available
  - Empty: reports failure
- `repairProject()` constructs a repair prompt with validation output, project files listing with content excerpts, and repair instructions; re-applies `FILE:` sections from the LLM response
- Configurable repair retries via `--repair-retries` flag (default 2)
- Validation artifacts saved to `.routerforge/artifacts/validation-N.json`

#### Shared FILE: Section Parser (internal/orchestrator/file_sections.go)
- `writeFileSections()` extracted from `TaskRunner.Execute` and shared between generation and repair
- `safeProjectPath()` directory traversal protection
- Improved content trimming: strips `---` separators more reliably

#### Tests (internal/orchestrator/repair_test.go)
- `TestWriteFileSectionsStripsSeparator`: verifies `---` does not leak into generated source
- `TestWriteFileSectionsRejectsTraversal`: verifies paths like `../escape.txt` are rejected
- `TestValidateProjectStaticWebMissingAsset`: verifies missing JS/CSS in `index.html` is detected
- `TestValidateProjectGoRequiresModule`: verifies Go projects without `go.mod` fail validation

### Changed
- Build pipeline: `Execute → Validate → Repair → Revalidate → Review` (RepairUntilValid injected between Execute and Review)
- `Review()` no longer fails on task count or task failure rate. Task reliability is reported as observability only. Software validation determines build success.
- `RunFullPipeline()`: includes RepairUntilValid(2) step
- Review gates relaxed — task status tracked but validation is the success gate

### Fixed
- Generated source files no longer contain literal `---` separator lines
- Path traversal rejection in generated file paths

## [1.7.1] — 2026-06-10

### Fixed (Execution Reliability — Build produces real source files)

#### Execution creates source files (internal/orchestrator/micro_agent.go)
- `TaskRunner.Execute` now parses `FILE: <path>` prefix from LLM output and writes source files to the project directory at the correct relative path
- Added `ProjectDir` field to `Context` — passed through from `HeadManager` to every agent
- LLM system prompt instructs model to return `FILE: path/to/file.ext` format with file content
- Fallback `inferFilename()` heuristic when LLM returns content without `FILE:` prefix

#### Plan/build disconnect (cmd/build.go, internal/orchestrator/head_manager.go)
- Extracted `applyPlan()` from `Design()` so it can be called independently
- Added `RestorePlan()` to rebuild teams/agents from a saved `plan.json` without re-invoking the LLM
- `build.go` loads `.routerforge/artifacts/plan.json` first; only calls `Design()` if no saved plan exists
- Added `projectDir` field and `SetProjectDir()` to `HeadManager`

#### Review gates detect failures (internal/orchestrator/head_manager.go)
- `Review()` now counts total/failed tasks across all agents
- Fails if 0 tasks were executed (build produced no output)
- Fails if >50% of tasks failed (exceeds failure threshold)

#### Models
- Added `pkg/models` import to `cmd/build.go` for `models.Plan` deserialization

### Added (Deepening Go Analysis — Proven Understanding)

#### Request Flow Tracing (internal/repo/flow_extractor.go)
- **ExtractRequestFlows()**: Traces every detected HTTP route through the full system — route → handler → service → repository → data model — derived entirely from AST analysis and capability graph traversal
- Walks call graph edges from handler function nodes, classifying callees as service/repo/model by package path
- Maps databases to flows via package→database import detection
- Attaches entrypoint information to every flow

#### Layer Violation Detection (internal/repo/violations.go)
- **DetectLayerViolations()**: Five architectural constraint checks:
  - Handler→Repository bypass (skipping service layer) — severity: high
  - Service→Infrastructure/Database import (should use repository) — severity: medium
  - Domain→Infrastructure import (DDD violation) — severity: high
  - Circular dependency detection via DFS cycle finding
  - Handler→Model direct import (should use DTOs) — severity: low
- Fixed `classifySingle()` in `arch_v2.go` to correctly handle path segment aliases (e.g., "repo" → "repository", "store", "db")

#### Ownership Analysis (internal/repo/ownership.go)
- **AnalyzeOwnership()**: Maps every capability graph node back to its owning package
- Per-package counts: routes, handlers, middleware, services, repositories, data models, interfaces, entrypoints
- Total capability score per package

#### Semantic Model Enrichment (pkg/models/codebase.go)
- **RequestFlow/FlowStep**: Complete trace of an HTTP request through the system
- **LayerViolation**: Architectural constraint violation with source, target, description, severity, location
- **OwnershipInfo**: Per-package capability ownership with typed counts
- All three types added to the `Codebase` struct — serialized in JSON output

#### Comprehensive Test Fixture (internal/repo/deep_understanding_test.go)
- **TestDeepUnderstanding_RealisticApp**: 11 subtests verifying architecture (layered, 100% confidence), 5 routes (GET/POST/DELETE), 5 handlers, 2 middleware, 1 service, repositories, 1 data model (json tags), 5 request flows (4 with handler resolution), 0 layer violations (clean architecture), 6 owned packages with 17 total capability assignments, 24 nodes & 78 edges in capability graph
- **TestDeepUnderstanding_WithViolations**: Deliberately creates handler→repo bypass — correctly detected as high-severity violation
- Both tests prove the system derives trustworthy understanding without reading a README

#### Fixed
- `classifySingle()` in arch_v2.go: Now correctly maps path aliases (repo→repository, store, db) so layer import matrices work for all naming conventions
- Violation test fixture: Fixed incorrect module name in import path

### Full Test Suite
- 55+ tests across 12 test packages — `go test ./...` and `go vet ./...` both pass clean

### Added (Capability Graph — Connected Intelligence Layer)

#### Capability Graph Model (pkg/models/codebase.go)
- **CapabilityGraph**: Graph with typed nodes and labeled edges — replaces flat capability lists
- **CapabilityNode**: Typed nodes (route, handler, middleware, service, repository, data_model, interface, implementation, package, entrypoint, database) with traceable source locations and key-value properties
- **CapabilityEdge**: Labeled edges (routes_to, calls, implements, depends_on, defined_in, method_of, registers, wraps, contract_for, uses) forming a connected graph
- Graph query methods: `NodesByType`, `EdgesFrom`, `EdgesTo`, `NodeByID`

#### Deep Code Understanding (internal/repo/capability_builder.go)
- **Route detection**: Extracts HTTP routes from call sites matching `Handle`, `HandleFunc`, `GET`, `POST`, `PUT`, `DELETE`, `PATCH` patterns. Parses Go 1.22+ method-qualified patterns (`"GET /api/users"` → method=GET, path=/api/users)
- **Handler extraction**: Detects `ServeHTTP` methods and functions with `(http.ResponseWriter, *http.Request)` signature
- **Middleware detection**: Identifies middleware by name patterns and `Use()`/`With()` call arguments
- **Service boundary detection**: Packages with "service"/"services" in path or interface contracts
- **Repository pattern detection**: Types with 2+ CRUD methods (Create, Find, Get, Update, Delete, Save, etc.) in packages with store/repo/db paths
- **Data model extraction**: Structs with JSON/XML/BSON/GORM/YAML tags
- **Interface → implementation mapping**: Linked via "implements" edges
- **Database detection**: 20+ known database drivers recognized from imports (postgres, mysql, sqlite, mongo, redis, etc.)

#### Enhanced Call Site Extraction (internal/repo/unified_analyzer.go)
- **Fixed**: Caller attribution now correctly uses function name (was using package name) — call graph grew from 538→1013 nodes
- **Added**: String literal argument capture in call sites — enables route path extraction
- **Added**: `argToString()` helper for AST expression → string conversion
- **Added**: Current function tracking during AST inspection for proper call site ownership

#### What the Capability Graph Answers
- "What does this system do?" → Routes, handlers, entrypoints
- "How does it do it?" → Middleware chains, service calls, repository access
- "What components are responsible?" → Package ownership, interface contracts, implementation bindings

#### Tests
- 4 new capability graph tests: route/handler/middleware/repo detection, node counts, data model tags, layer edges
- Real-world validation: RouterForge capability graph = 130 nodes, 4,243 edges

## [1.5.0] — 2026-06-09

### Added (Repository Intelligence V2 — Semantic Code Model Foundation)

#### Semantic Code Model (pkg/models/codebase.go)
- **Codebase**: Language-agnostic top-level semantic model — single source of truth for all future systems
- **Package/Type/Function/Interface/Field/Import/CallSite**: Connected object graph with bidirectional references, JSON-serializable, every entity carries `Location{File, Line}` for full source traceability
- **DepGraph/CallGraph/ArchProfile/Capability**: First-class model types for dependency analysis, function-level call tracking, architecture classification, and capability extraction

#### Unified Go Parser (internal/repo/unified_analyzer.go)
- **AnalyzeToCodebase(root)**: Single-pass analysis — one directory walk, each file parsed once, produces complete Codebase
- Call sites extracted from both `pkg.Func()` selector calls and direct `func()` calls
- Interface implementor resolution via method name matching
- Proper type expression formatting (handles `*ast.StarExpr`, `SelectorExpr`, `ArrayType`, `MapType`, `FuncType`, `ChanType`, `Ellipsis`, `InterfaceType`)
- Graceful test package skipping, vendor/hidden directory filtering

#### Architecture Fingerprinting v2 (internal/repo/arch_v2.go)
- **code-relationship-based architecture detection**: Analyzes import direction between layers, layer isolation, dependency inversion, interface implementations
- Evidence-based scoring: layered (handler→service→repo), hexagonal (port interfaces + adapter imports), DDD (domain isolation + infrastructure depends on domain), MVC
- Layer violation detection (e.g., handler→repo bypassing service)
- Replaces directory-name heuristics with actual code structure analysis

#### Capability Extraction with Traceability
- 20+ capability detectors: http_server, database, serialization, message_queue, observability, authentication, configuration, cli_framework, graphql, has_tests, http_handler, runnable_service, cli_entrypoint, request_handler, middleware, database_migrations, json_models, testing_framework
- Every capability carries `[]Location` — traceable to specific file and line

#### Fusion Engine V2 Integration
- **DeepStudyCodebase(path)**: Single-call entry point returning the complete `*models.Codebase`
- Surface-level scan supplements AST-derived capabilities with basic source locations

#### Real-world Validation
- Analyzes RouterForge itself without reading the README: 14 packages, 25K+ LOC, 538 call-graph nodes, 4,415 edges, 50 dep-graph nodes, 9 capabilities with traceable source locations

#### Tests
- 9 new tests in internal/repo (analyzer, imports, capabilities, architecture, call graph, JSON, struct fields, interface implementors, real-world)
- 2 new tests in internal/fusion (DeepStudyCodebase, capabilities + architecture end-to-end)
- Full module test suite: 50+ tests, go vet clean, all packages compile

### Fixed
- **Deadlock in Scheduler.Schedule()**: Called `Start()` while holding internal mutex — deadlocked on recursive lock attempt. Fixed by releasing mutex before calling Start().
- **Receiver type formatting**: `fmt.Sprintf("%s", ast.Node)` on `*ast.StarExpr` produced Go struct representation (`{%!s(token.Pos=177) UserHandler}`). Fixed by using `typeExprToString()` helper for clean `*TypeName` output.

## [1.4.0] — 2026-06-09

### Added (Roadmap Phase A: Repository Intelligence v2)
- **Deep Repository Study** (`analyze fusion deep`): Full analysis pipeline — AST analysis, call graph, import graph, architecture fingerprinting, capability graph, feature matrix — wired together from the repo package into the fusion engine
- **DeepStudyResult**: Combined result type with all analysis dimensions, JSON and Markdown output
- **Fusion engine rewrite**: StudyRepo and DeepStudy now use `repo.ASTAnalyzer`, `repo.PatternDetector`, `repo.SourceAnalyzer` instead of shallow file-scraping

### Added (Roadmap Phase B: Dynamic Team Synthesis)
- **Team Synthesis Engine** (`internal/orchestrator/synthesis.go`): Requirement-driven agent generation — project goal, tech stack, and features are parsed to auto-select domains and generate concrete, relevant tasks
- **Cascading design pipeline**: `Design()` now tries (1) LLM plan generation → (2) rule-based synthesis → (3) interactive fallback
- **Requirement-aware task derivation**: `deriveFrontendTasks` and `deriveBackendTasks` generate tasks matched to project type (website vs CLI vs API)
- **SynthesizedAgent**: Structured agent definition with domain, role, description, tools, and tasks derived from project requirements

### Added (Roadmap Phase C: Browser Runtime)
- **Visual Testing** (`ScreenshotCompare`): Compare screenshots against baseline for visual regression detection
- **Storage Access** (`GetLocalStorage`, `GetSessionStorage`): Read browser localStorage and sessionStorage via JavaScript evaluation
- **Auto-wait after navigation**: `Navigate()` now calls `WaitLoad()` automatically
- **Viewport control** (`SetViewport`): Set browser viewport dimensions
- **browser_action tool**: Now launches a real headless browser and dispatches navigate/click/type/screenshot/html/evaluate/cookies actions, instead of returning an error message

### Added (Roadmap Phase D: Architecture Generator)
- **Architecture Documentation Generator** (`analyze archgen`): Produces full Markdown architecture docs with Mermaid diagrams (dependency graph, call graph, architecture diagram) and service map table
- **Service Map Builder** (`analyze servicemap`): Infers services from AST packages, their exports, and dependency relationships; outputs Mermaid graph
- **ArchitectureDoc**: Complete documentation structure with all diagram types, architecture profile, and patterns
- **ServiceMap/Service types**: Package-level service discovery with type classification (cmd/internal/pkg)

### Added (Roadmap Phase E: Company Runtime)
- **Concurrent Agent Scheduler** (`internal/orchestrator/scheduler.go`): Pooled execution of agents with configurable concurrency limit, context-based cancellation, and per-process status tracking
- **Process lifecycle**: AgentProcess struct with running/completed/failed/cancelled states, timing, and error capture
- **Integration**: HeadManager now initializes and starts a Scheduler (max 3 concurrent) on construction

## [1.3.0] — 2026-06-09

### Added
- **Four-Phase Development Lifecycle** (`routerforge lifecycle`): Demo → Prototype → MVP → Production phase state machine with transition rules and per-phase deliverables
- **Governance Layer** (`routerforge gate`): Review gate manager with Architecture Review, Security Review, Testing Requirements, Documentation, and Phase Approval gates
- **Capability Fusion Engine** (`routerforge analyze study`): Study local and remote repositories, discover capabilities, build fusion graphs and feature matrices
- **Enhanced Browser Intelligence**: GetCookies/SetCookie/ClearCookies, ConsoleLogs capture, NewTab/SwitchTab/CloseTab multi-tab management, WaitForSelector
- **Cost Tracking**: Per-model, per-agent, per-phase cost tracking with aggregate totals and budget limits via CostTracker
- **Enhanced TUI Dashboard**: Lifecycle phase display, cost overview, token usage, and task board with real-time status
- **Structured Agent Communication Protocol**: Decision, review, report, task, action_item, escalation message types with CommunicationHub routing
- **Additional Tools**: web_search (DuckDuckGo), api_call (generic HTTP), browser_action (proxy to browser command) registered in tool registry
- **Deployment & Observability** (`routerforge deploy`): deploy check/build/health commands, health/ready HTTP endpoints, go vet/build/git checks
- **Updated free model list**: big-pickle, deepseek-v4-flash-free, mimo-v2.5-free, nemotron-3-super-free, nemotron-3-ultra-free (tested working)

### Changed
- `cmd/serve.go` added /health and /health/ready endpoints
- `internal/engine/browser_engine.go` and `browser_session.go` extended with cookie, console, tab, and wait methods
- `internal/engine/llm_client.go` wired cost callback into response parsing
- `internal/orchestrator/head_manager.go` integrated lifecycle, cost tracker, review gates, model list
- `internal/orchestrator/micro_agent.go` passes CostHandler through Context to LLMClient
- `internal/tool/tools.go` added web_search, api_call, browser_action registrations
- `internal/tui/model.go` added lifecycle, cost, token, and task board rendering

## [1.2.0] — 2026-06-09

### Added
- **LLM-Only Agent Spawning (Gap 1)**: Deleted `agent_registry.go` static templates. Every agent now receives a unique LLM-generated system prompt via `GenerateSystemPromptFromLLM()` — role, description, tools, tasks, and project goal are synthesized into a dynamic prompt. `BuildAgentFromPlan()` constructs full `models.Agent` objects from `PlanAgent`. `agent spawn <role> <task>` CLI command.
- **Function-Level Call Graph (Gap 2)**: `analyze callgraph <path>` walks Go source AST and builds a pkg.func → pkg.func call graph. Exported as `CallGraph` with Mermaid output.
- **Architecture Fingerprinting (Gap 2)**: `analyze arch <path>` detects architecture style (layered, hexagonal, MVC, DDD) from directory structure and package naming with confidence scoring. Exported as `ArchitectureProfile`.
- **Browser Session Manager (Gap 3)**: `internal/engine/browser_session.go` implements a pooled browser session manager with acquire/release/eviction, concurrent session limits, and idle timeout. Wraps go-rod `BrowserEngine`.
- **Multi-Terminal Manager (Gap 3)**: `internal/engine/terminal_manager.go` supports multiple concurrent shell sessions with command history, timeout, and session lifecycle management.
- **Dynamic Sub-Agent Spawner (Gap 3)**: `internal/engine/agent_spawner.go` creates sub-agents on demand via LLM with `Spawn`, `SpawnWithTools`, and `SpawnBatch`. Integrated into HeadManager for runtime delegation.
- **Token Budget Tracking (Gap 4)**: `internal/orchestrator/token_budget.go` provides per-agent, per-phase, and global token limits with automatic 90% usage warnings and hard caps. Includes `TokenTracker` for per-agent/per-phase token auditing.
- **Memory Access Policies (Gap 4)**: `internal/orchestrator/memory_policy.go` implements scope-based access rules with read/write/admin access levels, wildcard scope matching, and a `MemoryPolicyEnforcer` for runtime checks.
- **Tool Sandboxing (Gap 4)**: `internal/orchestrator/sandbox.go` provides per-agent allowed directories, allowed/blocked command lists, network toggle, file size limits, and path validation via `ToolSandbox`.

### Changed
- `head_manager.go` now initializes TokenBudget, TokenTracker, MemoryPolicy, MemoryPolicyEnforcer, ToolSandbox, and AgentSpawner on construction
- `head_manager.go.Design()` uses `BuildAgentFromPlan()` and `AdoptPreBuiltAgent()` to create dynamically-prompted agents, then registers their memory policies and sandbox permissions
- `cmd/agent.go` replaced static template listing with dynamic sub-agent spawning (`agent spawn`)
- `cmd/analyze.go` added `callgraph` and `arch` subcommands
- `team_manager.go` added `AdoptPreBuiltAgent()` method for consuming pre-constructed agents from the plan

### Removed
- `internal/agent/agent_registry.go` — static agent template map deleted entirely

## [1.1.0] — 2026-06-08

### Added
- **Dynamic Agent Creation**: HeadManager.GeneratePlan() sends project requirements to LLM and receives structured JSON plan with teams, agents, tasks. No more hardcoded agent lists.
- **AST-based Repository Intelligence**: `internal/repo/ast_analyzer.go` parses Go source into AST, extracts packages, functions (params/results/exported), types, interfaces, and builds Mermaid dependency graphs. `analyze ast <path>` and `analyze detect --ast <path>` commands.
- **Traceable Artifacts**: Every build now produces `plan.json` (LLM-generated plan), `trace.jsonl` (newline-delimited execution trace with timestamps), and `summary.json` (build summary).
- **serve.go Dashboard**: Rebuilt as a full artifact browser with embedded trace viewer showing the last 50 entries.
- **New commands**: `analyze ast`, `serve` (dashboard mode)

### Changed
- `build.go` removed all hardcoded agent creation (`api_designer`, `component_builder`, etc.) — agents are now synthesized by LLM during the Design phase
- `head_manager.go.Design()` now calls LLM for plan generation before falling back to interactive mode
- `head_manager.go` now writes trace entries for every phase transition, team execution, and task lifecycle event

## [1.0.0] — 2026-06-08

### Added
- 13 Go packages implementing the RouterForge operating system
- Head Manager with 5-phase state machine (Idle → Understand → Design → Execute → Review)
- Team Manager with Micro Agent task execution
- LLM Provider Gateway with OpenCode and OpenAI adapters
- Tool Registry with 7 built-in tools (shell, file, search, web, ask_user)
- EventBus with pub/sub, wildcard topics, lifecycle events
- Memory system with in-memory store, context compression, checkpoint/restore
- Repository Intelligence Engine (clone, detect, matrix, recommend)
- Bubbletea TUI for live pipeline visualization
- Pipeline profiles (quick, full) with config-driven team composition
- Encrypted secret store (AES-256-GCM)
- CI workflow (build, test, vet, fmt-check)
- 28+ tests across 7 test packages
