# LoopGoal for Claude Code

Add this file to your repository as `CLAUDE.md` or `.claude/commands/loopgoal.md` to enable the `/loopgoal` workflow in Claude Code.

## Custom Command: /loopgoal

When the user runs `/loopgoal [goal]`:

1. **Check Arguments**:
   - If user provided `/loopgoal status`: read `.loopgoal/state.json` and display summary.
   - If user provided `/loopgoal stop`: set status to `"stopped"` in `.loopgoal/state.json` and finish.
   - If user provided `/loopgoal <goal>`: use `<goal>` as the development goal.
   - If user provided `/loopgoal`: read goal from `.loopgoal/config.yaml`.

2. **Execute Autonomous Loop**:
   Loop through bounded iterations:
   - **Observe**: Inspect `git status`. Do not modify pre-existing uncommitted user changes.
   - **Select**: Pick ONE single, bounded, high-value improvement toward the goal.
   - **Implement**: Apply code changes cleanly.
   - **Verify**: Run the project's verification commands (e.g. `npm test`, `go test ./...`).
   - **Diff Review**: Confirm that changes are isolated and minimal.
   - **Commit**: `git add <files>` and `git commit -m "<type>: <summary>"`. Never push.
   - **State**: Write updated iteration and commit hash to `.loopgoal/state.json`.
   - **Repeat**: Proceed to the next bounded improvement until the goal is satisfied.
