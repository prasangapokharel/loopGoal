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
Observe & Command Audit → Build Scope Matrix → Refactor / Write Tests → Empirical Verification → Review Diff → Commit → Next Target → Repeat
```

LoopGoal is **100% generic, project-agnostic, and command-driven**. It works across any language, framework, or architecture (Go, Python, TypeScript, Rust, Java, C++, Monorepos, etc.).

---

## 1. EMPIRICAL COMMAND AUDITING & INVENTORY PROTOCOL

### Empirical File Discovery (Iteration 1)
When the user invokes `/loopgoal [goal]`, the agent MUST NOT guess or assume file lists. It MUST use shell commands and search tools to establish 100% complete empirical baselines:
1. **Discover all target files empirically**:
   - Run shell commands (`git ls-files <target_dir>`, `find <target_dir> -type f`, etc.).
   - Ensure zero files are missed in the target scope.
2. **Audit Rule & Skill Standards**:
   - Discover `.cursor/rules/*.mdc`, `.agents/rules/*.md`, `.agents/skills/*/SKILL.md`, `AGENTS.md`, `GEMINI.md`, etc.
3. **Run Deep Grep & Static Analysis Audits**:
   - Use `grep` / `ripgrep` to search for non-compliant patterns across all files (casing, error handling, clean imports, architecture).
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
       "path/to/file_N.ext"
     ],
     "completed_files": []
   }
   ```

---

## 2. SCALABLE STANDARDIZED TESTING PROTOCOL

When the goal involves writing, increasing coverage, or refactoring tests, always enforce a scalable, structured testing hierarchy:

### Standardized Test Directory Structure
```text
tests/
├── unit/
│   └── <module_or_service_name>/
│       └── test_<file_name>.py    # or <file_name>_test.go, <file_name>.test.ts
├── integration/
│   └── <service_or_flow_name>/
│       └── test_<flow_name>.py
└── e2e/
    └── <feature_name>/
        └── test_<feature_name>.py
```

### Testing Rules & Isolation Principles:
1. **True Unit Isolation (`tests/unit/<module>/`)**:
   - **Direct Module Imports**: Unit tests must import directly from the target module under test (e.g., `from backend.api.v1.levrage.views import LeverageViewSet`, `from backend.services.user import UserService`).
   - **Dependency Isolation**: Mock external services, I/O, database transactions, Redis cache, and network calls so unit tests execute in milliseconds and never fail from external state.
   - **Comprehensive Branch Coverage**: Test standard success paths, invalid inputs, edge cases, permission denials, and error handling.
2. **Integration Testing (`tests/integration/<module>/`)**:
   - Test interaction between controllers, services, repositories, and persistence layers.
3. **End-to-End Testing (`tests/e2e/<feature>/`)**:
   - Test full API request/response lifecycles, HTTP status codes, and serialized output schemas.
4. **Test Command Compatibility (`tests/command.yaml` / Manifests)**:
   - Respect project test configs (`pytest.ini`, `pyproject.toml`, `tests/command.yaml`, `go.mod`, `package.json`).
5. **Git Diff Hygiene & Cache Exclusion**:
   - Ensure test cache artifacts (`.pytest_cache/`, `.coverage`, `coverage/`, `htmlcov/`, `.nyc_output/`) are listed in `.gitignore` so test runs never pollute Git diffs or commits.

---

## 3. ZERO-PREMATURE-STOP & CONTINUOUS EXECUTION

- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified via shell commands, and committed.
- **DO NOT END YOUR TURN**: In chat-based agent environments (Antigravity, Claude Code, etc.), do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

---

## 4. Slash Commands & Arguments

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

## 5. SUB-SECOND VERIFICATION VIA LIVEFEED DAEMON (.loopgoal/livefeed.json)

LoopGoal includes an auto-detecting, polyglot background daemon (`loopgoal-daemon.mjs` / `loopgoal daemon`) that eliminates compiler cold-boot delays and prevents context bloat:

### How It Works
1. **0ms Sub-Second Agent Reads**:
   - The daemon runs continuously in the background and writes an atomic `.loopgoal/livefeed.json`.
   - The AI agent reads `.loopgoal/livefeed.json` directly from disk in **under 2ms**, eliminating 10–30s idle command waits.
2. **Context Window Optimization (Condensed 5-Line Errors)**:
   - Compilers and linters often dump hundreds of lines of noise and ANSI colors.
   - The livefeed condenses failures into clean 5-line JSON error objects:
     ```json
     {
       "source": "tsc",
       "file": "src/controllers/auth.ts",
       "line": 48,
       "col": 12,
       "code": "TS2339",
       "message": "Property 'organizationId' does not exist on type 'SessionUser'."
     }
     ```
   - Only the top 8 errors are preserved to protect context tokens for reasoning.
3. **Deterministic Hard Lock**:
   - The commit token is tied to `canCommit: true` / `status: "passing"`.
   - When passing, the daemon writes `.loopgoal/verified.token`, instantly unlocking git commit.
   - When failing, the token is automatically revoked to prevent accidental commits.
4. **Stale Feed Protection**:
   - Each cycle increments `heartbeat` and updates `updatedAt` (ISO timestamp).
   - If `livefeed.json` is older than 10 seconds or absent, the agent falls back to running `loopgoal verify`.

---

## 6. The Iteration Cycle (File by File)

For EVERY file in `"remaining_queue"`, follow this strict 7-step sequence:

### Step 1: Observe & Audit Target File
- Take the top file from `"remaining_queue"`.
- Lock target: run `loopgoal select <file>` to enforce the single-target barrier.
- Use code search or `grep` to inspect its functions, imports, type signatures, and standards against discovered rules.

### Step 2: Implement Bounded Refactoring / Tests
- Refactor or write unit tests for **only** the selected file to achieve 100% compliance.
- Do NOT touch unrelated files in the same iteration (out-of-scope edits are blocked).

### Step 3: Run Zero-Trust Empirical Verification
- **Fast Path**: Check `.loopgoal/livefeed.json`. If fresh and `canCommit === true`, verification is already complete and token is unlocked!
- **Standard Path**: If livefeed is failing or inactive, inspect condensed errors or run `loopgoal verify` to produce `.loopgoal/verified.token`.
- Base success strictly on empirical command logs and zero-exit codes.
- If verification fails:
  - Analyze exact error log output.
  - Fix issues in the file immediately (or run `loopgoal rollback` if spiraling).
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
- Note: Pre-commit hook enforces that `.loopgoal/verified.token` exists; commits without passing verification will be blocked with exit code 1.
- **NEVER** run `git push`.

### Step 6: Update State & Queue
- Remove the processed file from `"remaining_queue"` and append it to `"completed_files"`.
- Update `.loopgoal/state.json` and `.loopgoal/taskmap.json`.

### Step 7: Continuous Progression
- **If `"remaining_queue"` is not empty**: IMMEDIATELY proceed to Step 1 for the next file without waiting for user input.
- **If `"remaining_queue"` is empty**: Set `"status": "goal_reached"`, print final summary of all refactored files, and finish.

---

## 7. Safety Constraints

- **No Remote Push**: Never run `git push`. LoopGoal operates exclusively on local Git.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: Always identify pre-existing dirty files and exclude them from staging.
- **Stop Conditions**: Halt immediately if:
  - The user requests `/loopgoal stop`.
  - Max iterations reached.
  - Verification fails repeatedly and cannot be resolved.
  - All files in the target scope match 100% of project rules.

