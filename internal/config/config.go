package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"loopgoal/internal/resolve"
)

const (
	DefaultDir        = ".loopgoal"
	DefaultConfigFile = "config.yaml"
	DefaultStateFile  = "state.json"
	DefaultIterations = 20
	DefaultMaxRetries = 3
)

// Config represents the LoopGoal project configuration.
type Config struct {
	Goal   string       `yaml:"goal"`
	Agent  AgentConfig  `yaml:"agent"`
	Verify []string     `yaml:"verify"`
	Limits LimitsConfig `yaml:"limits"`
}

// AgentConfig holds agent execution configuration.
type AgentConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args,omitempty"`
}

// LimitsConfig holds loop execution thresholds.
type LimitsConfig struct {
	Iterations int `yaml:"iterations"`
	MaxRetries int `yaml:"max_retries,omitempty"`
}

// DefaultConfig returns a starter configuration.
func DefaultConfig() *Config {
	return &Config{
		Goal: "Continuously improve this project with small, safe, production-quality changes.",
		Agent: AgentConfig{
			Command: "agy",
			Args:    resolve.HeadlessArgs("agy"),
		},
		Verify: []string{
			"go test ./...",
		},
		Limits: LimitsConfig{
			Iterations: DefaultIterations,
			MaxRetries: DefaultMaxRetries,
		},
	}
}

// DetectConfig returns a configuration tailored to the repository in projectDir.
// It automatically populates the correct headless args for known AI agents.
func DetectConfig(projectDir string, verifyCommands []string) *Config {
	cfg := DefaultConfig()
	if len(verifyCommands) > 0 {
		cfg.Verify = verifyCommands
	}
	return cfg
}

// ConfigForAgent returns a config preset for the specified agent command,
// including the correct headless args required for non-interactive execution.
func ConfigForAgent(command string) *Config {
	cfg := DefaultConfig()
	cfg.Agent.Command = command
	cfg.Agent.Args = resolve.HeadlessArgs(command)
	return cfg
}

// DefaultConfigYAML returns the formatted default YAML string.
func DefaultConfigYAML() string {
	return FormatConfigYAML(DefaultConfig())
}

// FormatConfigYAML converts a Config into formatted, well-commented YAML.
func FormatConfigYAML(cfg *Config) string {
	var sb strings.Builder
	sb.WriteString("# LoopGoal Configuration\n")
	sb.WriteString("goal: >\n")
	for _, line := range strings.Split(strings.TrimSpace(cfg.Goal), "\n") {
		sb.WriteString("  " + strings.TrimSpace(line) + "\n")
	}
	sb.WriteString("\nagent:\n")
	sb.WriteString(fmt.Sprintf("  command: %q\n", cfg.Agent.Command))
	if len(cfg.Agent.Args) > 0 {
		sb.WriteString("  args:\n")
		for _, a := range cfg.Agent.Args {
			sb.WriteString(fmt.Sprintf("    - %q\n", a))
		}
	}
	sb.WriteString("\nverify:\n")
	for _, v := range cfg.Verify {
		sb.WriteString(fmt.Sprintf("  - %q\n", v))
	}
	sb.WriteString(fmt.Sprintf("\nlimits:\n  iterations: %d\n  max_retries: %d\n", cfg.Limits.Iterations, cfg.Limits.MaxRetries))
	return sb.String()
}

// Load reads and parses a YAML configuration file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	cfg.applyDefaults()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

// Save writes the configuration to the specified path.
func Save(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}

	return nil
}

func (c *Config) applyDefaults() {
	if c.Limits.Iterations <= 0 {
		c.Limits.Iterations = DefaultIterations
	}
	if c.Limits.MaxRetries <= 0 {
		c.Limits.MaxRetries = DefaultMaxRetries
	}
}

// Validate checks the configuration for required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Goal) == "" {
		return errors.New("goal cannot be empty")
	}
	if strings.TrimSpace(c.Agent.Command) == "" {
		return errors.New("agent.command cannot be empty")
	}
	return nil
}
