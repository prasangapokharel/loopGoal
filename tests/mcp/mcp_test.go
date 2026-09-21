package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"loopgoal/internal/mcp"
)

func setupTestGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	for _, args := range [][]string{
		{"init"},
		{"config", "user.name", "MCP Tester"},
		{"config", "user.email", "mcp@loopgoal.local"},
		{"config", "commit.gpgsign", "false"},
	} {
		c := exec.Command("git", args...)
		c.Dir = dir
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %s (%v)", args, string(out), err)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# MCP Test Repo\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	c := exec.Command("git", "add", "README.md")
	c.Dir = dir
	_ = c.Run()
	c = exec.Command("git", "commit", "-m", "initial")
	c.Dir = dir
	_ = c.Run()

	return dir
}

func TestMCPServerTools(t *testing.T) {
	dir := setupTestGitRepo(t)

	// Create .loopgoal/config.yaml
	loopDir := filepath.Join(dir, ".loopgoal")
	_ = os.MkdirAll(loopDir, 0o755)
	_ = os.WriteFile(filepath.Join(loopDir, "config.yaml"), []byte("verify:\n  - echo 'ok'\n"), 0o644)
	_ = os.WriteFile(filepath.Join(loopDir, "state.json"), []byte(`{"goal":"MCP Test","iteration":1,"status":"running"}`), 0o644)

	// Simulate MCP client input
	requests := []string{
		// 1. initialize
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		// 2. tools/list
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		// 3. call loopgoal_status
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"loopgoal_status","arguments":{}}}`,
		// 4. call loopgoal_select_target
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"loopgoal_select_target","arguments":{"file":"README.md","objective":"Add badge"}}}`,
		// 5. call loopgoal_verify
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"loopgoal_verify","arguments":{}}}`,
		// 6. call loopgoal_scan
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"loopgoal_scan","arguments":{}}}`,
	}

	inBuf := bytes.NewBufferString(strings.Join(requests, "\n") + "\n")
	var outBuf bytes.Buffer

	server := mcp.NewServer(dir, inBuf, &outBuf)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := server.Serve(ctx); err != nil {
		t.Fatalf("Serve error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) < len(requests) {
		t.Fatalf("expected at least %d responses, got %d. Output: %s", len(requests), len(lines), outBuf.String())
	}

	for i, line := range lines {
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Errorf("line %d invalid JSON: %s (%v)", i+1, line, err)
			continue
		}
		if resp["error"] != nil {
			t.Errorf("line %d returned error: %v", i+1, resp["error"])
		}
	}
}
