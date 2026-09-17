package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected valid default config, got: %v", err)
	}
	if cfg.Limits.Iterations != DefaultIterations {
		t.Errorf("expected iterations %d, got %d", DefaultIterations, cfg.Limits.Iterations)
	}
	if cfg.Limits.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected max_retries %d, got %d", DefaultMaxRetries, cfg.Limits.MaxRetries)
	}
}

func TestLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".loopgoal", "config.yaml")

	original := DefaultConfig()
	original.Goal = "Custom Goal"
	original.Agent.Command = "test-agent"
	original.Verify = []string{"npm test"}
	original.Limits.Iterations = 10

	if err := Save(configPath, original); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := Load(configPath)
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
	cfg := &Config{}
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
	_, err := Load(filepath.Join(os.TempDir(), "nonexistent_file_12345.yaml"))
	if err == nil {
		t.Fatal("expected error loading non-existent file, got nil")
	}
}
