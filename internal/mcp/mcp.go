package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"loopgoal/internal/config"
	"loopgoal/internal/git"
	"loopgoal/internal/hook"
	"loopgoal/internal/inventory"
	"loopgoal/internal/state"
	"loopgoal/internal/verify"
)

// Server coordinates the MCP JSON-RPC protocol over stdio.
type Server struct {
	workDir  string
	stateMgr *state.Manager
	git      *git.Git
	verifier *verify.Runner
	in       io.Reader
	out      io.Writer
}

// NewServer creates a new LoopGoal MCP server.
func NewServer(workDir string, in io.Reader, out io.Writer) *Server {
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	statePath := filepath.Join(workDir, config.DefaultDir, config.DefaultStateFile)
	return &Server{
		workDir:  workDir,
		stateMgr: state.NewManager(statePath),
		git:      git.New(workDir),
		verifier: verify.NewRunner(workDir),
		in:       in,
		out:      out,
	}
}

// JSON-RPC 2.0 types
type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type toolDefinition struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type toolCallResult struct {
	Content []toolContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Serve reads JSON-RPC messages line by line and dispatches responses.
func (s *Server) Serve(ctx context.Context) error {
	reader := bufio.NewReader(s.in)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		trimmed := strings.TrimSpace(string(line))
		if trimmed == "" {
			continue
		}

		var req jsonrpcRequest
		if err := json.Unmarshal([]byte(trimmed), &req); err != nil {
			continue
		}

		resp := s.handleRequest(ctx, req)
		if resp != nil {
			data, err := json.Marshal(resp)
			if err == nil {
				_, _ = fmt.Fprintf(s.out, "%s\n", data)
			}
		}
	}
}

func (s *Server) handleRequest(ctx context.Context, req jsonrpcRequest) *jsonrpcResponse {
	switch req.Method {
	case "initialize":
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities": map[string]interface{}{
					"tools": map[string]interface{}{},
				},
				"serverInfo": map[string]interface{}{
					"name":    "loopgoal",
					"version": "1.0.1",
				},
			},
		}

	case "notifications/initialized":
		return nil

	case "tools/list":
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.getTools(),
			},
		}

	case "tools/call":
		var params toolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return &jsonrpcResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32602, Message: "Invalid params"},
			}
		}

		res := s.executeTool(ctx, params.Name, params.Arguments)
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  res,
		}

	default:
		return &jsonrpcResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("Method '%s' not found", req.Method)},
		}
	}
}

func (s *Server) getTools() []toolDefinition {
	return []toolDefinition{
		{
			Name:        "loopgoal_status",
			Description: "Get the current LoopGoal autonomous loop state, active target, remaining file queue, and last commit.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "loopgoal_select_target",
			Description: "Lock a single file target for the current iteration. Out-of-scope modifications outside this target will be blocked.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file": map[string]interface{}{
						"type":        "string",
						"description": "Path to the target file to modify in this iteration.",
					},
					"objective": map[string]interface{}{
						"type":        "string",
						"description": "The specific, bounded improvement to implement.",
					},
				},
				"required": []string{"file"},
			},
		},
		{
			Name:        "loopgoal_verify",
			Description: "Execute project verification commands (tests, linters). If passed, generates a one-time cryptographic token unlocking git commit.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "loopgoal_commit",
			Description: "Commit verified iteration changes. Requires prior passing verification via loopgoal_verify.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"message": map[string]interface{}{
						"type":        "string",
						"description": "Commit message describing the bounded improvement.",
					},
				},
				"required": []string{"message"},
			},
		},
		{
			Name:        "loopgoal_rollback",
			Description: "Automatically restore working tree to clean state, discarding unverified or broken changes from the current iteration.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
		{
			Name:        "loopgoal_scan",
			Description: "Scan 100% of workspace inventory and return discovered files categorized by type.",
			InputSchema: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) executeTool(ctx context.Context, name string, args map[string]interface{}) toolCallResult {
	switch name {
	case "loopgoal_status":
		st, err := s.stateMgr.Load()
		if err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: "No LoopGoal state found. Run loopgoal init."}},
				IsError: true,
			}
		}
		data, _ := json.MarshalIndent(st, "", "  ")
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: string(data)}},
		}

	case "loopgoal_select_target":
		fileVal, _ := args["file"].(string)
		if fileVal == "" {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: "Error: 'file' parameter is required."}},
				IsError: true,
			}
		}
		st, _ := s.stateMgr.Load()
		if st == nil {
			st = &state.State{Status: state.StatusRunning}
		}
		st.ActiveTarget = fileVal
		if obj, ok := args["objective"].(string); ok && obj != "" {
			st.LastTask = obj
		}
		_ = s.stateMgr.Save(st)
		return toolCallResult{
			Content: []toolContent{{
				Type: "text",
				Text: fmt.Sprintf("[●] Target locked: %s\nOut-of-scope edits will be automatically blocked.", fileVal),
			}},
		}

	case "loopgoal_verify":
		cfgPath := filepath.Join(s.workDir, config.DefaultDir, config.DefaultConfigFile)
		cfg, err := config.Load(cfgPath)
		var verifyCmds []string
		if err == nil {
			verifyCmds = cfg.Verify
		}
		if len(verifyCmds) == 0 {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: "No verify commands configured in .loopgoal/config.yaml."}},
				IsError: true,
			}
		}

		summary, err := s.verifier.Run(ctx, verifyCmds)
		if err != nil {
			_ = hook.ConsumeToken(s.workDir)
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Verification failed: %v", err)}},
				IsError: true,
			}
		}

		if !summary.Passed {
			_ = hook.ConsumeToken(s.workDir)
			return toolCallResult{
				Content: []toolContent{{
					Type: "text",
					Text: fmt.Sprintf("❌ Verification FAILED on '%s':\n%s", summary.FailedCommand, summary.ErrorOutput()),
				}},
				IsError: true,
			}
		}

		token, err := hook.WriteToken(s.workDir, "Verified via MCP tool")
		if err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Error writing token: %v", err)}},
				IsError: true,
			}
		}

		return toolCallResult{
			Content: []toolContent{{
				Type: "text",
				Text: fmt.Sprintf("✓ Verification PASSED! One-time commit token generated: %s\nYou are unlocked to call loopgoal_commit.", token),
			}},
		}

	case "loopgoal_commit":
		msg, _ := args["message"].(string)
		if strings.TrimSpace(msg) == "" {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: "Error: 'message' parameter cannot be empty."}},
				IsError: true,
			}
		}

		if !hook.HasToken(s.workDir) {
			return toolCallResult{
				Content: []toolContent{{
					Type: "text",
					Text: "❌ [LoopGoal Blocker] Cannot commit: Verification has not passed!\nCall loopgoal_verify first.",
				}},
				IsError: true,
			}
		}

		// Stage changes
		changed, err := s.git.ChangedFiles(ctx)
		if err != nil || len(changed) == 0 {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: "No changes detected in working tree to commit."}},
				IsError: true,
			}
		}

		if err := s.git.Stage(ctx, changed...); err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Staging error: %v", err)}},
				IsError: true,
			}
		}

		hash, err := s.git.Commit(ctx, msg)
		_ = hook.ConsumeToken(s.workDir)
		if err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Commit error: %v", err)}},
				IsError: true,
			}
		}

		return toolCallResult{
			Content: []toolContent{{
				Type: "text",
				Text: fmt.Sprintf("✓ Committed successfully: %s\nCommit message: %s", hash, msg),
			}},
		}

	case "loopgoal_rollback":
		if err := s.git.Rollback(ctx); err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Rollback error: %v", err)}},
				IsError: true,
			}
		}
		_ = hook.ConsumeToken(s.workDir)
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: "✓ Working tree restored to clean state."}},
		}

	case "loopgoal_scan":
		scanner := inventory.NewScanner(s.workDir)
		inv, err := scanner.Scan(ctx)
		if err != nil {
			return toolCallResult{
				Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Scan error: %v", err)}},
				IsError: true,
			}
		}
		return toolCallResult{
			Content: []toolContent{{
				Type: "text",
				Text: fmt.Sprintf("Repository Inventory:\n%s\nSource: %d | Tests: %d | Docs: %d", inv.Summary(), len(inv.SourceFiles), len(inv.TestFiles), len(inv.DocFiles)),
			}},
		}

	default:
		return toolCallResult{
			Content: []toolContent{{Type: "text", Text: fmt.Sprintf("Unknown tool: %s", name)}},
			IsError: true,
		}
	}
}
