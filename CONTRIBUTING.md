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
cmd/              — CLI commands (cobra)
internal/
  agent/          — Agent registry, prompts, user proxy
  engine/         — LLM client, errors, file ops, search
  event/          — EventBus pub/sub
  llm/            — Provider gateway + adapters
  memory/         — In-memory store, compression
  orchestrator/   — Head Manager, state machine, micro agents, LLM plan gen
  repo/           — Repository Intelligence Engine (AST analysis, patterns, graphs)
  storage/        — Config, BoltDB store, encrypted secrets
  tool/           — Tool registry + implementations (shell, file, search, web)
  tui/            — Bubbletea terminal UI
pkg/models/       — Shared data types (project, agent, task, plan, trace)
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
