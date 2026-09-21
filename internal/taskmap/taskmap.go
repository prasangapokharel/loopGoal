// Package taskmap manages the bipartite Task-to-File and File-to-Task graph,
// requirement tracking, and evidence collection for LoopGoal.
package taskmap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"loopgoal/internal/inventory"
)

// Task and file execution status constants.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusVerified   = "verified"
	StatusFailed     = "failed"
	StatusMissing    = "missing"
	StatusUnknown    = "unknown"
)

// FileItem represents a tracked file in the repository task map.
type FileItem struct {
	Path            string    `json:"path"`
	Role            string    `json:"role"`
	TaskIDs         []string  `json:"task_ids,omitempty"`
	Status          string    `json:"status"` // pending, in_progress, verified, failed, missing, unknown
	ExpectedChanges string    `json:"expected_changes,omitempty"`
	ActualChanges   string    `json:"actual_changes,omitempty"`
	Evidence        []string  `json:"evidence,omitempty"`
	LastVerifiedAt  time.Time `json:"last_verified_at,omitempty"`
}

// TaskItem represents a single bounded requirement / unit of work.
type TaskItem struct {
	ID           string    `json:"id"`
	Requirement  string    `json:"requirement"`
	Files        []string  `json:"files"`
	Status       string    `json:"status"` // pending, in_progress, verified, failed, missing, unknown
	Dependencies []string  `json:"dependencies,omitempty"`
	Evidence     []string  `json:"evidence,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// TaskMap represents the complete bipartite graph of requirements, tasks, files, and evidence.
type TaskMap struct {
	Goal      string                `json:"goal"`
	RunID     string                `json:"run_id,omitempty"`
	Files     map[string]*FileItem  `json:"files"` // relative path -> FileItem
	Tasks     map[string]*TaskItem  `json:"tasks"` // task ID -> TaskItem
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

// New creates an empty TaskMap for a given goal.
func New(goal string) *TaskMap {
	now := time.Now().UTC()
	return &TaskMap{
		Goal:      goal,
		Files:     make(map[string]*FileItem),
		Tasks:     make(map[string]*TaskItem),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// BuildFromInventory populates a TaskMap using an existing repository inventory and goal analysis.
func BuildFromInventory(inv *inventory.Inventory, goal string) *TaskMap {
	tm := New(goal)
	if inv == nil {
		return tm
	}

	targetPatterns := ExtractTargetPatterns(goal)

	for path, fi := range inv.Files {
		if fi.Role == inventory.RoleIgnored || fi.Role == inventory.RoleInstruction {
			continue
		}

		isTarget := false
		if len(targetPatterns) > 0 {
			for _, pat := range targetPatterns {
				cleanPat := strings.Trim(pat, "()/\"'`")
				if strings.Contains(strings.ToLower(path), strings.ToLower(cleanPat)) {
					isTarget = true
					break
				}
			}
		}

		if isTarget {
			tm.Files[path] = &FileItem{
				Path:   path,
				Role:   fi.Role,
				Status: StatusPending,
			}
			taskID := fmt.Sprintf("task-%03d", len(tm.Tasks)+1)
			tm.Tasks[taskID] = &TaskItem{
				ID:          taskID,
				Requirement: fmt.Sprintf("Refactor / implement %s", path),
				Files:       []string{path},
				Status:      StatusPending,
				CreatedAt:   time.Now().UTC(),
				UpdatedAt:   time.Now().UTC(),
			}
			tm.Files[path].TaskIDs = []string{taskID}
		}
	}

	return tm
}

// ExtractTargetPatterns searches a goal text for explicit directory or file path mentions.
func ExtractTargetPatterns(goal string) []string {
	var patterns []string
	// Match words with slashes or extensions (e.g. backend/api/v1/auth/, src/auth/, auth.go)
	re := regexp.MustCompile(`[a-zA-Z0-9_\-\.\/]+(?:/[a-zA-Z0-9_\-\.\/]*|\.[a-zA-Z0-9]+)`)
	matches := re.FindAllString(goal, -1)
	for _, m := range matches {
		m = strings.Trim(m, "()[]{}<>,;:'\"`")
		if strings.Contains(m, "/") || strings.Contains(m, ".") {
			patterns = append(patterns, m)
		}
	}
	return patterns
}

// CanComplete checks if all requirements and evidence gates are satisfied.
func (tm *TaskMap) CanComplete() bool {
	// Rule 1: No file may be in Failed or Missing state
	for _, fi := range tm.Files {
		if fi.Status == StatusFailed || fi.Status == StatusMissing {
			return false
		}
	}

	// Rule 2: If explicit tasks are registered, all tasks must be verified
	for _, ti := range tm.Tasks {
		if ti.Status != StatusVerified {
			return false
		}
	}

	return true
}

// AllVerified returns true if all active tasks are verified.
func (tm *TaskMap) AllVerified() bool {
	return tm.CanComplete()
}

// PendingFiles returns all file paths that are not yet verified.
func (tm *TaskMap) PendingFiles() []string {
	var list []string
	for p, fi := range tm.Files {
		if fi.Status == StatusPending || fi.Status == StatusInProgress || fi.Status == StatusMissing {
			list = append(list, p)
		}
	}
	sort.Strings(list)
	return list
}

// VerifiedFiles returns all verified file paths.
func (tm *TaskMap) VerifiedFiles() []string {
	var list []string
	for p, fi := range tm.Files {
		if fi.Status == StatusVerified {
			list = append(list, p)
		}
	}
	sort.Strings(list)
	return list
}

// MissingFiles returns all files marked missing.
func (tm *TaskMap) MissingFiles() []string {
	var list []string
	for p, fi := range tm.Files {
		if fi.Status == StatusMissing {
			list = append(list, p)
		}
	}
	sort.Strings(list)
	return list
}

// AddOrUpdateFile ensures a file is registered with the specified status and role.
func (tm *TaskMap) AddOrUpdateFile(path string, role string, taskID string, status string) *FileItem {
	rel := filepath.ToSlash(filepath.Clean(path))
	item, ok := tm.Files[rel]
	if !ok {
		item = &FileItem{
			Path:   rel,
			Role:   role,
			Status: status,
		}
		tm.Files[rel] = item
	} else {
		if status != "" {
			item.Status = status
		}
		if role != "" && item.Role == "" {
			item.Role = role
		}
	}

	if taskID != "" {
		hasTask := false
		for _, tid := range item.TaskIDs {
			if tid == taskID {
				hasTask = true
				break
			}
		}
		if !hasTask {
			item.TaskIDs = append(item.TaskIDs, taskID)
		}
	}

	tm.UpdatedAt = time.Now().UTC()
	return item
}

// AddEvidence attaches a verifiable evidence snippet to a file.
func (tm *TaskMap) AddEvidence(filePath string, evidence string) {
	rel := filepath.ToSlash(filepath.Clean(filePath))
	item, ok := tm.Files[rel]
	if !ok {
		item = tm.AddOrUpdateFile(rel, inventory.ClassifyRole(rel), "", StatusUnknown)
	}
	item.Evidence = append(item.Evidence, evidence)
	tm.UpdatedAt = time.Now().UTC()
}

// MarkFileVerified sets a file to verified status with evidence and timestamp.
func (tm *TaskMap) MarkFileVerified(filePath string, evidence string) {
	rel := filepath.ToSlash(filepath.Clean(filePath))
	item, ok := tm.Files[rel]
	if !ok {
		item = tm.AddOrUpdateFile(rel, inventory.ClassifyRole(rel), "", StatusVerified)
	}
	item.Status = StatusVerified
	item.LastVerifiedAt = time.Now().UTC()
	if evidence != "" {
		item.Evidence = append(item.Evidence, evidence)
	}

	// Also update associated tasks if all files for that task are now verified
	for _, tid := range item.TaskIDs {
		if task, ok := tm.Tasks[tid]; ok {
			allFilesDone := true
			for _, tf := range task.Files {
				if fItem, exists := tm.Files[tf]; !exists || fItem.Status != StatusVerified {
					allFilesDone = false
					break
				}
			}
			if allFilesDone {
				task.Status = StatusVerified
				task.UpdatedAt = time.Now().UTC()
			}
		}
	}

	tm.UpdatedAt = time.Now().UTC()
}

// Summary returns a human-readable progress overview.
func (tm *TaskMap) Summary() string {
	var sb strings.Builder
	total := len(tm.Files)
	verified := len(tm.VerifiedFiles())
	pending := len(tm.PendingFiles())
	missing := len(tm.MissingFiles())

	sb.WriteString(fmt.Sprintf("TaskMap: %d files total | %d verified | %d pending | %d missing\n",
		total, verified, pending, missing))

	if pending > 0 {
		sb.WriteString("Pending queue:\n")
		pList := tm.PendingFiles()
		limit := 10
		for i, p := range pList {
			if i >= limit {
				sb.WriteString(fmt.Sprintf("  ... and %d more pending files\n", len(pList)-limit))
				break
			}
			st := tm.Files[p].Status
			sb.WriteString(fmt.Sprintf("  - [%s] %s (%s)\n", st, p, tm.Files[p].Role))
		}
	}

	return sb.String()
}

// Save atomically persists the task map to disk.
func (tm *TaskMap) Save(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating taskmap directory: %w", err)
	}

	tm.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(tm, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling taskmap: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, "taskmap-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary taskmap file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temporary taskmap file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing temporary taskmap file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temporary taskmap file: %w", err)
	}

	if err := os.Rename(tmpName, filePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomically renaming taskmap file: %w", err)
	}

	return nil
}

// Load reads a TaskMap from disk.
func Load(filePath string) (*TaskMap, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var tm TaskMap
	if err := json.Unmarshal(data, &tm); err != nil {
		return nil, fmt.Errorf("unmarshaling taskmap: %w", err)
	}

	if tm.Files == nil {
		tm.Files = make(map[string]*FileItem)
	}
	if tm.Tasks == nil {
		tm.Tasks = make(map[string]*TaskItem)
	}

	return &tm, nil
}
