# LoopGoal for Codex & OpenCode

Instructions for running LoopGoal with OpenAI Codex CLI or OpenCode.

## Configuration

Place a `.loopgoal/config.yaml` in the project root:
```yaml
goal: >
  Continuously improve this project with small,
  safe, production-quality changes.
verify:
  - "go test ./..."
limits:
  iterations: 20
```

## Execution Prompt

Invoke Codex with:
```bash
codex "Follow the LoopGoal protocol:
1. Inspect the repository.
2. Select ONE small improvement toward: {goal}
3. Implement and run verification: {verify}
4. Commit only the changed files with a conventional commit message.
5. Record state in .loopgoal/state.json."
```
