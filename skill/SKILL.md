---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal: Autonomous Development Loop Protocol

LoopGoal is a local-first autonomous development supervisor for AI coding agents. It keeps the agent working toward a user-defined goal through continuous, small, verifiable iterations.

The fundamental loop is:
```text
Observe & Command Audit → Build Scope Matrix → Refactor File → Empirical Shell Verification → Review Diff → Commit → Next File → Repeat
```

LoopGoal is **100% generic, project-agnostic, and command-driven**. It works across any language, framework, or architecture (Go, Python, TypeScript, Rust, Java, C++, Monorepos, etc.).

---

## EMPIRICAL COMMAND AUDITING & DEEP GREP PROTOCOL

### 1. Empirical Shell & Grep File Discovery (Iteration 1)
When the user invokes `/loopgoal [goal]`, the agent MUST NOT guess or assume file lists. It MUST use shell commands and search tools to establish 100% complete empirical baselines:
1. **Discover all target files empirically**:
   - Run shell commands or directory listings (`git ls-files <target_dir>`, `find <target_dir> -type f`, etc.).
   - Ensure zero files are missed in the target scope.
2. **Audit Rule & Skill Standards**:
   - Discover `.cursor/rules/*.mdc`, `.agents/rules/*.md`, `.agents/skills/*/SKILL.md`, `AGENTS.md`, `GEMINI.md`, etc.
3. **Run Deep Grep & Static Analysis Audits**:
   - Use `grep` / `ripgrep` to search for non-compliant patterns across all files:
     - Non-compliant variable / function casing (e.g. camelCase vs snake_case).
     - Deprecated API usage or missing type annotations.
     - Unhandled error cases or missing logging/docstrings.
   - Run configured linters and checkers (e.g. `ruff check`, `mypy`, `golangci-lint`, `eslint`, `tsc --noEmit`, `cargo clippy`).
4. **Construct the Task Matrix in `.loopgoal/state.json`**:
   Save the full file queue, rules applied, and baseline check status:
   ```json
   {
     "goal": "<user_defined_goal>",
     "iteration": 1,
     "status": "running",
     "rules_applied": [
       ".cursor/rules/*.mdc",
       ".agents/rules/*.md"
     ],
     "remaining_queue": [
       "path/to/file_1.ext",
       "path/to/file_2.ext",
       "path/to/file_3.ext",
       "path/to/file_N.ext"
     ],
     "completed_files": []
   }
   ```

---

### 2. Zero-Premature-Stop & Continuous Execution
- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified via shell commands, and committed.
- **DO NOT END YOUR TURN**: In chat-based agent environments (Antigravity, Claude Code, etc.), do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

---

## Slash Commands & Arguments

1. **`/loopgoal <goal>`**:
   Starts the autonomous loop toward the specified goal.
   - Executes empirical discovery (`git ls-files`, `find`, `grep`).
   - Audits project rules/skills (`.cursor/rules/*.mdc`, `.agents/`, `AGENTS.md`, etc.).
   - Configures verification commands (`pytest`, `ruff check`, `go test ./...`, `npm test`, etc.).

2. **`/loopgoal`**:
   Starts or resumes the autonomous loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
   - If items remain in `"remaining_queue"`, resumes directly with the next pending file!

3. **`/loopgoal status`**:
   Displays current status, iteration count, last commit, verification status, and remaining queue:
   ```text
   LoopGoal Status
   ────────────────────────────
   Status:      <running | completed | blocked | stopped | goal_reached>
   Iteration:   <current_iteration>
   Goal:        <goal>
   Last task:   <summary_of_last_change>
   Last commit: <git_commit_hash>
   Last check:  <passed | failed>
   Completed:   <count> files
   Remaining:   <list of files left in queue>
   ```

4. **`/loopgoal stop`**:
   Gracefully halts the autonomous loop. Writes `"status": "stopped"` to `.loopgoal/state.json`.

---

## The Iteration Cycle (File by File)

For EVERY file in `"remaining_queue"`, follow this strict 7-step sequence:

### Step 1: Observe & Audit Target File via Commands & Grep
- Take the top file from `"remaining_queue"`.
- Use `grep` / `ripgrep` or code search to inspect its functions, imports, type signatures, error handling, and formatting against discovered rules (`.cursor/rules/*.mdc`, `AGENTS.md`, etc.).
- Identify all non-compliant lines needing refactoring.

### Step 2: Implement Bounded Refactoring
- Refactor **only** the selected file to achieve 100% compliance with project rules and skills.
- Rename files/classes/functions if required by project standards.
- Do NOT touch unrelated files in the same iteration.

### Step 3: Run Empirical Verification Commands
- Execute project verification shell commands (e.g., `pytest`, `ruff check`, `go test ./...`, `npm test`, `tsc --noEmit`, `golangci-lint`).
- Base success strictly on empirical command logs and zero-exit codes.
- If verification fails:
  - Analyze exact error log output.
  - Fix issues in the file immediately.
  - Re-run verification until it passes (up to 3 attempts).
  - NEVER commit changes when verification is failing.

### Step 4: Review Diff
- Run `git diff` and `git status`.
- Ensure pre-existing uncommitted developer work is untouched.
- Ensure diff is minimal, clean, and isolated to the target file.

### Step 5: Stage & Commit
- Stage the changed file: `git add <file>`
- Commit with a clear conventional commit message:
  ```bash
  git commit -m "<type>(<scope>): <concise description matching rules>"
  ```
- **NEVER** run `git push`.

### Step 6: Update State & Queue
- Remove the processed file from `"remaining_queue"` and append it to `"completed_files"`.
- Update `.loopgoal/state.json`:
  - `iteration`: incremented count
  - `status`: `"running"` (unless `"remaining_queue"` is empty)
  - `last_task`: summary of change
  - `last_commit`: short git hash
  - `last_check`: empirical verification output log summary
  - `remaining_queue`: updated list of pending files
  - `completed_files`: list of completed files

### Step 7: Continuous Progression
- **If `"remaining_queue"` is not empty**: IMMEDIATELY proceed to Step 1 for the next file without waiting for user input.
- **If `"remaining_queue"` is empty**: Set `"status": "goal_reached"`, print final summary of all refactored files, and finish.

---

## Safety Constraints

- **No Remote Push**: Never run `git push`. LoopGoal operates exclusively on local Git.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: Always identify pre-existing dirty files and exclude them from staging.
- **Stop Conditions**: Halt immediately if:
  - The user requests `/loopgoal stop`.
  - Max iterations reached.
  - Verification fails repeatedly and cannot be resolved.
  - All files in the target scope match 100% of project rules.
