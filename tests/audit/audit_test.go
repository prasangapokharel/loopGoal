package audit_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"loopgoal/internal/audit"
)

func TestDiscoverRules_TopLevelFiles(t *testing.T) {
	dir := t.TempDir()

	// Write AGENTS.md and GEMINI.md
	_ = os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# Rules"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "GEMINI.md"), []byte("# Gemini"), 0o644)

	a := audit.New(dir)
	rules, err := a.DiscoverRules(context.Background())
	if err != nil {
		t.Fatalf("DiscoverRules failed: %v", err)
	}

	found := map[string]bool{}
	for _, r := range rules {
		found[r.Kind] = true
	}
	if !found["agents_md"] {
		t.Error("expected to find agents_md rule")
	}
	if !found["gemini_md"] {
		t.Error("expected to find gemini_md rule")
	}
}

func TestDiscoverRules_CursorAndAgentSkills(t *testing.T) {
	dir := t.TempDir()

	// .cursor/rules/style.mdc
	cursorDir := filepath.Join(dir, ".cursor", "rules")
	_ = os.MkdirAll(cursorDir, 0o755)
	_ = os.WriteFile(filepath.Join(cursorDir, "style.mdc"), []byte("use snake_case"), 0o644)

	// .agents/skills/myskill/SKILL.md
	skillDir := filepath.Join(dir, ".agents", "skills", "myskill")
	_ = os.MkdirAll(skillDir, 0o755)
	_ = os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("---\nname: myskill\n---"), 0o644)

	// .agents/rules/coding.md
	agentRulesDir := filepath.Join(dir, ".agents", "rules")
	_ = os.MkdirAll(agentRulesDir, 0o755)
	_ = os.WriteFile(filepath.Join(agentRulesDir, "coding.md"), []byte("# Coding rules"), 0o644)

	a := audit.New(dir)
	rules, err := a.DiscoverRules(context.Background())
	if err != nil {
		t.Fatalf("DiscoverRules failed: %v", err)
	}

	found := map[string]int{}
	for _, r := range rules {
		found[r.Kind]++
	}
	if found["cursor_rule"] != 1 {
		t.Errorf("expected 1 cursor_rule, got %d", found["cursor_rule"])
	}
	if found["skill"] != 1 {
		t.Errorf("expected 1 skill, got %d", found["skill"])
	}
	if found["agent_rule"] != 1 {
		t.Errorf("expected 1 agent_rule, got %d", found["agent_rule"])
	}
}

func TestGrepPattern_FindsMatches(t *testing.T) {
	dir := t.TempDir()

	// Init git repo first, then add the target file as a tracked file.
	initGitRepo(t, dir)
	_ = os.WriteFile(filepath.Join(dir, "bad.go"), []byte("package main\n\nfunc BadCamelCase() {}\n"), 0o644)
	runGit(t, dir, "add", "bad.go")
	runGit(t, dir, "commit", "-m", "add bad.go")

	a := audit.New(dir)
	findings, err := a.GrepPattern(context.Background(), "BadCamelCase", []string{"bad.go"})
	if err != nil {
		t.Fatalf("GrepPattern failed: %v", err)
	}

	if len(findings) == 0 {
		t.Error("expected at least 1 finding for BadCamelCase pattern")
		return
	}
	if !strings.Contains(findings[0].Text, "BadCamelCase") {
		t.Errorf("unexpected finding text: %s", findings[0].Text)
	}
}

func TestGrepPattern_NoMatches(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	_ = os.WriteFile(filepath.Join(dir, "clean.go"), []byte("package main\n\nfunc clean_func() {}\n"), 0o644)
	runGit(t, dir, "add", "clean.go")
	runGit(t, dir, "commit", "-m", "add clean.go")

	a := audit.New(dir)
	findings, err := a.GrepPattern(context.Background(), "nonexistent_pattern_xyz_9999", []string{"clean.go"})
	if err != nil {
		t.Fatalf("GrepPattern failed: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(findings))
	}
}

func TestAudit_PassedWhenNoViolations(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	_ = os.WriteFile(filepath.Join(dir, "app.go"), []byte("package main\n"), 0o644)

	a := audit.New(dir)
	report, err := a.Audit(context.Background(), []string{"panic_xyz_not_real"})
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}
	if !report.Passed {
		t.Errorf("expected Passed=true, got findings: %v", report.Findings)
	}
}

func TestAudit_SummaryNonEmpty(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	_ = os.WriteFile(filepath.Join(dir, "risky.go"), []byte("package main\nfunc main() { panic(\"fatal\") }\n"), 0o644)

	a := audit.New(dir)
	// Search for a real pattern — the file is untracked so grep falls back to dir scan.
	report, err := a.Audit(context.Background(), []string{`panic\(`})
	if err != nil {
		t.Fatalf("Audit failed: %v", err)
	}
	// Just verify the report is non-nil and Summary works without panicking.
	summary := report.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestReportSummary_Format(t *testing.T) {
	r := &audit.Report{
		Rules: []audit.RuleFile{
			{Path: "AGENTS.md", Kind: "agents_md"},
		},
		Findings: []audit.Finding{
			{File: "bad.go", Line: 3, Pattern: "TODO", Text: "// TODO: fix this"},
		},
		Passed: false,
	}
	s := r.Summary()
	if !strings.Contains(s, "bad.go") {
		t.Error("summary should include finding file name")
	}
	if !strings.Contains(s, "FAILED") {
		t.Error("summary should say FAILED")
	}
}

// ─── helpers ────────────────────────────────────────────────────────────────

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "t@test.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		runGit(t, dir, args...)
	}
	// Create and commit an initial file so HEAD exists.
	_ = os.WriteFile(filepath.Join(dir, ".gitkeep"), nil, 0o644)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "init")
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
	}
}
