package taskmap_test

import (
	"path/filepath"
	"testing"

	"loopgoal/internal/inventory"
	"loopgoal/internal/taskmap"
)

func TestTaskMap_ExtractTargetPatterns(t *testing.T) {
	goal := "Refactor backend auth (backend/api/v1/auth/) and ensure mfa.py conforms to standard"
	patterns := taskmap.ExtractTargetPatterns(goal)

	foundDir := false
	foundFile := false
	for _, p := range patterns {
		if p == "backend/api/v1/auth/" {
			foundDir = true
		}
		if p == "mfa.py" {
			foundFile = true
		}
	}

	if !foundDir {
		t.Errorf("expected to find directory 'backend/api/v1/auth/' in patterns, got %v", patterns)
	}
	if !foundFile {
		t.Errorf("expected to find file 'mfa.py' in patterns, got %v", patterns)
	}
}

func TestTaskMap_BuildFromInventory(t *testing.T) {
	inv := &inventory.Inventory{
		TotalFiles: 3,
		Files: map[string]*inventory.FileInfo{
			"backend/api/v1/auth/login.py": {
				Path: "backend/api/v1/auth/login.py",
				Role: inventory.RoleSource,
			},
			"backend/api/v1/auth/token.py": {
				Path: "backend/api/v1/auth/token.py",
				Role: inventory.RoleSource,
			},
			"frontend/app.tsx": {
				Path: "frontend/app.tsx",
				Role: inventory.RoleSource,
			},
		},
	}

	goal := "Refactor backend auth (backend/api/v1/auth/)"
	tm := taskmap.BuildFromInventory(inv, goal)

	// Should match only the 2 auth files
	if len(tm.Files) != 2 {
		t.Errorf("expected 2 mapped files, got %d", len(tm.Files))
	}
	if len(tm.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tm.Tasks))
	}

	pending := tm.PendingFiles()
	if len(pending) != 2 {
		t.Errorf("expected 2 pending files, got %d", len(pending))
	}

	// Cannot complete while tasks are pending
	if tm.CanComplete() {
		t.Error("expected CanComplete() = false when tasks are pending")
	}

	// Mark 1 verified
	tm.MarkFileVerified("backend/api/v1/auth/login.py", "pytest passed")
	if tm.CanComplete() {
		t.Error("expected CanComplete() = false when 1 task still pending")
	}

	// Mark 2nd verified
	tm.MarkFileVerified("backend/api/v1/auth/token.py", "pytest passed")
	if !tm.CanComplete() {
		t.Error("expected CanComplete() = true when all tasks verified")
	}

	verified := tm.VerifiedFiles()
	if len(verified) != 2 {
		t.Errorf("expected 2 verified files, got %d", len(verified))
	}
}

func TestTaskMap_Persistence(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "taskmap.json")

	tm := taskmap.New("Test Goal")
	tm.AddOrUpdateFile("src/service.go", inventory.RoleSource, "task-001", taskmap.StatusPending)
	tm.AddEvidence("src/service.go", "go vet clean")

	if err := tm.Save(filePath); err != nil {
		t.Fatalf("failed to save taskmap: %v", err)
	}

	loaded, err := taskmap.Load(filePath)
	if err != nil {
		t.Fatalf("failed to load taskmap: %v", err)
	}

	if loaded.Goal != tm.Goal {
		t.Errorf("expected goal %q, got %q", tm.Goal, loaded.Goal)
	}
	if len(loaded.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(loaded.Files))
	}
	if loaded.Files["src/service.go"].Evidence[0] != "go vet clean" {
		t.Errorf("unexpected evidence: %v", loaded.Files["src/service.go"].Evidence)
	}
}
