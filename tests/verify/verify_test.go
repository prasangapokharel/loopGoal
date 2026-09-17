package verify_test

import (
	"context"
	"testing"

	"loopgoal/internal/verify"
)

func TestVerifyRunnerSuccess(t *testing.T) {
	ctx := context.Background()
	runner := verify.NewRunner(t.TempDir())

	commands := []string{
		"echo 'check 1'",
		"echo 'check 2'",
	}

	summary, err := runner.Run(ctx, commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !summary.Passed {
		t.Fatal("expected all verification commands to pass")
	}
	if len(summary.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(summary.Results))
	}
	if summary.ErrorOutput() != "" {
		t.Fatalf("expected empty error output, got %q", summary.ErrorOutput())
	}
}

func TestVerifyRunnerFailure(t *testing.T) {
	ctx := context.Background()
	runner := verify.NewRunner(t.TempDir())

	commands := []string{
		"echo 'step 1'",
		"sh -c 'echo \"syntax error on line 42\" >&2; exit 1'",
		"echo 'should not reach'",
	}

	summary, err := runner.Run(ctx, commands)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.Passed {
		t.Fatal("expected verification to fail")
	}
	if len(summary.Results) != 2 {
		t.Fatalf("expected execution to halt after failure, got %d results", len(summary.Results))
	}
	if summary.FailedCommand != commands[1] {
		t.Errorf("expected failed command %q, got %q", commands[1], summary.FailedCommand)
	}
	errOut := summary.ErrorOutput()
	if errOut == "" {
		t.Fatal("expected non-empty ErrorOutput()")
	}
}
