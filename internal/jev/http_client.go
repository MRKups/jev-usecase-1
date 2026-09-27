package jev

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

// HTTPClient calls the official TypeSafe Jev Decision Engine API.
type HTTPClient struct {
	endpoint   string
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewHTTPClient creates an HTTP client for the TypeSafe Jev API.
func NewHTTPClient(cfg config.JevConfig) *HTTPClient {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		endpoint = "https://api.typesafe.ai"
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "jev-latest"
	}
	return &HTTPClient{
		endpoint: endpoint,
		apiKey:   strings.TrimSpace(cfg.APIKey),
		model:    model,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func resolveSystemOneURL(endpoint string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "https://api.typesafe.ai"
	}
	if strings.HasSuffix(base, "/systemone") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/systemone"
	}
	return base + "/v1/systemone"
}

func resolveModelsURL(endpoint string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "https://api.typesafe.ai"
	}
	if strings.HasSuffix(base, "/systemone") {
		base = strings.TrimSuffix(base, "/systemone")
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/models"
	}
	return base + "/v1/models"
}

type systemOneQuestionReq struct {
	Type         string            `json:"type"`
	Instructions string            `json:"instructions,omitempty"`
	Criteria     map[string]string `json:"criteria"`
}

type systemOneRequest struct {
	State     any                             `json:"state"`
	Model     string                          `json:"model"`
	Questions map[string]systemOneQuestionReq `json:"questions"`
}

type systemOneAnswerResp struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Noul          *float64           `json:"noul,omitempty"`
	Score         *float64           `json:"score,omitempty"`
}

func (a *systemOneAnswerResp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		a.Type = "choice"
		a.Choice = s
		return nil
	}
	type alias systemOneAnswerResp
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*a = systemOneAnswerResp(aux)
	return nil
}

type systemOneResponse struct {
	Model   string                         `json:"model"`
	Answers map[string]systemOneAnswerResp `json:"answers"`
	Usage   *systemOneUsage                `json:"usage,omitempty"`
	Error   *systemOneError                `json:"error,omitempty"`
}

type systemOneUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type systemOneError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func buildTicketState(t *ticket.Ticket) map[string]any {
	return map[string]any{
		"ticket_id":        t.ID,
		"summary":          t.Summary,
		"description":      t.Description,
		"department":       t.Department,
		"category":         t.Category,
		"affected_system":  t.AffectedSystem,
		"reported_urgency": t.ReportedUrgency,
		"reporter_name":    t.ReporterName,
		"reporter_email":   t.ReporterEmail,
		"created_at":       t.CreatedAt.Format(time.RFC3339),
	}
}

func buildQuestionsMap(questions []triage.Question) map[string]systemOneQuestionReq {
	reqMap := make(map[string]systemOneQuestionReq, len(questions))
	for _, q := range questions {
		qType := q.Type
		if qType == "" {
			if len(q.Options) == 2 && ((strings.EqualFold(q.Options[0], "yes") && strings.EqualFold(q.Options[1], "no")) ||
				(strings.EqualFold(q.Options[0], "true") && strings.EqualFold(q.Options[1], "false"))) {
				qType = "noul"
			} else {
				qType = "choice"
			}
		}

		instructions := q.Instructions
		if instructions == "" {
			instructions = q.Text
		}

		criteria := q.Criteria
		if criteria == nil {
			criteria = make(map[string]string)
			if qType == "noul" {
				criteria["true"] = q.Text
				criteria["false"] = "Negative or not applicable."
			} else {
				for _, opt := range q.Options {
					criteria[opt] = fmt.Sprintf("Assign to %s when applicable.", opt)
				}
			}
		}

		reqMap[q.ID] = systemOneQuestionReq{
			Type:         qType,
			Instructions: instructions,
			Criteria:     criteria,
		}
	}
	return reqMap
}

func (c *HTTPClient) doRequest(ctx context.Context, body []byte) (*systemOneResponse, error) {
	url := resolveSystemOneURL(c.endpoint)

	var lastErr error
	backoffs := []time.Duration{200 * time.Millisecond, 400 * time.Millisecond}

	for attempt := 0; attempt <= len(backoffs); attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoffs[attempt-1]):
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("create systemone request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("execute systemone call: %w", err)
			continue
		}

		respBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read systemone response: %w", err)
			continue
		}

		if resp.StatusCode == 429 || resp.StatusCode == 529 {
			lastErr = fmt.Errorf("systemone transient error %d: %s", resp.StatusCode, string(respBytes))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("systemone status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBytes)))
		}

		var parsed systemOneResponse
		if err := json.Unmarshal(respBytes, &parsed); err != nil {
			return nil, fmt.Errorf("unmarshal systemone response: %w", err)
		}
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("systemone api error (%s): %s", parsed.Error.Code, parsed.Error.Message)
		}

		return &parsed, nil
	}

	return nil, lastErr
}

// Ping verifies connectivity and authentication with the TypeSafe Jev API.
func (c *HTTPClient) Ping(ctx context.Context) error {
	url := resolveModelsURL(c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create ping request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute ping: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: invalid or missing TypeSafe API key (status 401)")
	case http.StatusForbidden:
		return fmt.Errorf("forbidden: access denied by TypeSafe API (status 403)")
	default:
		return fmt.Errorf("typesafe ping returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

// Classify calls the TypeSafe System One API to classify and label a ticket.
func (c *HTTPClient) Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error) {
	start := time.Now()

	reqPayload := systemOneRequest{
		State:     buildTicketState(t),
		Model:     c.model,
		Questions: buildQuestionsMap(questions),
	}

	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal systemone request: %w", err)
	}

	resp, err := c.doRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	latency := time.Since(start).Milliseconds()

	answers := make(map[string]string, len(questions))
	var confSum float64
	var confCount int

	for _, q := range questions {
		ans, ok := resp.Answers[q.ID]
		if !ok {
			continue
		}

		var answerVal string
		var conf float64 = 1.0

		switch ans.Type {
		case "choice":
			answerVal = ans.Choice
			if ans.Confidence != nil {
				conf = *ans.Confidence
			}
		case "noul":
			if ans.Noul != nil {
				prob := *ans.Noul
				if prob >= 0.5 {
					answerVal = "Yes"
				} else {
					answerVal = "No"
				}
				conf = math.Abs(prob-0.5) * 2
				if conf > 1.0 {
					conf = 1.0
				}
			} else if ans.Choice != "" {
				answerVal = ans.Choice
			}
		case "score":
			if ans.Score != nil {
				answerVal = fmt.Sprintf("%.2f", *ans.Score)
			}
			if ans.Confidence != nil {
				conf = *ans.Confidence
			}
		default:
			if ans.Choice != "" {
				answerVal = ans.Choice
				if ans.Confidence != nil {
					conf = *ans.Confidence
				}
			}
		}

		answers[q.ID] = answerVal
		confSum += conf
		confCount++
	}

	var overallConfidence float64
	if confCount > 0 {
		avg := confSum / float64(confCount)
		overallConfidence = math.Round(avg*10000) / 10000
	}

	actualModel := c.model
	if resp.Model != "" {
		actualModel = resp.Model
	}

	return &ticket.TriageResult{
		TicketID:          t.ID,
		EngineName:        c.Name(),
		Model:             actualModel,
		EvaluatedAt:       time.Now().UTC(),
		LatencyMs:         latency,
		Confidence:        overallConfidence,
		Answers:           answers,
		RecommendedAction: "Jev is not able to generate a response to this question.",
	}, nil
}

// Name returns the descriptive name of the Jev engine and model.
func (c *HTTPClient) Name() string {
	return "TypeSafe Jev (" + c.model + ")"
}

// Factory creates a Jev Client based on JevConfig.
func Factory(cfg config.JevConfig) Client {
	return NewHTTPClient(cfg)
}
