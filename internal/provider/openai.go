package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

// OpenAIProvider wraps the official OpenAI Go SDK.
type OpenAIProvider struct {
	cfg    config.OpenAIConfig
	client *openai.Client
}

// NormalizeOpenAIBaseURL ensures OpenAI-compatible base URLs include the /v1 path and trailing slash.
func NormalizeOpenAIBaseURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	trimmed := strings.TrimRight(raw, "/")
	if !strings.HasSuffix(trimmed, "/v1") {
		trimmed += "/v1"
	}
	return trimmed + "/"
}

// NewOpenAIProvider creates a new provider using the official OpenAI Go SDK.
func NewOpenAIProvider(cfg config.OpenAIConfig) (*OpenAIProvider, error) {
	if cfg.Model == "" {
		cfg.Model = "gpt-4o-mini"
	}

	opts := []option.RequestOption{}

	apiKey := strings.TrimSpace(cfg.APIKey)
	baseURL := strings.TrimSpace(cfg.BaseURL)
	isDefaultOpenAI := baseURL == "" || strings.Contains(baseURL, "api.openai.com")

	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	} else if isDefaultOpenAI {
		return nil, fmt.Errorf("openai api key is required when using api.openai.com")
	} else {
		// Local or custom OpenAI-compatible servers (such as LM Studio, Ollama, LocalAI, vLLM)
		// do not require authentication. Supply a placeholder token so clients expecting
		// an Authorization header are satisfied without blocking local execution.
		opts = append(opts, option.WithAPIKey("dummy"))
	}

	if baseURL != "" {
		normalized := NormalizeOpenAIBaseURL(baseURL)
		cfg.BaseURL = normalized
		opts = append(opts, option.WithBaseURL(normalized))
	}

	client := openai.NewClient(opts...)
	return &OpenAIProvider{
		cfg:    cfg,
		client: &client,
	}, nil
}

// GenerateTicket invokes OpenAI to generate a ticket.
func (p *OpenAIProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	respFormat := shared.NewResponseFormatJSONObjectParam()
	params := openai.ChatCompletionNewParams{
		Model: p.cfg.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are an IT helpdesk simulator. Output valid JSON adhering strictly to the requested schema. Return raw JSON only with no markdown fences or preambles."),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &respFormat,
		},
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil && (strings.Contains(err.Error(), "response_format") || strings.Contains(err.Error(), "json_object")) {
		// Some local engines (such as older Ollama, llama.cpp, or LM Studio builds) do not support response_format.
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{}
		resp, err = p.client.Chat.Completions.New(ctx, params)
	}
	if err != nil {
		return nil, fmt.Errorf("openai completions: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned 0 choices")
	}

	content := resp.Choices[0].Message.Content
	return parseTicketJSON(content, prompt)
}

// Ping verifies connectivity and credentials with the OpenAI-compatible endpoint by listing models.
func (p *OpenAIProvider) Ping(ctx context.Context) error {
	_, err := p.client.Models.List(ctx)
	if err != nil {
		return fmt.Errorf("openai list models: %w", err)
	}
	return nil
}

// Name returns the descriptive name of the OpenAI provider and model.
func (p *OpenAIProvider) Name() string {
	return "OpenAI (" + p.cfg.Model + ")"
}

// Classify evaluates a ticket against standard triage questions using OpenAI.
func (p *OpenAIProvider) Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error) {
	start := time.Now()
	prompt := BuildTriagePrompt(t, questions)

	respFormat := shared.NewResponseFormatJSONObjectParam()
	params := openai.ChatCompletionNewParams{
		Model: p.cfg.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are an expert IT service management triage engine. Output valid JSON adhering strictly to the requested schema. Return raw JSON only with no markdown fences or preambles."),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &respFormat,
		},
	}

	resp, err := p.client.Chat.Completions.New(ctx, params)
	if err != nil && (strings.Contains(err.Error(), "response_format") || strings.Contains(err.Error(), "json_object")) {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{}
		resp, err = p.client.Chat.Completions.New(ctx, params)
	}
	if err != nil {
		return nil, fmt.Errorf("openai triage completions: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned 0 choices")
	}

	latency := time.Since(start).Milliseconds()
	content := resp.Choices[0].Message.Content

	answers, conf, action, err := ParseTriageJSON(content, questions)
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
