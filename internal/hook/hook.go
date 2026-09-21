package hook

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// VerifiedToken contains metadata verifying that tests passed.
type VerifiedToken struct {
	Token     string    `json:"token"`
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
}

// TokenFile is the relative path to the verified token.
const TokenFile = ".loopgoal/verified.token"

// TokenPath returns the absolute path to the token file.
func TokenPath(workDir string) string {
	return filepath.Join(workDir, TokenFile)
}

// WriteToken generates and writes a one-time verification token.
func WriteToken(workDir string, summary string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}
	tokenStr := hex.EncodeToString(bytes)

	tok := VerifiedToken{
		Token:     tokenStr,
		Timestamp: time.Now().UTC(),
		Summary:   summary,
	}

	p := TokenPath(workDir)
	_ = os.MkdirAll(filepath.Dir(p), 0o755)

	data, err := json.MarshalIndent(tok, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", fmt.Errorf("writing verified token: %w", err)
	}
	return tokenStr, nil
}

// HasToken checks if a valid verification token currently exists.
func HasToken(workDir string) bool {
	p := TokenPath(workDir)
	_, err := os.Stat(p)
	return err == nil
}

// ConsumeToken deletes the token once a commit has been safely made.
func ConsumeToken(workDir string) error {
	p := TokenPath(workDir)
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

const PreCommitHookScript = `#!/usr/bin/env bash
# LoopGoal Deterministic Hard Enforcement Hook
# Prevents unverified commits by AI agents and commands

if [ -f ".loopgoal/state.json" ] || [ -f ".loopgoal/config.yaml" ]; then
  if [ ! -f ".loopgoal/verified.token" ]; then
    echo "❌ [LoopGoal Blocker] You cannot commit! Verification has NOT passed." >&2
    echo "   Required step: Run 'loopgoal verify' or pass project verification first." >&2
    exit 1
  fi
  # Verification verified. Consume the token on commit.
  rm -f ".loopgoal/verified.token"
fi
exit 0
`

const PrePushHookScript = `#!/usr/bin/env bash
# LoopGoal Remote Push Lock Hook
# Strictly enforces local development: zero silent/unauthorized remote pushes

if [ -f ".loopgoal/state.json" ] || [ -f ".loopgoal/config.yaml" ]; then
  echo "❌ [LoopGoal Blocker] Remote git push is locked by LoopGoal safety policy." >&2
  echo "   All commits must remain local on this machine until manual verification." >&2
  echo "   To push manually, uninstall the hook via 'loopgoal hook remove'." >&2
  exit 1
fi
exit 0
`

// Install installs the git pre-commit and pre-push hooks into .git/hooks/
func Install(workDir string) error {
	gitDir := filepath.Join(workDir, ".git")
	if stat, err := os.Stat(gitDir); err != nil || !stat.IsDir() {
		return fmt.Errorf("not a git repository: %s", workDir)
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("creating hooks directory: %w", err)
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	if err := os.WriteFile(preCommitPath, []byte(PreCommitHookScript), 0o755); err != nil {
		return fmt.Errorf("writing pre-commit hook: %w", err)
	}

	prePushPath := filepath.Join(hooksDir, "pre-push")
	if err := os.WriteFile(prePushPath, []byte(PrePushHookScript), 0o755); err != nil {
		return fmt.Errorf("writing pre-push hook: %w", err)
	}

	return nil
}

// Remove removes the installed hooks.
func Remove(workDir string) error {
	hooksDir := filepath.Join(workDir, ".git", "hooks")
	_ = os.Remove(filepath.Join(hooksDir, "pre-commit"))
	_ = os.Remove(filepath.Join(hooksDir, "pre-push"))
	return nil
}

// Status reports whether the hooks are installed.
func Status(workDir string) (bool, bool) {
	hooksDir := filepath.Join(workDir, ".git", "hooks")
	_, errCommit := os.Stat(filepath.Join(hooksDir, "pre-commit"))
	_, errPush := os.Stat(filepath.Join(hooksDir, "pre-push"))
	return errCommit == nil, errPush == nil
}

// WrapperScript returns the wrapper bash script for proxying git
func WrapperScript() string {
	return `#!/usr/bin/env bash
# loopgoal-git: Deterministic Git Proxy Wrapper
if [[ "$1" == "commit" ]]; then
  if [ ! -f ".loopgoal/verified.token" ]; then
    echo "❌ [LoopGoal Blocker] You cannot commit! Verification has NOT passed." >&2
    echo "Required step: Run 'loopgoal verify' first." >&2
    exit 1
  fi
fi

if [[ "$1" == "push" ]]; then
  echo "❌ [LoopGoal Blocker] Remote git push is locked by LoopGoal safety policy." >&2
  exit 1
fi

exec git "$@"
`
}
