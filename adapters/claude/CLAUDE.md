# LoopGoal for Claude Code

Add this file to your repository as `CLAUDE.md` or `.claude/commands/loopgoal.md` to enable the `/loopgoal` autonomous loop workflow in Claude Code.

---

## Headless Configuration

When running `loopgoal run` with `claude` as the executor, pass `--dangerously-skip-permissions` so Claude operates without interactive permission prompts:

```yaml
# .loopgoal/config.yaml
goal: >
  Refactor the codebase and enforce coding standards.

agent:
  command: "claude"
  args:
    - "--dangerously-skip-permissions"

verify:
  - "go test ./..."

limits:
  iterations: 20
  max_retries: 3
```

`loopgoal init` writes this automatically when it detects `claude` on your PATH.

---

## Custom Command: /loopgoal

When the user runs `/loopgoal [goal]`:

1. **Check Arguments**:
   - `/loopgoal status`: read `.loopgoal/state.json` / `taskmap.json` and display summary.
   - `/loopgoal stop`: set status to `"stopped"` in `.loopgoal/state.json` and finish.
   - `/loopgoal <goal>`: set/override development goal.
   - `/loopgoal`: read goal from `.loopgoal/config.yaml`.

2. **Empirical Inventory & Discovery (Iteration 1 only)**:
   - Run `git ls-files` to get the complete list of tracked files without guessing.
   - Discover rule files: `AGENTS.md`, `GEMINI.md`, `.cursor/rules/*.mdc`, `.agents/rules/*.md`, `.agents/skills/*/SKILL.md`.
   - Run grep/ripgrep patterns to identify non-compliant code.
   - Build `remaining_queue` in `.loopgoal/state.json` with the full file list to process.

3. **Scalable Standardized Testing Protocol**:
   When writing or updating tests, follow the standardized hierarchy:
   ```text
   tests/
   ├── unit/<module_name>/test_<file>.py      # Unit tests (direct imports + mocks)
   ├── integration/<flow_name>/test_<flow>.py # Integration tests
   └── e2e/<feature_name>/test_<feature>.py   # E2E API tests
   ```
   - Unit tests import directly from target modules and mock external DB/Redis/network calls.
   - Exclude test caches (`.pytest_cache/`, `htmlcov/`, `coverage/`) in `.gitignore`.

4. **Execute Autonomous Loop** — for each file in `remaining_queue`:
   - **Observe**: Inspect `git status`. Never modify pre-existing uncommitted user changes.
   - **Select**: Pick ONE single, bounded, high-value improvement toward the goal.
   - **Implement / Test**: Apply code or test changes cleanly, following all discovered rules.
   - **Verify**: Run the project's verification commands (e.g. `pytest`, `npm test`, `go test ./...`, `ruff check`). Fix failures before proceeding.
   - **Diff Review**: Confirm that changes are isolated to the target file only.
   - **Commit**: `git add <file>` and `git commit -m "<type>(<scope>): <summary>"`. Never push.
   - **State**: Write updated iteration, commit hash, remaining queue to `.loopgoal/state.json`.
   - **Repeat**: Immediately proceed to the next file without waiting for user input.

5. **Stop Conditions & Evidence Gate**:
   - `remaining_queue` is empty & all tests pass → set `"status": "goal_reached"`.
   - User sends `/loopgoal stop` → set `"status": "stopped"`.
   - Max iterations reached → set `"status": "completed"`.
   - Verification fails after max retries → set `"status": "blocked"`.

---

## Safety Rules

- Never run `git push`
- Never run `git reset --hard` or `git clean -fd`
- Never stage pre-existing uncommitted files
- Never commit when verification is failing
