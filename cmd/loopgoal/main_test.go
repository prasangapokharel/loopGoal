package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestGitProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "CLI Tester"},
		{"config", "user.email", "cli@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}

	// Create initial file
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write main.go: %v", err)
	}

	c := exec.Command("git", "add", "main.go")
	c.Dir = dir
	_ = c.Run()
	c = exec.Command("git", "commit", "-m", "initial")
	c.Dir = dir
	_ = c.Run()

	return dir
}

func TestCLIInitAndStatus(t *testing.T) {
	dir := setupTestGitProject(t)

	// Test init
	if err := runInit([]string{"-dir", dir}); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Re-init without force should fail
	if err := runInit([]string{"-dir", dir}); err == nil {
		t.Fatal("expected error on re-init without --force, got nil")
	}

	// Re-init with force should succeed
	if err := runInit([]string{"-dir", dir, "-force"}); err != nil {
		t.Fatalf("runInit with -force failed: %v", err)
	}

	// Test status
	if err := runStatus([]string{"-dir", dir}); err != nil {
		t.Fatalf("runStatus failed: %v", err)
	}

	// Test stop (when not running)
	if err := runStop([]string{"-dir", dir}); err != nil {
		t.Fatalf("runStop failed: %v", err)
	}
}

func TestCLIRunWithMockScript(t *testing.T) {
	dir := setupTestGitProject(t)

	// Run init
	if err := runInit([]string{"-dir", dir}); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	// Write mock agent script
	mockScript := filepath.Join(dir, "agent.sh")
	scriptContent := `#!/bin/sh
echo "Working..."
echo "file content" >> main.go
echo "- Change made: added code to main.go"
echo "Goal reached: yes"
`
	if err := os.WriteFile(mockScript, []byte(scriptContent), 0o755); err != nil {
		t.Fatalf("failed to write agent script: %v", err)
	}

	// Overwrite config to use this script and custom verify
	cfgContent := `goal: Test CLI execution
agent:
  command: "` + mockScript + `"
verify:
  - "grep 'file content' main.go"
limits:
  iterations: 1
`
	cfgPath := filepath.Join(dir, ".loopgoal", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Run loop
	if err := runLoop([]string{"-dir", dir}); err != nil {
		t.Fatalf("runLoop failed: %v", err)
	}

	// Check git log
	cmd := exec.Command("git", "log", "-1", "--pretty=%B")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log failed: %s (%v)", string(out), err)
	}
	if !strings.Contains(string(out), "added code to main.go") {
		t.Errorf("expected commit message to contain task, got: %s", string(out))
	}
}
