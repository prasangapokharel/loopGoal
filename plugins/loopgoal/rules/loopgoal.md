# LoopGoal Autonomous Development Protocol Rule

This workspace is governed by **LoopGoal**, an autonomous, verifiable development supervisor for AI coding agents.

Whenever the user invokes `/loopgoal [goal]`, references LoopGoal, or assigns an autonomous continuous task:

## 1. Zero-Premature-Stop Principle
- Processing one file or one bug is **ONE iteration**.
- **DO NOT stop after 1 file**: The agent must NEVER set status to `completed` or `goal_reached` until 100% of target files or items in the task queue are fully processed.
- **DO NOT END YOUR TURN PREMATURELY**: Keep `"status": "running"` in `.loopgoal/state.json` and immediately advance to the next file or task item in the same turn.

## 2. The 7-Step Iteration Cycle
For every single task item or target file:
1. **Observe & Audit**: Inspect the file, imports, types, and applicable rules (`.cursor/rules/*.mdc`, `AGENTS.md`, `.agents/rules/*.md`).
2. **Implement Bounded Work**: Make changes ONLY for this specific item. Never modify unrelated files.
3. **Run Empirical Verification**: Run shell verification commands (`npm test`, `pytest`, `go test ./...`, `eslint`, `tsc`). Verification MUST succeed with exit code 0.
4. **Self-Correction on Failure**: If verification fails, read the log, fix the code immediately, and re-verify. Never leave broken code.
5. **Inspect Git Diff**: Run `git diff` and ensure only the intended file was modified. Pre-existing uncommitted work must remain untouched.
6. **Local Stage & Commit**: Run `git add <file>` and commit with a concise conventional commit message.
7. **Advance Queue**: Update `.loopgoal/state.json` and proceed immediately to the next pending item.

## 3. Safety Guarantees
- **NEVER run `git push`**: All commits are strictly local unless explicitly commanded by the user.
- **NEVER run destructive commands**: Do not run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: Isolate changes and do not overwrite the developer's dirty working tree.
