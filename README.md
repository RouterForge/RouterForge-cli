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
| `agent` | Agent management |
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

# Build capability matrix
routerforge analyze matrix

# Get integration recommendations
routerforge analyze recommend api_server auth_system
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

**Operating Flow**: `Idle → Understand → Design → Execute → Review`

## Features

- **Multi-Agent Orchestration**: Head Manager delegates to Team Managers who coordinate Micro Agents
- **Provider Gateway**: Pluggable LLM provider support (OpenCode, OpenAI-compatible)
- **Tool Registry**: Extensible tool system (shell, file ops, search, web fetch)
- **Event Bus**: Pub/sub lifecycle events for real-time monitoring
- **Memory System**: In-memory store with context compression and checkpoint/restore
- **Terminal UI**: Bubbletea-powered live pipeline visualization (`build --tui`)
- **Repository Intelligence**: Clone, analyze, detect patterns, build feature matrices
- **Pipeline Profiles**: Configurable build profiles (quick, full)

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
