package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Recognized status values.
const (
	StatusIdle        = "idle"
	StatusRunning     = "running"
	StatusCompleted   = "completed"
	StatusBlocked     = "blocked"
	StatusFailed      = "failed"
	StatusGoalReached = "goal_reached"
	StatusStopped     = "stopped"
)

// State tracks the persistent execution state of LoopGoal.
type State struct {
	Goal           string    `json:"goal"`
	Iteration      int       `json:"iteration"`
	Status         string    `json:"status"`
	LastTask       string    `json:"last_task"`
	LastCommit     string    `json:"last_commit"`
	LastCheck      string    `json:"last_check,omitempty"`
	RemainingQueue []string  `json:"remaining_queue,omitempty"`
	StartedAt      time.Time `json:"started_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	PID            int       `json:"pid,omitempty"`
}

// UnmarshalJSON implements custom JSON unmarshaling to support both snake_case and camelCase keys.
func (s *State) UnmarshalJSON(data []byte) error {
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	getString := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				if str, ok := v.(string); ok {
					return str
				}
			}
		}
		return ""
	}

	getInt := func(keys ...string) int {
		for _, k := range keys {
			if v, ok := raw[k]; ok {
				if num, ok := v.(float64); ok {
					return int(num)
				}
			}
		}
		return 0
	}

	getTime := func(keys ...string) time.Time {
		str := getString(keys...)
		if str == "" {
			return time.Time{}
		}
		for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05Z", "2006-01-02 15:04:05"} {
			if t, err := time.Parse(layout, str); err == nil {
				return t
			}
		}
		return time.Time{}
	}

	s.Goal = getString("goal", "Goal")
	s.Status = getString("status", "Status")
	s.LastTask = getString("last_task", "lastTask", "LastTask")
	s.LastCommit = getString("last_commit", "lastCommit", "LastCommit")
	s.LastCheck = getString("last_check", "lastCheck", "LastCheck")
	s.Iteration = getInt("iteration", "Iteration")
	s.PID = getInt("pid", "PID")
	s.StartedAt = getTime("started_at", "startedTime", "StartedAt", "startedAt")
	s.UpdatedAt = getTime("updated_at", "updatedTime", "UpdatedAt", "updatedAt")

	// Parse remaining queue if present
	for _, k := range []string{"remaining_queue", "pending_files", "pendingFiles", "remainingTasks"} {
		if v, ok := raw[k]; ok {
			if list, ok := v.([]interface{}); ok {
				var items []string
				for _, item := range list {
					if str, ok := item.(string); ok {
						items = append(items, str)
					}
				}
				s.RemainingQueue = items
				break
			}
		}
	}

	return nil
}

// Manager handles reading and atomically persisting state.
type Manager struct {
	filePath string
}

// NewManager creates a state manager for the given state file path.
func NewManager(filePath string) *Manager {
	return &Manager{filePath: filePath}
}

// Init creates a fresh initial state file if it does not already exist.
func (m *Manager) Init(goal string) (*State, error) {
	if _, err := os.Stat(m.filePath); err == nil {
		return nil, errors.New("state file already exists")
	}

	now := time.Now().UTC()
	st := &State{
		Goal:      goal,
		Iteration: 0,
		Status:    StatusIdle,
		StartedAt: now,
		UpdatedAt: now,
	}

	if err := m.Save(st); err != nil {
		return nil, err
	}
	return st, nil
}

// Load reads the state from disk.
func (m *Manager) Load() (*State, error) {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("state file not found at %s: %w", m.filePath, err)
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("unmarshaling state file: %w", err)
	}
	return &st, nil
}

// Save atomically writes the state to disk using a temporary file and rename.
func (m *Manager) Save(st *State) error {
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}

	st.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	// Write to temp file first in the same directory for atomic rename
	tmpFile, err := os.CreateTemp(dir, "state-*.json.tmp")
	if err != nil {
		return fmt.Errorf("creating temporary state file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("writing temporary state file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("syncing temporary state file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("closing temporary state file: %w", err)
	}

	if err := os.Rename(tmpName, m.filePath); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomically renaming state file: %w", err)
	}

	return nil
}

// Exists checks if the state file exists.
func (m *Manager) Exists() bool {
	_, err := os.Stat(m.filePath)
	return err == nil
}
