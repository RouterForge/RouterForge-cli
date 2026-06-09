# RouterForge

**AI Software Company Operating System**

RouterForge is a Go CLI tool that orchestrates multi-agent AI teams to build software projects. It implements an 8-phase operating flow with 4 core powers, turning ideas into production-ready code through autonomous agent collaboration.

[![Go Version](https://img.shields.io/badge/go-1.24.4-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Overview

```
┌─────────────────────────────────────────────┐
│              RouterForge                     │
│     AI Software Company Operating System     │
├─────────────────────────────────────────────┤
│  Core Powers                                 │
│  • Head Manager    — Orchestrates agents     │
│  • Architecture    — Generates architecture   │
│  • Dynamic Teams   — Creates agent teams     │
│  • RI Engine       — Repository Intelligence │
└─────────────────────────────────────────────┘
```

## Quick Start

```bash
# Build from source
make build

# Or use the prebuilt binary
./routerforge

# Initialize a project
./routerforge init

# Build with the quick profile (frontend-only)
./routerforge build --profile quick

# Full pipeline
./routerforge build --profile full
```

## Installation

```bash
git clone https://github.com/RouterForge/RouterForge-cli.git
cd RouterForge-cli
make build
```

## Commands

| Command | Description |
|---------|-------------|
| `init` | Initialize a new RouterForge project |
| `plan` | Interactive planning phase |
| `build` | Execute the build pipeline |
| `analyze` | Repository Intelligence Engine |
| `agent` | Spawn dynamic sub-agents |
| `serve` | Serve generated artifacts |
| `run` | Run a shell command |

### Build Profiles

```bash
# Quick: frontend only
routerforge build --profile quick

# Full: backend + frontend + security + QA
routerforge build --profile full
```

### Repository Intelligence

```bash
# Clone a repo for analysis
routerforge analyze clone https://github.com/user/repo.git

# Detect language and patterns
routerforge analyze detect /path/to/repo

# Full AST analysis (Go) — packages, types, interfaces, dependency graph
routerforge analyze ast /path/to/go/repo

# Detect with AST-powered capability extraction
routerforge analyze detect --ast /path/to/go/repo

# Function-level call graph (Go) — tracks pkg.func → pkg.func calls
routerforge analyze callgraph /path/to/go/repo

# Architecture fingerprinting (Go) — detects layered / hexagonal / MVC / DDD
routerforge analyze arch /path/to/go/repo

# Build capability matrix
routerforge analyze matrix

# Get integration recommendations
routerforge analyze recommend api_server auth_system
```

### Dynamic Agent Spawning

```bash
# Spawn a sub-agent on demand (LLM creates the agent, runs the task, returns result)
routerforge agent spawn "code_reviewer" "Review the auth package for security issues"
```

### Artifacts & Tracing

```bash
# Build generates traceable artifacts
routerforge build --profile quick
# -> .routerforge/artifacts/plan.json
# -> .routerforge/artifacts/trace.jsonl
# -> .routerforge/artifacts/summary.json

# Serve the artifact dashboard
routerforge serve
```

### Terminal UI

```bash
routerforge build --profile quick --tui
```

## Architecture

```
                   ┌─────────────┐
                   │ Head Manager │
                   └──────┬──────┘
                          │
           ┌──────────────┼──────────────┐
           │              │              │
    ┌──────▼─────┐ ┌─────▼──────┐ ┌─────▼──────┐
    │ Team Mgr   │ │ Team Mgr   │ │ Team Mgr   │
    │ (Frontend) │ │ (Backend)  │ │ (Security) │
    └──────┬─────┘ └─────┬──────┘ └─────┬──────┘
           │             │              │
    ┌──────▼─────┐ ┌─────▼──────┐ ┌─────▼──────┐
    │ MicroAgent │ │ MicroAgent │ │ MicroAgent │
    │ Component  │ │ API        │ │ Security   │
    │ Builder    │ │ Designer   │ │ Reviewer   │
    └────────────┘ └────────────┘ └────────────┘
```

**Operating Flow**: `Idle → Understand → Design (LLM-generated) → Execute → Review`

## Features

- **Dynamic Agent Creation**: Agents and teams are synthesized by LLM from project requirements, not selected from a static list. Each agent gets a unique LLM-generated system prompt based on its role, description, tools, and tasks — no static templates.
- **Multi-Agent Orchestration**: Head Manager delegates to Team Managers who coordinate Micro Agents. Agents can spawn sub-agents dynamically.
- **Provider Gateway**: Pluggable LLM provider support (OpenCode, OpenAI-compatible)
- **Tool Registry**: Extensible tool system (shell, file ops, search, web fetch) with per-agent permission sandboxing (allowed dirs, allowed/blocked commands, network toggle, file size limits)
- **Event Bus**: Pub/sub lifecycle events for real-time monitoring
- **Memory System**: In-memory store with context compression, checkpoint/restore, and scope-based access policies (read/write/admin levels with wildcard matching)
- **Token Budget Tracking**: Per-agent, per-phase, and global token limits with automatic 90% usage warnings and hard caps
- **Terminal UI**: Bubbletea-powered live pipeline visualization (`build --tui`)
- **Repository Intelligence**: AST-based analysis (packages, functions, types, interfaces, dependency graphs), function-level call graphs, and architecture fingerprinting (layered, hexagonal, MVC, DDD)
- **Pipeline Profiles**: Configurable build profiles (quick, full)
- **Traceable Artifacts**: Every build generates plan.json, trace.jsonl, and summary.json for full auditability
- **Browser Session Management**: Pooled headless Chromium sessions with acquire/release/eviction lifecycle
- **Multi-Terminal Management**: Multiple concurrent shell sessions with command history

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Format
make fmt
```

All packages (13) compile clean, `go vet` passes, and 28+ tests pass across 7 test packages.

## Tech Stack

- **Language**: Go 1.24.4
- **CLI Framework**: Cobra + PTerm
- **TUI**: Bubbletea + Lipgloss
- **Database**: BoltDB
- **LLM Integration**: REST API (OpenCode / OpenAI-compatible)
- **Browser**: go-rod (headless Chromium)

## License

MIT
