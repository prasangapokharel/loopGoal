# LoopGoal for Google Antigravity (agy)

Install this file as `.agents/skills/loopgoal/SKILL.md` in your project or use the global skill in `~/.gemini/config/skills/loopgoal/SKILL.md`.

---

## Headless Configuration

When running `loopgoal run` with `agy` as the executor, you **must** pass `--dangerously-skip-permissions` so the agent can operate without interactive prompts:

```yaml
# .loopgoal/config.yaml
goal: >
  Refactor backend auth and enforce coding standards.

agent:
  command: "agy"
  args:
    - "--dangerously-skip-permissions"

verify:
  - "go test ./..."
  - "go vet ./..."

limits:
  iterations: 20
  max_retries: 3
```

`loopgoal init` writes this automatically when it detects `agy` on your PATH.

---

## 1. EMPIRICAL COMMAND AUDITING & INVENTORY PROTOCOL

### Empirical File Discovery (Iteration 1)
When the user invokes `/loopgoal [goal]`, the agent MUST NOT guess or assume file lists. It MUST use shell commands and search tools to establish 100% complete empirical baselines:
1. **Discover all target files empirically**:
   - Run shell commands (`git ls-files <target_dir>`, `find <target_dir> -type f`, etc.).
   - Ensure zero files are missed in the target scope.
2. **Audit Rule & Skill Standards**:
   - Discover `.cursor/rules/*.mdc`, `.agents/rules/*.md`, `.agents/skills/*/SKILL.md`, `AGENTS.md`, `GEMINI.md`, etc.
3. **Run Deep Grep & Static Analysis Audits**:
   - Use `grep` / `ripgrep` to search for non-compliant patterns across all files (casing, error handling, clean imports, architecture).
4. **Construct the Task Matrix in `.loopgoal/state.json`**:
   Save the full file queue, rules applied, and baseline check status:
   ```json
   {
     "goal": "<user_defined_goal>",
     "iteration": 1,
     "status": "running",
     "rules_applied": [
       ".cursor/rules/*.mdc",
       ".agents/rules/*.md"
     ],
     "remaining_queue": [
       "path/to/file_1.ext",
       "path/to/file_2.ext",
       "path/to/file_N.ext"
     ],
     "completed_files": []
   }
   ```

---

## 2. SCALABLE STANDARDIZED TESTING PROTOCOL

When the goal involves writing, increasing coverage, or refactoring tests, always enforce a scalable, structured testing hierarchy:

### Standardized Test Directory Structure
```text
tests/
├── unit/
│   └── <module_or_service_name>/
│       └── test_<file_name>.py    # or <file_name>_test.go, <file_name>.test.ts
├── integration/
│   └── <service_or_flow_name>/
│       └── test_<flow_name>.py
└── e2e/
    └── <feature_name>/
        └── test_<feature_name>.py
```

### Testing Rules & Isolation Principles:
1. **True Unit Isolation (`tests/unit/<module>/`)**:
   - **Direct Module Imports**: Unit tests must import directly from the target module under test (e.g., `from backend.api.v1.levrage.views import LeverageViewSet`).
   - **Dependency Isolation**: Mock external services, I/O, database transactions, Redis cache, and network calls so unit tests execute in milliseconds and never fail from external state.
   - **Comprehensive Branch Coverage**: Test standard success paths, invalid inputs, edge cases, permission denials, and error handling.
2. **Integration Testing (`tests/integration/<module>/`)**:
   - Test interaction between controllers, services, repositories, and persistence layers.
3. **End-to-End Testing (`tests/e2e/<feature>/`)**:
   - Test full API request/response lifecycles, HTTP status codes, and serialized output schemas.
4. **Test Command Compatibility (`tests/command.yaml` / Manifests)**:
   - Respect project test configs (`pytest.ini`, `pyproject.toml`, `tests/command.yaml`, `go.mod`, `package.json`).
5. **Git Diff Hygiene & Cache Exclusion**:
   - Ensure test cache artifacts (`.pytest_cache/`, `.coverage`, `coverage/`, `htmlcov/`, `.nyc_output/`) are listed in `.gitignore` so test runs never pollute Git diffs or commits.

---

## 3. Zero-Premature-Stop & Continuous Execution
- **DO NOT STOP after 1 file**: Processing one file is ONE iteration. You must NOT set `"status": "completed"` or `"status": "goal_reached"` until **100% of target files** in `"remaining_queue"` are audited, refactored, verified via shell commands, and committed.
- **DO NOT END YOUR TURN**: In Antigravity chat, do NOT stop tool execution after a single file to ask the user. Immediately call tools on the next file (`Iteration 2: Inspecting path/to/file_2.ext...`) in the same turn!
- Keep `"status": "running"` in `.loopgoal/state.json` until `"remaining_queue"` is completely empty (`[]`).

---

## 4. Slash Commands

- `/loopgoal`: Start or resume the loop using `.loopgoal/config.yaml` and `.loopgoal/state.json`.
- `/loopgoal <goal>`: Set/override the goal and begin multi-file rule-compliance iterations.
- `/loopgoal status`: Display current state, iteration count, last commit, remaining queue, and completed files.
- `/loopgoal stop`: Gracefully halt the autonomous loop.

---

## 5. The Iteration Cycle (File by File)

1. **Observe & Audit Target File via Commands & Grep**: Pick top file from `"remaining_queue"`. Audit filename, imports, typing, and formatting using `grep`/`ripgrep` and static analyzers against discovered rules (`.cursor/rules/*.mdc`, `AGENTS.md`, etc.).
2. **Implement Refactoring / Tests**: Apply minimal, high-quality refactoring or unit tests strictly matching project rules.
3. **Run Empirical Verification Commands**: Execute verification commands (`pytest`, `ruff check`, `go test ./...`, `npm test`, etc.). Base success strictly on empirical command output logs. Fix errors until tests pass.
4. **Review Diff**: Ensure pre-existing uncommitted developer files are untouched.
5. **Stage & Commit**: Stage changed file (`git add <file>`) and commit locally with conventional commit message. Never push.
6. **Update State & Queue**: Remove processed file from `"remaining_queue"`, add to `"completed_files"`, update `.loopgoal/state.json` and `.loopgoal/taskmap.json`.
7. **Continuous Progression**: If `"remaining_queue"` is not empty, IMMEDIATELY proceed to Step 1 for the next file without ending your turn.

---

## 6. Safety Constraints

- **No Remote Push**: Never run `git push`.
- **No Destructive Commands**: Never run `git reset --hard` or `git clean -fd`.
- **Preserve User Changes**: Always identify pre-existing dirty files and exclude them from staging.
- **Stop Conditions**: Halt if `/loopgoal stop` is issued, max iterations reached, or verification fails repeatedly.
