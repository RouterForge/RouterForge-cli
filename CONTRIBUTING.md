# Contributing

## Getting Started

```bash
git clone https://github.com/RouterForge/RouterForge-cli.git
cd RouterForge-cli
make build
make test
```

## Project Structure

```
cmd/              — CLI commands (cobra) — root command launches multi-agent OS
internal/
  agent/          — Agent prompts, user proxy, silent proxy for TUI mode
  engine/         — LLM client, errors, file ops, search, browser
  event/          — EventBus pub/sub with wildcard/prefix routing
  llm/            — Provider gateway + adapters
  memory/         — In-memory store, compression, scope-based policies
  orchestrator/   — Head Manager, state machine, micro agents, teams, repair
  repo/           — Repository Intelligence Engine (AST, patterns, graphs)
  storage/        — Config, BoltDB store, encrypted secrets
  tool/           — Tool registry + implementations (shell, file, search, web)
  tui/            — Multi-window OS interface (model.go + program.go)
pkg/models/       — Shared data types (project, agent, task, plan, trace, codebase)
```

## Standards

- **Go version**: 1.24.4
- **Formatting**: `go fmt` before commits
- **Linting**: `go vet ./...` must pass
- **Tests**: `go test ./...` must pass
- **No new direct dependencies** without justification

## PR Process

1. Fork the repo
2. Create a feature branch
3. Make changes with tests
4. Run `make test && make lint`
5. Open a PR
