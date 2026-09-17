package agent

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// CommandAgent executes an external CLI agent process.
type CommandAgent struct {
	Command string
	Args    []string
	WorkDir string
}

// NewCommandAgent creates an adapter that delegates tasks to a CLI command.
func NewCommandAgent(command string, args []string, workDir string) *CommandAgent {
	return &CommandAgent{
		Command: command,
		Args:    args,
		WorkDir: workDir,
	}
}

// Run executes the external command, piping the task prompt via stdin or arguments.
func (c *CommandAgent) Run(ctx context.Context, task string) (Result, error) {
	var cmd *exec.Cmd

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

	if len(finalArgs) > 0 {
		cmd = exec.CommandContext(ctx, c.Command, finalArgs...)
	} else if strings.Contains(c.Command, " ") {
		// Command contains arguments or shell symbols
		cmd = exec.CommandContext(ctx, "sh", "-c", c.Command)
	} else {
		cmd = exec.CommandContext(ctx, c.Command)
	}

	cmd.Dir = c.WorkDir

	// If prompt was not embedded in arguments, stream it via stdin
	if !hasPromptPlaceholder {
		cmd.Stdin = strings.NewReader(task)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	combinedOutput := stdout.String()
	if stderr.Len() > 0 {
		if combinedOutput != "" && !strings.HasSuffix(combinedOutput, "\n") {
			combinedOutput += "\n"
		}
		combinedOutput += stderr.String()
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
