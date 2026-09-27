package webapp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/generator"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

type testLLMProvider struct{}

func (p *testLLMProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	return &ticket.Ticket{
		ID:              "INC-GEN-001",
		CreatedAt:       time.Now().UTC(),
		ReporterName:    "Generated User",
		ReporterEmail:   "user@example.com",
		Department:      "Engineering",
		Category:        "Hardware",
		ReportedUrgency: "High",
		Summary:         "Generated hardware issue",
		Description:     "Test description",
		AffectedSystem:  "Laptop",
	}, nil
}

func (p *testLLMProvider) Ping(ctx context.Context) error {
	return nil
}

type testJevClient struct{}

func (c *testJevClient) Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error) {
	answers := make(map[string]string)
	for _, q := range questions {
		answers[q.ID] = "TestAnswer"
	}
	return &ticket.TriageResult{
		TicketID:          t.ID,
		EvaluatedAt:       time.Now().UTC(),
		LatencyMs:         15,
		Confidence:        0.95,
		Answers:           answers,
		RecommendedAction: "Route to Engineering",
	}, nil
}

func (c *testJevClient) Ping(ctx context.Context) error {
	return nil
}

func (c *testJevClient) Name() string {
	return "Test Decision Engine"
}

func setupTestServer() *Server {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	engineA := &testJevClient{}
	engineB := &testJevClient{}

	initial := []*ticket.Ticket{
		{
			ID:              "INC-TEST-001",
			CreatedAt:       time.Now(),
			ReporterName:    "Test User",
			ReporterEmail:   "test@example.com",
			Department:      "Engineering",
			Category:        "Hardware",
			ReportedUrgency: "High",
			Summary:         "Battery swelling test",
			Description:     "Trackpad is lifted due to swollen battery",
			AffectedSystem:  "Laptop Workstation",
		},
	}

	cfg := config.Default()
	return NewServer(cfg, gen, engineA, engineB, initial)
}

func TestIndexEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("expected text/html, got %s", ct)
	}
}

func TestListTicketsEndpoint(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var tickets []*ticket.TriagedTicket
	if err := json.NewDecoder(w.Body).Decode(&tickets); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].ID != "INC-TEST-001" {
		t.Errorf("expected ID INC-TEST-001, got %s", tickets[0].ID)
	}
}

func TestGenerateAndTriageFlow(t *testing.T) {
	srv := setupTestServer()

	// 1. Generate new ticket
	genReq := httptest.NewRequest(http.MethodPost, "/api/tickets/generate", nil)
	genRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(genRec, genReq)

	if genRec.Code != http.StatusOK {
		t.Fatalf("generate ticket expected 200, got %d: %s", genRec.Code, genRec.Body.String())
	}

	var generated ticket.TriagedTicket
	if err := json.NewDecoder(genRec.Body).Decode(&generated); err != nil {
		t.Fatalf("failed to decode generated ticket: %v", err)
	}

	if generated.ID == "" {
		t.Fatal("expected non-empty ticket ID")
	}

	// 2. Triage newly generated ticket
	triageReq := httptest.NewRequest(http.MethodPost, "/api/tickets/"+generated.ID+"/triage", nil)
	triageRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(triageRec, triageReq)

	if triageRec.Code != http.StatusOK {
		t.Fatalf("triage expected 200, got %d: %s", triageRec.Code, triageRec.Body.String())
	}

	var triaged ticket.TriagedTicket
	if err := json.NewDecoder(triageRec.Body).Decode(&triaged); err != nil {
		t.Fatalf("failed to decode triaged ticket: %v", err)
	}

	if triaged.TriageA == nil {
		t.Fatal("expected non-nil TriageA result")
	}
	if triaged.TriageA.TicketID != generated.ID {
		t.Errorf("expected TriageA.TicketID %s, got %s", generated.ID, triaged.TriageA.TicketID)
	}
	if len(triaged.TriageA.Answers) != 6 {
		t.Errorf("expected 6 triage answers, got %d", len(triaged.TriageA.Answers))
	}
}

func TestConfigEndpoints(t *testing.T) {
	srv := setupTestServer()

	// GET /api/config
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var cfg config.AppConfig
	if err := json.NewDecoder(rec.Body).Decode(&cfg); err != nil {
		t.Fatalf("failed to decode config: %v", err)
	}
	if cfg.ActiveGenerator != "openai" {
		t.Errorf("expected openai active generator, got %s", cfg.ActiveGenerator)
	}
}

func TestDatasetEndpoints(t *testing.T) {
	srv := setupTestServer()

	// Generate batch
	genBody := strings.NewReader(`{"count": 3}`)
	req := httptest.NewRequest(http.MethodPost, "/api/dataset/generate", genBody)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Generate with single count (arbitrary number 1)
	genBody1 := strings.NewReader(`{"count": 1}`)
	req1 := httptest.NewRequest(http.MethodPost, "/api/dataset/generate", genBody1)
	rec1 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}
	var res1 map[string]any
	if err := json.NewDecoder(rec1.Body).Decode(&res1); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if res1["generated"] != float64(1) {
		t.Errorf("expected 1 generated, got %v", res1["generated"])
	}

	// Triage all
	triageReq := httptest.NewRequest(http.MethodPost, "/api/triage/run", nil)
	triageRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(triageRec, triageReq)
	if triageRec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", triageRec.Code)
	}

	// Verify dataset count
	getReq := httptest.NewRequest(http.MethodGet, "/api/dataset", nil)
	getRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(getRec, getReq)
	var tickets []*ticket.TriagedTicket
	if err := json.NewDecoder(getRec.Body).Decode(&tickets); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if len(tickets) < 5 { // 1 initial + 3 generated + 1 generated
		t.Errorf("expected at least 5 tickets, got %d", len(tickets))
	}
}

func TestTestProviderEndpoint(t *testing.T) {
	srv := setupTestServer()

	// 1. Test missing API key
	reqBad := httptest.NewRequest(http.MethodPost, "/api/config/test-provider", strings.NewReader(`{
		"active_generator": "openai",
		"openai": {"api_key": ""}
	}`))
	recBad := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recBad, reqBad)

	if recBad.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", recBad.Code)
	}
	var resBad map[string]any
	if err := json.NewDecoder(recBad.Body).Decode(&resBad); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resBad["ok"] != false {
		t.Errorf("expected ok=false for missing key, got %v", resBad["ok"])
	}

	// 2. Test unknown generator
	reqUnknown := httptest.NewRequest(http.MethodPost, "/api/config/test-provider", strings.NewReader(`{
		"active_generator": "nonexistent"
	}`))
	recUnknown := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recUnknown, reqUnknown)

	var resUnknown map[string]any
	if err := json.NewDecoder(recUnknown.Body).Decode(&resUnknown); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resUnknown["ok"] != false {
		t.Errorf("expected ok=false for unknown generator, got %v", resUnknown["ok"])
	}

	// 3. Test mock OpenAI provider with base URL lacking /v1
	mockOpenAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer mockOpenAI.Close()

	// BaseURL intentionally does not have /v1 (e.g. mockOpenAI.URL is "http://127.0.0.1:port")
	payload, _ := json.Marshal(map[string]any{
		"active_generator": "openai",
		"openai": map[string]string{
			"api_key":  "test-secret-key",
			"model":    "gpt-4o-mini",
			"base_url": mockOpenAI.URL,
		},
	})

	reqOK := httptest.NewRequest(http.MethodPost, "/api/config/test-provider", strings.NewReader(string(payload)))
	recOK := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recOK, reqOK)

	if recOK.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recOK.Code, recOK.Body.String())
	}
	var resOK map[string]any
	if err := json.NewDecoder(recOK.Body).Decode(&resOK); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resOK["ok"] != true {
		t.Errorf("expected ok=true, got %v with error: %v", resOK["ok"], resOK["error"])
	}
	if resOK["provider"] != "openai" {
		t.Errorf("expected provider openai, got %v", resOK["provider"])
	}

	// 4. Test local server (LM Studio / Ollama) with empty API key
	payloadLocal, _ := json.Marshal(map[string]any{
		"active_generator": "openai",
		"openai": map[string]string{
			"api_key":  "",
			"model":    "llama3.2",
			"base_url": mockOpenAI.URL,
		},
	})

	reqLocal := httptest.NewRequest(http.MethodPost, "/api/config/test-provider", strings.NewReader(string(payloadLocal)))
	recLocal := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recLocal, reqLocal)

	if recLocal.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", recLocal.Code, recLocal.Body.String())
	}
	var resLocal map[string]any
	if err := json.NewDecoder(recLocal.Body).Decode(&resLocal); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resLocal["ok"] != true {
		t.Errorf("expected ok=true for local model without key, got %v with error: %v", resLocal["ok"], resLocal["error"])
	}
	if resLocal["model"] != "llama3.2" {
		t.Errorf("expected model llama3.2, got %v", resLocal["model"])
	}
}

type slowLLMProvider struct {
	started chan struct{}
}

func (p *slowLLMProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	select {
	case p.started <- struct{}{}:
	default:
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(2 * time.Second):
		return &ticket.Ticket{
			ID:              "INC-SLOW-001",
			CreatedAt:       time.Now().UTC(),
			ReporterName:    "Slow User",
			ReporterEmail:   "slow@example.com",
			Department:      "Operations",
			Category:        "Access",
			ReportedUrgency: "Low",
			Summary:         "Slow ticket",
			Description:     "Slow description",
			AffectedSystem:  "Portal",
		}, nil
	}
}

func (p *slowLLMProvider) Ping(ctx context.Context) error {
	return nil
}

func TestTestJevEndpoint(t *testing.T) {
	srv := setupTestServer()

	mockJev := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer valid-jev-key" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"jev-latest"}]}`))
	}))
	defer mockJev.Close()

	reqGood := httptest.NewRequest(http.MethodPost, "/api/config/test-jev", strings.NewReader(fmt.Sprintf(`{
		"jev": {
			"endpoint": %q,
			"api_key": "valid-jev-key",
			"model": "jev-latest"
		}
	}`, mockJev.URL)))
	recGood := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recGood, reqGood)

	var resGood map[string]any
	if err := json.NewDecoder(recGood.Body).Decode(&resGood); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resGood["ok"] != true {
		t.Errorf("expected ok=true, got %v: %v", resGood["ok"], resGood["error"])
	}

	reqBad := httptest.NewRequest(http.MethodPost, "/api/config/test-jev", strings.NewReader(fmt.Sprintf(`{
		"jev": {
			"endpoint": %q,
			"api_key": "invalid-key"
		}
	}`, mockJev.URL)))
	recBad := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recBad, reqBad)

	var resBad map[string]any
	if err := json.NewDecoder(recBad.Body).Decode(&resBad); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resBad["ok"] != false {
		t.Errorf("expected ok=false for bad key, got %v", resBad["ok"])
	}
}

func TestCancelDatasetEndpoint(t *testing.T) {
	srv := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/dataset/cancel", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "cancelled" {
		t.Errorf("expected status cancelled, got %v", res["status"])
	}
}

func TestCancelInFlightDatasetGeneration(t *testing.T) {
	slowProv := &slowLLMProvider{started: make(chan struct{}, 1)}
	gen := generator.NewGenerator(slowProv, generator.DefaultMatrix())
	jevClient := &testJevClient{}
	srv := NewServer(config.Default(), gen, jevClient, nil, nil)

	genDone := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		genReq := httptest.NewRequest(http.MethodPost, "/api/dataset/generate", strings.NewReader(`{"count": 5}`))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, genReq)
		genDone <- rec
	}()

	<-slowProv.started

	cancelReq := httptest.NewRequest(http.MethodPost, "/api/dataset/cancel", nil)
	cancelRec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(cancelRec, cancelReq)

	if cancelRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cancel, got %d", cancelRec.Code)
	}

	genRec := <-genDone
	if genRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for generation, got %d: %s", genRec.Code, genRec.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(genRec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode generation response: %v", err)
	}

	if res["cancelled"] != true {
		t.Errorf("expected cancelled true, got %v", res["cancelled"])
	}
	if res["generated"] != float64(0) {
		t.Errorf("expected 0 generated, got %v", res["generated"])
	}
}

func TestCancelTriageEndpoint(t *testing.T) {
	srv := setupTestServer()

	req := httptest.NewRequest(http.MethodPost, "/api/triage/cancel", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "cancelled" {
		t.Errorf("expected status cancelled, got %v", res["status"])
	}
}

func TestTestEngineEndpointDisabled(t *testing.T) {
	srv := setupTestServer()
	reqBody := `{"slot": "b", "provider": "none"}`
	req := httptest.NewRequest(http.MethodPost, "/api/config/test-engine", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["ok"] != true {
		t.Errorf("expected ok true, got %v", res["ok"])
	}
	if res["provider"] != "none" {
		t.Errorf("expected provider none, got %v", res["provider"])
	}
}

func TestRunTriageDualEngines(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/triage/run", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["triaged_count"] != float64(1) {
		t.Errorf("expected 1 triaged ticket, got %v", res["triaged_count"])
	}
	if res["has_engine_b"] != true {
		t.Errorf("expected has_engine_b true, got %v", res["has_engine_b"])
	}
	if res["engine_a_name"] != "Test Decision Engine" {
		t.Errorf("expected engine_a_name 'Test Decision Engine', got %v", res["engine_a_name"])
	}

	// Verify the ticket in memory has both triage results
	listReq := httptest.NewRequest(http.MethodGet, "/api/dataset", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	var tickets []*ticket.TriagedTicket
	if err := json.NewDecoder(listW.Body).Decode(&tickets); err != nil {
		t.Fatalf("failed to decode dataset: %v", err)
	}
	if len(tickets) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(tickets))
	}
	if tickets[0].TriageA == nil {
		t.Errorf("expected TriageA not nil")
	}
	if tickets[0].TriageB == nil {
		t.Errorf("expected TriageB not nil")
	}
}

func TestSingleTriageDualEngines(t *testing.T) {
	srv := setupTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/triage/INC-TEST-001", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var updated ticket.TriagedTicket
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if updated.TriageA == nil {
		t.Errorf("expected TriageA not nil")
	}
	if updated.TriageB == nil {
		t.Errorf("expected TriageB not nil")
	}
}

func TestRunTriageEngineBOnly(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	engineB := &testJevClient{}
	initial := []*ticket.Ticket{
		{
			ID:              "INC-TEST-B-001",
			CreatedAt:       time.Now(),
			ReporterName:    "Test User",
			ReporterEmail:   "test@example.com",
			Department:      "Engineering",
			Category:        "Hardware",
			ReportedUrgency: "High",
			Summary:         "Single engine B test",
			Description:     "Test description",
			AffectedSystem:  "Server",
		},
	}
	srv := NewServer(config.Default(), gen, nil, engineB, initial)

	req := httptest.NewRequest(http.MethodPost, "/api/triage/run", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["triaged_count"] != float64(1) {
		t.Errorf("expected 1 triaged ticket, got %v", res["triaged_count"])
	}
	if res["has_engine_a"] != false {
		t.Errorf("expected has_engine_a false, got %v", res["has_engine_a"])
	}
	if res["has_engine_b"] != true {
		t.Errorf("expected has_engine_b true, got %v", res["has_engine_b"])
	}
}

func TestRunTriageBothEnginesDisabled(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	initial := []*ticket.Ticket{
		{
			ID:              "INC-TEST-002",
			CreatedAt:       time.Now(),
			ReporterName:    "Test User",
			ReporterEmail:   "test@example.com",
			Department:      "Engineering",
			Category:        "Hardware",
			ReportedUrgency: "High",
			Summary:         "No engine test",
			Description:     "Test description",
			AffectedSystem:  "Server",
		},
	}
	srv := NewServer(config.Default(), gen, nil, nil, initial)

	req := httptest.NewRequest(http.MethodPost, "/api/triage/run", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSingleTriageEngineBOnly(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	engineB := &testJevClient{}
	initial := []*ticket.Ticket{
		{
			ID:              "INC-TEST-B-002",
			CreatedAt:       time.Now(),
			ReporterName:    "Test User",
			ReporterEmail:   "test@example.com",
			Department:      "Engineering",
			Category:        "Hardware",
			ReportedUrgency: "High",
			Summary:         "Single engine B test",
			Description:     "Test description",
			AffectedSystem:  "Server",
		},
	}
	srv := NewServer(config.Default(), gen, nil, engineB, initial)

	req := httptest.NewRequest(http.MethodPost, "/api/triage/INC-TEST-B-002", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var updated ticket.TriagedTicket
	if err := json.NewDecoder(w.Body).Decode(&updated); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if updated.TriageA != nil {
		t.Errorf("expected TriageA nil when Engine A disabled")
	}
	if updated.TriageB == nil {
		t.Errorf("expected TriageB not nil")
	}
}

func TestSingleTriageBothEnginesDisabled(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	initial := []*ticket.Ticket{
		{
			ID:              "INC-TEST-003",
			CreatedAt:       time.Now(),
			ReporterName:    "Test User",
			ReporterEmail:   "test@example.com",
			Department:      "Engineering",
			Category:        "Hardware",
			ReportedUrgency: "High",
			Summary:         "No engine test",
			Description:     "Test description",
			AffectedSystem:  "Server",
		},
	}
	srv := NewServer(config.Default(), gen, nil, nil, initial)

	req := httptest.NewRequest(http.MethodPost, "/api/triage/INC-TEST-003", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCriteriaEndpoints(t *testing.T) {
	srv := setupTestServer()

	// 1. GET /api/criteria should return standard questions
	reqGet := httptest.NewRequest(http.MethodGet, "/api/criteria", nil)
	recGet := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recGet, reqGet)
	if recGet.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recGet.Code)
	}
	var questions []triage.Question
	if err := json.NewDecoder(recGet.Body).Decode(&questions); err != nil {
		t.Fatalf("decode criteria failed: %v", err)
	}
	if len(questions) != 6 {
		t.Fatalf("expected 6 criteria questions, got %d", len(questions))
	}
	if questions[0].ID != "ticket_type" {
		t.Errorf("expected first question ticket_type, got %s", questions[0].ID)
	}

	// 2. POST /api/criteria to update criteria
	customCriteria := []triage.Question{
		{
			ID:      "custom_q",
			Type:    "choice",
			Text:    "Custom question text?",
			Options: []string{"Opt1", "Opt2"},
		},
	}
	customBytes, _ := json.Marshal(customCriteria)
	reqPost := httptest.NewRequest(http.MethodPost, "/api/criteria", strings.NewReader(string(customBytes)))
	recPost := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recPost, reqPost)
	if recPost.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recPost.Code, recPost.Body.String())
	}

	// Verify updated criteria
	recGet2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recGet2, reqGet)
	var updated []triage.Question
	_ = json.NewDecoder(recGet2.Body).Decode(&updated)
	if len(updated) != 1 || updated[0].ID != "custom_q" {
		t.Fatalf("expected custom criteria, got %v", updated)
	}

	// 3. POST /api/criteria/reset restores standard questions
	reqReset := httptest.NewRequest(http.MethodPost, "/api/criteria/reset", nil)
	recReset := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recReset, reqReset)
	if recReset.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recReset.Code, recReset.Body.String())
	}
	recGet3 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recGet3, reqGet)
	var restored []triage.Question
	_ = json.NewDecoder(recGet3.Body).Decode(&restored)
	if len(restored) != 6 {
		t.Fatalf("expected 6 restored questions, got %d", len(restored))
	}

	// 4. POST /api/criteria/save to persist criteria to file
	tmpCriteriaFile := filepath.Join(t.TempDir(), "test_questions.json")
	savePayload, _ := json.Marshal(map[string]string{"path": tmpCriteriaFile})
	reqSave := httptest.NewRequest(http.MethodPost, "/api/criteria/save", strings.NewReader(string(savePayload)))
	recSave := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recSave, reqSave)
	if recSave.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recSave.Code, recSave.Body.String())
	}

	// 5. POST /api/criteria/load to load criteria from file
	reqLoad := httptest.NewRequest(http.MethodPost, "/api/criteria/load", strings.NewReader(string(savePayload)))
	recLoad := httptest.NewRecorder()
	srv.Handler().ServeHTTP(recLoad, reqLoad)
	if recLoad.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recLoad.Code, recLoad.Body.String())
	}
	var loadResp struct {
		Status    string            `json:"status"`
		Questions []triage.Question `json:"questions"`
	}
	_ = json.NewDecoder(recLoad.Body).Decode(&loadResp)
	if len(loadResp.Questions) != 6 {
		t.Fatalf("expected 6 loaded questions, got %d", len(loadResp.Questions))
	}
}

func TestStaticAndTemplateServing(t *testing.T) {
	cfg := config.LoadDefault()
	srv := NewServer(cfg, nil, nil, nil, nil)
	handler := srv.Handler()

	// 1. Root dashboard serves pre-rendered HTML with all four modular page templates
	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	recRoot := httptest.NewRecorder()
	handler.ServeHTTP(recRoot, reqRoot)
	if recRoot.Code != http.StatusOK {
		t.Fatalf("expected 200 for root, got %d", recRoot.Code)
	}
	body := recRoot.Body.String()
	requiredElements := []string{
		`id="page-config"`,
		`id="page-dataset"`,
		`id="page-criteria"`,
		`id="page-showcase"`,
		`/static/css/style.css`,
		`/static/js/app.js`,
		`/static/js/config.js`,
		`/static/js/dataset.js`,
		`/static/js/criteria.js`,
		`/static/js/decision.js`,
	}
	for _, elem := range requiredElements {
		if !strings.Contains(body, elem) {
			t.Errorf("expected dashboard HTML to contain %q", elem)
		}
	}

	// 2. Static CSS file is served
	reqCSS := httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil)
	recCSS := httptest.NewRecorder()
	handler.ServeHTTP(recCSS, reqCSS)
	if recCSS.Code != http.StatusOK {
		t.Fatalf("expected 200 for style.css, got %d", recCSS.Code)
	}
	if recCSS.Body.Len() == 0 {
		t.Fatal("expected non-empty CSS content")
	}

	// 3. Static JS files are served
	jsFiles := []string{
		"/static/js/app.js",
		"/static/js/config.js",
		"/static/js/dataset.js",
		"/static/js/criteria.js",
		"/static/js/decision.js",
	}
	for _, jsPath := range jsFiles {
		reqJS := httptest.NewRequest(http.MethodGet, jsPath, nil)
		recJS := httptest.NewRecorder()
		handler.ServeHTTP(recJS, reqJS)
		if recJS.Code != http.StatusOK {
			t.Errorf("expected 200 for %s, got %d", jsPath, recJS.Code)
		}
		if recJS.Body.Len() == 0 {
			t.Errorf("expected non-empty JS content for %s", jsPath)
		}
	}

	// 4. Unknown route returns 404
	req404 := httptest.NewRequest(http.MethodGet, "/unknown-page", nil)
	rec404 := httptest.NewRecorder()
	handler.ServeHTTP(rec404, req404)
	if rec404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown route, got %d", rec404.Code)
	}
}

func TestRunTriageTrickleFeed(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	engineA := &testJevClient{}
	engineB := &testJevClient{}

	var initial []*ticket.Ticket
	for i := 1; i <= 6; i++ {
		initial = append(initial, &ticket.Ticket{
			ID:              fmt.Sprintf("INC-BATCH-%03d", i),
			CreatedAt:       time.Now(),
			ReporterName:    "Tester",
			ReporterEmail:   "tester@example.com",
			Department:      "Engineering",
			Category:        "Software",
			ReportedUrgency: "Medium",
			Summary:         fmt.Sprintf("Issue %d", i),
			Description:     fmt.Sprintf("Description %d", i),
			AffectedSystem:  "App",
		})
	}

	srv := NewServer(config.Default(), gen, engineA, engineB, initial)

	body := strings.NewReader(`{"concurrency": 3, "interval_ms": 10}`)
	req := httptest.NewRequest(http.MethodPost, "/api/triage/run", body)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["triaged_count"] != float64(6) {
		t.Errorf("expected 6 triaged tickets, got %v", res["triaged_count"])
	}

	// Verify all tickets in dataset have both triage results
	listReq := httptest.NewRequest(http.MethodGet, "/api/dataset", nil)
	listW := httptest.NewRecorder()
	srv.Handler().ServeHTTP(listW, listReq)

	var tickets []*ticket.TriagedTicket
	if err := json.NewDecoder(listW.Body).Decode(&tickets); err != nil {
		t.Fatalf("failed to decode dataset: %v", err)
	}
	if len(tickets) != 6 {
		t.Fatalf("expected 6 tickets, got %d", len(tickets))
	}
	for i, tick := range tickets {
		if tick.TriageA == nil {
			t.Errorf("ticket %d: expected TriageA not nil", i)
		}
		if tick.TriageB == nil {
			t.Errorf("ticket %d: expected TriageB not nil", i)
		}
		if tick.Status != "" {
			t.Errorf("ticket %d: expected empty status after completion, got %q", i, tick.Status)
		}
	}
}

func TestRunTriageSerialMode(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := generator.NewGenerator(testProv, generator.DefaultMatrix())
	engineA := &testJevClient{}
	engineB := &testJevClient{}

	var initial []*ticket.Ticket
	for i := 1; i <= 3; i++ {
		initial = append(initial, &ticket.Ticket{
			ID:              fmt.Sprintf("INC-SERIAL-%03d", i),
			CreatedAt:       time.Now(),
			ReporterName:    "Tester",
			ReporterEmail:   "tester@example.com",
			Department:      "Engineering",
			Category:        "Software",
			ReportedUrgency: "Low",
			Summary:         fmt.Sprintf("Serial Issue %d", i),
			Description:     fmt.Sprintf("Description %d", i),
			AffectedSystem:  "App",
		})
	}

	cfg := config.Default()
	cfg.EvaluationMode = "serial"
	srv := NewServer(cfg, gen, engineA, engineB, initial)

	req := httptest.NewRequest(http.MethodPost, "/api/triage/run", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	var res map[string]any
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["triaged_count"] != float64(3) {
		t.Errorf("expected 3 triaged tickets, got %v", res["triaged_count"])
	}
}

func TestSSEStream(t *testing.T) {
	srv := setupTestServer()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/stream", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to stream: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("expected Content-Type text/event-stream, got %q", ct)
	}

	reader := bufio.NewReader(resp.Body)
	// Verify initial connected comment
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read initial line: %v", err)
	}
	if strings.TrimSpace(line) != ": connected" {
		t.Fatalf("expected initial ': connected', got %q", line)
	}

	// Trigger single generation to verify ticket broadcast
	genReq, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL+"/api/dataset/generate", strings.NewReader(`{"count":1}`))
	if err != nil {
		t.Fatalf("failed to create generate request: %v", err)
	}
	genReq.Header.Set("Content-Type", "application/json")
	genResp, err := http.DefaultClient.Do(genReq)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	defer genResp.Body.Close()
	if genResp.StatusCode != http.StatusOK {
		t.Fatalf("expected generate status 200, got %d", genResp.StatusCode)
	}

	// Read event from stream until data: is found
	var foundData bool
	for i := 0; i < 10; i++ {
		l, rErr := reader.ReadString('\n')
		if rErr != nil {
			t.Fatalf("failed reading event: %v", rErr)
		}
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "data: ") {
			jsonPayload := strings.TrimPrefix(trimmed, "data: ")
			var item ticket.TriagedTicket
			if jErr := json.Unmarshal([]byte(jsonPayload), &item); jErr != nil {
				t.Fatalf("failed to unmarshal ticket from SSE: %v", jErr)
			}
			if item.ID == "" {
				t.Fatalf("expected non-empty ticket ID from SSE")
			}
			foundData = true
			break
		}
	}
	if !foundData {
		t.Fatalf("expected to receive ticket data event from SSE stream")
	}
}
