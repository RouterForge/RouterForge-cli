# Changelog

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
