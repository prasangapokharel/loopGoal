# AGENTS.md

## Project

LoopGoal is a local autonomous development loop for AI coding agents.

Its purpose is to keep an AI agent working toward a user-defined goal through small, verifiable iterations.

The core loop is:

```text
Observe → Select → Execute → Verify → Commit → Repeat
```

## MVP Principles

* Keep the implementation small.
* Prefer the Go standard library.
* Do not couple the core loop to a specific AI provider.
* Do not duplicate agent, Git, verification, or state logic.
* One iteration should produce one bounded improvement.
* Never make unrelated changes.
* Never silently push to a remote repository.
* Preserve the existing project's architecture.
* Prefer existing project patterns over introducing new abstractions.
* Stop when the goal is reached, the iteration limit is reached, or the agent is blocked.

## Architecture

```text
cmd/loopgoal
      │
      ▼
   loop
      │
 ┌────┼─────────┐
 ▼    ▼         ▼
agent git     verify
      │
      ▼
    state
```

### `cmd/loopgoal`

CLI entry point only.

Do not put business logic here.

### `internal/loop`

Owns the autonomous loop.

Responsibilities:

* Load configuration.
* Build the iteration prompt.
* Run the agent.
* Verify changes.
* Commit successful changes.
* Persist state.
* Decide whether another iteration should run.

The loop must not know implementation details of a specific agent.

### `internal/agent`

Defines the agent interface and agent execution.

The core interface should remain small:

```go
type Agent interface {
    Run(ctx context.Context, task string) (Result, error)
}
```

Agent implementations must be replaceable without modifying the loop engine.

For the MVP, a command-based adapter is sufficient.

### `internal/git`

Owns Git operations.

Examples:

```text
Status
Diff
Commit
```

Do not spread Git shell commands throughout the project.

### `internal/verify`

Runs project verification commands.

Examples:

```text
go test ./...
go vet ./...
npm run lint
npm run typecheck
```

Verification commands come from configuration rather than being hardcoded for every project.

### `internal/state`

Persists LoopGoal execution state.

MVP state should include:

```text
goal
iteration
status
last commit
last task
started time
updated time
```

Use a simple local state file for the MVP.

Avoid introducing a database until it is necessary.

## Agent Execution

LoopGoal should treat an AI coding agent as an external executor.

Conceptually:

```text
LoopGoal
   │
   │ task
   ▼
Agent
   │
   │ changes repository
   ▼
Project
   │
   ▼
Verification
```

LoopGoal does not need to understand how the agent reasons internally.

## Iteration Rules

Every iteration must:

1. Inspect the repository.
2. Identify one small, valuable improvement.
3. Implement only that improvement.
4. Run configured verification.
5. Inspect the resulting Git diff.
6. Commit if verification succeeds.
7. Save state.
8. Start the next iteration if allowed.

The agent should never be instructed to perform unlimited unrelated work in a single iteration.

Prefer:

```text
Fix one error-handling inconsistency.
```

over:

```text
Improve the entire backend.
```

## Failure Handling

If verification fails:

```text
Agent
  ↓
Verification failed
  ↓
Give failure output back to agent
  ↓
Agent fixes the change
  ↓
Verify again
```

If the iteration cannot be safely completed, mark it as blocked.

Do not create a commit for a failed iteration.

## Git Safety

MVP defaults:

```text
commit: enabled
push: disabled
```

LoopGoal must never push to a remote repository unless explicit support is added later.

Before committing:

* Verify the working tree.
* Verify the configured checks passed.
* Ensure the diff belongs to the current task.
* Avoid committing unrelated existing user changes.

## Configuration

Use a small configuration file:

```yaml
goal: >
  Continuously improve this project with small,
  safe, production-quality changes.

agent:
  command: "codex"

verify:
  - "go test ./..."

limits:
  iterations: 20
```

Do not introduce configuration for features that do not exist yet.

## Prompt Design

Every agent task should contain:

* The primary goal.
* Current iteration.
* Repository context.
* One bounded objective.
* Safety constraints.
* Verification requirements.

Example:

```text
Goal:
Continuously improve this project.

Iteration:
12

Instructions:
Inspect the repository and identify ONE small,
valuable improvement toward the goal.

Rules:
- Make only the required changes.
- Do not rewrite unrelated code.
- Preserve existing architecture.
- Run the configured verification commands.
- Do not modify unrelated files.
- Stop after completing this improvement.

After implementation, report:
- what changed
- verification result
- whether the iteration is complete or blocked
```

## Completion States

The MVP should recognize these states:

```text
completed
blocked
failed
goal_reached
stopped
```

Do not rely exclusively on natural-language output to determine state.

Git status and verification results are authoritative for repository state.

## Code Style

Use idiomatic modern Go.

Prefer:

```go
context.Context
errors
fmt
os/exec
encoding/json
```

and other standard-library packages where practical.

Keep functions focused.

Avoid:

* unnecessary interfaces
* premature dependency injection
* global mutable state
* duplicate helpers
* deeply nested packages
* generic abstractions without multiple concrete uses

## MVP Non-Goals

Do not implement these in the first version:

* Web dashboard
* Cloud service
* User accounts
* Multi-user execution
* Remote agent marketplace
* Automatic Git push
* Complex AI memory
* Vector database
* Distributed workers
* Plugin marketplace
* Large configuration system

The MVP should first prove that the autonomous development loop is reliable.

## FODLER STRUCURE
loopgoal/
├── cmd/
│   └── loopgoal/
│       └── main.go
│
├── internal/
│   ├── agent/
│   │   ├── agent.go
│   │   └── command.go
│   │
│   ├── loop/
│   │   └── loop.go
│   │
│   ├── git/
│   │   └── git.go
│   │
│   ├── verify/
│   │   └── verify.go
│   │
│   └── state/
│       └── state.go
│
├── AGENTS.md
├── go.mod
├── go.sum
└── README.md

## Future Direction

The architecture should allow:

```text
Antigravity
Codex
Claude Code
OpenCode
Other CLI agents
```

to become agent adapters without changing the loop engine.

Future architecture:

```text
                 LoopGoal
                    │
              Agent Interface
                    │
       ┌────────────┼────────────┐
       ▼            ▼            ▼
 Antigravity      Codex       Claude
```

The core product remains the loop, not the individual AI provider.
