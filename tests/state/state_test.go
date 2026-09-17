package state_test

import (
	"path/filepath"
	"testing"
	"time"

	"loopgoal/internal/state"
)

func TestStateInitAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".loopgoal", "state.json")
	mgr := state.NewManager(statePath)

	if mgr.Exists() {
		t.Fatal("expected state file not to exist initially")
	}

	st, err := mgr.Init("Test Goal")
	if err != nil {
		t.Fatalf("failed to init state: %v", err)
	}

	if st.Goal != "Test Goal" {
		t.Errorf("expected goal 'Test Goal', got %q", st.Goal)
	}
	if st.Iteration != 0 {
		t.Errorf("expected iteration 0, got %d", st.Iteration)
	}
	if st.Status != state.StatusIdle {
		t.Errorf("expected status 'idle', got %q", st.Status)
	}

	// Re-init should error
	if _, err := mgr.Init("Test Goal 2"); err == nil {
		t.Fatal("expected error on re-init, got nil")
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	if loaded.Goal != "Test Goal" {
		t.Errorf("expected loaded goal 'Test Goal', got %q", loaded.Goal)
	}
}

func TestStateSaveUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".loopgoal", "state.json")
	mgr := state.NewManager(statePath)

	st, err := mgr.Init("Improve code")
	if err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	st.Iteration = 1
	st.Status = state.StatusRunning
	st.LastTask = "Fix error handling"
	st.LastCommit = "abc1234"
	st.PID = 12345

	if err := mgr.Save(st); err != nil {
		t.Fatalf("failed to save state: %v", err)
	}

	loaded, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load updated state: %v", err)
	}

	if loaded.Iteration != 1 {
		t.Errorf("expected iteration 1, got %d", loaded.Iteration)
	}
	if loaded.Status != state.StatusRunning {
		t.Errorf("expected status running, got %q", loaded.Status)
	}
	if loaded.LastTask != "Fix error handling" {
		t.Errorf("expected last task 'Fix error handling', got %q", loaded.LastTask)
	}
	if loaded.LastCommit != "abc1234" {
		t.Errorf("expected last commit 'abc1234', got %q", loaded.LastCommit)
	}
	if loaded.PID != 12345 {
		t.Errorf("expected PID 12345, got %d", loaded.PID)
	}
	if !loaded.UpdatedAt.After(loaded.StartedAt) {
		t.Errorf("expected UpdatedAt %v to be after StartedAt %v", loaded.UpdatedAt, loaded.StartedAt)
	}
}
