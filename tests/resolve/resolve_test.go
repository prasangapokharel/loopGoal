package resolve_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"loopgoal/internal/resolve"
)

func TestResolve_AbsolutePath_Exists(t *testing.T) {
	// Create a temporary executable.
	dir := t.TempDir()
	bin := filepath.Join(dir, "myagent")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho ok"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolve.Resolve(bin)
	if err != nil {
		t.Fatalf("expected Resolve to succeed for absolute path, got: %v", err)
	}
	if got != bin {
		t.Errorf("expected %q, got %q", bin, got)
	}
}

func TestResolve_AbsolutePath_Missing(t *testing.T) {
	_, err := resolve.Resolve("/nonexistent/path/to/binary_xyz_not_real")
	if err == nil {
		t.Error("expected error for missing absolute path, got nil")
	}
}

func TestResolve_CommandOnPath(t *testing.T) {
	// "echo" or "true" are universally available on Linux/macOS.
	// On Windows "cmd" is always present.
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cmd"
	} else {
		cmd = "sh"
	}

	got, err := resolve.Resolve(cmd)
	if err != nil {
		t.Fatalf("expected to resolve %q from PATH, got: %v", cmd, err)
	}
	if got == "" {
		t.Error("expected non-empty resolved path")
	}
}

func TestResolve_UnknownCommand_ErrorContainsFix(t *testing.T) {
	_, err := resolve.Resolve("completelymadeupagent_xyz_not_real")
	if err == nil {
		t.Fatal("expected error for unknown command, got nil")
	}
	msg := err.Error()
	// Must contain the command name and a fix hint.
	if !strings.Contains(msg, "completelymadeupagent_xyz_not_real") {
		t.Errorf("error should mention command name, got: %s", msg)
	}
	if !strings.Contains(msg, "Fix:") && !strings.Contains(msg, "config.yaml") {
		t.Errorf("error should contain fix instructions, got: %s", msg)
	}
}

func TestResolve_KnownAgent_ErrorContainsInstallHint(t *testing.T) {
	// "antigravity" is a known agent — error must include install URL.
	if _, err := resolve.Resolve("antigravity"); err != nil {
		msg := err.Error()
		if !strings.Contains(msg, "antigravity") {
			t.Errorf("error should mention 'antigravity', got: %s", msg)
		}
		if !strings.Contains(msg, "https://") {
			t.Errorf("error should contain install URL, got: %s", msg)
		}
		if !strings.Contains(msg, ".loopgoal/config.yaml") {
			t.Errorf("error should mention config.yaml alternative, got: %s", msg)
		}
	}
	// If it's actually installed, that's fine too — skip assertion.
}

func TestResolve_KnownAgents_ErrorHintsForAll(t *testing.T) {
	knownNames := []string{"agy", "claude", "codex", "opencode", "gemini"}
	for _, name := range knownNames {
		t.Run(name, func(t *testing.T) {
			_, err := resolve.Resolve(name)
			if err == nil {
				t.Skipf("%s is installed on this system — skipping not-found test", name)
			}
			msg := err.Error()
			if !strings.Contains(msg, name) {
				t.Errorf("[%s] error should mention command name, got: %s", name, msg)
			}
			if !strings.Contains(msg, "https://") {
				t.Errorf("[%s] error should contain install URL, got: %s", name, msg)
			}
		})
	}
}

func TestResolve_ShellComposedCommand_Passthrough(t *testing.T) {
	// Commands with spaces should pass through without resolution (run via sh -c).
	cmd := "echo hello world"
	got, err := resolve.Resolve(cmd)
	if err != nil {
		t.Fatalf("shell command with spaces should not error, got: %v", err)
	}
	if got != cmd {
		t.Errorf("expected passthrough %q, got %q", cmd, got)
	}
}

func TestResolve_ResolveOrCommand_FallbackOnError(t *testing.T) {
	cmd := "nonexistent_bin_xyz_abc"
	got, err := resolve.ResolveOrCommand(cmd)
	// Must return the original command even on error.
	if got != cmd {
		t.Errorf("expected original command %q on error, got %q", cmd, got)
	}
	if err == nil {
		t.Error("expected error from ResolveOrCommand for missing binary")
	}
}

func TestResolve_ResolveOrCommand_Success(t *testing.T) {
	// Create a temporary executable in a temp dir and add it to PATH.
	dir := t.TempDir()
	bin := filepath.Join(dir, "mytestagent")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho ok"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Prepend dir to PATH so LookPath finds it.
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)

	got, err := resolve.ResolveOrCommand("mytestagent")
	if err != nil {
		t.Fatalf("expected success after adding to PATH, got: %v", err)
	}
	if got == "" || got == "mytestagent" {
		t.Errorf("expected resolved absolute path, got %q", got)
	}
}
