package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider wraps the official Anthropic Go SDK.
type AnthropicProvider struct {
	cfg    config.AnthropicConfig
	client *anthropic.Client
}

// NewAnthropicProvider creates a new provider using the official Anthropic Go SDK.
func NewAnthropicProvider(cfg config.AnthropicConfig) (*AnthropicProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("anthropic api key is required")
	}
	if cfg.Model == "" {
		cfg.Model = "claude-3-5-haiku-latest"
	}

	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" && !strings.Contains(cfg.BaseURL, "api.anthropic.com") {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}

	client := anthropic.NewClient(opts...)
	return &AnthropicProvider{
		cfg:    cfg,
		client: &client,
	}, nil
}

// GenerateTicket invokes Claude to generate a ticket.
func (p *AnthropicProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.cfg.Model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: "You are an IT helpdesk simulator. Output valid JSON adhering strictly to the requested schema. Return raw JSON only with no markdown fences or preambles."},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic messages: %w", err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned 0 content blocks")
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "thinking" || block.Type == "redacted_thinking" {
			continue
		}
		if block.Text != "" {
			sb.WriteString(block.Text)
		}
	}

	return parseTicketJSON(sb.String(), prompt)
}

// Ping verifies connectivity and credentials with Anthropic by listing models.
func (p *AnthropicProvider) Ping(ctx context.Context) error {
	_, err := p.client.Models.List(ctx, anthropic.ModelListParams{})
	if err != nil {
		return fmt.Errorf("anthropic list models: %w", err)
	}
	return nil
}

// Name returns the descriptive name of the Anthropic provider and model.
func (p *AnthropicProvider) Name() string {
	return "Anthropic (" + p.cfg.Model + ")"
}

// Classify evaluates a ticket against standard triage questions using Claude.
func (p *AnthropicProvider) Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error) {
	start := time.Now()
	prompt := BuildTriagePrompt(t, questions)

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.cfg.Model,
		MaxTokens: 1024,
		System: []anthropic.TextBlockParam{
			{Text: "You are an expert IT service management triage engine. Output valid JSON adhering strictly to the requested schema. Return raw JSON only with no markdown fences or preambles."},
		},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("anthropic triage call: %w", err)
	}

	if len(resp.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned 0 content blocks")
	}

	var sb strings.Builder
	for _, block := range resp.Content {
		if block.Type == "thinking" || block.Type == "redacted_thinking" {
			continue
		}
		if block.Text != "" {
			sb.WriteString(block.Text)
		}
	}

	latency := time.Since(start).Milliseconds()
	answers, conf, action, err := ParseTriageJSON(sb.String(), questions)
	if err != nil {
		return nil, err
	}

	return &ticket.TriageResult{
		TicketID:          t.ID,
		EngineName:        p.Name(),
		Model:             p.cfg.Model,
		EvaluatedAt:       time.Now().UTC(),
		LatencyMs:         latency,
		Confidence:        conf,
		Answers:           answers,
		RecommendedAction: action,
	}, nil
}
