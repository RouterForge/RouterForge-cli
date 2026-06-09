# Changelog

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
