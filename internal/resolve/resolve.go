// Package resolve finds AI agent executables across all major platforms and
// common non-standard install locations, and produces actionable error messages
// when a command cannot be found.
package resolve

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// knownAgents maps canonical agent names and their common aliases / binary names
// to a set of platform-specific extra search paths beyond $PATH.
var knownAgents = map[string]agentInfo{
	"antigravity": {
		aliases:     []string{"antigravity", "agy"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install Google Antigravity: https://antigravity.dev\n" +
			"   After install, add the binary to your PATH or set agent.command: \"agy\"",
		extraPaths: commonNodePaths("antigravity", "agy"),
	},
	"agy": {
		aliases:     []string{"agy", "antigravity"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install Google Antigravity (agy): https://antigravity.dev\n" +
			"   After install, the binary is usually at ~/.local/bin/agy or /usr/local/bin/agy",
		extraPaths: commonNodePaths("agy", "antigravity"),
	},
	"claude": {
		aliases:     []string{"claude"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install Claude Code CLI: https://docs.anthropic.com/claude-code\n" +
			"   Run: npm install -g @anthropic-ai/claude-code\n" +
			"   Then: claude --version",
		extraPaths: commonNodePaths("claude"),
	},
	"codex": {
		aliases:     []string{"codex"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install OpenAI Codex CLI: https://github.com/openai/codex\n" +
			"   Run: npm install -g @openai/codex\n" +
			"   Then: codex --version",
		extraPaths: commonNodePaths("codex"),
	},
	"opencode": {
		aliases:     []string{"opencode"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install OpenCode: https://opencode.ai\n" +
			"   Run: npm install -g opencode-ai   or   brew install opencode\n" +
			"   Then: opencode --version",
		extraPaths: commonNodePaths("opencode"),
	},
	"gemini": {
		aliases:     []string{"gemini"},
		headlessArgs: []string{"--dangerously-skip-permissions"},
		installHint: "Install Gemini CLI: https://github.com/google-gemini/gemini-cli\n" +
			"   Run: npm install -g @google/gemini-cli\n" +
			"   Then: gemini --version",
		extraPaths: commonNodePaths("gemini"),
	},
}

type agentInfo struct {
	aliases      []string
	headlessArgs []string
	installHint  string
	extraPaths   []string
}

// HeadlessArgs returns the CLI arguments required for the named agent to run
// non-interactively (headless mode). Returns nil if the agent is unknown or
// requires no special flags.
func HeadlessArgs(command string) []string {
	base := filepath.Base(command)
	if info, ok := knownAgents[base]; ok {
		return info.headlessArgs
	}
	return nil
}

// Resolve returns the absolute path to the given command. It searches:
//  1. The command as-is (handles absolute paths)
//  2. $PATH (standard exec.LookPath)
//  3. Common platform-specific install dirs for known AI agents
//
// If the command cannot be found, it returns a detailed, actionable error
// message including install instructions for known agents.
func Resolve(command string) (string, error) {
	// Absolute path — use directly if the file is executable.
	if filepath.IsAbs(command) {
		if isExecutable(command) {
			return command, nil
		}
		return "", fmt.Errorf("agent binary not found at absolute path: %s", command)
	}

	// Shell-command string (contains spaces) — no resolution needed; will run via sh -c.
	if strings.Contains(command, " ") {
		return command, nil
	}

	// Standard PATH lookup.
	if p, err := exec.LookPath(command); err == nil {
		return p, nil
	}

	// Extended search through known install dirs.
	baseName := filepath.Base(command)
	if resolved := searchKnownPaths(baseName); resolved != "" {
		return resolved, nil
	}

	// Build helpful error message.
	return "", buildNotFoundError(baseName)
}

// ResolveOrCommand returns the resolved path, or the original command string if
// resolution fails. It also returns the error so callers can surface it early.
func ResolveOrCommand(command string) (string, error) {
	resolved, err := Resolve(command)
	if err != nil {
		return command, err
	}
	return resolved, nil
}

// searchKnownPaths walks the extra search paths for a given binary name,
// including aliases registered for known AI agents.
func searchKnownPaths(name string) string {
	// Collect search paths: from the agent's own entry + any agent whose alias matches.
	paths := extraSearchPaths(name)

	candidates := []string{name}
	if info, ok := knownAgents[name]; ok {
		candidates = append(candidates, info.aliases...)
	}

	for _, dir := range paths {
		for _, candidate := range candidates {
			full := filepath.Join(dir, candidate)
			if runtime.GOOS == "windows" {
				for _, ext := range []string{".exe", ".cmd", ".bat"} {
					if isExecutable(full + ext) {
						return full + ext
					}
				}
			}
			if isExecutable(full) {
				return full
			}
		}
	}
	return ""
}

// extraSearchPaths returns the combined extra search paths for a given command
// name, merging paths from matched known agents.
func extraSearchPaths(name string) []string {
	seen := map[string]bool{}
	var dirs []string

	add := func(paths []string) {
		for _, p := range paths {
			if !seen[p] {
				seen[p] = true
				dirs = append(dirs, p)
			}
		}
	}

	if info, ok := knownAgents[name]; ok {
		add(info.extraPaths)
	}
	// Also check generic common dirs.
	add(genericSearchPaths())

	return dirs
}

// buildNotFoundError produces an actionable error with install hints.
func buildNotFoundError(name string) error {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("agent command %q not found in PATH\n\n", name))

	if info, ok := knownAgents[name]; ok {
		sb.WriteString("  Fix: ")
		sb.WriteString(info.installHint)
		sb.WriteString("\n\n")
	} else {
		sb.WriteString("  Fix: ensure the command is installed and its directory is in $PATH\n\n")
	}

	sb.WriteString("  Searched locations:\n")
	for _, p := range extraSearchPaths(name) {
		sb.WriteString(fmt.Sprintf("    %s\n", p))
	}

	sb.WriteString("\n  Alternative: specify the full absolute path in .loopgoal/config.yaml:\n")
	sb.WriteString("    agent:\n")
	if runtime.GOOS == "windows" {
		sb.WriteString(fmt.Sprintf("      command: \"C:\\\\Users\\\\YourUser\\\\AppData\\\\Roaming\\\\npm\\\\%s.cmd\"\n", name))
	} else {
		sb.WriteString(fmt.Sprintf("      command: \"/usr/local/bin/%s\"\n", name))
	}

	return fmt.Errorf("%s", sb.String())
}

// ─── platform-specific path builders ─────────────────────────────────────────

// commonNodePaths returns directories where npm global packages install their
// binaries across Linux, macOS, and Windows, for the given binary names.
func commonNodePaths(names ...string) []string {
	_ = names // names used conceptually; dirs are the same regardless of binary name
	home, _ := os.UserHomeDir()

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		localAppData := os.Getenv("LOCALAPPDATA")
		return []string{
			filepath.Join(appData, "npm"),
			filepath.Join(localAppData, "npm"),
			filepath.Join(localAppData, "Programs", "node"),
			filepath.Join(home, "AppData", "Roaming", "npm"),
			`C:\Program Files\nodejs`,
			`C:\Program Files (x86)\nodejs`,
		}
	case "darwin":
		return []string{
			"/usr/local/bin",
			"/opt/homebrew/bin",
			"/opt/homebrew/sbin",
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, "Library", "pnpm"),
			"/usr/local/lib/node_modules/.bin",
			filepath.Join(home, ".npm-global", "bin"),
			"/usr/local/share/npm/bin",
		}
	default: // linux and others
		return []string{
			"/usr/local/bin",
			"/usr/bin",
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".npm-global", "bin"),
			filepath.Join(home, ".kimi-code", "bin"),
			filepath.Join(home, "go", "bin"),
			"/snap/bin",
			"/usr/local/lib/node_modules/.bin",
			"/usr/lib/node_modules/.bin",
		}
	}
}

// genericSearchPaths returns common binary install directories for all platforms.
func genericSearchPaths() []string {
	home, _ := os.UserHomeDir()
	paths := []string{
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, "bin"),
		filepath.Join(home, "go", "bin"),
	}
	switch runtime.GOOS {
	case "windows":
		paths = append(paths,
			filepath.Join(home, "AppData", "Roaming", "npm"),
			filepath.Join(home, "AppData", "Local", "Programs"),
		)
	case "darwin":
		paths = append(paths,
			"/opt/homebrew/bin",
			"/usr/local/bin",
		)
	default:
		paths = append(paths,
			"/usr/local/bin",
			"/snap/bin",
		)
	}
	return paths
}

// isExecutable reports whether path points to a regular, executable file.
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		// On Windows, existence is sufficient — execution rights are determined by extension.
		return true
	}
	return info.Mode()&0o111 != 0
}
