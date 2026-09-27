package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/MRKups/jev-usecase-1/internal/config"
)

// Factory creates an LLMProvider based on the current AppConfig.
func Factory(ctx context.Context, cfg config.AppConfig) (LLMProvider, error) {
	switch strings.ToLower(cfg.ActiveGenerator) {
	case "openai":
		return NewOpenAIProvider(cfg.OpenAI)
	case "anthropic":
		return NewAnthropicProvider(cfg.Anthropic)
	case "gemini":
		return NewGeminiProvider(ctx, cfg.Gemini)
	default:
		return nil, fmt.Errorf("unknown active generator %q (valid: openai, anthropic, gemini)", cfg.ActiveGenerator)
	}
}
