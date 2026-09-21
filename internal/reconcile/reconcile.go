// Package reconcile compares expected vs actual repository changes, tracks
// missing and unexpected files, dynamically updates the task graph, and collects
// verifiable evidence after each LoopGoal iteration.
package reconcile

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"loopgoal/internal/inventory"
	"loopgoal/internal/taskmap"
)

// Default maximum dynamically discovered files per run to avoid infinite loop sprawl.
const DefaultMaxTaskExpansion = 50

// Options configures the reconciliation pass.
type Options struct {
	MaxTaskExpansion int
}

// Result contains the output of the reconciliation pass.
type Result struct {
	MissingFiles    []string
	UnexpectedFiles []string
	VerifiedFiles   []string
	FailedFiles     []string
	NewlyDiscovered []string
	AllPassed       bool
	BlockedReason   string
}

// Engine performs post-iteration reconciliation.
type Engine struct {
	opts Options
}

// New creates a new reconciliation engine.
func New(opts Options) *Engine {
	if opts.MaxTaskExpansion <= 0 {
		opts.MaxTaskExpansion = DefaultMaxTaskExpansion
	}
	return &Engine{opts: opts}
}

// Reconcile evaluates the actual changed files against the expected files and task map.
func (e *Engine) Reconcile(
	tm *taskmap.TaskMap,
	inv *inventory.Inventory,
	expectedFiles []string,
	actualChangedFiles []string,
	verifyPassed bool,
	verifyEvidence string,
) *Result {
	res := &Result{
		MissingFiles:    make([]string, 0),
		UnexpectedFiles: make([]string, 0),
		VerifiedFiles:   make([]string, 0),
		FailedFiles:     make([]string, 0),
		NewlyDiscovered: make([]string, 0),
	}

	actualMap := make(map[string]bool, len(actualChangedFiles))
	for _, f := range actualChangedFiles {
		clean := filepath.ToSlash(filepath.Clean(f))
		actualMap[clean] = true
	}

	expectedMap := make(map[string]bool, len(expectedFiles))
	for _, f := range expectedFiles {
		clean := filepath.ToSlash(filepath.Clean(f))
		expectedMap[clean] = true
	}

	// 1. Detect missing files: expected but NOT changed
	for exp := range expectedMap {
		if !actualMap[exp] {
			res.MissingFiles = append(res.MissingFiles, exp)
			if tm != nil {
				item := tm.AddOrUpdateFile(exp, inventory.ClassifyRole(exp), "", taskmap.StatusMissing)
				item.ActualChanges = "File was expected to change in this task but was not modified."
			}
		}
	}

	// 2. Detect unexpected files and newly discovered files
	for act := range actualMap {
		if len(expectedMap) > 0 && !expectedMap[act] {
			res.UnexpectedFiles = append(res.UnexpectedFiles, act)
		}

		if tm != nil {
			if _, exists := tm.Files[act]; !exists {
				// Newly discovered file!
				if len(res.NewlyDiscovered) < e.opts.MaxTaskExpansion {
					res.NewlyDiscovered = append(res.NewlyDiscovered, act)
					role := inventory.ClassifyRole(act)
					if inv != nil && inv.Files[act] != nil {
						role = inv.Files[act].Role
					}
					tm.AddOrUpdateFile(act, role, "", taskmap.StatusPending)
				} else {
					res.BlockedReason = fmt.Sprintf("Max task expansion limit (%d) exceeded with newly discovered files", e.opts.MaxTaskExpansion)
				}
			}
		}
	}

	// 3. Update verification statuses and collect evidence
	for act := range actualMap {
		if tm == nil {
			continue
		}

		if verifyPassed {
			res.VerifiedFiles = append(res.VerifiedFiles, act)
			tm.MarkFileVerified(act, fmt.Sprintf("Verified with test suite: %s", verifyEvidence))
		} else {
			res.FailedFiles = append(res.FailedFiles, act)
			item := tm.AddOrUpdateFile(act, inventory.ClassifyRole(act), "", taskmap.StatusFailed)
			item.Evidence = append(item.Evidence, fmt.Sprintf("Verification failure: %s", verifyEvidence))
		}
	}

	sort.Strings(res.MissingFiles)
	sort.Strings(res.UnexpectedFiles)
	sort.Strings(res.VerifiedFiles)
	sort.Strings(res.FailedFiles)
	sort.Strings(res.NewlyDiscovered)

	res.AllPassed = verifyPassed && len(res.MissingFiles) == 0 && res.BlockedReason == ""

	return res
}

// Summary formats a reconciliation result.
func (r *Result) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Reconciliation: %d verified, %d failed, %d missing, %d unexpected, %d new\n",
		len(r.VerifiedFiles),
		len(r.FailedFiles),
		len(r.MissingFiles),
		len(r.UnexpectedFiles),
		len(r.NewlyDiscovered),
	))

	if len(r.MissingFiles) > 0 {
		sb.WriteString("Missing expected files:\n")
		for _, f := range r.MissingFiles {
			sb.WriteString(fmt.Sprintf("  - %s (NOT MODIFIED)\n", f))
		}
	}

	if len(r.UnexpectedFiles) > 0 {
		sb.WriteString("Unexpected modified files:\n")
		for _, f := range r.UnexpectedFiles {
			sb.WriteString(fmt.Sprintf("  + %s (UNPLANNED CHANGE)\n", f))
		}
	}

	if r.BlockedReason != "" {
		sb.WriteString(fmt.Sprintf("Blocker: %s\n", r.BlockedReason))
	}

	return sb.String()
}
