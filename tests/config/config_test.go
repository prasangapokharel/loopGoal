package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"loopgoal/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid default config, got: %v", err)
	}
	if cfg.Limits.Iterations != config.DefaultIterations {
		t.Errorf("expected iterations %d, got %d", config.DefaultIterations, cfg.Limits.Iterations)
	}
	if cfg.Limits.MaxRetries != config.DefaultMaxRetries {
		t.Errorf("expected max_retries %d, got %d", config.DefaultMaxRetries, cfg.Limits.MaxRetries)
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".loopgoal", "config.yaml")

	original := config.DefaultConfig()
	original.Goal = "Custom Goal"
	original.Agent.Command = "test-agent"
	original.Verify = []string{"npm test"}
	original.Limits.Iterations = 10

	if err := config.Save(configPath, original); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Goal != original.Goal {
		t.Errorf("expected goal %q, got %q", original.Goal, loaded.Goal)
	}
	if loaded.Agent.Command != original.Agent.Command {
		t.Errorf("expected agent command %q, got %q", original.Agent.Command, loaded.Agent.Command)
	}
	if len(loaded.Verify) != 1 || loaded.Verify[0] != "npm test" {
		t.Errorf("expected verify ['npm test'], got %v", loaded.Verify)
	}
	if loaded.Limits.Iterations != 10 {
		t.Errorf("expected iterations 10, got %d", loaded.Limits.Iterations)
	}
}

func TestValidateEmptyFields(t *testing.T) {
	cfg := &config.Config{}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error on empty config, got nil")
	}

	cfg.Goal = "Test Goal"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error on empty agent command, got nil")
	}

	cfg.Agent.Command = "codex"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(os.TempDir(), "nonexistent_file_12345.yaml"))
	if err == nil {
		t.Fatal("expected error loading non-existent file, got nil")
	}
}
