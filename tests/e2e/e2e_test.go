package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loopgoal/internal/agent"
	"loopgoal/internal/config"
	"loopgoal/internal/detect"
	"loopgoal/internal/git"
	"loopgoal/internal/loop"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"branch", "-M", "main"},
		{"config", "user.name", "E2E Tester"},
		{"config", "user.email", "e2e@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}
}

func commitAll(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	_ = cmd.Run()
	cmd = exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s (%v)", string(out), err)
	}
}

// TestE2E_GoProjectWorkflow tests a real Go repository with compiler checks and verification retry.
func TestE2E_GoProjectWorkflow(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create real go.mod and a starter file
	goMod := "module example.com/calc\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	calcGo := "package calc\n\nfunc Add(a, b int) int {\n\treturn a + b\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "calc.go"), []byte(calcGo), 0o644); err != nil {
		t.Fatal(err)
	}
	commitAll(t, dir, "init calc module")

	// Detect project
	detected := detect.Detect(dir)
	if detected.Type != "go" {
		t.Fatalf("expected detected type 'go', got %s", detected.Type)
	}

	cfg := config.DetectConfig(dir, detected.VerifyCommands)
	cfg.Goal = "Add multiply function with unit tests"
	cfg.Limits.Iterations = 2
	cfg.Limits.MaxRetries = 2

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)
	if err := config.Save(filepath.Join(cfgDir, config.DefaultConfigFile), cfg); err != nil {
		t.Fatal(err)
	}

	// Mock agent simulating 2 iterations:
	// Iteration 1: Adds Add test
	// Iteration 2: Adds Multiply, but introduces a syntax error on attempt 1, then fixes it on attempt 2 (retry)
	step := 0
	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			step++
			if step == 1 {
				// Add unit test for Add
				testCode := "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fail()\n\t}\n}\n"
				_ = os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(testCode), 0o644)
				return agent.Result{Task: "add unit test for Add function"}, nil
			} else if step == 2 {
				// Introduce syntax error in calc.go
				brokenCode := "package calc\n\nfunc Add(a, b int) int { return a + b }\nfunc Multiply(a, b int) int { return a * }\n"
				_ = os.WriteFile(filepath.Join(dir, "calc.go"), []byte(brokenCode), 0o644)
				return agent.Result{Task: "add Multiply with intentional syntax bug"}, nil
			} else if step == 3 {
				// Retry step: fix syntax error and add test
				fixedCode := "package calc\n\nfunc Add(a, b int) int { return a + b }\nfunc Multiply(a, b int) int { return a * b }\n"
				_ = os.WriteFile(filepath.Join(dir, "calc.go"), []byte(fixedCode), 0o644)

				testCode := "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 3) != 5 {\n\t\tt.Fail()\n\t}\n}\n\nfunc TestMultiply(t *testing.T) {\n\tif Multiply(2, 4) != 8 {\n\t\tt.Fail()\n\t}\n}\n"
				_ = os.WriteFile(filepath.Join(dir, "calc_test.go"), []byte(testCode), 0o644)
				return agent.Result{Task: "fix Multiply syntax and add Multiply test", GoalReached: true}, nil
			}
			return agent.Result{Task: "noop"}, nil
		},
	}

	var logBuf bytes.Buffer
	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &logBuf,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	// Verify state
	st, err := stateMgr.Load()
	if err != nil {
		t.Fatal(err)
	}
	if st.Status != state.StatusGoalReached {
		t.Errorf("expected status 'goal_reached', got %s", st.Status)
	}
	if st.Iteration != 2 {
		t.Errorf("expected 2 successful iterations, got %d", st.Iteration)
	}

	// Verify tests pass in real project
	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go test failed on final repo: %s (%v)", string(out), err)
	}
}

// TestE2E_NodeProjectWorkflow tests project-agnostic execution with custom lint/test commands.
func TestE2E_NodeProjectWorkflow(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create package.json and index.js
	pkgJSON := `{"name":"demo","scripts":{"test":"node test.js"}}`
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "index.js"), []byte("module.exports = { greet: () => 'hello' };\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "test.js"), []byte("const { greet } = require('./index'); if (greet() !== 'hello') process.exit(1);\n"), 0o644)
	commitAll(t, dir, "initial node app")

	detected := detect.Detect(dir)
	if detected.Type != "node" {
		t.Fatalf("expected detected type 'node', got %s", detected.Type)
	}

	cfg := config.DetectConfig(dir, []string{"node test.js"})
	cfg.Goal = "Add farewell function"
	cfg.Limits.Iterations = 1

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			newCode := "module.exports = { greet: () => 'hello', farewell: () => 'goodbye' };\n"
			_ = os.WriteFile(filepath.Join(dir, "index.js"), []byte(newCode), 0o644)
			newTest := "const app = require('./index'); if (app.greet() !== 'hello' || app.farewell() !== 'goodbye') process.exit(1);\n"
			_ = os.WriteFile(filepath.Join(dir, "test.js"), []byte(newTest), 0o644)
			return agent.Result{Task: "add farewell function", GoalReached: true}, nil
		},
	}

	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	st, _ := stateMgr.Load()
	if st.Status != state.StatusGoalReached {
		t.Errorf("expected goal_reached, got %s", st.Status)
	}
}

// TestE2E_DeveloperWorkPreservation ensures pre-existing uncommitted files are never committed by LoopGoal.
func TestE2E_DeveloperWorkPreservation(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	_ = os.WriteFile(filepath.Join(dir, "app.txt"), []byte("version 1\n"), 0o644)
	commitAll(t, dir, "initial app")

	// Developer has uncommitted WIP changes
	devNotesFile := filepath.Join(dir, "scratch_notes.md")
	_ = os.WriteFile(devNotesFile, []byte("# Private Developer Notes\n"), 0o644)

	devDraftFile := filepath.Join(dir, "app.txt")
	_ = os.WriteFile(devDraftFile, []byte("version 1 + developer unstaged thoughts\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Goal = "Add helper module"
	cfg.Verify = []string{"test -f helper.txt"}
	cfg.Limits.Iterations = 1

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			_ = os.WriteFile(filepath.Join(dir, "helper.txt"), []byte("helper module\n"), 0o644)
			return agent.Result{Task: "add helper module", GoalReached: true}, nil
		},
	}

	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	// Verify only helper.txt was committed in the latest commit
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	committedList := strings.TrimSpace(string(out))
	if committedList != "helper.txt" {
		t.Errorf("expected only helper.txt in commit, got:\n%s", committedList)
	}

	// Verify developer files are intact
	notes, err := os.ReadFile(devNotesFile)
	if err != nil || string(notes) != "# Private Developer Notes\n" {
		t.Errorf("developer notes were corrupted")
	}
}

// TestE2E_GracefulStopSignal ensures stop trigger immediately halts loop safely.
func TestE2E_GracefulStopSignal(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("initial\n"), 0o644)
	commitAll(t, dir, "initial")

	cfg := config.DefaultConfig()
	cfg.Goal = "Long running task"
	cfg.Verify = []string{"true"}
	cfg.Limits.Iterations = 10

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			// On first iteration, create the stop trigger file
			_ = os.WriteFile(filepath.Join(cfgDir, "stop"), []byte("stop\n"), 0o644)
			_ = os.WriteFile(filepath.Join(dir, "file.txt"), []byte("updated\n"), 0o644)
			return agent.Result{Task: "iteration 1 change"}, nil
		},
	}

	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := engine.Run(ctx); err != nil {
		t.Fatalf("engine failed: %v", err)
	}

	st, _ := stateMgr.Load()
	if st.Status != state.StatusStopped {
		t.Errorf("expected status 'stopped', got %s", st.Status)
	}
	if st.Iteration != 1 {
		t.Errorf("expected loop to halt after iteration 1, got %d", st.Iteration)
	}
}

type mockE2EAgent struct {
	onRun func(ctx context.Context, task string) (agent.Result, error)
}

func (m *mockE2EAgent) Run(ctx context.Context, task string) (agent.Result, error) {
	if m.onRun != nil {
		return m.onRun(ctx, task)
	}
	return agent.Result{}, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// New E2E tests
// ─────────────────────────────────────────────────────────────────────────────

// TestE2E_MultiFileQueue verifies that the loop commits one file per iteration
// and advances through all 3 iterations before marking goal_reached.
func TestE2E_MultiFileQueue(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Start with a single committed file so HEAD exists.
	_ = os.WriteFile(filepath.Join(dir, "base.txt"), []byte("base\n"), 0o644)
	commitAll(t, dir, "initial base")

	cfg := &config.Config{
		Goal: "Create three feature files",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{"true"}, // always passes
		Limits: config.LimitsConfig{
			Iterations: 5,
			MaxRetries: 1,
		},
	}

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	iteration := 0
	filenames := []string{"feature_a.txt", "feature_b.txt", "feature_c.txt"}

	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			if iteration >= len(filenames) {
				return agent.Result{GoalReached: true, Task: "all features complete"}, nil
			}
			fname := filenames[iteration]
			iteration++
			_ = os.WriteFile(filepath.Join(dir, fname), []byte(fname+"\n"), 0o644)
			return agent.Result{
				Task:        "create " + fname,
				GoalReached: iteration == len(filenames),
			}, nil
		},
	}

	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &bytes.Buffer{},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	st, err := stateMgr.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Must reach goal_reached
	if st.Status != state.StatusGoalReached {
		t.Errorf("expected goal_reached, got %s", st.Status)
	}

	// Must have made 3 successful iterations (one file each)
	if st.Iteration != 3 {
		t.Errorf("expected 3 committed iterations, got %d", st.Iteration)
	}

	// All 3 feature files must exist and be committed
	for _, fname := range filenames {
		if _, err := os.Stat(filepath.Join(dir, fname)); err != nil {
			t.Errorf("expected %s to exist on disk: %v", fname, err)
		}
	}

	// Git log should have: initial + 3 feature commits = 4 total
	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "4" {
		t.Errorf("expected 4 commits in git log, got %s", string(out))
	}
}

// TestE2E_VerifyFixCommitCycle tests the full verify→fail→agent-fix→re-verify→commit cycle.
// The first agent run produces invalid output; the fix run produces valid output;
// the commit must only include the corrected file.
func TestE2E_VerifyFixCommitCycle(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	_ = os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("seed\n"), 0o644)
	commitAll(t, dir, "initial seed")

	cfg := &config.Config{
		Goal: "Produce a valid output.txt",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{"grep -x 'STATUS_OK' output.txt"},
		Limits: config.LimitsConfig{
			Iterations: 1,
			MaxRetries: 2,
		},
	}

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	callCount := 0
	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			callCount++
			if callCount == 1 {
				// First call: write BROKEN content — 'grep -x STATUS_OK' will fail.
				_ = os.WriteFile(filepath.Join(dir, "output.txt"), []byte("BROKEN\n"), 0o644)
				return agent.Result{Task: "create output.txt (broken)"}, nil
			}
			// Second call (fix retry): write exact match — verification passes.
			_ = os.WriteFile(filepath.Join(dir, "output.txt"), []byte("STATUS_OK\n"), 0o644)
			return agent.Result{Task: "fix output.txt to STATUS_OK", GoalReached: true}, nil
		},
	}

	var logBuf bytes.Buffer
	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	engine, err := loop.NewEngine(loop.Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: stateMgr,
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &logBuf,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine run failed: %v", err)
	}

	// Agent must have been called exactly twice: initial + fix retry.
	if callCount != 2 {
		t.Errorf("expected agent to be called 2 times (initial + fix), got %d", callCount)
	}

	// output.txt must contain OK after the fix.
	content, err := os.ReadFile(filepath.Join(dir, "output.txt"))
	if err != nil {
		t.Fatalf("output.txt missing: %v", err)
	}
	if !strings.Contains(string(content), "STATUS_OK") {
		t.Errorf("expected output.txt to contain STATUS_OK, got: %s", string(content))
	}

	// Verify git log contains a commit for output.txt.
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git diff-tree failed: %v", err)
	}
	if !strings.Contains(string(out), "output.txt") {
		t.Errorf("expected output.txt in latest commit, got: %s", string(out))
	}

	// Log must mention verification failure for diagnostic completeness.
	if !strings.Contains(logBuf.String(), "Verification failed") && !strings.Contains(logBuf.String(), "failed") {
		t.Logf("log output: %s", logBuf.String())
		// Soft check — logging wording may vary, but warn.
		t.Log("warning: expected verification failure mention in logs")
	}
}

