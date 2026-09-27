// Package config provides runtime configuration for LLM providers, Jev decision engine, and web storage.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// OpenAIConfig holds configuration for the OpenAI API generator.
type OpenAIConfig struct {
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

// AnthropicConfig holds configuration for the Anthropic API generator.
type AnthropicConfig struct {
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

// GeminiConfig holds configuration for the Google Gemini API generator.
type GeminiConfig struct {
	APIKey string `json:"api_key"`
	Model  string `json:"model"`
}

// JevConfig holds configuration for the TypeSafe Jev Decision Engine.
type JevConfig struct {
	Endpoint  string `json:"endpoint"`
	APIKey    string `json:"api_key"`
	Model     string `json:"model"`
	TimeoutMs int    `json:"timeout_ms"`
}

// DecisionEngineConfig specifies the provider and options for an engine slot.
type DecisionEngineConfig struct {
	Provider string `json:"provider"` // "jev", "openai", "anthropic", "gemini", or "none"
	Endpoint string `json:"endpoint,omitempty"`
	Model    string `json:"model,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

// AppConfig represents the unified configuration saved to config.json.
type AppConfig struct {
	Addr            string               `json:"addr"`
	ActiveGenerator string               `json:"active_generator"`
	OpenAI          OpenAIConfig         `json:"openai"`
	Anthropic       AnthropicConfig      `json:"anthropic"`
	Gemini          GeminiConfig         `json:"gemini"`
	Jev             JevConfig            `json:"jev"`
	EngineA         DecisionEngineConfig `json:"engine_a"`
	EngineB         DecisionEngineConfig `json:"engine_b"`
	DatasetPath     string               `json:"dataset_path"`
	EvaluationMode  string               `json:"evaluation_mode,omitempty"`
}

// Default returns a freshly initialized default configuration.
func Default() AppConfig {
	openAIKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	jevEndpoint := os.Getenv("TYPESAFE_ENDPOINT")
	if jevEndpoint == "" {
		jevEndpoint = os.Getenv("JEV_ENDPOINT")
	}

	jevKey := os.Getenv("TYPESAFE_API_KEY")
	if jevKey == "" {
		jevKey = os.Getenv("JEV_API_KEY")
	}

	jevModel := os.Getenv("TYPESAFE_MODEL")
	if jevModel == "" {
		jevModel = os.Getenv("JEV_MODEL")
	}
	if jevModel == "" {
		jevModel = "jev-latest"
	}

	activeGen := os.Getenv("JEV_LLM_PROVIDER")
	if activeGen == "" {
		activeGen = "openai"
	}

	addr := os.Getenv("JEV_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}

	return AppConfig{
		Addr:            addr,
		ActiveGenerator: activeGen,
		OpenAI: OpenAIConfig{
			APIKey:  openAIKey,
			Model:   "gpt-4o-mini",
			BaseURL: "https://api.openai.com/v1",
		},
		Anthropic: AnthropicConfig{
			APIKey:  anthropicKey,
			Model:   "claude-3-5-haiku-latest",
			BaseURL: "https://api.anthropic.com",
		},
		Gemini: GeminiConfig{
			APIKey: geminiKey,
			Model:  "gemini-2.5-flash",
		},
		Jev: JevConfig{
			Endpoint:  jevEndpoint,
			APIKey:    jevKey,
			Model:     jevModel,
			TimeoutMs: 5000,
		},
		EngineA: DecisionEngineConfig{
			Provider: "jev",
			Endpoint: jevEndpoint,
			Model:    jevModel,
			APIKey:   jevKey,
		},
		EngineB: DecisionEngineConfig{
			Provider: "openai",
			Endpoint: "https://api.openai.com/v1",
			Model:    "gpt-4o-mini",
			APIKey:   openAIKey,
		},
		DatasetPath:    "dataset.json",
		EvaluationMode: "parallel",
	}
}

// LoadFromFile reads an AppConfig from a JSON file.
func LoadFromFile(path string) (AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, fmt.Errorf("read config file: %w", err)
	}
	var rawCheck struct {
		EngineA        *DecisionEngineConfig `json:"engine_a"`
		EngineB        *DecisionEngineConfig `json:"engine_b"`
		ExecutionMode  string                `json:"execution_mode"`
		EvaluationMode string                `json:"evaluation_mode"`
	}
	_ = json.Unmarshal(data, &rawCheck)

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("unmarshal config: %w", err)
	}

	if rawCheck.EvaluationMode != "" {
		cfg.EvaluationMode = rawCheck.EvaluationMode
	} else if rawCheck.ExecutionMode != "" {
		cfg.EvaluationMode = rawCheck.ExecutionMode
	} else if cfg.EvaluationMode == "" {
		cfg.EvaluationMode = "parallel"
	}

	// Backwards compatibility migration for older config files
	if rawCheck.EngineA == nil {
		cfg.EngineA = DecisionEngineConfig{
			Provider: "jev",
			Endpoint: cfg.Jev.Endpoint,
			Model:    cfg.Jev.Model,
			APIKey:   cfg.Jev.APIKey,
		}
	}
	if rawCheck.EngineB == nil {
		cfg.EngineB = DecisionEngineConfig{
			Provider: "openai",
			Endpoint: cfg.OpenAI.BaseURL,
			Model:    cfg.OpenAI.Model,
			APIKey:   cfg.OpenAI.APIKey,
		}
	}
	if cfg.EngineA.Provider == "jev" {
		cfg.Jev.Endpoint = cfg.EngineA.Endpoint
		cfg.Jev.Model = cfg.EngineA.Model
		cfg.Jev.APIKey = cfg.EngineA.APIKey
	}

	return cfg, nil
}

// SaveToFile writes an AppConfig to a JSON file.
func (c AppConfig) SaveToFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}
	return nil
}

// Config is retained as an alias / compatibility struct for CLI flags.
type Config = AppConfig

// LoadDefault returns the configuration, checking for a local config.json first.
func LoadDefault() Config {
	if cfg, err := LoadFromFile("config.json"); err == nil {
		return cfg
	}
	return Default()
}

// StreamInterval returns the default polling interval.
func (c AppConfig) StreamInterval() time.Duration {
	return 2 * time.Second
}
