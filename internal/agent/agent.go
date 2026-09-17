package agent

import (
	"context"
)

// Result contains the outcome of an agent's execution.
type Result struct {
	Output      string   `json:"output"`
	Task        string   `json:"task,omitempty"`
	Files       []string `json:"files,omitempty"`
	GoalReached bool     `json:"goal_reached"`
	Blocked     bool     `json:"blocked"`
}

// Agent is the unified interface for AI coding agent executors.
type Agent interface {
	Run(ctx context.Context, task string) (Result, error)
}
