---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal: Autonomous Development Loop Protocol

LoopGoal is a local-first autonomous development supervisor for AI coding agents. It keeps the agent working toward a user-defined goal through small, verifiable iterations.

The fundamental loop is:
```text
Observe → Select ONE improvement → Implement → Verify → Review Diff → Commit → Save State → Repeat
```

---

## Slash Commands & Arguments

When the user invokes `/loopgoal`:

1. **`/loopgoal <goal>`**:
   Starts the autonomous loop toward `<goal>` (e.g., `/loopgoal improve ICPay API reliability`).
   - If `.loopgoal/config.yaml` exists, update or use its verification rules.
   - If `.loopgoal/config.yaml` does not exist, inspect the repository to infer default verification commands (e.g., `go test ./...` or `npm test`) and initialize `.loopgoal/`.

2. **`/loopgoal`**:
   Starts the autonomous loop using the persistent goal defined in `.loopgoal/config.yaml`.
   - If `.loopgoal/config.yaml` does not exist, ask the user for a goal or initialize with a default goal.

3. **`/loopgoal status`**:
   Reads `.loopgoal/state.json` and prints the current status report:
   ```text
   LoopGoal Status
   ────────────────────────────
   Status:      <running | completed | blocked | stopped | goal_reached>
   Iteration:   <current_iteration>
   Goal:        <goal>
   Last task:   <summary_of_last_change>
   Last commit: <git_commit_hash>
   Last check:  <passed | failed>
   ```

4. **`/loopgoal stop`**:
   Gracefully stops the autonomous loop.
   - Updates `.loopgoal/state.json` with status `"stopped"`.
   - Halts any pending iteration.
   - Summarizes all progress made during the session.

---

## The Iteration Cycle

In each iteration, you MUST follow this strict sequence:

### 1. Observe
- Check current Git status: `git status`
- Identify any pre-existing user changes that must NOT be touched or reverted.
- Inspect project code, tests, or documentation related to the goal.

### 2. Select ONE Bounded Improvement
- Identify **ONE** small, valuable, and self-contained improvement directly contributing to the goal.
- Avoid large refactors or modifying multiple unrelated systems in a single iteration.
- Preferred examples:
  - Extract duplicated error handling in one module.
  - Add test coverage for an untested edge case.
  - Fix a type-safety inconsistency in one file.
  - Simplify an overly complex function.

### 3. Implement
- Implement **only** the selected improvement.
- Preserve existing project architecture and conventions.
- Do not make stylistic or whitespace rewrites in unrelated files.

### 4. Verify
- Run the configured verification commands (from `.loopgoal/config.yaml` or project defaults like `go test ./...`, `npm test`, etc.).
- If verification fails:
  - Analyze the error output.
  - Fix the issue immediately.
  - Re-run verification until it passes (up to 3 attempts).
  - If it cannot be fixed cleanly, stop and mark status as `blocked`.
- NEVER commit changes when verification is failing.

### 5. Review Diff
- Inspect `git diff` and `git status`.
- Ensure only files relevant to the current improvement were touched.
- Ensure pre-existing uncommitted user files are untouched.

### 6. Commit
- Stage only the relevant changed files: `git add <file1> <file2> ...`
- Commit with a clear, conventional commit message:
  ```bash
  git commit -m "<type>(<scope>): <concise description>"
  ```
  Example: `git commit -m "refactor(api): centralize error response helpers"`
- Record the new commit hash.
- **NEVER** run `git push`.

### 7. Save State & Report
- Update `.loopgoal/state.json` with:
  - `iteration`: incremented count
  - `status`: `"running"`
  - `last_task`: concise summary of the improvement
  - `last_commit`: short git hash
  - `updated_at`: ISO timestamp
- Provide a concise progress update:
  ```text
  Iteration <N>
  → Found: <identified issue>
  → Implemented: <improvement>
  ✓ Verification: passed (<commands>)
  ✓ Committed: <hash> (<commit message>)
  ```

### 8. Repeat
- Check if the overarching goal has been reached or if the maximum iteration limit has been met.
- If more improvements are needed, proceed directly to the next iteration.
- If the goal is reached, mark status as `"goal_reached"`, report final summary, and complete.

---

## Safety Constraints

- **No Remote Push**: Never run `git push`.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: If files were modified before an iteration began, exclude them from staging.
- **Stop Conditions**: Halt immediately if:
  - The user requests `/loopgoal stop`.
  - Max iterations reached (default 20).
  - Verification fails repeatedly and cannot be resolved.
  - Overall goal is reached.
