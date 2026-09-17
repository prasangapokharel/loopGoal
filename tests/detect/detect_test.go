package detect_test

import (
	"os"
	"path/filepath"
	"testing"

	"loopgoal/internal/detect"
)

func TestDetectGoProject(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module testmod\n"), 0o644)

	info := detect.Detect(dir)
	if info.Type != "go" {
		t.Errorf("expected type 'go', got %q", info.Type)
	}
	if len(info.VerifyCommands) != 2 || info.VerifyCommands[0] != "go test ./..." {
		t.Errorf("unexpected verify commands: %v", info.VerifyCommands)
	}
}

func TestDetectNodeProject(t *testing.T) {
	dir := t.TempDir()
	pkgJSON := `{
		"name": "my-app",
		"scripts": {
			"lint": "eslint .",
			"typecheck": "tsc --noEmit",
			"test": "vitest run"
		}
	}`
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644)

	info := detect.Detect(dir)
	if info.Type != "node" {
		t.Errorf("expected type 'node', got %q", info.Type)
	}
	if len(info.VerifyCommands) != 3 {
		t.Fatalf("expected 3 verify commands, got %d: %v", len(info.VerifyCommands), info.VerifyCommands)
	}
	if info.VerifyCommands[0] != "npm run lint" || info.VerifyCommands[1] != "npm run typecheck" || info.VerifyCommands[2] != "npm test" {
		t.Errorf("unexpected verify commands: %v", info.VerifyCommands)
	}
}

func TestDetectRustProject(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0o644)

	info := detect.Detect(dir)
	if info.Type != "rust" {
		t.Errorf("expected type 'rust', got %q", info.Type)
	}
	if len(info.VerifyCommands) < 1 || info.VerifyCommands[0] != "cargo test" {
		t.Errorf("unexpected verify commands: %v", info.VerifyCommands)
	}
}

func TestDetectPythonProject(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[tool.pytest]\n"), 0o644)

	info := detect.Detect(dir)
	if info.Type != "python" {
		t.Errorf("expected type 'python', got %q", info.Type)
	}
	if len(info.VerifyCommands) != 1 || info.VerifyCommands[0] != "pytest" {
		t.Errorf("unexpected verify commands: %v", info.VerifyCommands)
	}
}

func TestDetectGenericFallback(t *testing.T) {
	dir := t.TempDir()
	info := detect.Detect(dir)
	if info.Type != "generic" {
		t.Errorf("expected type 'generic', got %q", info.Type)
	}
}
