# LoopGoal for Codex & OpenCode

Instructions for running LoopGoal with OpenAI Codex CLI or OpenCode.

---

## Headless Configuration

Both `codex` and `opencode` require `--dangerously-skip-permissions` (or `--auto-approve`) to run non-interactively:

```yaml
# .loopgoal/config.yaml — for Codex CLI
goal: >
  Continuously improve this project with small,
  safe, production-quality changes.

agent:
  command: "codex"
  args:
    - "--dangerously-skip-permissions"

verify:
  - "go test ./..."
  - "go vet ./..."

limits:
  iterations: 20
  max_retries: 3
```

```yaml
# .loopgoal/config.yaml — for OpenCode
agent:
  command: "opencode"
  args:
    - "--dangerously-skip-permissions"
```

`loopgoal init` writes the correct args automatically when it detects `codex` or `opencode` on your PATH.

---

## Scalable Testing Protocol

Follow the multi-tier structured test hierarchy:
```text
tests/
├── unit/<module_name>/test_<file>.py      # Unit tests (direct imports + mocks)
├── integration/<flow_name>/test_<flow>.py # Integration tests
└── e2e/<feature_name>/test_<feature>.py   # E2E tests
```

- Unit tests must import directly from the target module under test.
- External dependencies (DB, Redis, network) must be isolated with mocks.

---

## Iteration Protocol

For every iteration, the agent must:

1. **Observe** — run `git status`, `git ls-files`, and grep/ripgrep to inspect the repository state empirically.
2. **Discover Rules** — check `AGENTS.md`, `.cursor/rules/*.mdc`, `.agents/rules/*.md`, `.agents/skills/*/SKILL.md`.
3. **Select** — identify ONE small, bounded improvement or unit test toward the goal.
4. **Implement** — apply changes or tests to only the target file.
5. **Verify** — run configured verification commands (`pytest`, `ruff check`, `go test ./...`, etc.). Fix failures before proceeding.
6. **Review Diff** — confirm changes are isolated. Never touch pre-existing uncommitted files.
7. **Commit** — `git add <file>` + `git commit -m "<type>(<scope>): <desc>"`. Never push.
8. **Update State** — write updated `.loopgoal/state.json` with new iteration count, commit hash, and remaining queue.
9. **Continue** — immediately pick the next file from `remaining_queue` without stopping.

---

## Safety Rules

- Never run `git push`
- Never run `git reset --hard` or `git clean -fd`
- Never stage pre-existing uncommitted developer files
- Never commit when any verification command is failing
- Stop only when: `remaining_queue` is empty, max iterations reached, or permanently blocked
