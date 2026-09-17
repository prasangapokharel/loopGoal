package loop

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"loopgoal/internal/agent"
	"loopgoal/internal/config"
	"loopgoal/internal/git"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

// Options configures the Loop engine.
type Options struct {
	WorkDir    string
	Config     *config.Config
	StateMgr   *state.Manager
	Git        *git.Git
	Verifier   *verify.Runner
	Agent      agent.Agent
	Logger     io.Writer
	PID        int
	StopSignal string // path to stop trigger file if any
}

// Engine coordinates the autonomous iteration cycle.
type Engine struct {
	opts    Options
	cfg     *config.Config
	state   *state.State
	git     *git.Git
	verify  *verify.Runner
	agent   agent.Agent
	out     io.Writer
	pidFile string
}

// NewEngine constructs a new loop engine.
func NewEngine(opts Options) (*Engine, error) {
	if opts.Logger == nil {
		opts.Logger = os.Stdout
	}
	if opts.PID == 0 {
		opts.PID = os.Getpid()
	}
	if opts.WorkDir == "" {
		opts.WorkDir = "."
	}

	pidFile := filepath.Join(opts.WorkDir, config.DefaultDir, "loopgoal.pid")

	return &Engine{
		opts:    opts,
		cfg:     opts.Config,
		git:     opts.Git,
		verify:  opts.Verifier,
		agent:   opts.Agent,
		out:     opts.Logger,
		pidFile: pidFile,
	}, nil
}

// Run starts the autonomous development supervisor loop.
func (e *Engine) Run(ctx context.Context) error {
	// Write PID file
	if err := os.WriteFile(e.pidFile, []byte(fmt.Sprintf("%d\n", e.opts.PID)), 0o644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}
	defer os.Remove(e.pidFile)

	// Ensure git repo
	isRepo, err := e.git.IsRepo(ctx)
	if err != nil || !isRepo {
		return fmt.Errorf("not a git repository (or any of the parent directories)")
	}

	// Load or initialize state
	st, err := e.opts.StateMgr.Load()
	if err != nil {
		st, err = e.opts.StateMgr.Init(e.cfg.Goal)
		if err != nil {
			return fmt.Errorf("initializing state: %w", err)
		}
	}
	e.state = st
	e.state.Goal = e.cfg.Goal
	e.state.Status = state.StatusRunning
	e.state.PID = e.opts.PID
	if err := e.opts.StateMgr.Save(e.state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	e.logHeader()

	for {
		// 1. Check stop conditions
		if err := ctx.Err(); err != nil {
			e.log("\nLoop interrupted by user.")
			e.updateStatus(state.StatusStopped)
			return nil
		}

		if e.checkStopSignal() {
			e.log("\nStop signal detected. Gracefully stopping.")
			e.updateStatus(state.StatusStopped)
			return nil
		}

		if e.state.Iteration >= e.cfg.Limits.Iterations {
			e.log(fmt.Sprintf("\nIteration limit reached (%d/%d). Stopping.", e.state.Iteration, e.cfg.Limits.Iterations))
			e.updateStatus(state.StatusCompleted)
			return nil
		}

		// 2. Begin iteration
		nextIter := e.state.Iteration + 1
		e.log(fmt.Sprintf("\nIteration %d of %d", nextIter, e.cfg.Limits.Iterations))

		iterSuccess, shouldStop, err := e.runIteration(ctx, nextIter)
		if err != nil {
			if ctx.Err() != nil {
				e.updateStatus(state.StatusStopped)
				return nil
			}
			e.log(fmt.Sprintf("✗ Iteration failed: %v", err))
			e.updateStatus(state.StatusFailed)
			return err
		}

		if iterSuccess {
			e.state.Iteration = nextIter
			if err := e.opts.StateMgr.Save(e.state); err != nil {
				return fmt.Errorf("saving state: %w", err)
			}
		}

		if shouldStop {
			return nil
		}

		// Brief pause between iterations
		select {
		case <-ctx.Done():
			e.updateStatus(state.StatusStopped)
			return nil
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// runIteration executes one bounded Observe-Select-Execute-Verify-Commit cycle.
// Returns (success bool, shouldStop bool, err error)
func (e *Engine) runIteration(ctx context.Context, iter int) (bool, bool, error) {
	e.log("→ Inspecting repository")

	// Snapshot pre-existing dirty files so we don't commit unrelated changes
	preExistingDirty, err := e.git.Snapshot(ctx)
	if err != nil {
		return false, false, fmt.Errorf("inspecting git status: %w", err)
	}

	// Build initial task prompt
	prompt := e.buildPrompt(iter, preExistingDirty)

	e.log("→ Running agent")
	res, err := e.agent.Run(ctx, prompt)
	if err != nil {
		if ctx.Err() != nil {
			return false, true, ctx.Err()
		}
		return false, false, fmt.Errorf("agent execution failed: %w", err)
	}

	if res.Blocked {
		e.log("! Agent reported blocked.")
		e.updateStatus(state.StatusBlocked)
		return false, true, nil
	}

	// Check for diff right after execution
	changedFiles, err := e.git.IterationChanges(ctx, preExistingDirty)
	if err != nil {
		return false, false, fmt.Errorf("checking git changes: %w", err)
	}

	if len(changedFiles) == 0 {
		e.log("ℹ No file changes detected in this iteration.")
		if res.GoalReached {
			e.log("✓ Agent reports goal has been reached!")
			e.updateStatus(state.StatusGoalReached)
			return false, true, nil
		}
		// If no changes were made and goal is not reached, advance iteration count without commit
		return true, false, nil
	}

	e.log(fmt.Sprintf("→ Verifying changes (%d files changed)", len(changedFiles)))

	// Verification loop with retries
	verifyPassed := false
	maxRetries := e.cfg.Limits.MaxRetries
	if maxRetries <= 0 {
		maxRetries = config.DefaultMaxRetries
	}

	for retry := 0; retry <= maxRetries; retry++ {
		if ctx.Err() != nil {
			return false, true, ctx.Err()
		}

		vSummary, err := e.verify.Run(ctx, e.cfg.Verify)
		if err != nil {
			return false, false, fmt.Errorf("running verification: %w", err)
		}

		if vSummary.Passed {
			verifyPassed = true
			e.log("✓ Verification passed")
			break
		}

		e.log(fmt.Sprintf("✗ Verification failed on command: %s", vSummary.FailedCommand))

		if retry == maxRetries {
			e.log("! Max verification retries exceeded for this iteration.")
			e.updateStatus(state.StatusBlocked)
			return false, true, nil
		}

		e.log(fmt.Sprintf("→ Retrying with agent (attempt %d/%d)", retry+1, maxRetries))
		fixPrompt := e.buildFixPrompt(iter, vSummary.ErrorOutput())
		fixRes, err := e.agent.Run(ctx, fixPrompt)
		if err != nil {
			return false, false, fmt.Errorf("agent fix retry failed: %w", err)
		}
		if fixRes.Blocked {
			e.log("! Agent reported blocked during fix attempt.")
			e.updateStatus(state.StatusBlocked)
			return false, true, nil
		}
		if fixRes.GoalReached {
			res.GoalReached = true
		}
		if fixRes.Task != "" {
			res.Task = fixRes.Task
		}
	}

	if !verifyPassed {
		return false, true, nil
	}

	// Diff review
	e.log("→ Reviewing diff")
	finalChanges, err := e.git.IterationChanges(ctx, preExistingDirty)
	if err != nil {
		return false, false, fmt.Errorf("reviewing iteration diff: %w", err)
	}
	if len(finalChanges) == 0 {
		e.log("ℹ No changes remained after verification.")
		return true, false, nil
	}

	// Commit iteration changes
	taskDesc := res.Task
	if taskDesc == "" {
		taskDesc = fmt.Sprintf("Iteration %d improvement", iter)
	}
	commitMsg := formatCommitMessage(taskDesc, finalChanges)

	e.log(fmt.Sprintf("→ Committing changes: %s", commitMsg))
	if err := e.git.Stage(ctx, finalChanges...); err != nil {
		return false, false, fmt.Errorf("staging iteration changes: %w", err)
	}

	commitHash, err := e.git.Commit(ctx, commitMsg)
	if err != nil {
		return false, false, fmt.Errorf("committing iteration changes: %w", err)
	}
	e.log(fmt.Sprintf("✓ Committed: %s", commitHash))

	// Update state
	e.state.LastTask = taskDesc
	e.state.LastCommit = commitHash
	e.state.Status = state.StatusRunning

	if res.GoalReached {
		e.log("✓ Agent reports goal has been reached!")
		e.updateStatus(state.StatusGoalReached)
		return true, true, nil
	}

	return true, false, nil
}

func (e *Engine) buildPrompt(iter int, preExistingDirty map[string]bool) string {
	var sb strings.Builder

	sb.WriteString("You are working inside an existing software repository.\n\n")
	sb.WriteString(fmt.Sprintf("Primary goal:\n%s\n\n", strings.TrimSpace(e.cfg.Goal)))
	sb.WriteString(fmt.Sprintf("Current iteration:\n%d\n\n", iter))

	if len(preExistingDirty) > 0 {
		sb.WriteString("Note: The working tree already has the following files modified by the user:\n")
		for f := range preExistingDirty {
			sb.WriteString(fmt.Sprintf("- %s\n", f))
		}
		sb.WriteString("Do NOT overwrite or revert these pre-existing changes.\n\n")
	}

	sb.WriteString("Your task:\n")
	sb.WriteString("Inspect the repository and identify ONE small, valuable improvement that directly contributes to the primary goal.\n\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Make only the changes necessary for this improvement.\n")
	sb.WriteString("- Do not rewrite unrelated code.\n")
	sb.WriteString("- Preserve the existing architecture.\n")
	sb.WriteString("- Reuse existing abstractions where appropriate.\n")
	sb.WriteString("- Do not introduce unnecessary dependencies.\n")
	sb.WriteString("- Do not modify unrelated files.\n")
	sb.WriteString("- Do not remove working functionality.\n")
	if len(e.cfg.Verify) > 0 {
		sb.WriteString("- Ensure changes pass verification: " + strings.Join(e.cfg.Verify, ", ") + "\n")
	}
	sb.WriteString("- Stop after completing this single improvement.\n\n")
	sb.WriteString("After completing the task, report:\n")
	sb.WriteString("- Change made: <concise summary of change>\n")
	sb.WriteString("- Files changed: <list of files>\n")
	sb.WriteString("- Verification result: <passed/failed>\n")
	sb.WriteString("- Goal reached: <yes/no>\n")
	sb.WriteString("- Blocked: <yes/no>\n")

	return sb.String()
}

func (e *Engine) buildFixPrompt(iter int, errorOutput string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Verification failed for iteration %d.\n\n", iter))
	sb.WriteString("Failure details:\n")
	sb.WriteString(errorOutput)
	sb.WriteString("\n\nYour task:\n")
	sb.WriteString("Fix the failure while keeping your bounded improvement intact.\n")
	sb.WriteString("Do not make unrelated changes.\n")
	sb.WriteString("Report when done:\n")
	sb.WriteString("- Change made: <summary of fix>\n")
	sb.WriteString("- Blocked: <yes/no>\n")
	return sb.String()
}

func formatCommitMessage(task string, files []string) string {
	cleanTask := strings.TrimSpace(task)
	// Strip existing prefixes like "change made:" if present
	lower := strings.ToLower(cleanTask)
	prefixes := []string{"change made:", "task:", "summary:", "feat:", "fix:", "refactor:", "chore:", "test:", "docs:"}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			cleanTask = strings.TrimSpace(cleanTask[len(p):])
			lower = strings.ToLower(cleanTask)
		}
	}

	// Detect conventional commit prefix from task text
	prefix := "refactor"
	if strings.HasPrefix(lower, "add") || strings.HasPrefix(lower, "create") || strings.HasPrefix(lower, "implement") {
		prefix = "feat"
	} else if strings.HasPrefix(lower, "fix") || strings.HasPrefix(lower, "correct") || strings.HasPrefix(lower, "repair") {
		prefix = "fix"
	} else if strings.HasPrefix(lower, "test") {
		prefix = "test"
	} else if strings.HasPrefix(lower, "doc") {
		prefix = "docs"
	}

	// Try to extract a scope if files are unified under a directory
	scope := ""
	if len(files) > 0 {
		dir := filepath.Dir(files[0])
		if dir != "." && dir != "/" && !strings.Contains(dir, "/") {
			scope = fmt.Sprintf("(%s)", dir)
		}
	}

	return fmt.Sprintf("%s%s: %s", prefix, scope, cleanTask)
}

func (e *Engine) checkStopSignal() bool {
	stopPath := filepath.Join(e.opts.WorkDir, config.DefaultDir, "stop")
	if _, err := os.Stat(stopPath); err == nil {
		_ = os.Remove(stopPath)
		return true
	}
	return false
}

func (e *Engine) updateStatus(st string) {
	e.state.Status = st
	_ = e.opts.StateMgr.Save(e.state)
}

func (e *Engine) log(msg string) {
	fmt.Fprintln(e.out, msg)
}

func (e *Engine) logHeader() {
	e.log("LoopGoal")
	e.log("────────────────────────────")
	e.log(fmt.Sprintf("Goal:\n%s\n", strings.TrimSpace(e.cfg.Goal)))
}
