---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal for Google Antigravity

This skill teaches the Antigravity agent how to execute the LoopGoal autonomous development loop with 100% rule and skill compliance across any project.

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

### 2. Zero-Premature-Stop Guarantee
- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified, and committed.
- **DO NOT END YOUR TURN**: In Antigravity chat, do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

## Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
- `/loopgoal <goal>`: Set/override the goal and begin multi-file rule-compliance iterations.
- `/loopgoal status`: Display current state, iteration count, last commit, remaining queue, and completed files.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

## The Iteration Cycle (File by File)

For EVERY file in `"remaining_queue"`, follow this strict sequence:

1. **Observe & Audit Target File**: Pick top file from `"remaining_queue"`. Audit filename, imports, typing, and formatting against discovered rules (`.cursor/rules/*.mdc`, `AGENTS.md`, etc.).
2. **Implement Refactoring**: Apply minimal, high-quality refactoring strictly matching project rules.
3. **Run Verification**: Execute verification commands (`pytest`, `ruff check`, `go test ./...`, `npm test`, etc.). Fix errors until tests pass.
4. **Review Diff**: Ensure pre-existing uncommitted developer files are untouched.
5. **Stage & Commit**: Stage changed file (`git add <file>`) and commit locally with conventional commit message. Never push.
6. **Update State & Queue**: Remove processed file from `"remaining_queue"`, add to `"completed_files"`, update `.loopgoal/state.json`.
7. **Continuous Progression**: If `"remaining_queue"` is not empty, IMMEDIATELY proceed to Step 1 for the next file without ending your turn.
