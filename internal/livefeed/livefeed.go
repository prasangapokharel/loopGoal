package livefeed

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"loopgoal/internal/config"
	"loopgoal/internal/detect"
	"loopgoal/internal/hook"
)

// FeedPayload represents the schema written to .loopgoal/livefeed.json
type FeedPayload struct {
	Heartbeat         int         `json:"heartbeat"`
	UpdatedAt         string      `json:"updatedAt"`
	LatencyMs         string      `json:"latencyMs"`
	Status            string      `json:"status"`
	Stats             FeedStats   `json:"stats"`
	Errors            []FeedError `json:"errors"`
	CanCommit         bool        `json:"canCommit"`
	VerificationToken string      `json:"verificationToken,omitempty"`
}

// FeedStats aggregates summary counts and detected runtimes.
type FeedStats struct {
	TotalErrors int      `json:"totalErrors"`
	Runtimes    []string `json:"runtimes"`
}

// FeedError defines a condensed 5-line JSON error object.
type FeedError struct {
	Source  string `json:"source"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
	Col     int    `json:"col,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

// FeedFileName is the standard livefeed file relative to workDir.
const FeedFileName = ".loopgoal/livefeed.json"

// FeedFilePath returns the absolute path to livefeed.json.
func FeedFilePath(workDir string) string {
	return filepath.Join(workDir, FeedFileName)
}

// ReadFeed reads and parses the current .loopgoal/livefeed.json.
func ReadFeed(workDir string) (*FeedPayload, error) {
	p := FeedFilePath(workDir)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var feed FeedPayload
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("parsing livefeed.json: %w", err)
	}
	return &feed, nil
}

// IsFeedFresh checks if the livefeed was updated within maxAge.
func IsFeedFresh(feed *FeedPayload, maxAge time.Duration) bool {
	if feed == nil || feed.UpdatedAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, feed.UpdatedAt)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, feed.UpdatedAt)
		if err != nil {
			return false
		}
	}
	return time.Since(t) <= maxAge
}

// WriteFeedAtomic writes payload to livefeed.json atomically via temporary file rename.
func WriteFeedAtomic(workDir string, payload FeedPayload) error {
	p := FeedFilePath(workDir)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling livefeed: %w", err)
	}

	tempFile := p + ".tmp"
	if err := os.WriteFile(tempFile, data, 0o644); err != nil {
		return fmt.Errorf("writing temp livefeed: %w", err)
	}

	if err := os.Rename(tempFile, p); err != nil {
		return fmt.Errorf("atomic rename to %s: %w", p, err)
	}
	return nil
}

// GenerateToken produces a cryptographic sha256 token string.
func GenerateToken(heartbeat int) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", time.Now().UnixNano(), hex.EncodeToString(b), heartbeat)))
	return hex.EncodeToString(hash[:])
}

var (
	goVetRegex = regexp.MustCompile(`^(.+?):(\d+):(\d+):\s*(.+)`)
	tscRegex   = regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s*error\s*(TS\d+):\s*(.+)`)
)

// ParseCompilerOutput parses compiler or linter outputs into condensed FeedError slices.
func ParseCompilerOutput(source string, output string) []FeedError {
	var errors []FeedError
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if source == "go" || source == "go-vet" {
			if m := goVetRegex.FindStringSubmatch(line); len(m) == 5 {
				l, _ := strconv.Atoi(m[2])
				c, _ := strconv.Atoi(m[3])
				errors = append(errors, FeedError{
					Source:  "go-vet",
					File:    m[1],
					Line:    l,
					Col:     c,
					Code:    "compiler",
					Message: m[4],
				})
				continue
			}
		}

		if source == "typescript" || source == "tsc" {
			if m := tscRegex.FindStringSubmatch(line); len(m) == 6 {
				l, _ := strconv.Atoi(m[2])
				c, _ := strconv.Atoi(m[3])
				errors = append(errors, FeedError{
					Source:  "tsc",
					File:    m[1],
					Line:    l,
					Col:     c,
					Code:    m[4],
					Message: m[5],
				})
				continue
			}
		}

		// Generic fallback error line
		if strings.Contains(strings.ToLower(line), "error") || strings.Contains(line, "FAIL") {
			errors = append(errors, FeedError{
				Source:  source,
				Message: line,
			})
		}
	}

	if len(errors) > 8 {
		errors = errors[:8]
	}
	return errors
}

// CheckWorkspace runs auto-detected or configured commands and updates livefeed and verified.token.
func CheckWorkspace(workDir string, heartbeat int) (*FeedPayload, error) {
	startTime := time.Now()
	cfgPath := filepath.Join(workDir, config.DefaultDir, config.DefaultConfigFile)
	var commands []string
	var runtimes []string

	if cfg, err := config.Load(cfgPath); err == nil && len(cfg.Verify) > 0 {
		commands = cfg.Verify
		runtimes = []string{"config"}
	} else {
		detected := detect.Detect(workDir)
		commands = detected.VerifyCommands
		runtimes = []string{detected.Type}
	}

	var allErrors []FeedError
	for _, cmdStr := range commands {
		cmdStr = strings.TrimSpace(cmdStr)
		if cmdStr == "" {
			continue
		}
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = workDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			parsed := ParseCompilerOutput(runtimes[0], string(out))
			if len(parsed) == 0 {
				parsed = append(parsed, FeedError{
					Source:  runtimes[0],
					Message: strings.TrimSpace(string(out)),
				})
			}
			allErrors = append(allErrors, parsed...)
		}
	}

	latency := time.Since(startTime)
	isPassing := len(allErrors) == 0
	var token string
	if isPassing {
		token = GenerateToken(heartbeat)
	}

	payload := FeedPayload{
		Heartbeat: heartbeat,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339Nano),
		LatencyMs: fmt.Sprintf("%dms", latency.Milliseconds()),
		Status:    "failing",
		Stats: FeedStats{
			TotalErrors: len(allErrors),
			Runtimes:    runtimes,
		},
		Errors:            allErrors,
		CanCommit:         isPassing,
		VerificationToken: token,
	}
	if isPassing {
		payload.Status = "passing"
	}

	if err := WriteFeedAtomic(workDir, payload); err != nil {
		return nil, fmt.Errorf("writing livefeed: %w", err)
	}

	// Synchronize with verified.token
	if isPassing {
		_, _ = hook.WriteToken(workDir, fmt.Sprintf("Verified via LoopGoal Daemon (%s)", strings.Join(runtimes, ", ")))
	} else {
		_ = hook.ConsumeToken(workDir)
	}

	return &payload, nil
}

// RunDaemon executes the continuous debounced livefeed watcher loop.
func RunDaemon(workDir string, interval time.Duration, once bool, out io.Writer) error {
	if out == nil {
		out = os.Stdout
	}
	if interval < 500*time.Millisecond {
		interval = 1 * time.Second
	}

	heartbeat := 0
	for {
		heartbeat++
		feed, err := CheckWorkspace(workDir, heartbeat)
		if err != nil {
			fmt.Fprintf(out, "❌ [LoopGoal Daemon Error] %v\n", err)
		} else {
			if feed.CanCommit {
				fmt.Fprintf(out, "✓ [LoopGoal: PASSED] Heartbeat #%d (%s) — Git commit UNLOCKED\n", feed.Heartbeat, feed.LatencyMs)
			} else {
				fmt.Fprintf(out, "✗ [LoopGoal: BLOCKED] Heartbeat #%d (%s) — %d error(s) found\n", feed.Heartbeat, feed.LatencyMs, len(feed.Errors))
			}
		}

		if once {
			return nil
		}

		time.Sleep(interval)
	}
}
