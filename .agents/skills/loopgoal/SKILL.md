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
Observe → Select ONE File/Improvement → Implement → Verify → Review Diff → Commit → Save State → Continue to Next File → Repeat
```

---

## CRITICAL MULTI-FILE & CONTINUATION RULES

### 1. Never Stop After Just One File!
When a goal targets a directory, module, or broad objective (e.g., *"Refactor backend auth (backend/api/v1/auth/)"*):
- **DO NOT** stop after completing one file (e.g., `mfa.py`).
- **DO NOT** set `"status": "completed"` in `.loopgoal/state.json` when other files in the target scope remain untouched (`password.py`, `urls.py`, `serializers.py`, `views.py`, `auth.py`, `profile.py`).
- Keep `"status": "running"` in `.loopgoal/state.json`.

### 2. Maintain a Scope Queue
In your first iteration:
1. Scan all files within the goal's scope.
2. List them in your notes and save them in `.loopgoal/state.json` under `"remaining_queue"`:
   ```json
   {
     "goal": "Refactor backend auth",
     "iteration": 1,
     "status": "running",
     "last_task": "Refactored mfa.py to match senior standards",
     "last_commit": "fb2143db",
     "last_check": "passed: pytest && ruff",
     "remaining_queue": [
       "backend/api/v1/auth/password.py",
       "backend/api/v1/auth/urls.py",
       "backend/api/v1/auth/serializers.py",
       "backend/api/v1/auth/views.py",
       "backend/api/v1/auth/profile.py",
       "backend/api/v1/auth/auth.py"
     ]
   }
   ```
3. In each subsequent iteration, pick the **next file** from `"remaining_queue"`.
4. Only when `"remaining_queue"` is empty may you set `"status": "completed"` or `"status": "goal_reached"`.

### 3. Continuous Execution (No Premature Stopping)
- In chat-based agent environments (Antigravity, Claude Code, etc.), **do not stop your turn** after finishing 1 file and wait for user prompts.
- Immediately start the next iteration tool calls (`Iteration 2: Inspecting password.py...`, `Iteration 3: Inspecting urls.py...`) until all files are refactored or an iteration limit is reached.

---

## Slash Commands & Arguments

When the user types `/loopgoal`, determine the mode:

1. **`/loopgoal <goal>`**:
   Starts the autonomous loop toward `<goal>` (e.g., `/loopgoal "refactor backend auth to senior django standard"`).
   - If `.loopgoal/config.yaml` exists, use its verification rules.
   - If `.loopgoal/config.yaml` does not exist, auto-detect the project (e.g., `pytest`, `npm test`, `go test ./...`) and initialize `.loopgoal/`.

2. **`/loopgoal`**:
   Starts or resumes the autonomous loop using the persistent goal defined in `.loopgoal/config.yaml` and `.loopgoal/state.json`.
   - If a remaining queue exists, resume with the next pending file!

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
   Remaining:   <list of files still to be processed>
   ```

4. **`/loopgoal stop`**:
   Gracefully stops the autonomous loop.
   - Updates `.loopgoal/state.json` with `"status": "stopped"`.
   - Halts further iterations.

---

## The Iteration Cycle

In each iteration, you MUST follow this strict sequence:

### 1. Observe & Plan Next File
- Check current Git status: `git status`
- Identify any pre-existing user changes that must NOT be touched or reverted.
- Check the remaining files in the target scope. Pick the **next specific file**.

### 2. Select ONE Bounded Improvement
- Focus strictly on the single selected file/improvement for this iteration.
- Avoid modifying multiple unrelated files in one iteration.

### 3. Implement
- Implement the changes on that single file.
- Follow project standards, clean code, and conventions.

### 4. Verify
- Run configured project verification commands (e.g., `pytest`, `ruff check`, `npm test`, `go test ./...`).
- If verification fails:
  - Analyze the test/linter error output.
  - Fix the issue immediately in the active file.
  - Re-run verification until it passes (up to 3 attempts).
  - NEVER commit changes while verification is failing.

### 5. Review Diff
- Inspect `git diff` and `git status`.
- Ensure only files relevant to the current improvement were touched.
- Ensure pre-existing uncommitted user files are untouched.

### 6. Commit
- Stage only the relevant changed files: `git add <file>`
- Commit with a clear, conventional commit message:
  ```bash
  git commit -m "<type>(<scope>): <concise description>"
  ```
  Example: `git commit -m "refactor(auth): align password.py with senior django standards"`
- Record the new commit hash.
- **NEVER** run `git push`.

### 7. Save State & Update Queue
- Update `.loopgoal/state.json` with:
  - `iteration`: incremented count
  - `status`: `"running"` (unless remaining_queue is empty)
  - `last_task`: concise summary of the improvement
  - `last_commit`: short git hash
  - `last_check`: verification outcome
  - `remaining_queue`: updated list of files left to process
  - `updated_at`: ISO timestamp

### 8. Repeat
- **If files remain in `remaining_queue`**: IMMEDIATELY proceed to the next iteration without waiting for user input!
- **If all files are done**: Mark `"status": "completed"`, summarize all completed iterations, and finish.

---

## Safety Constraints

- **No Remote Push**: Never run `git push`.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: If files were modified before an iteration began, exclude them from staging.
- **Stop Conditions**: Halt immediately if:
  - The user requests `/loopgoal stop`.
  - Max iterations reached (default 20).
  - Verification fails repeatedly and cannot be resolved.
  - Overall goal is reached and all files are completed.
