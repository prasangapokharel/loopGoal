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
Observe → Select ONE Item → Implement → Verify → Review Diff → Commit → Save State → Continue to Next Item → Repeat
```

LoopGoal is **100% generic and project-agnostic**. It works identically across any language, framework, or architecture (Go, Python, TypeScript, Rust, Java, C++, Monorepos, etc.).

---

## Dynamic Multi-Item & Continuous Execution Protocol

### 1. Dynamic Scope Discovery
When a user provides a goal targeting a module, directory, feature, or broad objective:
1. **Dynamically inspect the repository**: Use file listing, directory inspection, or git status to discover the relevant candidate files, functions, or tasks.
2. **Formulate a dynamic task queue**: Keep track of the remaining items that need attention to fully achieve the goal.
3. Save or update the remaining items in `.loopgoal/state.json` under `"remaining_queue"`:
   ```json
   {
     "goal": "<user_defined_goal>",
     "iteration": 1,
     "status": "running",
     "last_task": "<concise_summary_of_improvement>",
     "last_commit": "<git_commit_hash>",
     "last_check": "passed: <configured_verification_command>",
     "remaining_queue": [
       "<path/to/pending_item_2>",
       "<path/to/pending_item_3>",
       "<path/to/pending_item_4>"
     ]
   }
   ```

### 2. Never Stop Prematurely
- **DO NOT** stop after completing only one file or subtask if more items in the goal scope remain pending.
- **DO NOT** set `"status": "completed"` or `"status": "goal_reached"` until all items required by the goal are processed and verified.
- Keep `"status": "running"` in `.loopgoal/state.json` while items remain in `"remaining_queue"`.

### 3. Continuous Autonomous Progression
- In conversational / chat-based agents, do NOT end your response after finishing a single bounded task and wait for the user to tell you to continue.
- Immediately proceed to the next item:
  - Iteration 1: Process first bounded item → Verify → Commit.
  - Iteration 2: Process next item from queue → Verify → Commit.
  - Iteration 3: Process next item from queue → Verify → Commit.
  - Continue until all items in scope are completed or the configured iteration limit is reached.

---

## Slash Commands & Arguments

When the user types `/loopgoal`, determine the mode:

1. **`/loopgoal <goal>`**:
   Starts the autonomous loop toward the specified `<goal>`.
   - If `.loopgoal/config.yaml` exists, use its configured verification commands.
   - If `.loopgoal/config.yaml` does not exist, dynamically detect project test commands (e.g., via `package.json`, `go.mod`, `Cargo.toml`, `pyproject.toml`, Makefile) and initialize `.loopgoal/`.

2. **`/loopgoal`**:
   Starts or resumes the autonomous loop using the persistent goal in `.loopgoal/config.yaml` and `.loopgoal/state.json`.
   - If items remain in `"remaining_queue"`, resume with the next pending item!

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
   Remaining:   <list of items still in queue>
   ```

4. **`/loopgoal stop`**:
   Gracefully stops the autonomous loop.
   - Updates `.loopgoal/state.json` with `"status": "stopped"`.
   - Halts further iterations safely.

---

## The Iteration Cycle

In each iteration, you MUST follow this strict sequence:

### 1. Observe & Select Next Item
- Check current Git status: `git status`.
- Ensure pre-existing uncommitted user files are never touched or staged.
- Check `"remaining_queue"` and select the **next single bounded item**.

### 2. Implement
- Implement **only** the selected item/improvement.
- Preserve existing project architecture and conventions.
- Do not make unnecessary changes in unrelated files.

### 3. Verify
- Run the configured verification commands (e.g., from `.loopgoal/config.yaml` or detected project tests).
- If verification fails:
  - Analyze the error output.
  - Fix the issue immediately in the active item.
  - Re-run verification until it passes (up to 3 attempts).
  - If it cannot be fixed cleanly, mark status as `blocked`.
  - **NEVER** commit changes while verification is failing.

### 4. Review Diff
- Inspect `git diff` and `git status`.
- Ensure only files relevant to the current bounded improvement were touched.
- Ensure pre-existing uncommitted user files are untouched.

### 5. Commit
- Stage only the relevant changed files: `git add <file...>`
- Commit with a clear, conventional commit message:
  ```bash
  git commit -m "<type>(<scope>): <concise description>"
  ```
- Record the new commit hash.
- **NEVER** run `git push`.

### 6. Save State & Update Queue
- Update `.loopgoal/state.json`:
  - `iteration`: incremented count
  - `status`: `"running"` (unless queue is empty)
  - `last_task`: concise summary of what was done
  - `last_commit`: short git hash
  - `last_check`: verification outcome
  - `remaining_queue`: updated list of items left to process
  - `updated_at`: ISO timestamp

### 7. Repeat
- **If items remain in `remaining_queue`**: IMMEDIATELY proceed to the next iteration without waiting for user input!
- **If all items are done**: Set `"status": "completed"`, summarize all completed iterations, and complete.

---

## Safety Constraints

- **No Remote Push**: Never run `git push`. LoopGoal operates exclusively on local Git.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: Always identify pre-existing dirty files and exclude them from staging.
- **Stop Conditions**: Halt immediately if:
  - The user requests `/loopgoal stop`.
  - Max iterations reached.
  - Verification fails repeatedly and cannot be resolved.
  - Overall goal is reached and all scoped items are completed.
