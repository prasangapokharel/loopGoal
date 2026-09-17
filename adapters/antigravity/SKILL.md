---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal for Google Antigravity

This skill teaches the Antigravity agent how to execute the LoopGoal autonomous development loop with empirical shell verification and grep pattern auditing across any project.

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

### 2. Zero-Premature-Stop Guarantee
- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified via shell commands, and committed.
- **DO NOT END YOUR TURN**: In Antigravity chat, do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

## Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
- `/loopgoal <goal>`: Set/override the goal and begin multi-file rule-compliance iterations.
- `/loopgoal status`: Display current state, iteration count, last commit, remaining queue, and completed files.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

## The Iteration Cycle (File by File)

1. **Observe & Audit Target File via Commands & Grep**: Pick top file from `"remaining_queue"`. Audit filename, imports, typing, and formatting using `grep`/`ripgrep` and static analyzers against discovered rules (`.cursor/rules/*.mdc`, `AGENTS.md`, etc.).
2. **Implement Refactoring**: Apply minimal, high-quality refactoring strictly matching project rules.
3. **Run Empirical Verification Commands**: Execute verification commands (`pytest`, `ruff check`, `go test ./...`, `npm test`, etc.). Base success strictly on empirical command output logs. Fix errors until tests pass.
4. **Review Diff**: Ensure pre-existing uncommitted developer files are untouched.
5. **Stage & Commit**: Stage changed file (`git add <file>`) and commit locally with conventional commit message. Never push.
6. **Update State & Queue**: Remove processed file from `"remaining_queue"`, add to `"completed_files"`, update `.loopgoal/state.json`.
7. **Continuous Progression**: If `"remaining_queue"` is not empty, IMMEDIATELY proceed to Step 1 for the next file without ending your turn.
