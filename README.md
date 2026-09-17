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

## 1. Using LoopGoal as an In-Agent Skill

Type slash-commands directly in your AI coding assistant:

```text
/loopgoal "improve API error handling"
```

### Supported Commands

- `/loopgoal`: Starts or resumes the autonomous loop using `.loopgoal/config.yaml`.
- `/loopgoal <goal>`: Runs the autonomous loop toward the specified goal.
- `/loopgoal status`: Displays current execution status, iteration count, last task, and last commit.
- `/loopgoal stop`: Gracefully halts the autonomous loop.

### Agent Adapters

LoopGoal provides native configuration and prompt adapters for leading agent environments in [`adapters/`](./adapters/):

- **Google Antigravity**: [`adapters/antigravity/SKILL.md`](./adapters/antigravity/SKILL.md) (also installed in `.agents/skills/loopgoal/`)
- **Claude Code**: [`adapters/claude/CLAUDE.md`](./adapters/claude/CLAUDE.md)
- **Codex / OpenCode**: [`adapters/codex/CODEX.md`](./adapters/codex/CODEX.md)
- **Universal LLM Prompt**: [`adapters/generic/SYSTEM_PROMPT.md`](./adapters/generic/SYSTEM_PROMPT.md)

---

## 2. Using the Standalone Go CLI Supervisor

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
│       ├── main.go            # CLI entrypoint (init, run, status, stop)
│       └── main_test.go       # CLI integration tests
├── internal/
│   ├── config/                # YAML configuration parser & validation
│   ├── state/                 # State persistence (.loopgoal/state.json)
│   ├── git/                   # Git operations wrapper & diff safety
│   ├── verify/                # Project-agnostic verification runner
│   ├── agent/                 # Agent interface & CommandAgent adapter
│   └── loop/                  # Autonomous supervisor loop engine
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
