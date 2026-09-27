package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MRKups/jev-usecase-1/internal/config"
)

func TestNormalizeOpenAIBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "whitespace only",
			input:    "   \t\n",
			expected: "",
		},
		{
			name:     "openai default without v1",
			input:    "https://api.openai.com",
			expected: "https://api.openai.com/v1/",
		},
		{
			name:     "openai default without v1 with trailing slash",
			input:    "https://api.openai.com/",
			expected: "https://api.openai.com/v1/",
		},
		{
			name:     "openai default with v1",
			input:    "https://api.openai.com/v1",
			expected: "https://api.openai.com/v1/",
		},
		{
			name:     "openai default with v1 and trailing slash",
			input:    "https://api.openai.com/v1/",
			expected: "https://api.openai.com/v1/",
		},
		{
			name:     "local ollama without v1",
			input:    "http://localhost:11434",
			expected: "http://localhost:11434/v1/",
		},
		{
			name:     "local ollama with trailing slash without v1",
			input:    "http://localhost:11434/",
			expected: "http://localhost:11434/v1/",
		},
		{
			name:     "local ollama with v1",
			input:    "http://localhost:11434/v1",
			expected: "http://localhost:11434/v1/",
		},
		{
			name:     "openrouter api without v1",
			input:    "https://openrouter.ai/api",
			expected: "https://openrouter.ai/api/v1/",
		},
		{
			name:     "openrouter api with v1",
			input:    "https://openrouter.ai/api/v1",
			expected: "https://openrouter.ai/api/v1/",
		},
		{
			name:     "openrouter api with v1 and trailing slash",
			input:    "https://openrouter.ai/api/v1/",
			expected: "https://openrouter.ai/api/v1/",
		},
		{
			name:     "custom port without v1",
			input:    "http://127.0.0.1:8000",
			expected: "http://127.0.0.1:8000/v1/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeOpenAIBaseURL(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeOpenAIBaseURL(%q) = %q; expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNewOpenAIProvider_APIKeyRequirement(t *testing.T) {
	// 1. Official OpenAI without key must fail
	_, err := NewOpenAIProvider(config.OpenAIConfig{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "",
	})
	if err == nil {
		t.Error("expected error for empty API key on api.openai.com, got nil")
	}

	// 2. Official OpenAI empty baseURL without key must fail
	_, err = NewOpenAIProvider(config.OpenAIConfig{
		BaseURL: "",
		APIKey:  "",
	})
	if err == nil {
		t.Error("expected error for empty API key and empty baseURL, got nil")
	}

	// 3. Local Ollama without key must succeed
	p, err := NewOpenAIProvider(config.OpenAIConfig{
		BaseURL: "http://localhost:11434",
		APIKey:  "",
	})
	if err != nil {
		t.Fatalf("expected success for local Ollama without key, got error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil provider")
	}

	// 4. Local LM Studio without key must succeed
	p, err = NewOpenAIProvider(config.OpenAIConfig{
		BaseURL: "http://localhost:1234/v1",
		APIKey:  "",
		Model:   "local-model",
	})
	if err != nil {
		t.Fatalf("expected success for local LM Studio without key, got error: %v", err)
	}
	if p.cfg.Model != "local-model" {
		t.Errorf("expected model local-model, got %s", p.cfg.Model)
	}
}

func TestOpenAIProvider_Ping(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"object": "list",
			"data": []map[string]any{
				{"id": "gpt-4o-mini", "object": "model"},
			},
		})
	}))
	defer mockServer.Close()

	p, err := NewOpenAIProvider(config.OpenAIConfig{
		BaseURL: mockServer.URL,
		APIKey:  "",
	})
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	if err := p.Ping(context.Background()); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
}
