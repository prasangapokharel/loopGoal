package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus represents a file's state in git status.
type FileStatus struct {
	Path     string
	Staging  byte
	WorkTree byte
}

// Git provides repository-level Git operations.
type Git struct {
	dir string
}

// New creates a new Git client for the specified working directory.
func New(dir string) *Git {
	return &Git{dir: dir}
}

// execGit executes a git command in the repository directory.
func (g *Git) execGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = g.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg == "" {
			errMsg = strings.TrimSpace(stdout.String())
		}
		if errMsg != "" {
			return "", fmt.Errorf("git %s failed: %s (%w)", strings.Join(args, " "), errMsg, err)
		}
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}

// IsRepo checks if the directory is inside a git work tree.
func (g *Git) IsRepo(ctx context.Context) (bool, error) {
	out, err := g.execGit(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(out) == "true", nil
}

// Status returns the current git status entries.
func (g *Git) Status(ctx context.Context) ([]FileStatus, error) {
	out, err := g.execGit(ctx, "status", "--porcelain=v1")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out, "\n")
	var statuses []FileStatus
	for _, line := range lines {
		if len(line) < 4 {
			continue
		}
		staging := line[0]
		workTree := line[1]
		path := strings.TrimSpace(line[3:])

		// Handle renamed files "old -> new"
		if idx := strings.Index(path, " -> "); idx != -1 {
			path = path[idx+4:]
		}

		statuses = append(statuses, FileStatus{
			Path:     path,
			Staging:  staging,
			WorkTree: workTree,
		})
	}
	return statuses, nil
}

// ChangedFiles returns a slice of paths for all modified, staged, or untracked files.
func (g *Git) ChangedFiles(ctx context.Context) ([]string, error) {
	statuses, err := g.Status(ctx)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, s := range statuses {
		paths = append(paths, s.Path)
	}
	return paths, nil
}

// Snapshot returns a set of paths currently dirty/untracked in the repository.
func (g *Git) Snapshot(ctx context.Context) (map[string]bool, error) {
	files, err := g.ChangedFiles(ctx)
	if err != nil {
		return nil, err
	}

	snap := make(map[string]bool, len(files))
	for _, f := range files {
		snap[f] = true
	}
	return snap, nil
}

// IterationChanges identifies files modified during an iteration, excluding files that were already dirty.
func (g *Git) IterationChanges(ctx context.Context, initialSnapshot map[string]bool) ([]string, error) {
	currentFiles, err := g.ChangedFiles(ctx)
	if err != nil {
		return nil, err
	}

	var changes []string
	for _, f := range currentFiles {
		if !initialSnapshot[f] {
			changes = append(changes, f)
		}
	}
	return changes, nil
}

// Diff returns the working tree diff, optionally filtered by file paths.
func (g *Git) Diff(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return g.execGit(ctx, args...)
}

// DiffCached returns the staged diff, optionally filtered by file paths.
func (g *Git) DiffCached(ctx context.Context, files ...string) (string, error) {
	args := []string{"diff", "--cached"}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return g.execGit(ctx, args...)
}

// Stage stages specific files using git add.
func (g *Git) Stage(ctx context.Context, files ...string) error {
	if len(files) == 0 {
		return errors.New("no files specified to stage")
	}
	args := append([]string{"add", "--"}, files...)
	_, err := g.execGit(ctx, args...)
	return err
}

// Commit creates a commit with the specified message.
func (g *Git) Commit(ctx context.Context, message string) (string, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return "", errors.New("commit message cannot be empty")
	}

	_, err := g.execGit(ctx, "commit", "-m", message)
	if err != nil {
		return "", err
	}

	return g.HeadCommit(ctx)
}

// HeadCommit returns the short commit hash of the current HEAD.
func (g *Git) HeadCommit(ctx context.Context) (string, error) {
	out, err := g.execGit(ctx, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branch returns the current branch name.
func (g *Git) Branch(ctx context.Context) (string, error) {
	out, err := g.execGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Rollback reverts modified files and removes untracked files from an iteration.
func (g *Git) Rollback(ctx context.Context, files ...string) error {
	if len(files) == 0 {
		_, _ = g.execGit(ctx, "checkout", "--", ".")
		_, _ = g.execGit(ctx, "clean", "-fd")
		return nil
	}

	for _, f := range files {
		_, _ = g.execGit(ctx, "checkout", "--", f)
		_, _ = g.execGit(ctx, "clean", "-f", "--", f)
	}
	return nil
}
