# RouterForge

**AI Software Company Operating System**

RouterForge is a Go CLI tool that orchestrates multi-agent AI teams to build software projects. It implements a 5-phase operating flow (Idle → Understand → Design → Execute → Review) within a four-phase development lifecycle (Demo → Prototype → MVP → Production), turning ideas into production-ready code through autonomous agent collaboration.

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
| `analyze` | Repository Intelligence Engine + Capability Fusion |
| `agent` | Spawn dynamic sub-agents |
| `serve` | Serve generated artifacts (with health endpoints) |
| `run` | Run a shell command |
| `lifecycle` | Manage development lifecycle phases |
| `gate` | Review and approve governance gates |
| `deploy` | Deployment readiness checks, build, and health |

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

### Development Lifecycle

```bash
# View current lifecycle phase and review gates
routerforge lifecycle status

# Advance to the next lifecycle phase
routerforge lifecycle advance
```

### Governance Gates

```bash
# List all gates and their approval status
routerforge gate list

# Approve a specific gate
routerforge gate approve architecture_review
```

### Deployment

```bash
# Run deployment readiness checks
routerforge deploy check

# Build for production
routerforge deploy build

# Health check
routerforge deploy health
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

# Health check endpoints
curl http://localhost:8080/health
curl http://localhost:8080/health/ready
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
- **Capability Fusion Engine**: Study local and remote repositories to discover capabilities and build integration graphs
- **Cost Tracking**: Per-model, per-agent, per-phase cost tracking with budget limits
- **Structured Agent Communication**: Decision, review, report, task, action item, and escalation message types with routing
- **Four-Phase Development Lifecycle**: Demo → Prototype → MVP → Production with gated transitions
- **Governance Layer**: Architecture reviews, security reviews, testing requirements, documentation, and phase approvals
- **Deployment & Observability**: Health check endpoints, deployment readiness checks, and production builds

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

All packages compile clean, `go vet` passes, and 39+ tests pass across 11 test packages.

## Tech Stack

- **Language**: Go 1.24.4
- **CLI Framework**: Cobra + PTerm
- **TUI**: Bubbletea + Lipgloss
- **Database**: BoltDB
- **LLM Integration**: REST API (OpenCode / OpenAI-compatible)
- **Browser**: go-rod (headless Chromium)

## License

MIT
