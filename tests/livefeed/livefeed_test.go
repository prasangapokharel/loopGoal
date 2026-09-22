package livefeed_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"loopgoal/internal/hook"
	"loopgoal/internal/livefeed"
)

func TestLivefeedAtomicWriteAndRead(t *testing.T) {
	dir := t.TempDir()

	payload := livefeed.FeedPayload{
		Heartbeat: 42,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		LatencyMs: "12ms",
		Status:    "passing",
		Stats: livefeed.FeedStats{
			TotalErrors: 0,
			Runtimes:    []string{"go"},
		},
		Errors:            []livefeed.FeedError{},
		CanCommit:         true,
		VerificationToken: "test-token-1234",
	}

	err := livefeed.WriteFeedAtomic(dir, payload)
	if err != nil {
		t.Fatalf("unexpected error writing feed: %v", err)
	}

	read, err := livefeed.ReadFeed(dir)
	if err != nil {
		t.Fatalf("unexpected error reading feed: %v", err)
	}

	if read.Heartbeat != 42 {
		t.Errorf("expected heartbeat 42, got %d", read.Heartbeat)
	}
	if !read.CanCommit {
		t.Error("expected canCommit to be true")
	}
	if read.Status != "passing" {
		t.Errorf("expected status 'passing', got %q", read.Status)
	}
	if !livefeed.IsFeedFresh(read, 5*time.Second) {
		t.Error("expected feed to be fresh")
	}
}

func TestParseCompilerOutputGoVet(t *testing.T) {
	raw := "internal/loop/loop.go:42:15: undefined: foo\ninternal/agent/agent.go:10:5: cannot use 123 (variable of type int)\n"
	parsed := livefeed.ParseCompilerOutput("go-vet", raw)

	if len(parsed) != 2 {
		t.Fatalf("expected 2 parsed errors, got %d", len(parsed))
	}
	if parsed[0].File != "internal/loop/loop.go" || parsed[0].Line != 42 || parsed[0].Col != 15 {
		t.Errorf("unexpected parsed details for error 0: %+v", parsed[0])
	}
	if parsed[0].Message != "undefined: foo" {
		t.Errorf("unexpected message: %q", parsed[0].Message)
	}
}

func TestCheckWorkspacePassing(t *testing.T) {
	dir := t.TempDir()
	// Create dummy go.mod so detect works
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testpkg\n\ngo 1.22\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)

	feed, err := livefeed.CheckWorkspace(dir, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !feed.CanCommit {
		t.Errorf("expected canCommit to be true, got false, errors: %+v", feed.Errors)
	}
	if !hook.HasToken(dir) {
		t.Error("expected hook verified.token to be generated on passing feed")
	}
}
