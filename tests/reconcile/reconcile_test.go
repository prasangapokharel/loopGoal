package reconcile_test

import (
	"testing"

	"loopgoal/internal/inventory"
	"loopgoal/internal/reconcile"
	"loopgoal/internal/taskmap"
)

func TestReconcile_DetectMissingAndUnexpected(t *testing.T) {
	engine := reconcile.New(reconcile.Options{MaxTaskExpansion: 10})
	tm := taskmap.New("Test Reconcile")

	tm.AddOrUpdateFile("api/users.go", inventory.RoleSource, "task-001", taskmap.StatusPending)
	tm.AddOrUpdateFile("api/auth.go", inventory.RoleSource, "task-002", taskmap.StatusPending)

	// Expected both files to change, but agent only modified api/users.go and also unexpectedly modified config.yaml
	expected := []string{"api/users.go", "api/auth.go"}
	actual := []string{"api/users.go", "config.yaml"}

	res := engine.Reconcile(tm, nil, expected, actual, true, "tests passed")

	// Missing: api/auth.go
	if len(res.MissingFiles) != 1 || res.MissingFiles[0] != "api/auth.go" {
		t.Errorf("expected missing file api/auth.go, got %v", res.MissingFiles)
	}

	// Unexpected: config.yaml
	if len(res.UnexpectedFiles) != 1 || res.UnexpectedFiles[0] != "config.yaml" {
		t.Errorf("expected unexpected file config.yaml, got %v", res.UnexpectedFiles)
	}

	// Dynamic discovery: config.yaml was added to tm.Files
	if tm.Files["config.yaml"] == nil {
		t.Error("expected config.yaml to be dynamically added to taskmap")
	}

	// AllPassed should be false because missing files exist
	if res.AllPassed {
		t.Error("expected res.AllPassed = false when missing files exist")
	}

	// TaskMap should NOT allow completion because api/auth.go is marked missing
	if tm.CanComplete() {
		t.Error("expected tm.CanComplete() = false when missing file present")
	}
}

func TestReconcile_VerificationFailureRecording(t *testing.T) {
	engine := reconcile.New(reconcile.Options{MaxTaskExpansion: 10})
	tm := taskmap.New("Test Reconcile Failure")

	actual := []string{"server.go"}
	res := engine.Reconcile(tm, nil, nil, actual, false, "syntax error on line 42")

	if res.AllPassed {
		t.Error("expected AllPassed = false on verification failure")
	}
	if len(res.FailedFiles) != 1 || res.FailedFiles[0] != "server.go" {
		t.Errorf("expected server.go in FailedFiles, got %v", res.FailedFiles)
	}
	if tm.Files["server.go"].Status != taskmap.StatusFailed {
		t.Errorf("expected server.go status = failed, got %s", tm.Files["server.go"].Status)
	}
	if tm.CanComplete() {
		t.Error("expected tm.CanComplete() = false on failed file")
	}
}

func TestReconcile_MaxExpansionLimit(t *testing.T) {
	engine := reconcile.New(reconcile.Options{MaxTaskExpansion: 2})
	tm := taskmap.New("Test Expansion")

	// Modify 3 new files when limit is 2
	actual := []string{"new1.go", "new2.go", "new3.go"}
	res := engine.Reconcile(tm, nil, nil, actual, true, "tests passed")

	if res.BlockedReason == "" {
		t.Error("expected blocked reason when expansion limit exceeded")
	}
	if res.AllPassed {
		t.Error("expected AllPassed = false when expansion limit exceeded")
	}
}
