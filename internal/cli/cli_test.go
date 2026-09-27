package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

func TestCLIHelpAndVersion(t *testing.T) {
	ctx := context.Background()

	var buf bytes.Buffer
	if err := RunWithInput(ctx, []string{"--help"}, nil, &buf); err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Usage:") {
		t.Errorf("expected usage string, got %s", buf.String())
	}

	buf.Reset()
	if err := RunWithInput(ctx, []string{"version"}, nil, &buf); err != nil {
		t.Fatalf("version failed: %v", err)
	}
	if !strings.Contains(buf.String(), "ticket-eval version") {
		t.Errorf("expected version output, got %s", buf.String())
	}
}

func TestCLITriageValidation(t *testing.T) {
	ctx := context.Background()

	// Missing --input
	var buf bytes.Buffer
	err := RunWithInput(ctx, []string{"triage"}, nil, &buf)
	if err == nil || !strings.Contains(err.Error(), "missing required --input flag") {
		t.Fatalf("expected missing input error, got: %v", err)
	}

	// Missing endpoint configuration
	fixturePath := filepath.Join("..", "..", "fixtures", "tickets", "sample_tickets.json")
	buf.Reset()
	os.Unsetenv("JEV_ENDPOINT")
	err = RunWithInput(ctx, []string{"triage", "--input=" + fixturePath}, nil, &buf)
	if err == nil || !strings.Contains(err.Error(), "jev endpoint is not configured") {
		t.Fatalf("expected missing endpoint error, got: %v", err)
	}
}

func TestCLITriageExecution(t *testing.T) {
	ctx := context.Background()

	// Test HTTP server acting as TypeSafe Jev endpoint for CLI test
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := ticket.TriageResult{
			TicketID:   "INC-20260925-A101",
			LatencyMs:  14,
			Confidence: 0.95,
			Answers: map[string]string{
				"technical_domain":        "Hardware",
				"operational_urgency":     "High",
				"security_incident":       "No",
				"target_resolution_group": "Service Desk",
				"blast_radius":            "Single User",
			},
			RecommendedAction: "Dispatch to Service Desk",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer server.Close()

	os.Setenv("JEV_ENDPOINT", server.URL)
	defer os.Unsetenv("JEV_ENDPOINT")

	var triageBuf bytes.Buffer
	fixturePath := filepath.Join("..", "..", "fixtures", "tickets", "sample_tickets.json")
	if err := RunWithInput(ctx, []string{"triage", "--input=" + fixturePath}, nil, &triageBuf); err != nil {
		t.Fatalf("triage failed: %v", err)
	}
	if !strings.Contains(triageBuf.String(), "Evaluating") {
		t.Errorf("expected evaluation output, got: %s", triageBuf.String())
	}
	if !strings.Contains(triageBuf.String(), "Service Desk") {
		t.Errorf("expected Service Desk in output, got: %s", triageBuf.String())
	}
}

func TestLoadInitialTickets(t *testing.T) {
	// Test 1: Starts empty when no custom or root file exists
	tickets, source := loadInitialTickets("non_existent_dataset.json")
	if len(tickets) != 0 {
		t.Fatalf("expected 0 tickets when file does not exist, got %d", len(tickets))
	}
	if source != "" {
		t.Errorf("expected empty source, got %q", source)
	}

	// Test 2: Custom dataset path is loaded when present
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "custom_dataset.json")
	sampleJSON := `[
		{
			"id": "INC-TEST-1",
			"created_at": "2026-09-25T12:00:00Z",
			"reporter_name": "Test User",
			"reporter_email": "test.user@example.com",
			"department": "Engineering",
			"category": "Hardware",
			"reported_urgency": "High",
			"summary": "Custom test ticket",
			"description": "Custom description",
			"affected_system": "Laptop"
		}
	]`
	if err := os.WriteFile(customPath, []byte(sampleJSON), 0644); err != nil {
		t.Fatalf("failed to write custom dataset: %v", err)
	}

	loaded, loadedSource := loadInitialTickets(customPath)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 ticket from custom dataset, got %d", len(loaded))
	}
	if loadedSource != customPath {
		t.Errorf("expected source %q, got %q", customPath, loadedSource)
	}
	if loaded[0].Summary != "Custom test ticket" {
		t.Errorf("expected 'Custom test ticket', got %q", loaded[0].Summary)
	}

	// Test 3: Empty dataset file starts empty rather than loading dummy fixture
	emptyPath := filepath.Join(tmpDir, "empty_dataset.json")
	if err := os.WriteFile(emptyPath, []byte("[]"), 0644); err != nil {
		t.Fatalf("failed to write empty dataset: %v", err)
	}

	fallbackTickets, fallbackSource := loadInitialTickets(emptyPath)
	if len(fallbackTickets) != 0 {
		t.Fatalf("expected 0 tickets for empty dataset file, got %d", len(fallbackTickets))
	}
	if fallbackSource != "" {
		t.Errorf("expected empty source, got %q", fallbackSource)
	}
}

func TestLoadInitialQuestions(t *testing.T) {
	// Test 1: Non-existent file returns nil, empty source
	qs, src := loadInitialQuestions("non_existent_questions.json")
	if qs != nil && len(qs) > 0 {
		t.Fatalf("expected nil or empty for non-existent file, got %d", len(qs))
	}
	if src != "" {
		t.Errorf("expected empty source, got %q", src)
	}

	// Test 2: Custom questions file is loaded when present
	tmpDir := t.TempDir()
	customPath := filepath.Join(tmpDir, "questions.json")
	sampleJSON := `[
		{
			"id": "test_q",
			"type": "choice",
			"text": "Is this a test?",
			"options": ["Yes", "No"]
		}
	]`
	if err := os.WriteFile(customPath, []byte(sampleJSON), 0644); err != nil {
		t.Fatalf("failed to write custom questions: %v", err)
	}

	loaded, loadedSource := loadInitialQuestions(customPath)
	if len(loaded) != 1 {
		t.Fatalf("expected 1 question, got %d", len(loaded))
	}
	if loadedSource != customPath {
		t.Errorf("expected source %q, got %q", customPath, loadedSource)
	}
	if loaded[0].ID != "test_q" {
		t.Errorf("expected 'test_q', got %q", loaded[0].ID)
	}
}
