---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal for Google Antigravity

This skill teaches the Antigravity agent how to execute the LoopGoal autonomous development loop across any codebase.

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
- In Antigravity chat, do NOT end your response after finishing a single bounded task and wait for the user to tell you to continue.
- Immediately proceed to the next item:
  - Iteration 1: Process first bounded item → Verify → Commit.
  - Iteration 2: Process next item from queue → Verify → Commit.
  - Iteration 3: Process next item from queue → Verify → Commit.
  - Continue until all items in scope are completed or the configured iteration limit is reached.

## Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
- `/loopgoal <goal>`: Set/override the goal and begin multi-item iterations.
- `/loopgoal status`: Display current state, iteration count, last commit, and remaining queue.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

## Workflow Per Iteration

1. **Observe**: Run `git status`, check `"remaining_queue"`, and select the **next single bounded item**.
2. **Select**: Identify ONE small, valuable, bounded improvement on that item.
3. **Implement**: Make clean, minimal code changes needed.
4. **Verify**: Run verification commands from `.loopgoal/config.yaml` or project defaults.
5. **Review Diff**: Ensure only intended files were changed.
6. **Commit**: Stage and commit locally with a conventional commit message. Never push.
7. **Persist State**: Update `.loopgoal/state.json` (remove processed item from `remaining_queue`, keep `status: "running"`).
8. **Repeat**: Immediately continue to the next item in `remaining_queue` until the queue is finished.
