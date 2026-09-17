package loop

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"loopgoal/internal/agent"
	"loopgoal/internal/config"
	"loopgoal/internal/git"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

// mockAgent implements agent.Agent for testing.
type mockAgent struct {
	workDir string
	step    int
	onRun   func(ctx context.Context, task string, step int) (agent.Result, error)
}

func (m *mockAgent) Run(ctx context.Context, task string) (agent.Result, error) {
	m.step++
	if m.onRun != nil {
		return m.onRun(ctx, task, m.step)
	}
	return agent.Result{Task: "default task"}, nil
}

func setupTestGitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Test Runner"},
		{"config", "user.email", "runner@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}

	// Create initial file
	if err := os.WriteFile(filepath.Join(dir, "initial.txt"), []byte("initial\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	c := exec.Command("git", "add", "initial.txt")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git add failed: %s (%v)", string(out), err)
	}

	c = exec.Command("git", "commit", "-m", "init")
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git commit failed: %s (%v)", string(out), err)
	}

	return dir
}

func TestLoopRunSuccessfulIterations(t *testing.T) {
	dir := setupTestGitDir(t)
	cfgPath := filepath.Join(dir, config.DefaultDir, config.DefaultConfigFile)
	statePath := filepath.Join(dir, config.DefaultDir, config.DefaultStateFile)

	cfg := &config.Config{
		Goal: "Improve the test project",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{
			"cat file*.txt",
		},
		Limits: config.LimitsConfig{
			Iterations: 2,
			MaxRetries: 2,
		},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	mock := &mockAgent{
		workDir: dir,
		onRun: func(ctx context.Context, task string, step int) (agent.Result, error) {
			fname := fmt.Sprintf("file%d.txt", step)
			_ = os.WriteFile(filepath.Join(dir, fname), []byte(fmt.Sprintf("content %d\n", step)), 0o644)
			return agent.Result{
				Task:        fmt.Sprintf("add %s", fname),
				GoalReached: step >= 2,
			}, nil
		},
	}

	var logBuf bytes.Buffer
	engine, err := NewEngine(Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: state.NewManager(statePath),
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &logBuf,
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	stMgr := state.NewManager(statePath)
	finalState, err := stMgr.Load()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if finalState.Iteration != 2 {
		t.Errorf("expected 2 iterations, got %d", finalState.Iteration)
	}
	if finalState.Status != state.StatusGoalReached {
		t.Errorf("expected status 'goal_reached', got %q", finalState.Status)
	}
	if finalState.LastCommit == "" {
		t.Error("expected LastCommit to be recorded")
	}

	// Verify git log has 3 commits (init + 2 iterations)
	c := exec.Command("git", "rev-list", "--count", "HEAD")
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git rev-list failed: %s (%v)", string(out), err)
	}
	if string(bytes.TrimSpace(out)) != "3" {
		t.Errorf("expected 3 commits in git log, got %s", string(out))
	}
}

func TestLoopVerificationFailureAndRetry(t *testing.T) {
	dir := setupTestGitDir(t)
	cfgPath := filepath.Join(dir, config.DefaultDir, config.DefaultConfigFile)
	statePath := filepath.Join(dir, config.DefaultDir, config.DefaultStateFile)

	// Verification expects valid.txt to contain "valid"
	cfg := &config.Config{
		Goal: "Ensure valid configuration",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{
			"grep valid target.txt",
		},
		Limits: config.LimitsConfig{
			Iterations: 1,
			MaxRetries: 2,
		},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// First call writes invalid file, second call (retry) fixes it to "valid"
	mock := &mockAgent{
		workDir: dir,
		onRun: func(ctx context.Context, task string, step int) (agent.Result, error) {
			targetFile := filepath.Join(dir, "target.txt")
			if step == 1 {
				_ = os.WriteFile(targetFile, []byte("broken\n"), 0o644)
				return agent.Result{Task: "create broken file"}, nil
			}
			// Fix step
			_ = os.WriteFile(targetFile, []byte("valid\n"), 0o644)
			return agent.Result{Task: "fix target to valid"}, nil
		},
	}

	var logBuf bytes.Buffer
	engine, err := NewEngine(Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: state.NewManager(statePath),
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &logBuf,
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	stMgr := state.NewManager(statePath)
	finalState, err := stMgr.Load()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}

	if finalState.Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", finalState.Iteration)
	}
	if finalState.Status != state.StatusCompleted {
		t.Errorf("expected status 'completed', got %q", finalState.Status)
	}
}

func TestLoopPreservesPreExistingUserChanges(t *testing.T) {
	dir := setupTestGitDir(t)
	cfgPath := filepath.Join(dir, config.DefaultDir, config.DefaultConfigFile)
	statePath := filepath.Join(dir, config.DefaultDir, config.DefaultStateFile)

	// User creates an uncommitted file before loop starts
	userFile := filepath.Join(dir, "user_wip.txt")
	if err := os.WriteFile(userFile, []byte("user work in progress\n"), 0o644); err != nil {
		t.Fatalf("failed to write user wip: %v", err)
	}

	cfg := &config.Config{
		Goal: "Add agent file",
		Agent: config.AgentConfig{
			Command: "mock",
		},
		Verify: []string{
			"test -f agent_feature.txt",
		},
		Limits: config.LimitsConfig{
			Iterations: 1,
			MaxRetries: 1,
		},
	}
	if err := config.Save(cfgPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	mock := &mockAgent{
		workDir: dir,
		onRun: func(ctx context.Context, task string, step int) (agent.Result, error) {
			_ = os.WriteFile(filepath.Join(dir, "agent_feature.txt"), []byte("agent code\n"), 0o644)
			return agent.Result{Task: "add agent feature"}, nil
		},
	}

	var logBuf bytes.Buffer
	engine, err := NewEngine(Options{
		WorkDir:  dir,
		Config:   cfg,
		StateMgr: state.NewManager(statePath),
		Git:      git.New(dir),
		Verifier: verify.NewRunner(dir),
		Agent:    mock,
		Logger:   &logBuf,
	})
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}

	if err := engine.Run(context.Background()); err != nil {
		t.Fatalf("engine.Run failed: %v", err)
	}

	// Verify user_wip.txt is NOT committed in the latest commit
	c := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git diff-tree failed: %s (%v)", string(out), err)
	}
	committedFiles := string(out)
	if bytes.Contains(out, []byte("user_wip.txt")) {
		t.Fatalf("user_wip.txt was incorrectly committed! Files committed: %s", committedFiles)
	}
	if !bytes.Contains(out, []byte("agent_feature.txt")) {
		t.Fatalf("agent_feature.txt was not committed! Files committed: %s", committedFiles)
	}

	// Verify user_wip.txt is still intact on disk
	content, err := os.ReadFile(userFile)
	if err != nil || string(content) != "user work in progress\n" {
		t.Fatalf("user_wip.txt content was damaged: %s (%v)", string(content), err)
	}
}
