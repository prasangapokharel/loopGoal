---
name: loopgoal
description: >-
  Autonomous development loop protocol. Run with /loopgoal [goal], /loopgoal status,
  or /loopgoal stop to guide the agent through continuous, verifiable, single-improvement
  iterations (Observe → Select → Implement → Verify → Commit → Repeat).
---

# LoopGoal for Google Antigravity

This skill teaches the Antigravity agent how to execute the LoopGoal autonomous development loop.

## Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml`.
- `/loopgoal <goal>`: Set/override the goal and begin iterations.
- `/loopgoal status`: Display the current state, iteration count, and last commit.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

## Workflow

1. **Observe**: Run `git status` to check the current repository state and note any existing user changes.
2. **Select**: Identify ONE small, valuable, bounded improvement toward the goal.
3. **Implement**: Make only the minimal code changes needed.
4. **Verify**: Run verification commands from `.loopgoal/config.yaml` or project defaults (`go test ./...`, `npm test`, `pytest`, etc.).
5. **Review Diff**: Ensure only intended files were changed.
6. **Commit**: Stage and commit locally with a conventional commit message. Never push.
7. **Persist State**: Update `.loopgoal/state.json`.
8. **Repeat**: Continue to next iteration until goal reached or limit reached.
