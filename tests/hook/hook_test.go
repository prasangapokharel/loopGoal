package hook_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"loopgoal/internal/hook"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Hook Tester"},
		{"config", "user.email", "hook@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}

	// Create initial commit
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	c := exec.Command("git", "add", "README.md")
	c.Dir = dir
	_ = c.Run()
	c = exec.Command("git", "commit", "-m", "initial commit")
	c.Dir = dir
	_ = c.Run()

	return dir
}

func TestHookInstallAndRemove(t *testing.T) {
	dir := setupTestGitRepo(t)

	// Install hooks
	if err := hook.Install(dir); err != nil {
		t.Fatalf("hook.Install failed: %v", err)
	}

	hasCommit, hasPush := hook.Status(dir)
	if !hasCommit || !hasPush {
		t.Fatalf("expected both hooks installed, got commit=%v, push=%v", hasCommit, hasPush)
	}

	// Remove hooks
	if err := hook.Remove(dir); err != nil {
		t.Fatalf("hook.Remove failed: %v", err)
	}

	hasCommit, hasPush = hook.Status(dir)
	if hasCommit || hasPush {
		t.Fatalf("expected both hooks removed, got commit=%v, push=%v", hasCommit, hasPush)
	}
}

func TestPreCommitHookBlocksUnverifiedCommit(t *testing.T) {
	dir := setupTestGitRepo(t)

	// Create .loopgoal directory and state to activate hook
	loopDir := filepath.Join(dir, ".loopgoal")
	_ = os.MkdirAll(loopDir, 0o755)
	_ = os.WriteFile(filepath.Join(loopDir, "state.json"), []byte(`{"status":"running"}`), 0o644)

	// Install hooks
	if err := hook.Install(dir); err != nil {
		t.Fatalf("hook.Install failed: %v", err)
	}

	// Make an unverified edit
	_ = os.WriteFile(filepath.Join(dir, "test.txt"), []byte("unverified edit\n"), 0o644)
	c := exec.Command("git", "add", "test.txt")
	c.Dir = dir
	_ = c.Run()

	// Commit attempt MUST fail
	commitCmd := exec.Command("git", "commit", "-m", "unverified commit")
	commitCmd.Dir = dir
	out, err := commitCmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected commit to be BLOCKED by hook, but succeeded! Output: %s", string(out))
	}
	if !hook.HasToken(dir) && !os.IsNotExist(err) {
		t.Logf("✓ Pre-commit correctly blocked unverified commit: %s", string(out))
	}

	// Now generate verification token
	token, err := hook.WriteToken(dir, "Verified test passed")
	if err != nil {
		t.Fatalf("failed to write token: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Commit should now succeed and consume token
	commitCmd = exec.Command("git", "commit", "-m", "verified commit")
	commitCmd.Dir = dir
	out, err = commitCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected commit to SUCCEED with verified token, failed: %s (%v)", string(out), err)
	}

	// Token should be consumed
	if hook.HasToken(dir) {
		t.Fatal("expected token to be consumed after commit, but still exists")
	}
}
