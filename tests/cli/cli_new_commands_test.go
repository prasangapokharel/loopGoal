package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"loopgoal/internal/cli"
	"loopgoal/internal/hook"
	"loopgoal/internal/state"
)

func TestCLISelectCommand(t *testing.T) {
	dir := setupTestGitProject(t)

	// Select a file
	err := cli.RunSelect([]string{"-dir", dir, "-objective", "Add typed error handling", "main.go"})
	if err != nil {
		t.Fatalf("RunSelect failed: %v", err)
	}

	// Verify state file
	statePath := filepath.Join(dir, ".loopgoal", "state.json")
	stateMgr := state.NewManager(statePath)
	st, err := stateMgr.Load()
	if err != nil {
		t.Fatalf("loading state failed: %v", err)
	}

	if st.ActiveTarget != "main.go" {
		t.Errorf("expected active target 'main.go', got '%s'", st.ActiveTarget)
	}
	if st.LastTask != "Add typed error handling" {
		t.Errorf("expected objective 'Add typed error handling', got '%s'", st.LastTask)
	}
}

func TestCLIHookCommand(t *testing.T) {
	dir := setupTestGitProject(t)

	// Install
	if err := cli.RunHook([]string{"install", "-dir", dir}); err != nil {
		t.Fatalf("RunHook install failed: %v", err)
	}

	hasCommit, hasPush := hook.Status(dir)
	if !hasCommit || !hasPush {
		t.Fatalf("expected hooks installed, got commit=%v, push=%v", hasCommit, hasPush)
	}

	// Status
	if err := cli.RunHook([]string{"status", "-dir", dir}); err != nil {
		t.Fatalf("RunHook status failed: %v", err)
	}

	// Remove
	if err := cli.RunHook([]string{"remove", "-dir", dir}); err != nil {
		t.Fatalf("RunHook remove failed: %v", err)
	}

	hasCommit, hasPush = hook.Status(dir)
	if hasCommit || hasPush {
		t.Fatalf("expected hooks removed, got commit=%v, push=%v", hasCommit, hasPush)
	}
}

func TestCLIRollbackCommand(t *testing.T) {
	dir := setupTestGitProject(t)

	// Create dirty file
	filePath := filepath.Join(dir, "broken.go")
	_ = os.WriteFile(filePath, []byte("broken syntax!!"), 0o644)

	// Run rollback
	if err := cli.RunRollback([]string{"-dir", dir}); err != nil {
		t.Fatalf("RunRollback failed: %v", err)
	}

	// File should be gone
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected broken.go to be removed by rollback, but still exists")
	}
}

func TestCLIVerifyCommand(t *testing.T) {
	dir := setupTestGitProject(t)

	// Initialize config
	_ = cli.RunInit([]string{"-dir", dir})

	// Run verify
	err := cli.RunVerify([]string{"-dir", dir})
	if err != nil {
		t.Fatalf("RunVerify failed: %v", err)
	}

	// Token should exist
	if !hook.HasToken(dir) {
		t.Fatalf("expected verification token to exist after successful RunVerify")
	}
}
