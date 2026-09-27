package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MRKups/jev-usecase-1/internal/config"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	if cfg.ActiveGenerator != "openai" {
		t.Errorf("expected default active generator 'openai', got %q", cfg.ActiveGenerator)
	}
	if cfg.EngineA.Provider != "jev" {
		t.Errorf("expected default EngineA provider 'jev', got %q", cfg.EngineA.Provider)
	}
	if cfg.EngineB.Provider != "openai" {
		t.Errorf("expected default EngineB provider 'openai', got %q", cfg.EngineB.Provider)
	}
	if cfg.EngineB.Model != "gpt-4o-mini" {
		t.Errorf("expected default EngineB model 'gpt-4o-mini', got %q", cfg.EngineB.Model)
	}
	if cfg.EvaluationMode != "parallel" {
		t.Errorf("expected default EvaluationMode 'parallel', got %q", cfg.EvaluationMode)
	}
}

func TestSaveAndLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := config.Default()
	cfg.EngineA.Provider = "anthropic"
	cfg.EngineA.Model = "claude-3-5-haiku-latest"
	cfg.EngineB.Provider = "gemini"
	cfg.EngineB.Model = "gemini-2.5-flash"
	cfg.EvaluationMode = "serial"

	if err := cfg.SaveToFile(configPath); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.EngineA.Provider != "anthropic" {
		t.Errorf("expected loaded EngineA provider 'anthropic', got %q", loaded.EngineA.Provider)
	}
	if loaded.EngineA.Model != "claude-3-5-haiku-latest" {
		t.Errorf("expected loaded EngineA model 'claude-3-5-haiku-latest', got %q", loaded.EngineA.Model)
	}
	if loaded.EngineB.Provider != "gemini" {
		t.Errorf("expected loaded EngineB provider 'gemini', got %q", loaded.EngineB.Provider)
	}
	if loaded.EngineB.Model != "gemini-2.5-flash" {
		t.Errorf("expected loaded EngineB model 'gemini-2.5-flash', got %q", loaded.EngineB.Model)
	}
	if loaded.EvaluationMode != "serial" {
		t.Errorf("expected loaded EvaluationMode 'serial', got %q", loaded.EvaluationMode)
	}
}

func TestLegacyMigration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "legacy_config.json")

	legacyJSON := `{
		"addr": "127.0.0.1:9090",
		"active_generator": "anthropic",
		"jev": {
			"endpoint": "https://custom.typesafe.ai",
			"model": "jev-custom",
			"api_key": "sec-123"
		}
	}`

	if err := os.WriteFile(configPath, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("failed to write legacy config: %v", err)
	}

	loaded, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load legacy config: %v", err)
	}

	if loaded.EngineA.Provider != "jev" {
		t.Errorf("expected migrated EngineA provider 'jev', got %q", loaded.EngineA.Provider)
	}
	if loaded.EngineA.Endpoint != "https://custom.typesafe.ai" {
		t.Errorf("expected migrated EngineA endpoint 'https://custom.typesafe.ai', got %q", loaded.EngineA.Endpoint)
	}
	if loaded.EngineA.Model != "jev-custom" {
		t.Errorf("expected migrated EngineA model 'jev-custom', got %q", loaded.EngineA.Model)
	}
	if loaded.EngineA.APIKey != "sec-123" {
		t.Errorf("expected migrated EngineA api key 'sec-123', got %q", loaded.EngineA.APIKey)
	}
}

func TestLegacyExecutionModeMigration(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "legacy_exec_mode.json")

	legacyJSON := `{"execution_mode": "serial"}`
	if err := os.WriteFile(configPath, []byte(legacyJSON), 0644); err != nil {
		t.Fatalf("failed to write legacy execution mode config: %v", err)
	}

	loaded, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load legacy config: %v", err)
	}

	if loaded.EvaluationMode != "serial" {
		t.Errorf("expected migrated EvaluationMode 'serial', got %q", loaded.EvaluationMode)
	}
}
