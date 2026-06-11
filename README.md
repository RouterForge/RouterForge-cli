# RouterForge

**AI Multi-Agent Operating System**

RouterForge is a multi-agent operating system that orchestrates AI agent teams to build software projects. Just type `routerforge` and start chatting with your Head Manager — it iteratively understands your idea, spawns specialist teams, and builds your project in real time.

[![Go Version](https://img.shields.io/badge/go-1.24.4-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)

## Quick Start

```bash
# Build from source
make build

# Launch the multi-agent OS — chat with your Head Manager
./routerforge
```

That's it. Just `routerforge`. Describe your project idea, and watch as teams of AI agents form, plan, and build.

## What It Looks Like

```
┌─────────────────────────────────────────────────────────────┐
│ RouterForge 2.0    [Demo]    understand    ● online         │
├─────────────────────────────────────────────────────────────┤
│ [● Head Manager] [○ Backend Team] [○ Frontend Team]        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ● Head Manager                                             │
│  ─────────────────────────────────────────────────────      │
│  [14:23:01] ▶ RouterForge 2.0 — AI Multi-Agent OS          │
│  [14:23:01] 💬 Welcome! I'm your Head Manager.             │
│  [14:23:01] 💬 Tell me about the project you want to build.│
│  [14:23:15] 💬 You > Build a task management app            │
│  [14:23:15] ▶ Got it! Starting the multi-agent pipeline...  │
│  [14:23:20] ● Phase transition: design                     │
│  [14:23:25] ✓ Team frontend created and ready.             │
│  [14:23:25] ✓ Team backend created and ready.              │
│  [14:23:30] ✓ Agent online: frontend_component_builder     │
│  [14:23:30] ● Task started: Set up React project           │
│  [14:23:45] ✓ Task completed: Set up React project         │
│  [14:23:46] 📄 Generated: src/App.jsx                      │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ > _                                                  │  │
│  └──────────────────────────────────────────────────────┘  │
├─────────────────────────────────────────────────────────────┤
│ Phase: execute  │  Agents: 2/4  │  $0.0021  │  342 tok    │
├─────────────────────────────────────────────────────────────┤
│ Tab/←→:Switch  Enter:Chat  ↑↓:Scroll  0-9:Jump  h:Head   │
└─────────────────────────────────────────────────────────────┘
```

**Navigate**: `Tab/←→` switch windows, `Enter` chat, `↑↓` scroll, `0-9` jump, `h` head manager, `q` quit.

## Installation

```bash
git clone https://github.com/RouterForge/RouterForge-cli.git
cd RouterForge-cli
make build
```

## Commands

| Command | Description |
|---------|-------------|
| `routerforge` | Launch the multi-agent OS (chat with Head Manager) |
| `init` | Initialize a new RouterForge project |
| `plan` | Interactive planning phase |
| `build` | Execute the build pipeline |
| `build --tui` | RouterForge 2.0 multi-agent OS interface |
| `analyze` | Repository Intelligence Engine + Capability Fusion |
| `agent` | Spawn dynamic sub-agents |
| `serve` | Serve generated artifacts |
| `lifecycle` | Manage development lifecycle phases |
| `gate` | Review and approve governance gates |
| `deploy` | Deployment readiness checks |
| `inspect` | Inspect all build artifacts |

### Build Profiles

```bash
# Quick: frontend only
routerforge build --profile quick

# Full: backend + frontend + security + QA
routerforge build --profile full

# With TUI
routerforge build --profile quick --tui
```

### Terminal UI

```bash
# RouterForge 2.0 — just type routerforge
routerforge

# Or explicitly with build
routerforge build --tui
```

### Repository Intelligence V2

```bash
# Deep study — single-pass analysis returning full semantic model
routerforge analyze fusion deep /path/to/repo

# Clone a repo for analysis
routerforge analyze clone https://github.com/user/repo.git

# Detect language and patterns
routerforge analyze detect /path/to/repo

# Full AST analysis (Go) — packages, types, interfaces, dependency graph
routerforge analyze ast /path/to/go/repo

# Function-level call graph + import graph (Go)
routerforge analyze callgraph /path/to/go/repo

# Architecture fingerprinting (Go) — detects layered / hexagonal / MVC / DDD
routerforge analyze arch /path/to/go/repo

# Architecture documentation generator with Mermaid diagrams
routerforge analyze archgen /path/to/repo

# Service map generation
routerforge analyze servicemap /path/to/repo

# Build capability matrix
routerforge analyze matrix
```

### Dynamic Agent Spawning

```bash
routerforge agent spawn "code_reviewer" "Review the auth package for security issues"
```

### Development Lifecycle

```bash
routerforge lifecycle status
routerforge lifecycle advance
```

### Governance Gates

```bash
routerforge gate list
routerforge gate approve architecture_review
```

### Deployment

```bash
routerforge deploy check
routerforge deploy build
routerforge deploy health
```

### Artifacts & Tracing

```bash
routerforge build
# -> .routerforge/artifacts/plan.json
# -> .routerforge/artifacts/trace.jsonl
# -> .routerforge/artifacts/summary.json

# Serve the artifact dashboard
routerforge serve
```

## Architecture

```
                    ┌──────────────────────────┐
                    │     Semantic Code Model    │
                    │  (Codebase — single truth) │
                    └──────────┬───────────────┘
                               │ feeds
           ┌───────────────────┼───────────────────┐
           │                   │                   │
    ┌──────▼──────┐   ┌───────▼───────┐   ┌───────▼───────┐
    │ Repository   │   │ Architecture  │   │  Capability    │
    │ Intelligence │   │ Generator     │   │  Extraction    │
    │ (Analyze)    │   │ (Plan/Design) │   │  (Fusion)      │
    └──────┬───────┘   └───────┬───────┘   └───────┬───────┘
           │                   │                   │
           └───────────────────┼───────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │    Head Manager      │
                    │  (Orchestrator)      │
                    └──────────┬──────────┘
                               │
             ┌─────────────────┼─────────────────┐
             │                 │                  │
      ┌──────▼─────┐   ┌──────▼──────┐   ┌──────▼──────┐
      │ Team Mgr   │   │  Team Mgr   │   │  Team Mgr   │
      │ (Frontend) │   │  (Backend)  │   │  (Security) │
      └──────┬─────┘   └──────┬──────┘   └──────┬──────┘
             │                │                  │
      ┌──────▼─────┐   ┌──────▼──────┐   ┌──────▼──────┐
      │ MicroAgent │   │  MicroAgent │   │  MicroAgent │
      │ Component  │   │  API        │   │  Security   │
      │ Builder    │   │  Designer   │   │  Reviewer   │
      └────────────┘   └─────────────┘   └─────────────┘
```

**Pipeline**: `Idle → Understand → Design → Execute → Repair → Review`

## Features

- **Conversational Interface**: Just type `routerforge` and describe your idea to the Head Manager
- **Multi-Window OS**: Dark-themed tabbed interface — Head Manager, team panels, live agent windows
- **Filtered Intelligence**: Shows results and decisions, hides internal machinery (tool calls, search logs, file operations)
- **Dynamic Agent Creation**: Agents and teams are synthesized by LLM from project requirements, not selected from a static list
- **Multi-Agent Orchestration**: Head Manager delegates to Team Managers who coordinate Micro Agents
- **Provider Gateway**: Pluggable LLM provider support (OpenCode, OpenAI-compatible)
- **Tool Registry**: Extensible tool system (shell, file ops, search, web fetch) with per-agent permission sandboxing
- **Event Bus**: Pub/sub lifecycle events for real-time monitoring
- **Memory System**: In-memory store with context compression, checkpoint/restore, and scope-based access policies
- **Token Budget Tracking**: Per-agent, per-phase, and global token limits
- **Repair Engine**: Post-generation validation with closed repair loop
- **Repository Intelligence V2**: Unified Semantic Code Model with Go parsing, dependency graph, call graph
- **Capability Graph**: Connected graph of routes, handlers, middleware, services, repositories
- **Request Flow Tracing**: Traces every HTTP route through the full system
- **Layer Violation Detection**: Detects layer bypass, DDD violations, circular dependencies
- **Browser Session Management**: Pooled headless Chromium sessions
- **Cost Tracking**: Per-model, per-agent, per-phase cost tracking
- **Four-Phase Development Lifecycle**: Demo → Prototype → MVP → Production with gated transitions
- **Deployment & Observability**: Health check endpoints, readiness checks, production builds

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

## Tech Stack

- **Language**: Go 1.24.4
- **CLI Framework**: Cobra + PTerm
- **TUI**: Bubbletea + Lipgloss
- **Database**: BoltDB
- **LLM Integration**: REST API (OpenCode / OpenAI-compatible)
- **Browser**: go-rod (headless Chromium)

## License

MIT
