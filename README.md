# LoopGoal

**LoopGoal** is a local-first autonomous development supervisor and protocol for AI coding agents.

It keeps an AI coding agent focused on a persistent development goal through continuous, small, and independently verifiable improvements:

```text
Observe → Select ONE improvement → Implement → Verify → Review Diff → Commit → Repeat
```

LoopGoal works across arbitrary software repositories (Go, Next.js, Python, Rust, Node.js, etc.) without language lock-in or proprietary agent dependencies.

---

## Two-Layer Architecture

LoopGoal operates at two complementary levels:

```text
┌─────────────────────────────────────────────────────────────┐
│                    Layer 1: Agent Skill                     │
│  (Directly inside Antigravity, Claude Code, Codex, etc.)     │
│                                                             │
│  /loopgoal "improve API reliability"                        │
│  /loopgoal status                                           │
│  /loopgoal stop                                             │
└──────────────────────────────┬──────────────────────────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                   Layer 2: Go Supervisor                    │
│   (Standalone local process for external CLI executors)     │
│                                                             │
│  loopgoal init                                              │
│  loopgoal run                                               │
│  loopgoal status                                            │
│  loopgoal stop                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 1. Step-by-Step User Guides

### Guide A: Using LoopGoal Inside Your AI Agent (Antigravity, Claude Code, etc.)

Follow these 3 simple steps directly in your chat:

1. **Start the Loop**:
   Type the slash command with your desired goal:
   ```text
   /loopgoal "improve API error handling and validation"
   ```
2. **Observe Autonomous Progress**:
   The agent will run through disciplined, bounded iterations:
   ```text
   LoopGoal started
   Goal: improve API error handling and validation

   Iteration 1
   → Inspecting repository
   → Identified: duplicate 400 Bad Request error payloads in api/handlers.go
   → Implementing: extracted RespondBadRequest helper
   ✓ Verification: go test ./... (passed)
   ✓ Committed: refactor(api): centralize bad request error responses (9a1f23c)

   Iteration 2
   → Inspecting repository
   → Identified: missing validation for negative amounts in transfer service
   → Implementing: added validation rule and edge-case unit test
   ✓ Verification: go test ./... (passed)
   ✓ Committed: feat(transfer): validate non-negative transfer amounts (3b4d5e6)
   ```
3. **Monitor or Halt Anytime**:
   - Check current progress: `/loopgoal status`
   - Gracefully halt: `/loopgoal stop`

---

### Guide B: Using the Standalone CLI Supervisor

For headless execution, CI pipelines, or external coding agents:

1. **Initialize Your Project**:
   ```bash
   loopgoal init
   ```
   LoopGoal automatically inspects your repository (`go.mod`, `package.json`, `Cargo.toml`, `pyproject.toml`) and configures your project's verification test commands.

2. **Review or Customize `.loopgoal/config.yaml`**:
   ```yaml
   goal: "Add comprehensive unit tests for all edge cases"
   agent:
     command: "codex"
   verify:
     - "go test ./..."
     - "go vet ./..."
   limits:
     iterations: 10
   ```

3. **Start the Autonomous Supervisor**:
   ```bash
   loopgoal run
   ```
   To inspect live status in a separate terminal:
   ```bash
   loopgoal status
   ```
   To halt gracefully without corrupting state or in-flight Git changes:
   ```bash
   loopgoal stop
   ```

---

## 2. Real-World Use Cases

Here are the 4 most popular ways developers use LoopGoal:

### Use Case 1: Automated Test Coverage Expansion
- **The Problem**: A backend service has 45% unit test coverage; writing tests for dozens of edge cases is tedious.
- **The LoopGoal Approach**:
  ```text
  /loopgoal "increase test coverage for edge cases across all services"
  ```
  1. **Iteration 1**: Detects missing test for nil pointer in user authentication; adds test fixture; verifies with `go test ./...`; commits.
  2. **Iteration 2**: Detects untested timeout case in database connection pool; adds timeout mock test; verifies; commits.
  3. **Iteration 3**: Detects malformed JSON payload handling; adds test; verifies; commits.

### Use Case 2: Codebase Modernization & Technical Debt Reduction
- **The Problem**: A legacy codebase has inconsistent error handling, deprecated utility functions, or mixed async patterns.
- **The LoopGoal Approach**:
  ```text
  /loopgoal "modernize error handling and replace deprecated helpers"
  ```
  1. **Iteration 1**: Replaces deprecated `ioutil.ReadFile` with `os.ReadFile` in config module; verifies tests pass; commits.
  2. **Iteration 2**: Standardizes error wrapping using `fmt.Errorf("%w")` in auth module; verifies; commits.
  3. **Iteration 3**: Consolidates duplicate validation helpers in user service; verifies; commits.

### Use Case 3: Overnight / Unattended Continuous Improvement
- **The Problem**: You want to improve a project overnight while you sleep, but standard AI agents often go off the rails, hallucinate broad rewrites, or break working functionality.
- **The LoopGoal Safety Advantage**:
  - LoopGoal limits the agent to **one small bounded improvement per cycle**.
  - If any test or linter fails, LoopGoal triggers a fix loop. If it cannot be resolved, it halts safely without committing broken code.
  - Zero pushes to remote repository (`push = disabled`).
  - Pre-existing uncommitted work in your working tree is preserved and never overwritten.

### Use Case 4: Project-Agnostic Microservices & Monorepos
- **The Problem**: Engineering teams use Go, Next.js, Python, and Rust across different repositories.
- **The LoopGoal Solution**:
  - In a Go repo: runs `go test ./...`
  - In a Next.js repo: runs `npm run lint` && `npm run typecheck` && `npm test`
  - In a Python repo: runs `pytest`
  - The LoopGoal supervisor engine remains 100% identical; only project configuration adapts.

---

## 3. Supported In-Agent Commands & Adapters

### Slash Commands
- `/loopgoal`: Starts or resumes the autonomous loop using `.loopgoal/config.yaml`.
- `/loopgoal <goal>`: Runs the autonomous loop toward the specified goal.
- `/loopgoal status`: Displays current execution status, iteration count, last task, and last commit.
- `/loopgoal stop`: Gracefully halts the autonomous loop.

### Native Agent Adapters
LoopGoal provides native configuration and prompt adapters in [`adapters/`](./adapters/):
- **Google Antigravity**: [`adapters/antigravity/SKILL.md`](./adapters/antigravity/SKILL.md) (installed in `.agents/skills/loopgoal/`)
- **Claude Code**: [`adapters/claude/CLAUDE.md`](./adapters/claude/CLAUDE.md)
- **Codex / OpenCode**: [`adapters/codex/CODEX.md`](./adapters/codex/CODEX.md)
- **Universal LLM Prompt**: [`adapters/generic/SYSTEM_PROMPT.md`](./adapters/generic/SYSTEM_PROMPT.md)

---

## 4. Standalone Go CLI Reference

For workflows where a local command-line supervisor controls an external agent process (e.g. headless CI, local scripts, or external CLI adapters).

### Installation

```bash
go build -o loopgoal ./cmd/loopgoal
```

### CLI Commands

#### Initialize Project
```bash
loopgoal init
```
Creates `.loopgoal/config.yaml` and `.loopgoal/state.json`.

#### Start Autonomous Loop
```bash
loopgoal run
```
Optionally specify an iteration limit or custom directory:
```bash
loopgoal run --iterations 10 --dir ./my-project
```

#### Check Status
```bash
loopgoal status
```
Output:
```text
LoopGoal Status
────────────────────────────
Goal:        Continuously improve this project with small, safe, production-quality changes.
Iteration:   4
Status:      running (PID: 23145)
Last task:   centralize API error responses
Last commit: a1b2c3d
Started:     2026-09-17T12:00:00Z
Updated:     2026-09-17T12:05:00Z
```

#### Stop Running Loop
```bash
loopgoal stop
```
Gracefully requests termination without corrupting state or in-flight Git changes.

---

## Configuration (`.loopgoal/config.yaml`)

```yaml
goal: >
  Continuously improve this project with small,
  safe, production-quality changes.

agent:
  command: "codex"

verify:
  - "go test ./..."
  - "go vet ./..."

limits:
  iterations: 20
  max_retries: 3
```

LoopGoal is completely project-agnostic. For a Next.js / TypeScript project, simply configure:

```yaml
verify:
  - "npm run lint"
  - "npm run typecheck"
  - "npm test"
```

---

## Iteration Rules & Safety

1. **One Improvement Per Iteration**: The agent is restricted to one small, bounded, verifiable change per cycle.
2. **Authoritative Verification**: Changes are only committed when all configured verification commands pass. If verification fails, failure output is passed back to the agent for targeted fixes.
3. **Diff Review & User Change Isolation**: Pre-existing uncommitted user changes are detected and preserved. Only files changed by the current iteration are staged.
4. **Git Safety**:
   - `git push` is disabled. LoopGoal works entirely in local Git.
   - Destructive commands (`git reset --hard`, `git clean -fd`) are forbidden.

---

## Project Structure

```text
loopgoal/
├── cmd/
│   └── loopgoal/
│       └── main.go            # CLI entrypoint (thin binary wrapper)
├── internal/
│   ├── agent/                 # Agent interface & CommandAgent adapter
│   ├── cli/                   # CLI command dispatcher (init, run, status, stop)
│   ├── config/                # YAML configuration parser & validation
│   ├── detect/                # Smart project stack autodetection
│   ├── git/                   # Git operations wrapper & diff safety
│   ├── loop/                  # Autonomous supervisor loop engine
│   ├── state/                 # State persistence (.loopgoal/state.json)
│   └── verify/                # Project-agnostic verification runner
├── tests/
│   ├── agent/                 # Agent adapter unit tests
│   ├── cli/                   # CLI integration tests
│   ├── config/                # Configuration unit tests
│   ├── detect/                # Stack autodetection tests
│   ├── e2e/                   # Multi-stack E2E workflow tests
│   ├── git/                   # Git isolation & commit safety tests
│   ├── loop/                  # Autonomous loop engine tests
│   ├── state/                 # State persistence tests
│   └── verify/                # Verification runner tests
├── skill/
│   ├── SKILL.md               # Universal LoopGoal agent skill
│   └── loopgoal.md            # Deep protocol reference
├── adapters/
│   ├── antigravity/           # Antigravity skill adapter
│   ├── claude/                # Claude Code adapter
│   ├── codex/                 # OpenAI Codex & OpenCode adapter
│   └── generic/               # Universal LLM prompt
├── .agents/
│   └── skills/
│       └── loopgoal/          # Active workspace skill for Antigravity
├── go.mod
├── go.sum
└── README.md
```

---

## Running Tests

```bash
go test -v ./...
```
