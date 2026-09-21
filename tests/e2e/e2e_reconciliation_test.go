package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"loopgoal/internal/agent"
	"loopgoal/internal/cli"
	"loopgoal/internal/config"
	"loopgoal/internal/git"
	"loopgoal/internal/loop"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

// TestE2E_FalseDonePrevention tests that an AI claiming "Goal reached" early
// when files from the target queue are still unverified is rejected by LoopGoal's
// completion gate and kept running until all mapped targets are verified.
func TestE2E_FalseDonePrevention(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create initial target files
	authDir := filepath.Join(dir, "backend", "api", "v1", "auth")
	_ = os.MkdirAll(authDir, 0o755)
	_ = os.WriteFile(filepath.Join(authDir, "login.py"), []byte("# login\n"), 0o644)
	_ = os.WriteFile(filepath.Join(authDir, "mfa.py"), []byte("# mfa\n"), 0o644)

	c1 := exec.Command("git", "add", ".")
	c1.Dir = dir
	_ = c1.Run()
	c2 := exec.Command("git", "commit", "-m", "initial auth files")
	c2.Dir = dir
	_ = c2.Run()

	cfg := &config.Config{
		Goal: "Refactor backend auth (backend/api/v1/auth/)",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{"true"},
		Limits: config.LimitsConfig{
			Iterations: 5,
			MaxRetries: 1,
		},
	}

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	iteration := 0
	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			iteration++
			if iteration == 1 {
				// Modify only login.py, but falsely claim GoalReached: true!
				_ = os.WriteFile(filepath.Join(authDir, "login.py"), []byte("# refactored login\n"), 0o644)
				return agent.Result{
					Task:        "refactor login.py",
					GoalReached: true, // FALSE DONE CLAIM
				}, nil
			}

			// Iteration 2: AI continues and refactors mfa.py
			_ = os.WriteFile(filepath.Join(authDir, "mfa.py"), []byte("# refactored mfa\n"), 0o644)
			return agent.Result{
				Task:        "refactor mfa.py",
				GoalReached: true, // TRUE DONE CLAIM (all 2 files now done)
			}, nil
		},
	}

	stateMgr := state.NewManager(filepath.Join(cfgDir, config.DefaultStateFile))
	var logBuf bytes.Buffer
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

	st, err := stateMgr.Load()
	if err != nil {
		t.Fatal(err)
	}

	// Must have completed after 2 iterations, NOT after 1 iteration
	if st.Iteration != 2 {
		t.Errorf("expected 2 iterations (false done in iter 1 caught), got %d", st.Iteration)
	}
	if st.Status != state.StatusGoalReached {
		t.Errorf("expected status 'goal_reached', got %s", st.Status)
	}

	logs := logBuf.String()
	if !bytes.Contains(logBuf.Bytes(), []byte("remain pending")) && !bytes.Contains(logBuf.Bytes(), []byte("remain unverified")) {
		t.Errorf("expected log to indicate rejection of premature goal completion, got:\n%s", logs)
	}
}

// TestE2E_CLIScanAndPlan verifies loopgoal scan and loopgoal plan commands.
func TestE2E_CLIScanAndPlan(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Create files
	_ = os.WriteFile(filepath.Join(dir, "app.py"), []byte("print('hello')\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "app_test.py"), []byte("def test_app(): pass\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# App\n"), 0o644)

	c1 := exec.Command("git", "add", ".")
	c1.Dir = dir
	_ = c1.Run()
	c2 := exec.Command("git", "commit", "-m", "init")
	c2.Dir = dir
	_ = c2.Run()

	// 1. Initialize
	if err := cli.RunInit([]string{"-dir", dir}); err != nil {
		t.Fatalf("cli.RunInit failed: %v", err)
	}

	// 2. Scan
	if err := cli.RunScan([]string{"-dir", dir}); err != nil {
		t.Fatalf("cli.RunScan failed: %v", err)
	}

	// 3. Plan
	if err := cli.RunPlan([]string{"-dir", dir}); err != nil {
		t.Fatalf("cli.RunPlan failed: %v", err)
	}
}
