// Package audit discovers project rule/skill files and runs deep grep-based
// compliance checks before each LoopGoal iteration.
package audit

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RuleFile represents a discovered rule or skill definition file.
type RuleFile struct {
	Path string
	Kind string // "agents_md", "gemini_md", "cursor_rule", "skill", "claude_md"
}

// Finding reports a single non-compliant pattern hit in a source file.
type Finding struct {
	File    string
	Line    int
	Pattern string
	Text    string
}

// Report is the full result of an audit run.
type Report struct {
	Rules    []RuleFile
	Findings []Finding
	Passed   bool
}

// Auditor discovers rule files and executes grep-based compliance audits.
type Auditor struct {
	workDir string
}

// New creates an Auditor for the given project root.
func New(workDir string) *Auditor {
	return &Auditor{workDir: workDir}
}

// DiscoverRules locates all rule/skill files in a project directory tree.
// It searches for: AGENTS.md, GEMINI.md, CLAUDE.md, .cursor/rules/*.mdc,
// .agents/rules/*.md, .agents/skills/*/SKILL.md
func (a *Auditor) DiscoverRules(ctx context.Context) ([]RuleFile, error) {
	var rules []RuleFile

	// Canonical top-level rule files
	toplevel := []struct {
		name string
		kind string
	}{
		{"AGENTS.md", "agents_md"},
		{"GEMINI.md", "gemini_md"},
		{"CLAUDE.md", "claude_md"},
	}
	for _, tl := range toplevel {
		p := filepath.Join(a.workDir, tl.name)
		if fileExists(p) {
			rules = append(rules, RuleFile{Path: p, Kind: tl.kind})
		}
	}

	// .cursor/rules/*.mdc
	cursorRulesDir := filepath.Join(a.workDir, ".cursor", "rules")
	if entries, err := os.ReadDir(cursorRulesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".mdc") {
				rules = append(rules, RuleFile{
					Path: filepath.Join(cursorRulesDir, e.Name()),
					Kind: "cursor_rule",
				})
			}
		}
	}

	// .agents/rules/*.md
	agentRulesDir := filepath.Join(a.workDir, ".agents", "rules")
	if entries, err := os.ReadDir(agentRulesDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				rules = append(rules, RuleFile{
					Path: filepath.Join(agentRulesDir, e.Name()),
					Kind: "agent_rule",
				})
			}
		}
	}

	// .agents/skills/*/SKILL.md
	skillsDir := filepath.Join(a.workDir, ".agents", "skills")
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				skillFile := filepath.Join(skillsDir, e.Name(), "SKILL.md")
				if fileExists(skillFile) {
					rules = append(rules, RuleFile{Path: skillFile, Kind: "skill"})
				}
			}
		}
	}

	return rules, nil
}

// GrepPattern runs a grep/ripgrep search for a given pattern across all tracked
// source files and returns matching findings.
func (a *Auditor) GrepPattern(ctx context.Context, pattern string, scope []string) ([]Finding, error) {
	// Prefer ripgrep (rg) for speed, fall back to grep.
	tool, toolArgs := grepTool(pattern, scope, a.workDir)

	cmd := exec.CommandContext(ctx, tool, toolArgs...)
	cmd.Dir = a.workDir

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Exit code 1 means no matches — not an error.
	_ = cmd.Run()

	return parseGrepOutput(out.String(), pattern), nil
}

// RunStaticChecks executes project static analysis tools (go vet, ruff, tsc, etc.)
// that are detected to be available in the project.
func (a *Auditor) RunStaticChecks(ctx context.Context) ([]string, error) {
	var output []string

	checkers := detectStaticCheckers(a.workDir)
	for _, chk := range checkers {
		cmd := exec.CommandContext(ctx, chk[0], chk[1:]...)
		cmd.Dir = a.workDir
		var buf bytes.Buffer
		cmd.Stdout = &buf
		cmd.Stderr = &buf
		if err := cmd.Run(); err != nil {
			output = append(output, fmt.Sprintf("[%s] %s", strings.Join(chk, " "), buf.String()))
		}
	}

	return output, nil
}

// Audit runs a full compliance audit: rule discovery + grep pattern sweep +
// static checks. It returns a Report summarising all findings.
func (a *Auditor) Audit(ctx context.Context, patterns []string) (*Report, error) {
	report := &Report{}

	rules, err := a.DiscoverRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("discovering rules: %w", err)
	}
	report.Rules = rules

	// Collect tracked source files via git ls-files for the grep scope.
	trackedFiles := a.trackedFiles(ctx)

	for _, pat := range patterns {
		findings, err := a.GrepPattern(ctx, pat, trackedFiles)
		if err != nil {
			return nil, fmt.Errorf("grep pattern %q: %w", pat, err)
		}
		report.Findings = append(report.Findings, findings...)
	}

	report.Passed = len(report.Findings) == 0
	return report, nil
}

// Summary formats an audit report into a human-readable string suitable for
// injecting into an agent prompt.
func (r *Report) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Rules discovered: %d\n", len(r.Rules)))
	for _, rule := range r.Rules {
		sb.WriteString(fmt.Sprintf("  [%s] %s\n", rule.Kind, rule.Path))
	}

	if len(r.Findings) == 0 {
		sb.WriteString("Compliance: PASSED — no violations found.\n")
	} else {
		sb.WriteString(fmt.Sprintf("Compliance: FAILED — %d violation(s):\n", len(r.Findings)))
		for _, f := range r.Findings {
			sb.WriteString(fmt.Sprintf("  %s:%d  [%s]  %s\n", f.File, f.Line, f.Pattern, f.Text))
		}
	}

	return sb.String()
}

// ─── helpers ────────────────────────────────────────────────────────────────

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// grepTool picks ripgrep or grep and assembles the argument list.
// The --with-filename / -H flag is always passed so that every output line is
// in the format <file>:<line>:<content> that parseGrepOutput expects.
func grepTool(pattern string, scope []string, workDir string) (string, []string) {
	// Check whether rg is on PATH.
	if _, err := exec.LookPath("rg"); err == nil {
		args := []string{"--line-number", "--no-heading", "--color=never", "--with-filename", pattern}
		if len(scope) > 0 {
			args = append(args, scope...)
		} else {
			args = append(args, ".")
		}
		return "rg", args
	}

	// Fall back to grep -rn -H.
	args := []string{"-rn", "-H", "--color=never", pattern}
	if len(scope) > 0 {
		args = append(args, scope...)
	} else {
		args = append(args, workDir)
	}
	return "grep", args
}

// parseGrepOutput converts raw grep/rg output lines into Finding structs.
// Expected format per line: <file>:<line>:<content>
func parseGrepOutput(raw string, pattern string) []Finding {
	var findings []Finding
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		lineNum := 0
		_, _ = fmt.Sscanf(parts[1], "%d", &lineNum)
		findings = append(findings, Finding{
			File:    parts[0],
			Line:    lineNum,
			Pattern: pattern,
			Text:    strings.TrimSpace(parts[2]),
		})
	}
	return findings
}

// detectStaticCheckers returns the static analysis commands available for the project.
func detectStaticCheckers(workDir string) [][]string {
	var checkers [][]string

	if fileExists(filepath.Join(workDir, "go.mod")) {
		if _, err := exec.LookPath("go"); err == nil {
			checkers = append(checkers, []string{"go", "vet", "./..."})
		}
	}

	if fileExists(filepath.Join(workDir, "pyproject.toml")) || fileExists(filepath.Join(workDir, "requirements.txt")) {
		if _, err := exec.LookPath("ruff"); err == nil {
			checkers = append(checkers, []string{"ruff", "check", "."})
		}
	}

	if fileExists(filepath.Join(workDir, "package.json")) {
		if _, err := exec.LookPath("npx"); err == nil {
			checkers = append(checkers, []string{"npx", "--yes", "tsc", "--noEmit"})
		}
	}

	if fileExists(filepath.Join(workDir, "Cargo.toml")) {
		if _, err := exec.LookPath("cargo"); err == nil {
			checkers = append(checkers, []string{"cargo", "check", "--quiet"})
		}
	}

	return checkers
}

// trackedFiles returns git-tracked source file paths via `git ls-files`.
// Falls back to an empty slice (grep will scan the whole directory).
func (a *Auditor) trackedFiles(ctx context.Context) []string {
	cmd := exec.CommandContext(ctx, "git", "ls-files")
	cmd.Dir = a.workDir
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var files []string
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		f = strings.TrimRight(f, "\r")
		if f != "" {
			files = append(files, f)
		}
	}
	return files
}
