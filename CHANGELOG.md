# Changelog

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
