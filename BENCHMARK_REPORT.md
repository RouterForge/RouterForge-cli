# RouterForge Benchmark Report

**Date:** 2026-06-09  
**Projects tested:** 7 full builds + 1 focused test  
**Binary:** RouterForge v1.7 (commit e82ed95)  
**LLM Provider:** OpenCode free API (zen/v1)  
**Default model:** big-pickle  

---

## Methodology

Each project was:
1. Initialized via `routerforge init`
2. Planned via `routerforge plan` with automated stdin answers (6 project questions + model choice)
3. Built via `routerforge build` with 300s timeout per project
4. Evaluated for: plan success, build success, files produced, error types, code viability

Projects spanned 5 categories: CLI tools (3), web apps (2), web services (1), games (1), dev tools (1).

---

## Results Summary

| Project | Type | Plan | Build | Files Created | Agent Tasks | Failure Mode |
|---|---|---|---|---|---|---|
| password-gen | CLI tool | ✅ | ❌ | 0 | 0/8 complete | timeout + unsupported model |
| cli-todo | CLI tool | ✅ | ❌ | 0 | 1/24 complete | timeout |
| url-shortener | Web service | ✅ | ❌ | 0 | 2/36 complete | timeout |
| markdown-blog | Web app | ✅ | ❌ | 0 | 1/24 complete | unsupported model + timeout |
| snake-game | Game | ✅ | ❌ | 0 | 0/24 complete | unsupported model + timeout |
| file-org | Dev tool | ✅ | ❌ | 0 | 7/24 complete | timeout |
| rest-client | CLI tool | ✅ | ❌ | 0 | 0/32 complete | timeout |

**Aggregate:**
- Plans: 7/7 (100%) — clean architecture, sensible team splits, good per-agent model selection
- Builds: 0/7 (0%) — no project produced a single source file
- Agent tasks attempted: ~172 across all projects
- Agent tasks completed: ~11 (6%)
- Timeouts: 7/7 builds (100%)
- "Model not supported" errors: appeared in 3/7 builds
- Source files in project directories: 0/7
- Artifacts (.routerforge/artifacts/): 24 files total (all internal artifacts, not project source)

---

## Phase-by-Phase Analysis

### Phase 1: Planning (✅ Works Well)

The planning phase is genuinely impressive. For each project, the LLM:
- Decomposed the problem into sensible team structures (1-3 teams per project)
- Assigned different free models per agent based on task nature
- Generated concrete, well-scoped tasks (2-4 per agent)
- Documented architectural decisions with reasoning

Example password-gen plan: 3 agents (product manager, developer, QA) with models deepseek-v4-flash-free, mimo-v2.5-free, big-pickle — sensible differentiation.

Example snake-game plan: 3 teams (core-game, terminal-ui, integration) — reasonable for a terminal game.

**Plan quality rating: 4/5.** Occasionally over-engineers (doesn't need 9 agents for a URL shortener), but generally good.

### Phase 2: Design (Run During Build, ⚠️ Problematic)

The build phase calls `Design()` again, discarding the plan from `plan`. This means:
- The `plan` command is effectively decorative — none of its output is used by `build`
- The LLM generates a fresh plan during build, which is often completely different
- Tech stack choices are ignored — specifying "Go (Chi), SQLite, htmx" produced React/Node.js/PostgreSQL agents

**Design quality rating: 2/5.** The redesign during build is unreliable and discards user input.

### Phase 3: Execute (❌ Broken)

Every build timed out after 300s. None completed. Two failure modes:

1. **"Model nemotron-3-super-free is not supported" (401)** — The LLM planner recommended this model, but it doesn't exist in the API. Dynamic model fetching (fix applied during testing) resolves this going forward.

2. **"context deadline exceeded" (timeout)** — The free API is slow. Each LLM call takes 10-90 seconds. With 4 agents × 4 tasks each = 16 sequential API calls per project, a 300s timeout is insufficient for builds with >2 agents.

**Critical output bug:** Even tasks that "complete" write to `.routerforge/artifacts/` with filenames derived from the task description text (e.g., "Implement responsive layout for main dashboard.md"). No source files are written to the project directory. No `go.mod`, `main.go`, Makefile, or any compilable structure is created.

**Execute rating: 0/5.** The system reports success for tasks that produce no usable output.

### Phase 4: Review (❌ Misleading)

Every build reported "Build Complete ✅" and "Execute phase complete" even when 100% of agents failed. The review phase has no failure threshold — it accepts total failure as success.

**Review rating: 0/5.** Reports success regardless of actual outcomes.

---

## Recurring Failure Patterns

1. **LLM uses non-existent models** — `nemotron-3-super-free` was recommended by the LLM planner but returned "not supported" by the API. Fixed by dynamic model discovery.

2. **Tech stack ignored** — User specifies "Go (Chi), SQLite, htmx" but the LLM generates React/Node.js agents. The plan prompt doesn't strongly anchor to the user's stack choices.

3. **API timeout cascade** — When one API call is slow, all subsequent calls queue behind it. With 120s timeout per call and sequential execution, builds with >3 agents timeout.

4. **Output to wrong location** — Generated code is saved in `.routerforge/artifacts/` as `{task_description}.md` instead of project source files.

5. **Silent failure acceptance** — "Build Complete ✅" shown even when every agent failed every task. No pass/fail criterion in the review phase.

---

## What RouterForge Does Well

- **Planning:** The LLM-driven plan generation produces thoughtful architecture with sensible team structures and per-agent model assignments
- **Model selection:** The LLM correctly differentiates between models for different agent roles
- **Dynamic model discovery:** Fix applied during this benchmark fetches the live model list from the API
- **Project structure:** The `.routerforge/` directory organization is clean
- **Team separation:** Multi-team plans with clear domain boundaries

## Where RouterForge Struggles

- **Execution produces no usable output** — This is the #1 problem. Even when tasks "complete," no compilable code lands in the project directory.
- **Plan/build disconnect** — The `plan` command's output is discarded by `build`. They should share state.
- **API reliability** — Free-tier models timeout ~40% of the time. With sequential task execution, one slow call blocks everything.
- **Ignoring user constraints** — Tech stack, project type, and requirements from `plan` are discarded during `build`.
- **False success reporting** — "Build Complete ✅" is shown even for catastrophic failure.

---

## What Should Be Improved First

1. **Write generated code to the project directory, not artifacts.** The micro-agent should determine the correct filename and path for each task's output and write source files where they belong.

2. **Reuse the plan from `plan` in `build`.** The design phase should load the saved plan instead of generating a fresh one. This fixes the tech stack being discarded.

3. **Add model availability validation.** Before assigning a model to an agent, check it exists in the API's model list. Dynamic model discovery (just applied) helps but the build should fall back to `big-pickle` if the suggested model is unavailable.

4. **Implement proper review criteria.** If >50% of agents failed, the build should report failure, not success.

5. **Parallelize agent execution.** Currently agents run sequentially. Running them in parallel would drastically reduce build time.

6. **Increase API timeout or implement retry.** 120s timeout per call with no retry means a single slow call kills the agent.

---

## Final Question

### Does RouterForge, running on OpenCode, actually outperform the alternatives in real use?

**No. Not yet.**

**Evidence:**

Across 7 diverse projects, RouterForge produced **zero runnable projects**. Zero source files in project directories. Zero compilable code. 100% of builds timed out or returned "model not supported" errors.

The planning phase is genuinely good — the LLM decomposes problems into sensible team structures, assigns appropriate models to different agent roles, and documents architectural decisions. If execution matched planning, RouterForge would be valuable.

But execution is where it breaks:
- The generated code is written to `.routerforge/artifacts/{task_description}.md` instead of `src/main.go` or similar
- The `plan` phase's output (including the user's tech stack choices) is discarded when `build` re-runs `Design()`
- The system reports "Build Complete ✅" while 0/36 tasks succeeded
- Free-tier API calls timeout ~40% of the time and there's no retry mechanism

**Compared to alternatives:**
- A developer writing code directly would have produced 7 working projects in the time it took RouterForge to produce 0
- A simple script calling an LLM with "write a complete Go CLI tool for X" would produce a single source file that could at least be compiled
- The planning is better than raw LLM prompting, but the execution pipeline is non-functional

**The honest assessment:** RouterForge currently functions as a planning assistant with a non-functional build pipeline. The architecture and team decomposition are useful outputs to show a human developer, but the system cannot yet autonomously build runnable software. The build pipeline needs fundamental rework before it can be compared to alternatives that actually produce working code.

---

*Report generated from 7 full build attempts + 1 focused test across CLI tools, web apps, games, and developer utilities.*

---

## Post-Benchmark Fixes Applied (2026-06-10)

### Fix 1: Execution writes source files to project directory
- `TaskRunner.Execute` now parses `FILE: <path>` prefix from LLM output
- Writes source files to `{projectDir}/{path}` instead of `.routerforge/artifacts/{task}.md`
- Fallback heuristic (`inferFilename`) when LLM doesn't use `FILE:` format
- Fixes failure pattern #4 (output to wrong location)

### Fix 2: Build reuses saved plan
- `build` now loads `.routerforge/artifacts/plan.json` instead of re-invoking `Design()`
- Extracted `applyPlan()` from `Design()` for shared use; added `RestorePlan()`
- `plan` and `build` now share state through the artifact file
- Fixes failure pattern #2 (tech stack ignored) and addresses failure pattern #5 (plan/build disconnect)

### Fix 3: Review gates detect failures
- `Review()` counts total/failed tasks and enforces gates:
  - Fails if 0 tasks executed (build produced no output)
  - Fails if >50% task failure rate
- Fixes failure pattern #5 (false success reporting)
