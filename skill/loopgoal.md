# LoopGoal Protocol Reference

This document outlines the design principles and best practices for running LoopGoal inside an AI agent environment.

## Key Principles

1. **Protocol Over Implementation**:
   LoopGoal's power lies in the bounded iteration discipline (`Observe → Select → Implement → Verify → Review → Commit → Repeat`). Whether executed directly by an agent following this skill or by an external CLI supervisor, the protocol remains identical.

2. **One Improvement Per Iteration**:
   The most common failure mode in autonomous coding is uncontrolled blast radius. LoopGoal eliminates this by requiring the agent to pick one small, verifiable improvement per iteration.

3. **Verification As Gatekeeper**:
   An iteration is only complete when all configured verification checks pass. If tests or linters fail, the agent enters a corrective sub-loop before committing.

4. **Git Safety**:
   - Commits are atomic and local.
   - Pushes to remote repositories are explicitly prohibited.
   - User work-in-progress is isolated from automated commits.

## Supported Command Invocations

| Command | Action |
| :--- | :--- |
| `/loopgoal` | Runs autonomous loop with goal configured in `.loopgoal/config.yaml` |
| `/loopgoal <goal>` | Runs autonomous loop toward specified goal |
| `/loopgoal status` | Shows current iteration, status, last task, and last commit |
| `/loopgoal stop` | Gracefully terminates autonomous loop |

## State File Format (`.loopgoal/state.json`)

```json
{
  "goal": "Improve API reliability",
  "iteration": 3,
  "status": "running",
  "last_task": "Add validation for transfer edge case",
  "last_commit": "91ac72e",
  "started_at": "2026-09-17T12:00:00Z",
  "updated_at": "2026-09-17T12:05:00Z"
}
```
