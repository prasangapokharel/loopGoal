package detect

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProjectInfo contains detected project metadata and recommended verification commands.
type ProjectInfo struct {
	Type          string   `json:"type"`
	VerifyCommands []string `json:"verify_commands"`
	Description   string   `json:"description"`
}

// Detect inspects a project directory and suggests verification commands based on project files.
func Detect(dir string) ProjectInfo {
	// 1. Check Go
	if fileExists(filepath.Join(dir, "go.mod")) {
		return ProjectInfo{
			Type: "go",
			VerifyCommands: []string{
				"go test ./...",
				"go vet ./...",
			},
			Description: "Go module detected",
		}
	}

	// 2. Check Node / Next.js / TypeScript
	pkgJsonPath := filepath.Join(dir, "package.json")
	if fileExists(pkgJsonPath) {
		cmds := []string{}
		data, err := os.ReadFile(pkgJsonPath)
		if err == nil {
			var pkg struct {
				Scripts map[string]string `json:"scripts"`
			}
			if json.Unmarshal(data, &pkg) == nil && pkg.Scripts != nil {
				if _, ok := pkg.Scripts["lint"]; ok {
					cmds = append(cmds, "npm run lint")
				}
				if _, ok := pkg.Scripts["typecheck"]; ok {
					cmds = append(cmds, "npm run typecheck")
				}
				if _, ok := pkg.Scripts["test"]; ok {
					cmds = append(cmds, "npm test")
				}
			}
		}
		if len(cmds) == 0 {
			cmds = []string{"npm test"}
		}
		return ProjectInfo{
			Type:           "node",
			VerifyCommands: cmds,
			Description:    "Node.js / JavaScript / TypeScript project detected",
		}
	}

	// 3. Check Rust
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return ProjectInfo{
			Type: "rust",
			VerifyCommands: []string{
				"cargo test",
				"cargo check",
			},
			Description: "Rust Cargo project detected",
		}
	}

	// 4. Check Python
	if fileExists(filepath.Join(dir, "pyproject.toml")) ||
		fileExists(filepath.Join(dir, "pytest.ini")) ||
		fileExists(filepath.Join(dir, "requirements.txt")) {
		return ProjectInfo{
			Type: "python",
			VerifyCommands: []string{
				"pytest",
			},
			Description: "Python project detected",
		}
	}

	// Default generic fallback
	return ProjectInfo{
		Type: "generic",
		VerifyCommands: []string{
			"echo 'No verification command configured'",
		},
		Description: "Generic project",
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
