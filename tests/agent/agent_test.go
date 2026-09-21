package agent_test

import (
	"bytes"
	"context"
	"testing"

	"loopgoal/internal/agent"
)

func TestCommandAgentExecution(t *testing.T) {
	ctx := context.Background()
	ca := agent.NewCommandAgent("cat", nil, t.TempDir())

	res, err := ca.Run(ctx, "Hello from LoopGoal!\n- Change made: extracted validation helper\nGoal reached: no\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Task != "extracted validation helper" {
		t.Errorf("expected task 'extracted validation helper', got %q", res.Task)
	}
	if res.GoalReached {
		t.Error("expected GoalReached to be false")
	}
	if res.Blocked {
		t.Error("expected Blocked to be false")
	}
}

func TestCommandAgentGoalReachedAndBlocked(t *testing.T) {
	outputGoal := "Everything done.\n- task: final cleanup\ngoal reached: yes"
	res := agent.ParseAgentOutput(outputGoal)
	if !res.GoalReached {
		t.Error("expected GoalReached to be true")
	}
	if res.Task != "final cleanup" {
		t.Errorf("expected task 'final cleanup', got %q", res.Task)
	}

	outputBlocked := "Unable to progress.\nStatus: blocked"
	resBlocked := agent.ParseAgentOutput(outputBlocked)
	if !resBlocked.Blocked {
		t.Error("expected Blocked to be true")
	}
}

func TestCommandAgentPlaceholder(t *testing.T) {
	ctx := context.Background()
	ca := agent.NewCommandAgent("echo", []string{"received: {task}"}, t.TempDir())

	res, err := ca.Run(ctx, "sample task prompt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "received: sample task prompt\n"
	if res.Output != expected {
		t.Errorf("expected %q, got %q", expected, res.Output)
	}
}

// TestCommandAgentStreaming verifies that output is written to StreamWriter
// in real-time while the full output is also available in Result.Output.
func TestCommandAgentStreaming(t *testing.T) {
	ctx := context.Background()

	var streamBuf bytes.Buffer
	ca := agent.NewCommandAgent("echo", []string{"streamed output"}, t.TempDir()).
		WithStreaming(&streamBuf)

	res, err := ca.Run(ctx, "ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Both StreamWriter and Result.Output must contain the same content.
	if streamBuf.String() == "" {
		t.Error("StreamWriter received no output")
	}
	if res.Output != streamBuf.String() {
		t.Errorf("Result.Output %q differs from stream %q", res.Output, streamBuf.String())
	}
	if !bytes.Contains(streamBuf.Bytes(), []byte("streamed output")) {
		t.Errorf("expected 'streamed output' in stream, got: %q", streamBuf.String())
	}
}

// TestWithStreamingNil ensures Run works normally when StreamWriter is nil.
func TestWithStreamingNil(t *testing.T) {
	ctx := context.Background()
	ca := agent.NewCommandAgent("echo", []string{"hello"}, t.TempDir())
	ca.StreamWriter = nil // explicit nil — must not panic

	res, err := ca.Run(ctx, "task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Output == "" {
		t.Error("expected non-empty output")
	}
}
