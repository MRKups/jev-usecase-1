package jev

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

func TestJevHTTPClientClassify(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/systemone" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-jev-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req systemOneRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if req.Model != "jev-latest" {
			http.Error(w, "unexpected model", http.StatusBadRequest)
			return
		}

		confHigh := 0.94
		confMed := 0.88
		noulSec := 0.05
		resp := systemOneResponse{
			Model: "jev-1.13.0",
			Answers: map[string]systemOneAnswerResp{
				"ticket_type": {
					Type:       "choice",
					Choice:     "Incident",
					Confidence: &confHigh,
				},
				"technical_domain": {
					Type:       "choice",
					Choice:     "Network",
					Confidence: &confHigh,
				},
				"operational_urgency": {
					Type:       "choice",
					Choice:     "High",
					Confidence: &confMed,
				},
				"security_incident": {
					Type: "noul",
					Noul: &noulSec,
				},
				"target_resolution_group": {
					Type:       "choice",
					Choice:     "Network Operations",
					Confidence: &confHigh,
				},
				"blast_radius": {
					Type:       "choice",
					Choice:     "Single User",
					Confidence: &confMed,
				},
			},
			Usage: &systemOneUsage{
				InputTokens:  180,
				OutputTokens: 45,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(config.JevConfig{
		Endpoint:  server.URL,
		APIKey:    "test-jev-key",
		Model:     "jev-latest",
		TimeoutMs: 2000,
	})

	testTicket := &ticket.Ticket{
		ID:              "INC-999",
		Summary:         "VPN disconnects every 5 minutes",
		Description:     "Virtual private network client drops and cannot route packets",
		ReportedUrgency: "High",
		Department:      "Sales & Commercial",
		AffectedSystem:  "Virtual Private Network Client",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := client.Classify(ctx, testTicket, triage.StandardQuestions())
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}

	if result.TicketID != "INC-999" {
		t.Errorf("expected TicketID INC-999, got %s", result.TicketID)
	}
	if result.Answers["ticket_type"] != "Incident" {
		t.Errorf("expected ticket_type Incident, got %s", result.Answers["ticket_type"])
	}
	if result.Answers["technical_domain"] != "Network" {
		t.Errorf("expected Network, got %s", result.Answers["technical_domain"])
	}
	if result.Answers["security_incident"] != "No" {
		t.Errorf("expected security_incident No, got %s", result.Answers["security_incident"])
	}
	if result.Answers["target_resolution_group"] != "Network Operations" {
		t.Errorf("expected Network Operations, got %s", result.Answers["target_resolution_group"])
	}
	if result.Confidence <= 0.0 || result.Confidence > 1.0 {
		t.Errorf("unexpected confidence: %f", result.Confidence)
	}
	if result.RecommendedAction != "Jev is not able to generate a response to this question." {
		t.Errorf("expected inability notice, got %s", result.RecommendedAction)
	}
}

func TestJevHTTPClientPing(t *testing.T) {
	var authSeen string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		authSeen = r.Header.Get("Authorization")
		if authSeen != "Bearer test-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"jev-latest"}]}`))
	}))
	defer server.Close()

	client := NewHTTPClient(config.JevConfig{
		Endpoint: server.URL,
		APIKey:   "test-key",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	badClient := NewHTTPClient(config.JevConfig{
		Endpoint: server.URL,
		APIKey:   "wrong-key",
	})
	if err := badClient.Ping(ctx); err == nil {
		t.Errorf("expected ping to fail with wrong key")
	}
}

func TestJevHTTPClientRetryOnRateLimit(t *testing.T) {
	var attempts int32
	conf := 0.95
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&attempts, 1)
		if count == 1 {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		resp := systemOneResponse{
			Model: "jev-latest",
			Answers: map[string]systemOneAnswerResp{
				"technical_domain": {
					Type:       "choice",
					Choice:     "Hardware",
					Confidence: &conf,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPClient(config.JevConfig{
		Endpoint: server.URL,
	})

	testTicket := &ticket.Ticket{ID: "INC-123"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	questions := []triage.Question{
		{
			ID:   "technical_domain",
			Type: "choice",
			Text: "What is the primary technical domain?",
		},
	}
	result, err := client.Classify(ctx, testTicket, questions)
	if err != nil {
		t.Fatalf("Classify failed on retry: %v", err)
	}
	if result.Answers["technical_domain"] != "Hardware" {
		t.Errorf("expected Hardware, got %s", result.Answers["technical_domain"])
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected 2 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}
