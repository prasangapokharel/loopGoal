package inventory_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"loopgoal/internal/inventory"
)

func initTempGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runCmd(t, dir, "git", "init")
	runCmd(t, dir, "git", "config", "user.name", "Test")
	runCmd(t, dir, "git", "config", "user.email", "test@example.com")
	return dir
}

func runCmd(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	c := exec.Command(name, args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("%s %v failed: %s (%v)", name, args, string(out), err)
	}
}

func TestInventory_Classification(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", inventory.RoleSource},
		{"src/api/auth.py", inventory.RoleSource},
		{"frontend/src/app.tsx", inventory.RoleSource},
		{"server/contract.sol", inventory.RoleSource},
		{"main_test.go", inventory.RoleTest},
		{"tests/test_auth.py", inventory.RoleTest},
		{"src/components/button.test.ts", inventory.RoleTest},
		{"README.md", inventory.RoleDoc},
		{"docs/architecture.md", inventory.RoleDoc},
		{"go.mod", inventory.RoleConfig},
		{"package.json", inventory.RoleConfig},
		{"pyproject.toml", inventory.RoleConfig},
		{"AGENTS.md", inventory.RoleInstruction},
		{".cursor/rules/standards.mdc", inventory.RoleInstruction},
		{".agents/skills/clean/SKILL.md", inventory.RoleInstruction},
		{".git/config", inventory.RoleIgnored},
		{"node_modules/pkg/index.js", inventory.RoleIgnored},
		{".loopgoal/state.json", inventory.RoleIgnored},
	}

	for _, tt := range tests {
		got := inventory.ClassifyRole(tt.path)
		if got != tt.expected {
			t.Errorf("ClassifyRole(%q) = %q, expected %q", tt.path, got, tt.expected)
		}
	}
}

func TestInventory_Scan(t *testing.T) {
	dir := initTempGitRepo(t)

	// Create files
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main_test.go"), []byte("package main\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Title\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644)

	_ = os.MkdirAll(filepath.Join(dir, ".agents", "skills", "refactor"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, ".agents", "skills", "refactor", "SKILL.md"), []byte("# Skill\n"), 0o644)

	runCmd(t, dir, "git", "add", ".")
	runCmd(t, dir, "git", "commit", "-m", "init")

	scanner := inventory.NewScanner(dir)
	inv, err := scanner.Scan(context.Background())
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	if inv.TotalFiles < 5 {
		t.Errorf("expected at least 5 files, got %d", inv.TotalFiles)
	}
	if len(inv.SourceFiles) != 1 {
		t.Errorf("expected 1 source file, got %d", len(inv.SourceFiles))
	}
	if len(inv.TestFiles) != 1 {
		t.Errorf("expected 1 test file, got %d", len(inv.TestFiles))
	}
	if len(inv.DocFiles) != 1 {
		t.Errorf("expected 1 doc file, got %d", len(inv.DocFiles))
	}
	if len(inv.ConfigFiles) != 1 {
		t.Errorf("expected 1 config file, got %d", len(inv.ConfigFiles))
	}
	if len(inv.Instructions) != 1 {
		t.Errorf("expected 1 instruction file, got %d", len(inv.Instructions))
	}

	summary := inv.Summary()
	if summary == "" {
		t.Error("expected non-empty inventory summary")
	}
}
