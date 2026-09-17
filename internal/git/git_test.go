package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupTestRepo(t *testing.T) (string, *Git) {
	t.Helper()
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %s (%v)", string(out), err)
	}

	// Configure test user
	for _, args := range [][]string{
		{"config", "user.name", "LoopGoal Test"},
		{"config", "user.email", "test@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = tmpDir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}

	g := New(tmpDir)
	return tmpDir, g
}

func TestGitIsRepo(t *testing.T) {
	ctx := context.Background()
	tmpDir, g := setupTestRepo(t)

	isRepo, err := g.IsRepo(ctx)
	if err != nil {
		t.Fatalf("IsRepo returned error: %v", err)
	}
	if !isRepo {
		t.Fatal("expected IsRepo to be true")
	}

	nonRepo := New(t.TempDir())
	isRepo, err = nonRepo.IsRepo(ctx)
	if err != nil {
		t.Fatalf("IsRepo on non-repo returned error: %v", err)
	}
	if isRepo {
		t.Fatal("expected IsRepo to be false for empty directory")
	}
	_ = tmpDir
}

func TestGitWorkflow(t *testing.T) {
	ctx := context.Background()
	tmpDir, g := setupTestRepo(t)

	// Create initial file & commit
	initFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(initFile, []byte("# Test Repo\n"), 0o644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	if err := g.Stage(ctx, "README.md"); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	hash, err := g.Commit(ctx, "initial commit")
	if err != nil {
		t.Fatalf("failed to commit: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty commit hash")
	}

	// Snapshot before iteration
	// Say pre-existing dirty file exists
	preExistingFile := filepath.Join(tmpDir, "unrelated.txt")
	if err := os.WriteFile(preExistingFile, []byte("user notes\n"), 0o644); err != nil {
		t.Fatalf("failed to write pre-existing file: %v", err)
	}

	snapshot, err := g.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	if !snapshot["unrelated.txt"] {
		t.Fatal("expected unrelated.txt in snapshot")
	}

	// Now an iteration modifies README.md and creates feature.go
	if err := os.WriteFile(initFile, []byte("# Test Repo\n\nUpdated.\n"), 0o644); err != nil {
		t.Fatalf("failed to modify README: %v", err)
	}
	featureFile := filepath.Join(tmpDir, "feature.go")
	if err := os.WriteFile(featureFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatalf("failed to write feature: %v", err)
	}

	iterationChanges, err := g.IterationChanges(ctx, snapshot)
	if err != nil {
		t.Fatalf("iteration changes failed: %v", err)
	}

	// Should contain README.md and feature.go, NOT unrelated.txt!
	foundReadme := false
	foundFeature := false
	for _, c := range iterationChanges {
		if c == "README.md" {
			foundReadme = true
		}
		if c == "feature.go" {
			foundFeature = true
		}
		if c == "unrelated.txt" {
			t.Errorf("unrelated.txt should NOT be part of iteration changes: %v", iterationChanges)
		}
	}
	if !foundReadme || !foundFeature {
		t.Fatalf("expected README.md and feature.go in iteration changes, got: %v", iterationChanges)
	}

	// Stage only iteration changes
	if err := g.Stage(ctx, iterationChanges...); err != nil {
		t.Fatalf("failed to stage iteration changes: %v", err)
	}

	commitHash, err := g.Commit(ctx, "feat: add feature")
	if err != nil {
		t.Fatalf("failed to commit iteration changes: %v", err)
	}
	if commitHash == "" {
		t.Fatal("expected valid commit hash")
	}

	// Verify that unrelated.txt is STILL untracked / untouched
	statuses, err := g.Status(ctx)
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}
	foundUnrelated := false
	for _, s := range statuses {
		if s.Path == "unrelated.txt" {
			foundUnrelated = true
			if s.Staging != '?' && s.WorkTree != '?' {
				t.Errorf("expected unrelated.txt to remain untracked (??), got %c%c", s.Staging, s.WorkTree)
			}
		}
	}
	if !foundUnrelated {
		t.Fatal("expected unrelated.txt to still exist in working tree status")
	}
}
