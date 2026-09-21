package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"

	"loopgoal/internal/resolve"
)

// CommandAgent executes an external CLI agent process.
// Set StreamWriter to an io.Writer (e.g. os.Stdout) to receive live output.
type CommandAgent struct {
	Command      string
	Args         []string
	WorkDir      string
	StreamWriter io.Writer // optional: receives real-time stdout/stderr
}

// NewCommandAgent creates an adapter that delegates tasks to a CLI command.
func NewCommandAgent(command string, args []string, workDir string) *CommandAgent {
	return &CommandAgent{
		Command: command,
		Args:    args,
		WorkDir: workDir,
	}
}

// WithStreaming returns a copy of CommandAgent configured to stream live output
// to w. This is a fluent helper for callers that do not use struct literals.
func (c *CommandAgent) WithStreaming(w io.Writer) *CommandAgent {
	cp := *c
	cp.StreamWriter = w
	return &cp
}

// Validate checks that the agent command can be resolved before the loop
// starts. Returns a detailed, actionable error if the binary is not found.
func (c *CommandAgent) Validate() error {
	// Skip validation for shell-composed commands (contain spaces) — they go
	// through sh -c and cannot be statically resolved.
	if strings.Contains(c.Command, " ") {
		return nil
	}
	_, err := resolve.Resolve(c.Command)
	return err
}

// Run executes the external command, piping the task prompt via stdin or
// arguments. If StreamWriter is set, output is forwarded in real-time; it is
// also buffered internally so ParseAgentOutput can analyse the full text.
func (c *CommandAgent) Run(ctx context.Context, task string) (Result, error) {
	// Resolve the binary path with cross-platform search + helpful errors.
	resolvedCmd := c.Command
	if !strings.Contains(c.Command, " ") {
		if r, err := resolve.Resolve(c.Command); err != nil {
			return Result{}, fmt.Errorf("agent not found: %w", err)
		} else {
			resolvedCmd = r
		}
	}

	hasPromptPlaceholder := false
	var finalArgs []string
	for _, arg := range c.Args {
		if strings.Contains(arg, "{task}") || strings.Contains(arg, "{prompt}") {
			arg = strings.ReplaceAll(arg, "{task}", task)
			arg = strings.ReplaceAll(arg, "{prompt}", task)
			hasPromptPlaceholder = true
		}
		finalArgs = append(finalArgs, arg)
	}

	var cmd *exec.Cmd
	if len(finalArgs) > 0 {
		cmd = exec.CommandContext(ctx, resolvedCmd, finalArgs...)
	} else if strings.Contains(c.Command, " ") {
		// Command string contains embedded arguments / shell syntax.
		cmd = exec.CommandContext(ctx, "sh", "-c", c.Command)
	} else {
		cmd = exec.CommandContext(ctx, resolvedCmd)
	}

	cmd.Dir = c.WorkDir

	// Stream task via stdin unless it was embedded in args.
	if !hasPromptPlaceholder {
		cmd.Stdin = strings.NewReader(task)
	}

	// Buffer always captures output for parsing.
	// If StreamWriter is set, output is tee'd there in real-time.
	var stdoutBuf, stderrBuf bytes.Buffer
	if c.StreamWriter != nil {
		cmd.Stdout = io.MultiWriter(&stdoutBuf, c.StreamWriter)
		cmd.Stderr = io.MultiWriter(&stderrBuf, c.StreamWriter)
	} else {
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
	}

	runErr := cmd.Run()

	combinedOutput := stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		if combinedOutput != "" && !strings.HasSuffix(combinedOutput, "\n") {
			combinedOutput += "\n"
		}
		combinedOutput += stderrBuf.String()
	}

	res := parseAgentOutput(combinedOutput)

	if runErr != nil {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		return res, fmt.Errorf("agent command failed: %w\noutput:\n%s", runErr, combinedOutput)
	}

	return res, nil
}

var (
	reGoalReached = regexp.MustCompile(`(?i)(?:goal\s*(?:appears\s*)?reached\s*[:=]\s*(?:true|yes)|\[goal_reached\]|status:\s*goal_reached)`)
	reBlocked     = regexp.MustCompile(`(?i)(?:blocked\s*[:=]\s*(?:true|yes)|\[blocked\]|status:\s*blocked|iteration\s*(?:is\s*)?blocked)`)
	reTask        = regexp.MustCompile(`(?i)(?:[-*]\s*)?(?:change(?:s)?\s*(?:made)?|task|summary)\s*[:=]\s*(.+)`)
)

// ParseAgentOutput extracts completion signals, task summaries, and flags from agent output.
func ParseAgentOutput(output string) Result {
	return parseAgentOutput(output)
}

func parseAgentOutput(output string) Result {
	res := Result{
		Output: output,
	}

	res.GoalReached = reGoalReached.MatchString(output)
	res.Blocked = reBlocked.MatchString(output)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if m := reTask.FindStringSubmatch(line); len(m) > 1 {
			taskText := strings.TrimSpace(m[1])
			if taskText != "" && res.Task == "" {
				res.Task = taskText
			}
		}
	}

	return res
}
