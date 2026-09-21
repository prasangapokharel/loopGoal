# LoopGoal

> **The autonomous development supervisor that keeps AI coding agents focused, safe, and productive — iteration by iteration.**

LoopGoal solves a core problem with AI coding agents: **they drift**. Without structure, they rewrite entire systems, break working tests, or hallucinate broad changes. LoopGoal enforces a tight, repeatable loop that constrains the agent to **one small, verified, committed improvement per cycle** — forever, until your goal is reached.

```text
Observe → Select ONE improvement → Implement → Verify → Review Diff → Commit → Repeat
```

Works with any AI agent (Antigravity, Claude Code, Codex, OpenCode) and any project stack (Go, Python, TypeScript, Rust, Node.js, Java, monorepos).

---

## Why LoopGoal Exists

| Without LoopGoal | With LoopGoal |
|---|---|
| Agent rewrites 40 files in one shot | One file per bounded iteration |
| Breaks existing tests silently | Verification gates every commit |
| Commits broken or unrelated code | Only staged changes from current task |
| Overwrites your in-progress WIP | Developer work is always preserved |
| Pushes to remote without warning | Push is permanently disabled |
| No way to safely stop mid-run | Graceful stop with `loopgoal stop` |
| No audit trail | Every iteration has a committed diff |

---

## Two-Layer Architecture

LoopGoal operates at two complementary levels — choose the one that fits your workflow:

```text
┌─────────────────────────────────────────────────────────────────┐
│                    LAYER 1 — Agent Skill                        │
│    Runs inside Antigravity, Claude Code, Codex, etc.            │
│                                                                 │
│  /loopgoal "improve API reliability"     → starts the loop      │
│  /loopgoal status                        → shows progress       │
│  /loopgoal stop                          → halts safely         │
│                                                                 │
│  No binary needed. The agent IS the executor.                   │
└──────────────────────────────┬──────────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                  LAYER 2 — Go Supervisor CLI                    │
│   Headless process that drives any external CLI agent.          │
│                                                                 │
│  loopgoal init    → detect stack, write config                  │
│  loopgoal run     → launch autonomous loop                      │
│  loopgoal status  → check iteration, last commit, PID           │
│  loopgoal stop    → signal graceful shutdown                    │
│                                                                 │
│  Use for CI pipelines, overnight runs, external agent CLIs.     │
└─────────────────────────────────────────────────────────────────┘
```

---

---

## Install

Install LoopGoal across all your AI coding agents in one command:

```bash
npx loopgoal install
```

Or via curl (macOS & Linux):
```bash
curl -fsSL https://raw.githubusercontent.com/prasangapokharel/loopGoal/main/install.sh | bash
```

Or via Go:
```bash
go install github.com/prasangapokharel/loopGoal/cmd/loopgoal@latest
```

---

## Quick Start

### Option A — Use Inside Your AI Agent (Fastest)

#### 1. Google Antigravity (AGY)
```bash
# Install globally for all projects:
mkdir -p ~/.gemini/config/skills/loopgoal
cp adapters/antigravity/SKILL.md ~/.gemini/config/skills/loopgoal/SKILL.md

# Or install for current project only:
mkdir -p .agents/skills/loopgoal
cp adapters/antigravity/SKILL.md .agents/skills/loopgoal/SKILL.md
```
**Run in Chat:**
```text
/loopgoal Refactor backend/api/v1/ and write unit tests in tests/unit/
```

#### 2. Claude Code (CLI)
```bash
# Install globally for all projects:
mkdir -p ~/.claude/commands
cp adapters/claude/CLAUDE.md ~/.claude/commands/loopgoal.md

# Or install for current project only:
mkdir -p .claude/commands
cp adapters/claude/CLAUDE.md .claude/commands/loopgoal.md
```
**Run in Claude CLI:**
```text
/loopgoal Refactor backend/api/v1/ and write unit tests in tests/unit/
```

#### 3. Cursor / Roo Code / Cline
```bash
# Install as a Cursor Rule:
mkdir -p .cursor/rules
cp adapters/antigravity/SKILL.md .cursor/rules/loopgoal.mdc
```
**Run in Composer / Chat:**
```text
@loopgoal Refactor backend/api/v1/ and write unit tests in tests/unit/
```

---

### Option B — Standalone CLI Supervisor

For headless execution, CI integration, or driving external CLI agents (Antigravity `agy`, Claude `claude`, Codex `codex`, OpenCode `opencode`):

**Step 1 — Build and install the binary:**
```bash
go install ./cmd/loopgoal
```

**Step 2 — Initialize your project:**
```bash
cd /path/to/your-project
loopgoal init
```
LoopGoal auto-detects your stack and writes `.loopgoal/config.yaml`:
```text
✓ Initialized LoopGoal in .loopgoal/ (Go module detected)
  Config: .loopgoal/config.yaml
  State:  .loopgoal/state.json
```

**Step 3 — Run pre-flight checks:**
```bash
loopgoal test --smoke
```

**Step 4 — Inspect inventory & plan:**
```bash
loopgoal scan   # View all discovered files categorized
loopgoal plan   # View active task map & pending queue
```

**Step 5 — Start the autonomous loop:**
```bash
loopgoal run
```

**Step 6 — In another terminal, monitor live progress:**
```bash
loopgoal status
```

**Step 7 — Halt cleanly whenever you want:**
```bash
loopgoal stop
```

**Step 3 — Edit the config to your goal:**
```yaml
# .loopgoal/config.yaml
goal: >
  Increase test coverage for all edge cases
  in the authentication and payment services.

agent:
  command: "codex"

verify:
  - "go test ./..."
  - "go vet ./..."

limits:
  iterations: 20
  max_retries: 3
```

**Step 4 — Start the autonomous loop:**
```bash
loopgoal run
```

**Step 5 — In another terminal, monitor live progress:**
```bash
loopgoal status
```
```text
LoopGoal Status
────────────────────────────
Goal:        Increase test coverage for all edge cases
Iteration:   7
Status:      running (PID: 23145)
Last task:   add nil pointer test for token validator
Last commit: a1b2c3d
Last check:  passed
Started:     2026-09-17T22:00:00Z
Updated:     2026-09-17T22:35:00Z
```

**Step 6 — Halt cleanly whenever you want:**
```bash
loopgoal stop
```
```text
✓ Sent graceful stop signal to LoopGoal (PID: 23145).
```

---

## Real-World Use Cases

### 🔴 Problem 1: "My test coverage is stuck at 40% and writing tests for hundreds of edge cases manually is impossible"

**Root Cause**: Writing unit tests is repetitive and context-heavy — humans avoid it; unconstrained AI agents over-generate unrelated tests.

**Solution — LoopGoal targeted coverage expansion:**
```text
/loopgoal "increase test coverage to 80% — one edge case per iteration"
```

| Iteration | What the agent identified | What it fixed | Verification |
|---|---|---|---|
| 1 | Nil pointer in `auth/token.go:ValidateToken` uncovered | Added `TestValidateToken_NilInput` | `go test ./...` ✅ |
| 2 | DB timeout in `store/pool.go` not tested | Added `TestPool_ConnectionTimeout` with mock | `go test ./...` ✅ |
| 3 | Malformed JSON in `api/handler.go` not handled | Added `TestHandleRequest_MalformedJSON` | `go test ./...` ✅ |
| 4 | Race condition in `worker/queue.go` | Added `TestQueue_ConcurrentPush` with `-race` flag | `go test -race ./...` ✅ |
| ... | Continues autonomously... | | |

**Outcome**: Coverage grows from 40% → 80%+, one test at a time, each commit independently reviewable.

---

### 🔴 Problem 2: "Our codebase has years of technical debt — deprecated APIs, inconsistent error handling, dead code — and we can't afford to refactor everything at once"

**Root Cause**: Big-bang refactors break things. Refactoring one function at a time takes months manually.

**Solution — LoopGoal incremental debt elimination:**
```text
/loopgoal "modernize error handling: replace deprecated helpers, standardize fmt.Errorf wrapping"
```

| Iteration | File targeted | Change made | Risk |
|---|---|---|---|
| 1 | `internal/config/config.go` | `ioutil.ReadFile` → `os.ReadFile` | Zero — API-identical |
| 2 | `internal/auth/handler.go` | Raw `errors.New` → `fmt.Errorf("%w", err)` for wrapping | Zero |
| 3 | `internal/user/service.go` | Duplicate `validateEmail()` helpers consolidated into one | Zero |
| 4 | `internal/api/middleware.go` | `log.Printf` → structured `slog.Error` | Zero |
| ... | Continues autonomously... | | |

**Outcome**: Every commit is atomic, reviewable, and passes all tests. Debt is eliminated with no regression risk.

---

### 🔴 Problem 3: "I want to run AI-assisted improvement overnight, but I'm scared the agent will rewrite everything and break production code"

**Root Cause**: Unconstrained AI agents given broad goals will make large sweeping changes, break tests, and commit broken code if not stopped.

**Solution — LoopGoal's safety guarantees make overnight runs safe:**

```bash
# Start before bed
loopgoal run --iterations 50
```

LoopGoal enforces a strict 7-layer safety system:

| Safety Layer | What it prevents |
|---|---|
| **One improvement per iteration** | No broad rewrites or multi-file sprints |
| **Verification gate** | No broken commit ever reaches git history |
| **Retry loop (max 3)** | Transient failures are fixed; persistent failures halt cleanly |
| **Diff isolation** | Only files from current task are staged — not your WIP |
| **Developer change preservation** | Pre-existing dirty files are detected and excluded |
| **No push** | Remote repository is never touched |
| **Stop signal** | `loopgoal stop` or `Ctrl+C` halts cleanly after current iteration |

```text
# What you wake up to:
git log --oneline -10
a7f3b2c test(auth): add expired token edge case
92e1d4a test(store): cover DB timeout with mock
3c5f8e1 refactor(config): replace ioutil with os.ReadFile
7b1a9c2 fix(api): standardize 400 error response format
d4e2f1b test(worker): add concurrent queue race test
...
```

**Outcome**: 8 hours of sleep = 20–50 clean, reviewable, tested commits. No surprises.

---

### 🔴 Problem 4: "We run Go, Python, TypeScript, and Rust microservices — we can't use different tools for each"

**Root Cause**: Most AI automation tools are language-specific. Teams waste time maintaining separate automation pipelines per stack.

**Solution — LoopGoal is 100% project-agnostic:**

**Go microservice:**
```yaml
verify:
  - "go test ./..."
  - "go vet ./..."
```

**Python Django API:**
```yaml
verify:
  - "pytest"
  - "ruff check ."
  - "mypy src/"
```

**Next.js / TypeScript frontend:**
```yaml
verify:
  - "npm run lint"
  - "npm run typecheck"
  - "npm test"
```

**Rust service:**
```yaml
verify:
  - "cargo test"
  - "cargo clippy -- -D warnings"
```

**Monorepo with all of the above:**
```yaml
verify:
  - "go test ./services/..."
  - "pytest backend/"
  - "npm test --workspace=frontend"
  - "cargo test -p payments"
```

The loop engine is identical. Only the `verify:` commands differ. The same `loopgoal` binary manages all of them.

---

### 🔴 Problem 5: "We use custom code standards (cursor rules, agent rules, skill files) — how do we make sure the AI follows ALL of them?"

**Root Cause**: AI agents often ignore `.cursor/rules/*.mdc`, `AGENTS.md`, or `.agents/skills/` files unless explicitly forced to read them at every iteration.

**Solution — LoopGoal's Audit Layer enforces compliance automatically:**

Before each iteration, LoopGoal's `audit` package:
1. **Discovers all rule files** in the project:
   - `AGENTS.md`, `GEMINI.md`, `CLAUDE.md`
   - `.cursor/rules/*.mdc`
   - `.agents/rules/*.md`
   - `.agents/skills/*/SKILL.md`
2. **Runs grep/ripgrep pattern sweeps** across all git-tracked source files to detect violations
3. **Runs static analysis** (`go vet`, `ruff check`, `tsc --noEmit`, `cargo clippy`) automatically
4. **Injects findings into the agent prompt** so the agent knows exactly what to fix

```text
Iteration 5 of 20
→ Auditing rules: AGENTS.md, .cursor/rules/naming.mdc, .agents/skills/loopgoal/SKILL.md
→ Grep scan: found 2 violations
    internal/api/handler.go:42  [camelCase]  func handleUserRequest()
    internal/store/db.go:88     [TODO]       // TODO: fix this
→ Running static checks: go vet ./... PASSED
→ Implementing: rename handleUserRequest → HandleUserRequest, resolve TODO
✓ Verification: go test ./... PASSED
✓ Committed: refactor(api): enforce naming and resolve TODOs per AGENTS.md
```

**Outcome**: Every file in the repository eventually reaches 100% compliance with your project rules — automatically.

---

### 🔴 Problem 6: "I run AI agents against large codebases but they only look at one file and then stop"

**Root Cause**: AI agents in chat interfaces naturally end their turn after one action. Without explicit queue management, they stop prematurely.

**Solution — LoopGoal's Zero-Premature-Stop guarantee:**

LoopGoal maintains a `remaining_queue` in `.loopgoal/state.json`. The protocol enforces:

```json
{
  "goal": "enforce naming conventions across all files",
  "iteration": 3,
  "status": "running",
  "remaining_queue": [
    "internal/auth/handler.go",
    "internal/store/db.go",
    "internal/api/middleware.go"
  ],
  "completed_files": [
    "internal/config/config.go",
    "internal/loop/loop.go"
  ]
}
```

- `"status"` stays `"running"` until `"remaining_queue"` is **completely empty**
- After each file, the agent immediately starts the next — **no waiting for user input**
- If interrupted (`/loopgoal stop`), state is preserved — resuming with `/loopgoal` picks up exactly where it left off

**Outcome**: Every file in scope gets processed. The agent never stops early or needs prompting to continue.

---

## Supported Agents & Adapters

Install the LoopGoal adapter for your AI coding agent:

| Agent | Adapter | Installation |
|---|---|---|
| **Google Antigravity** | [`adapters/antigravity/SKILL.md`](./adapters/antigravity/SKILL.md) | Copy to `.agents/skills/loopgoal/SKILL.md` |
| **Claude Code** | [`adapters/claude/CLAUDE.md`](./adapters/claude/CLAUDE.md) | Copy to `CLAUDE.md` or `.claude/commands/loopgoal.md` |
| **OpenAI Codex / OpenCode** | [`adapters/codex/CODEX.md`](./adapters/codex/CODEX.md) | Follow instructions in file |
| **Any LLM / Custom Agent** | [`adapters/generic/SYSTEM_PROMPT.md`](./adapters/generic/SYSTEM_PROMPT.md) | Inject as system prompt |

### Slash Commands (All Agents)

| Command | What it does |
|---|---|
| `/loopgoal <goal>` | Start autonomous loop toward the specified goal |
| `/loopgoal` | Resume from `.loopgoal/state.json` if items remain in queue |
| `/loopgoal status` | Print current iteration, status, last commit, and remaining queue |
| `/loopgoal stop` | Write `stopped` status to state file and halt after current iteration |

---

## CLI Reference

```bash
# Build
go build -o loopgoal ./cmd/loopgoal

# Initialize (auto-detects Go / Node / Python / Rust)
loopgoal init
loopgoal init --force          # overwrite existing config
loopgoal init --dir ./myrepo   # specify project directory

# Run the autonomous loop
loopgoal run
loopgoal run --iterations 10
loopgoal run --dir ./myrepo
loopgoal run --config ./custom-config.yaml

# Check status
loopgoal status
loopgoal status --dir ./myrepo

# Request graceful stop
loopgoal stop
loopgoal stop --dir ./myrepo
```

---

## Configuration Reference

```yaml
# .loopgoal/config.yaml

goal: >
  Continuously improve this project with small,
  safe, production-quality changes.

agent:
  command: "codex"          # any CLI command
  args: []                  # optional CLI arguments
                            # use {task} or {prompt} as placeholder for the task prompt

verify:
  - "go test ./..."         # all commands must exit 0 for a commit to proceed
  - "go vet ./..."

limits:
  iterations: 20            # maximum number of loop cycles
  max_retries: 3            # max fix attempts per iteration when verification fails
```

### Configuration Tips

- Set `iterations: 0` to run indefinitely until the goal is reached or you stop manually.
- Use `max_retries: 0` to halt immediately on verification failure without retry.
- Verification commands are run sequentially — fail-fast on first error.
- The `{task}` placeholder in `args` is replaced with the full iteration prompt at runtime.

---

## Iteration Rules & Safety Guarantees

Every iteration follows this strict sequence — no exceptions:

```text
1. INSPECT   → git status, git diff, identify pre-existing user changes
2. SELECT    → one small, bounded improvement toward the goal
3. IMPLEMENT → write code changes
4. VERIFY    → run all configured commands (exit 0 = pass)
5. RETRY     → if verify fails: send error output to agent, fix, re-verify (up to max_retries)
6. DIFF      → confirm changes are isolated to current task only
7. STAGE     → git add <changed files from this iteration only>
8. COMMIT    → git commit -m "<conventional commit message>"
9. PERSIST   → write updated state to .loopgoal/state.json
10. ADVANCE  → next iteration, or halt if goal reached / limit hit / stopped
```

**Hard rules that are never bypassed:**
- `git push` is **permanently disabled**
- `git reset --hard` and `git clean -fd` are **forbidden**
- Pre-existing uncommitted files are **never staged**
- A commit is **never created** when verification is failing

---

## Project Structure

```text
loopgoal/
├── cmd/loopgoal/
│   └── main.go                 # CLI entrypoint
├── internal/
│   ├── agent/                  # Agent interface & CommandAgent (streaming output)
│   ├── audit/                  # Rule discovery, grep/rg compliance scanning
│   ├── cli/                    # init / run / status / stop command handlers
│   ├── config/                 # YAML config parser, validation, defaults
│   ├── detect/                 # Auto-detect Go / Node / Python / Rust stack
│   ├── git/                    # Git wrapper: status, diff, stage, commit (no push)
│   ├── loop/                   # Autonomous supervisor loop engine
│   ├── inventory/              # Empirical repository scanner & classification
│   ├── loop/                   # Core autonomous engine
│   ├── reconcile/              # Expected vs actual diff reconciliation
│   ├── resolve/                # Cross-platform binary resolution & headless flags
│   ├── state/                  # JSON state persistence with atomic writes
│   ├── taskmap/                # Bipartite task/file graph & evidence gate
│   └── verify/                 # Project-agnostic verification runner
├── tests/
│   ├── agent/                  # Agent streaming & output parsing tests
│   ├── audit/                  # Rule discovery & grep compliance tests
│   ├── cli/                    # CLI integration tests (init/run/scan/plan/test/status/stop)
│   ├── config/                 # Config load/save/validation tests
│   ├── detect/                 # Stack autodetection tests
│   ├── e2e/                    # End-to-end loop tests (false-done, monorepo, polyglot)
│   ├── git/                    # Git safety & isolation tests
│   ├── inventory/              # File classification & inventory scan tests
│   ├── loop/                   # Autonomous loop engine unit tests
│   ├── reconcile/              # Reconciliation & dynamic discovery tests
│   ├── resolve/                # Path resolution & install hint tests
│   ├── state/                  # State persistence tests
│   ├── taskmap/                # Bipartite graph & evidence gate tests
│   └── verify/                 # Verification runner tests
├── adapters/
│   ├── antigravity/SKILL.md    # Antigravity agent adapter
│   ├── claude/CLAUDE.md        # Claude Code adapter
│   ├── codex/CODEX.md          # Codex / OpenCode adapter
│   └── generic/SYSTEM_PROMPT.md # Universal LLM system prompt
├── .agents/
│   └── skills/loopgoal/
│       └── SKILL.md            # Active Antigravity skill
├── go.mod
├── go.sum
└── README.md
```

---

## Running Tests

```bash
# Run all 14 test packages
go test ./...

# Run with verbose output
go test -v ./...

# Run pre-flight CLI diagnostics
loopgoal test --smoke
```

All 14 test packages pass. Zero `go vet` warnings.

```text
ok  loopgoal/tests/agent      — streaming, output parsing, placeholder substitution
ok  loopgoal/tests/audit      — rule discovery, grep patterns, report formatting
ok  loopgoal/tests/cli        — init/run/scan/plan/test/status/stop integration
ok  loopgoal/tests/config     — YAML load/save/validate/defaults
ok  loopgoal/tests/detect     — Go/Node/Python/Rust/generic detection
ok  loopgoal/tests/e2e        — false-done prevention, monorepo targeting, polyglot, verify→fix cycle
ok  loopgoal/tests/git        — IsRepo, staged/commit isolation
ok  loopgoal/tests/inventory  — repository scanning, role classification
ok  loopgoal/tests/loop       — successful iterations, retry, pre-existing change preservation
ok  loopgoal/tests/reconcile  — missing/unexpected diff detection, dynamic discovery
ok  loopgoal/tests/resolve    — cross-platform path resolution, install hints
ok  loopgoal/tests/state      — init/load/save/status transitions
ok  loopgoal/tests/taskmap    — bipartite task/file graph, evidence gate
ok  loopgoal/tests/verify     — pass/fail command execution
```

---

## Completion States

| Status | Meaning |
|---|---|
| `idle` | Initialized but not yet started |
| `running` | Autonomous loop is active |
| `completed` | Iteration limit reached; goal not explicitly flagged |
| `goal_reached` | Agent confirmed the goal is fully achieved |
| `stopped` | Halted by user (`loopgoal stop` or `Ctrl+C`) |
| `blocked` | Agent cannot make progress; fix loop exhausted |
| `failed` | Unrecoverable error during execution |

---

## Architecture

```text
cmd/loopgoal
      │
      ▼
   internal/cli          ← command dispatcher
      │
      ▼
   internal/loop         ← autonomous loop engine
      │
 ┌────┼────────┬──────────┐
 ▼    ▼        ▼          ▼
agent git    verify     audit
 │
 ▼
CommandAgent (streaming)
 │
 ▼
External CLI (codex / agy / claude / opencode / any command)
```

The loop engine is decoupled from every agent implementation. To add a new agent, implement one interface:

```go
type Agent interface {
    Run(ctx context.Context, task string) (Result, error)
}
```

---

## MVP Non-Goals

The following are **explicitly not implemented** to keep the core loop focused and reliable:

- Web dashboard or GUI
- Cloud service or user accounts
- Remote agent marketplace
- Automatic `git push`
- Vector database or complex AI memory
- Distributed workers
- Plugin marketplace

The MVP proves that the autonomous development loop itself is reliable. Extensions come later.

---

*Built with Go. Local-first. No cloud. No telemetry. Just clean, verifiable iterations.*
