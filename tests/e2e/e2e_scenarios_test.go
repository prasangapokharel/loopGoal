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

// TestScenario_MonorepoScopedTargeting tests Scenario 17 from docs/test/plan:
// In a monorepo with apps/web, apps/api, packages/ui, and docs/, a task targeting
// apps/web scopes only apps/web and does not treat other apps as pending targets.
func TestScenario_MonorepoScopedTargeting(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	// Monorepo tree
	_ = os.MkdirAll(filepath.Join(dir, "apps", "web", "src"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "apps", "api", "src"), 0o755)
	_ = os.MkdirAll(filepath.Join(dir, "packages", "ui"), 0o755)

	_ = os.WriteFile(filepath.Join(dir, "apps", "web", "src", "index.tsx"), []byte("// web\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "apps", "web", "src", "button.tsx"), []byte("// button\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "apps", "api", "src", "server.go"), []byte("package main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "packages", "ui", "theme.ts"), []byte("// theme\n"), 0o644)

	c1 := exec.Command("git", "add", ".")
	c1.Dir = dir
	_ = c1.Run()
	c2 := exec.Command("git", "commit", "-m", "init monorepo")
	c2.Dir = dir
	_ = c2.Run()

	cfg := &config.Config{
		Goal: "Upgrade frontend UI components in apps/web/",
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
				_ = os.WriteFile(filepath.Join(dir, "apps", "web", "src", "index.tsx"), []byte("// upgraded index\n"), 0o644)
				return agent.Result{
					Task:        "upgrade index.tsx",
					GoalReached: false,
				}, nil
			}
			_ = os.WriteFile(filepath.Join(dir, "apps", "web", "src", "button.tsx"), []byte("// upgraded button\n"), 0o644)
			return agent.Result{
				Task:        "upgrade button.tsx",
				GoalReached: true,
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

	if st.Status != state.StatusGoalReached {
		t.Errorf("expected goal_reached for apps/web target, got %s", st.Status)
	}
	if st.Iteration != 2 {
		t.Errorf("expected 2 iterations for 2 apps/web files, got %d", st.Iteration)
	}
}

// TestScenario_CrossLanguagePolyglot tests Scenario 18 from docs/test/plan:
// Verifies LoopGoal accurately scans and operates in a multi-language repo (TS, Go, Python, Rust).
func TestScenario_CrossLanguagePolyglot(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	_ = os.WriteFile(filepath.Join(dir, "web.ts"), []byte("// ts\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "worker.py"), []byte("# python\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "lib.rs"), []byte("// rust\n"), 0o644)

	c1 := exec.Command("git", "add", ".")
	c1.Dir = dir
	_ = c1.Run()
	c2 := exec.Command("git", "commit", "-m", "init polyglot")
	c2.Dir = dir
	_ = c2.Run()

	if err := cli.RunInit([]string{"-dir", dir}); err != nil {
		t.Fatal(err)
	}

	if err := cli.RunTest([]string{"-dir", dir, "-smoke"}); err != nil {
		t.Errorf("cli.RunTest failed on polyglot repo: %v", err)
	}
}

// TestScenario_InfiniteLoopConvergenceProtection tests Scenario 15 from docs/test/plan:
// When an agent repeatedly changes a file but verification never passes, LoopGoal
// enforces max verification retries and halts cleanly in StatusBlocked.
func TestScenario_InfiniteLoopConvergenceProtection(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	cfg := &config.Config{
		Goal: "Fix failing build",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{"false"}, // always fails
		Limits: config.LimitsConfig{
			Iterations: 5,
			MaxRetries: 2,
		},
	}

	cfgDir := filepath.Join(dir, config.DefaultDir)
	_ = os.MkdirAll(cfgDir, 0o755)

	mock := &mockE2EAgent{
		onRun: func(ctx context.Context, task string) (agent.Result, error) {
			_ = os.WriteFile(filepath.Join(dir, "broken.go"), []byte("package main\n"), 0o644)
			return agent.Result{
				Task: "modify broken.go",
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

	if st.Status != state.StatusBlocked {
		t.Errorf("expected StatusBlocked when verification never converges, got %s", st.Status)
	}
}
