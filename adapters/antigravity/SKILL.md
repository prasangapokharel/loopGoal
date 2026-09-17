---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal for Google Antigravity

This skill teaches the Antigravity agent how to execute the LoopGoal autonomous development loop.

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
- In Antigravity chat, **do not stop your turn** after finishing 1 file and wait for user prompts.
- Immediately start the next iteration tool calls (`Iteration 2: Inspecting password.py...`, `Iteration 3: Inspecting urls.py...`) until all files are refactored or an iteration limit is reached.

## Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
- `/loopgoal <goal>`: Set/override the goal and begin multi-file iterations.
- `/loopgoal status`: Display current state, iteration count, last commit, and remaining queue.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

## Workflow Per Iteration

1. **Observe**: Run `git status`, check `remaining_queue`, and select the **next single file**.
2. **Select**: Identify ONE small, valuable, bounded improvement on that file.
3. **Implement**: Make clean, minimal code changes needed.
4. **Verify**: Run verification commands from `.loopgoal/config.yaml` or project defaults (`pytest`, `ruff`, `npm test`, `go test ./...`).
5. **Review Diff**: Ensure only intended files were changed.
6. **Commit**: Stage and commit locally with a conventional commit message. Never push.
7. **Persist State**: Update `.loopgoal/state.json` (remove processed file from `remaining_queue`, keep `status: "running"`).
8. **Repeat**: Immediately continue to the next file in `remaining_queue` until the queue is finished.
