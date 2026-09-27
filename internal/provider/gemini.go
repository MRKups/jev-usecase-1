package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
	"google.golang.org/genai"
)

// GeminiProvider wraps the official Google GenAI Go SDK.
type GeminiProvider struct {
	cfg    config.GeminiConfig
	client *genai.Client
}

// NewGeminiProvider creates a new provider using the official Google GenAI Go SDK.
func NewGeminiProvider(ctx context.Context, cfg config.GeminiConfig) (*GeminiProvider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("gemini api key is required")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-flash"
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}

	return &GeminiProvider{
		cfg:    cfg,
		client: client,
	}, nil
}

// GenerateTicket invokes Gemini to generate a ticket.
func (p *GeminiProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	resp, err := p.client.Models.GenerateContent(ctx, p.cfg.Model, []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: "You are an IT helpdesk simulator. Output valid JSON adhering strictly to the requested schema."},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gemini generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Thought {
			continue
		}
		if part.Text != "" {
			sb.WriteString(part.Text)
		}
	}

	return parseTicketJSON(sb.String(), prompt)
}

// Ping verifies connectivity and credentials with Gemini by listing models.
func (p *GeminiProvider) Ping(ctx context.Context) error {
	_, err := p.client.Models.List(ctx, nil)
	if err != nil {
		return fmt.Errorf("gemini list models: %w", err)
	}
	return nil
}

// Name returns the descriptive name of the Gemini provider and model.
func (p *GeminiProvider) Name() string {
	return "Gemini (" + p.cfg.Model + ")"
}

// Classify evaluates a ticket against standard triage questions using Gemini.
func (p *GeminiProvider) Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error) {
	start := time.Now()
	prompt := BuildTriagePrompt(t, questions)

	resp, err := p.client.Models.GenerateContent(ctx, p.cfg.Model, []*genai.Content{
		{
			Role: "user",
			Parts: []*genai.Part{
				{Text: prompt},
			},
		},
	}, &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{
				{Text: "You are an expert IT service management triage engine. Output valid JSON adhering strictly to the requested schema."},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("gemini triage call: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("gemini returned no candidates")
	}

	var sb strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if part.Thought {
			continue
		}
		if part.Text != "" {
			sb.WriteString(part.Text)
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
