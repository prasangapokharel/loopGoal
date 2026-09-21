// Package inventory provides deterministic repository scanning, file
// categorization, and metadata collection for LoopGoal.
package inventory

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// File role constants.
const (
	RoleSource      = "source"
	RoleTest        = "test"
	RoleDoc         = "doc"
	RoleConfig      = "config"
	RoleInstruction = "instruction"
	RoleIgnored     = "ignored"
	RoleOther       = "other"
)

// FileInfo contains metadata and classification for a single repository file.
type FileInfo struct {
	Path      string `json:"path"`
	Role      string `json:"role"`
	Extension string `json:"extension"`
	SizeBytes int64  `json:"size_bytes"`
}

// Inventory captures the complete empirical state of files in a repository.
type Inventory struct {
	RootDir      string               `json:"root_dir"`
	TotalFiles   int                  `json:"total_files"`
	Files        map[string]*FileInfo `json:"files"` // relative path -> FileInfo
	SourceFiles  []string             `json:"source_files"`
	TestFiles    []string             `json:"test_files"`
	DocFiles     []string             `json:"doc_files"`
	ConfigFiles  []string             `json:"config_files"`
	Instructions []string             `json:"instructions"`
	IgnoredFiles []string             `json:"ignored_files,omitempty"`
}

// Scanner scans a repository to build an Inventory.
type Scanner struct {
	RootDir string
}

// NewScanner creates a new repository scanner.
func NewScanner(rootDir string) *Scanner {
	if rootDir == "" {
		rootDir = "."
	}
	return &Scanner{RootDir: rootDir}
}

// Scan scans the repository using git ls-files if available, with a filesystem walk fallback.
func (s *Scanner) Scan(ctx context.Context) (*Inventory, error) {
	absRoot, err := filepath.Abs(s.RootDir)
	if err != nil {
		return nil, fmt.Errorf("resolving root directory: %w", err)
	}

	relPaths, err := s.collectFiles(ctx, absRoot)
	if err != nil {
		return nil, fmt.Errorf("collecting repository files: %w", err)
	}

	inv := &Inventory{
		RootDir:      s.RootDir,
		Files:        make(map[string]*FileInfo, len(relPaths)),
		SourceFiles:  make([]string, 0),
		TestFiles:    make([]string, 0),
		DocFiles:     make([]string, 0),
		ConfigFiles:  make([]string, 0),
		Instructions: make([]string, 0),
		IgnoredFiles: make([]string, 0),
	}

	for _, rel := range relPaths {
		rel = filepath.ToSlash(filepath.Clean(rel))
		if rel == "." || rel == "" {
			continue
		}

		fullPath := filepath.Join(absRoot, filepath.FromSlash(rel))
		fi, err := os.Stat(fullPath)
		if err != nil || fi.IsDir() {
			continue
		}

		role := ClassifyRole(rel)
		ext := strings.ToLower(filepath.Ext(rel))

		info := &FileInfo{
			Path:      rel,
			Role:      role,
			Extension: ext,
			SizeBytes: fi.Size(),
		}

		inv.Files[rel] = info
		inv.TotalFiles++

		switch role {
		case RoleSource:
			inv.SourceFiles = append(inv.SourceFiles, rel)
		case RoleTest:
			inv.TestFiles = append(inv.TestFiles, rel)
		case RoleDoc:
			inv.DocFiles = append(inv.DocFiles, rel)
		case RoleConfig:
			inv.ConfigFiles = append(inv.ConfigFiles, rel)
		case RoleInstruction:
			inv.Instructions = append(inv.Instructions, rel)
		case RoleIgnored:
			inv.IgnoredFiles = append(inv.IgnoredFiles, rel)
		}
	}

	return inv, nil
}

// collectFiles gathers all repository files.
func (s *Scanner) collectFiles(ctx context.Context, absRoot string) ([]string, error) {
	// Try git ls-files first (tracked + untracked except ignored)
	cmd := exec.CommandContext(ctx, "git", "ls-files", "--cached", "--others", "--exclude-standard")
	cmd.Dir = absRoot
	out, err := cmd.Output()
	if err == nil {
		var list []string
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				list = append(list, line)
			}
		}
		if len(list) > 0 {
			return list, nil
		}
	}

	// Fallback: walk filesystem skipping common ignore dirs
	var list []string
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if isIgnoredDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil
		}
		list = append(list, filepath.ToSlash(rel))
		return nil
	})

	return list, err
}

// ClassifyRole assigns a role to a relative file path.
func ClassifyRole(relPath string) string {
	lower := strings.ToLower(relPath)
	base := strings.ToLower(filepath.Base(relPath))
	ext := strings.ToLower(filepath.Ext(relPath))

	// Ignored / artifacts / cache
	if strings.HasPrefix(lower, ".git/") ||
		strings.HasPrefix(lower, "node_modules/") ||
		strings.HasPrefix(lower, "vendor/") ||
		strings.HasPrefix(lower, "__pycache__/") ||
		strings.HasPrefix(lower, ".pytest_cache/") ||
		strings.HasPrefix(lower, "htmlcov/") ||
		strings.HasPrefix(lower, "coverage/") ||
		strings.HasPrefix(lower, ".nyc_output/") ||
		strings.HasPrefix(lower, "dist/") ||
		strings.HasPrefix(lower, "build/") ||
		strings.HasPrefix(lower, ".loopgoal/") {
		return RoleIgnored
	}

	// Instruction files (AGENTS.md, CLAUDE.md, .cursor/rules/*, .agents/*)
	if base == "agents.md" || base == "gemini.md" || base == "claude.md" ||
		strings.HasPrefix(lower, ".cursor/rules/") ||
		strings.HasPrefix(lower, ".agents/") ||
		strings.HasPrefix(lower, ".opencode/") {
		return RoleInstruction
	}

	// Tests (unit, integration, e2e, and language-specific test conventions)
	if strings.HasSuffix(lower, "_test.go") ||
		strings.HasPrefix(base, "test_") ||
		strings.HasSuffix(lower, "_test.py") ||
		strings.HasSuffix(lower, ".test.ts") ||
		strings.HasSuffix(lower, ".test.js") ||
		strings.HasSuffix(lower, ".spec.ts") ||
		strings.HasSuffix(lower, ".spec.js") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "/testing/") ||
		strings.HasPrefix(lower, "tests/") ||
		strings.HasPrefix(lower, "test/") ||
		strings.HasPrefix(lower, "testing/") {
		return RoleTest
	}

	// Documentation
	if ext == ".md" || ext == ".rst" || ext == ".txt" || ext == ".adoc" ||
		strings.HasPrefix(lower, "docs/") || strings.Contains(lower, "/docs/") {
		return RoleDoc
	}

	// Configuration & manifests
	if ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".toml" ||
		ext == ".ini" || ext == ".env" || ext == ".xml" ||
		base == "makefile" || base == "dockerfile" || base == "go.mod" ||
		base == "go.sum" || base == "package.json" || base == "package-lock.json" ||
		base == "requirements.txt" || base == "pyproject.toml" || base == "cargo.toml" ||
		base == "pytest.ini" {
		return RoleConfig
	}

	// Source code files
	sourceExts := map[string]bool{
		".go": true, ".py": true, ".ts": true, ".js": true, ".tsx": true, ".jsx": true,
		".rs": true, ".java": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
		".cs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true, ".scala": true,
		".sh": true, ".bash": true, ".sql": true, ".html": true, ".css": true, ".scss": true,
		".mo": true, ".sol": true, ".zig": true, ".lua": true,
	}
	if sourceExts[ext] {
		return RoleSource
	}

	return RoleOther
}

// Summary returns a concise summary of the inventory.
func (inv *Inventory) Summary() string {
	return fmt.Sprintf("Total: %d files (Source: %d, Tests: %d, Docs: %d, Config: %d, Instructions: %d)",
		inv.TotalFiles,
		len(inv.SourceFiles),
		len(inv.TestFiles),
		len(inv.DocFiles),
		len(inv.ConfigFiles),
		len(inv.Instructions),
	)
}

func isIgnoredDir(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", "__pycache__", ".pytest_cache", "htmlcov", "coverage", ".nyc_output", "dist", "build", ".loopgoal", ".idea", ".vscode":
		return true
	default:
		return false
	}
}
