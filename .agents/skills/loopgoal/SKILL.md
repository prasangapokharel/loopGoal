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
Observe → Audit Rules & Skills → Build Scope Matrix → Refactor File → Verify → Commit → Save State → Next File → Repeat
```

LoopGoal is **100% generic, project-agnostic, and rule-driven**. It works across any language, framework, or architecture (Go, Python, TypeScript, Rust, Java, C++, Monorepos, etc.).

---

## DEEP RULE COMPLIANCE & ZERO-PREMATURE-STOP PROTOCOL

### 1. Rule & Skill Discovery Phase (Iteration 1)
When the user invokes `/loopgoal [goal]`, the agent MUST immediately perform a comprehensive project audit:
1. **Discover all project rules and skill guidelines**:
   - `.agents/rules/*.md` and `.agents/skills/*/SKILL.md`
   - `.cursor/rules/*.mdc` (Cursor rule files)
   - `.opencode/skills/` and `.claude/` / `CLAUDE.md`
   - `AGENTS.md` and `GEMINI.md`
2. **Read and extract all mandatory standards**:
   - Naming conventions (filenames, variable casing, module structures).
   - Architectural patterns (layer separation, error handling, typing/docstrings).
   - Code quality, linter rules, and testing standards.
3. **Discover 100% of Target Files**:
   - Scan every single file inside the target directory or module scope.
   - Do NOT skip any files.
4. **Construct the Full Task Matrix in `.loopgoal/state.json`**:
   Save the goal, full file list, and iteration status:
   ```json
   {
     "goal": "<user_defined_goal>",
     "iteration": 1,
     "status": "running",
     "rules_applied": [
       ".cursor/rules/coding-standard.mdc",
       ".agents/skills/project-style/SKILL.md",
       "AGENTS.md"
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

### 2. Zero-Premature-Stop Guarantee
- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified, and committed.
- **DO NOT END YOUR TURN**: In chat-based agent environments (Antigravity, Claude Code, etc.), do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

---

## Slash Commands & Arguments

1. **`/loopgoal <goal>`**:
   Starts the autonomous loop toward the specified goal.
   - Automatically scans project rules/skills (`.cursor/rules/*.mdc`, `.agents/`, `AGENTS.md`, etc.).
   - Builds the full file list for the target directory.
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

### Step 1: Observe & Audit Target File
- Take the top file from `"remaining_queue"`.
- Audit its filename, imports, functions, type signatures, error handling, and formatting against the project's discovered rules (`.cursor/rules/*.mdc`, `AGENTS.md`, etc.).
- Identify all discrepancies needing refactoring.

### Step 2: Implement Bounded Refactoring
- Refactor **only** the selected file to achieving 100% compliance with project rules and skills.
- Rename files/classes/functions if required by project standards.
- Do NOT touch unrelated files in the same iteration.

### Step 3: Run Verification
- Execute project verification commands (e.g., `pytest`, `ruff check`, `go test ./...`, `npm test`, `golangci-lint`).
- If verification fails:
  - Analyze error output.
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
  - `last_check`: verification output
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
