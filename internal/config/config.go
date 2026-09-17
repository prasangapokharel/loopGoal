package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
			Command: "codex",
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

// DefaultConfigYAML returns the formatted default YAML string.
func DefaultConfigYAML() string {
	return `# LoopGoal Configuration
goal: >
  Continuously improve this project with small,
  safe, production-quality changes.

agent:
  command: "codex"

verify:
  - "go test ./..."

limits:
  iterations: 20
  max_retries: 3
`
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
