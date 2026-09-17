package verify

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CommandResult records the outcome of an individual verification command.
type CommandResult struct {
	Command  string
	Passed   bool
	Output   string
	Duration time.Duration
	Err      error
}

// Summary aggregates the results of all verification commands.
type Summary struct {
	Passed        bool
	Results       []CommandResult
	FailedCommand string
	FailedOutput  string
}

// ErrorOutput formats the failure details suitable for feeding back into the agent prompt.
func (s *Summary) ErrorOutput() string {
	if s.Passed {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Verification command failed: %s\n", s.FailedCommand))
	sb.WriteString("Output:\n")
	sb.WriteString(strings.TrimSpace(s.FailedOutput))
	return sb.String()
}

// Runner executes verification commands sequentially.
type Runner struct {
	dir string
}

// NewRunner creates a verification runner for the given workspace directory.
func NewRunner(dir string) *Runner {
	return &Runner{dir: dir}
}

// Run executes the given list of verification commands sequentially.
// It aborts immediately if any command fails.
func (r *Runner) Run(ctx context.Context, commands []string) (*Summary, error) {
	summary := &Summary{
		Passed:  true,
		Results: make([]CommandResult, 0, len(commands)),
	}

	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}

		start := time.Now()
		cmd := exec.CommandContext(ctx, "sh", "-c", cmdStr)
		cmd.Dir = r.dir

		var outputBuf bytes.Buffer
		cmd.Stdout = &outputBuf
		cmd.Stderr = &outputBuf

		err := cmd.Run()
		duration := time.Since(start)

		outText := outputBuf.String()
		res := CommandResult{
			Command:  cmdStr,
			Passed:   err == nil,
			Output:   outText,
			Duration: duration,
			Err:      err,
		}

		summary.Results = append(summary.Results, res)

		if err != nil {
			summary.Passed = false
			summary.FailedCommand = cmdStr
			summary.FailedOutput = outText
			break
		}
	}

	return summary, nil
}
