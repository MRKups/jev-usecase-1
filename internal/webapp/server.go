// Package webapp provides the HTTP server, REST endpoints, SSE streaming, and presentation UI.
package webapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/generator"
	"github.com/MRKups/jev-usecase-1/internal/jev"
	"github.com/MRKups/jev-usecase-1/internal/provider"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

// Server coordinates configuration, generation, triage, and UI presentation.
type Server struct {
	mu           sync.RWMutex
	cfg          config.AppConfig
	gen          *generator.Generator
	engineA      triage.Engine
	engineB      triage.Engine
	tickets      []*ticket.TriagedTicket
	questions    []triage.Question
	listeners    map[chan *ticket.TriagedTicket]struct{}
	cancelGen    context.CancelFunc
	cancelTriage context.CancelFunc
}

// NewServer creates a new webapp server with Decision Engines A and B.
func NewServer(cfg config.AppConfig, gen *generator.Generator, engineA triage.Engine, engineB triage.Engine, initialTickets []*ticket.Ticket, initialQuestions ...[]triage.Question) *Server {
	var questions []triage.Question
	if len(initialQuestions) > 0 && len(initialQuestions[0]) > 0 {
		questions = initialQuestions[0]
	} else {
		candidates := []string{"questions.json", "criteria.json"}
		for _, c := range candidates {
			if data, err := os.ReadFile(c); err == nil {
				var loaded []triage.Question
				if err := json.Unmarshal(data, &loaded); err == nil && len(loaded) > 0 {
					questions = loaded
					break
				}
			}
		}
	}
	if len(questions) == 0 {
		questions = triage.StandardQuestions()
	}

	if len(initialTickets) == 0 {
		candidates := []string{"dataset.json", "data.json", "tickets.json"}
		if cfg.DatasetPath != "" {
			candidates = append([]string{cfg.DatasetPath}, candidates...)
		}
		for _, c := range candidates {
			if data, err := os.ReadFile(c); err == nil {
				if loaded, err := ticket.DecodeJSON(strings.NewReader(string(data))); err == nil && len(loaded) > 0 {
					initialTickets = loaded
					break
				}
			}
		}
	}

	s := &Server{
		cfg:       cfg,
		gen:       gen,
		engineA:   engineA,
		engineB:   engineB,
		questions: questions,
		listeners: make(map[chan *ticket.TriagedTicket]struct{}),
		tickets:   make([]*ticket.TriagedTicket, 0, len(initialTickets)),
	}

	for _, t := range initialTickets {
		s.tickets = append(s.tickets, &ticket.TriagedTicket{Ticket: *t})
	}

	return s
}

func CreateEngine(engineCfg config.DecisionEngineConfig, appCfg config.AppConfig) (triage.Engine, error) {
	providerName := strings.ToLower(strings.TrimSpace(engineCfg.Provider))
	switch providerName {
	case "jev", "":
		jevCfg := appCfg.Jev
		if engineCfg.Endpoint != "" {
			jevCfg.Endpoint = engineCfg.Endpoint
		}
		if engineCfg.APIKey != "" {
			jevCfg.APIKey = engineCfg.APIKey
		}
		if engineCfg.Model != "" {
			jevCfg.Model = engineCfg.Model
		}
		return jev.NewHTTPClient(jevCfg), nil

	case "openai":
		cfg := appCfg.OpenAI
		if engineCfg.APIKey != "" {
			cfg.APIKey = engineCfg.APIKey
		}
		if engineCfg.Model != "" {
			cfg.Model = engineCfg.Model
		}
		if engineCfg.Endpoint != "" {
			cfg.BaseURL = engineCfg.Endpoint
		}
		return provider.NewOpenAIProvider(cfg)

	case "anthropic":
		cfg := appCfg.Anthropic
		if engineCfg.APIKey != "" {
			cfg.APIKey = engineCfg.APIKey
		}
		if engineCfg.Model != "" {
			cfg.Model = engineCfg.Model
		}
		if engineCfg.Endpoint != "" {
			cfg.BaseURL = engineCfg.Endpoint
		}
		return provider.NewAnthropicProvider(cfg)

	case "gemini":
		cfg := appCfg.Gemini
		if engineCfg.APIKey != "" {
			cfg.APIKey = engineCfg.APIKey
		}
		if engineCfg.Model != "" {
			cfg.Model = engineCfg.Model
		}
		return provider.NewGeminiProvider(context.Background(), cfg)

	case "none", "disabled":
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown engine provider: %s", engineCfg.Provider)
	}
}

// Handler returns the HTTP handler with all registered endpoints.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))

	// Config endpoints
	mux.HandleFunc("GET /api/config", s.handleGetConfig)
	mux.HandleFunc("POST /api/config", s.handleUpdateConfig)
	mux.HandleFunc("POST /api/config/save", s.handleSaveConfig)
	mux.HandleFunc("POST /api/config/load", s.handleLoadConfig)
	mux.HandleFunc("POST /api/config/test-provider", s.handleTestProvider)
	mux.HandleFunc("POST /api/config/test-jev", s.handleTestJev)
	mux.HandleFunc("POST /api/config/test-engine", s.handleTestEngine)

	// Dataset endpoints
	mux.HandleFunc("GET /api/dataset", s.handleGetDataset)
	mux.HandleFunc("POST /api/dataset/generate", s.handleGenerateDataset)
	mux.HandleFunc("POST /api/dataset/cancel", s.handleCancelDataset)
	mux.HandleFunc("POST /api/dataset/save", s.handleSaveDataset)
	mux.HandleFunc("POST /api/dataset/load", s.handleLoadDataset)
	mux.HandleFunc("POST /api/dataset/clear", s.handleClearDataset)

	// Criteria endpoints
	mux.HandleFunc("GET /api/criteria", s.handleGetCriteria)
	mux.HandleFunc("POST /api/criteria", s.handleUpdateCriteria)
	mux.HandleFunc("POST /api/criteria/save", s.handleSaveCriteria)
	mux.HandleFunc("POST /api/criteria/load", s.handleLoadCriteria)
	mux.HandleFunc("POST /api/criteria/reset", s.handleResetCriteria)

	// Triage endpoints
	mux.HandleFunc("POST /api/triage/run", s.handleRunTriage)
	mux.HandleFunc("POST /api/triage/cancel", s.handleCancelTriage)
	mux.HandleFunc("POST /api/triage/{id}", s.handleTriageSingle)

	// Stream
	mux.HandleFunc("GET /api/stream", s.handleSSEStream)

	// Backwards compatibility with previous endpoints
	mux.HandleFunc("GET /api/tickets", s.handleGetDataset)
	mux.HandleFunc("POST /api/tickets/generate", s.handleGenerateSingle)
	mux.HandleFunc("POST /api/tickets/{id}/triage", s.handleTriageSingle)
	mux.HandleFunc("POST /api/triage/all", s.handleRunTriage)

	return mux
}

func (s *Server) broadcast(item *ticket.TriagedTicket) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for ch := range s.listeners {
		select {
		case ch <- item:
		default:
		}
	}
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(IndexHTML))
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.cfg)
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var newCfg config.AppConfig
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		http.Error(w, fmt.Sprintf("invalid config: %v", err), http.StatusBadRequest)
		return
	}

	p, err := provider.Factory(r.Context(), newCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize generator: %v", err), http.StatusBadRequest)
		return
	}

	engA, err := CreateEngine(newCfg.EngineA, newCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize decision engine A: %v", err), http.StatusBadRequest)
		return
	}
	engB, err := CreateEngine(newCfg.EngineB, newCfg)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize decision engine B: %v", err), http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	s.cfg = newCfg
	s.gen = generator.NewGenerator(p, generator.DefaultMatrix())
	s.engineA = engA
	s.engineB = engB
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "updated",
		"config": newCfg,
	})
}

func (s *Server) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := req.Path
	if targetPath == "" {
		targetPath = "config.json"
	}

	if err := cfg.SaveToFile(targetPath); err != nil {
		http.Error(w, fmt.Sprintf("save config failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": "saved",
		"path":   targetPath,
	})
}

func (s *Server) handleLoadConfig(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := req.Path
	if targetPath == "" {
		targetPath = "config.json"
	}

	loaded, err := config.LoadFromFile(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("load config failed: %v", err), http.StatusBadRequest)
		return
	}

	p, err := provider.Factory(r.Context(), loaded)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize generator from loaded config: %v", err), http.StatusBadRequest)
		return
	}
	engA, err := CreateEngine(loaded.EngineA, loaded)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to initialize decision engine A from loaded config: %v", err), http.StatusBadRequest)
		return
	}
	engB, _ := CreateEngine(loaded.EngineB, loaded)

	s.mu.Lock()
	s.cfg = loaded
	s.gen = generator.NewGenerator(p, generator.DefaultMatrix())
	s.engineA = engA
	s.engineB = engB
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "loaded",
		"config": loaded,
	})
}

func (s *Server) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	testCfg := s.cfg
	s.mu.RUnlock()

	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&testCfg)
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	p, err := provider.Factory(ctx, testCfg)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":       false,
			"provider": testCfg.ActiveGenerator,
			"error":    fmt.Sprintf("provider initialization failed: %v", err),
		})
		return
	}

	model := ""
	switch strings.ToLower(testCfg.ActiveGenerator) {
	case "openai":
		model = testCfg.OpenAI.Model
	case "anthropic":
		model = testCfg.Anthropic.Model
	case "gemini":
		model = testCfg.Gemini.Model
	}

	start := time.Now()
	err = p.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         false,
			"provider":   testCfg.ActiveGenerator,
			"model":      model,
			"latency_ms": latency,
			"error":      err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         true,
		"provider":   testCfg.ActiveGenerator,
		"model":      model,
		"latency_ms": latency,
		"message":    fmt.Sprintf("Connected successfully: %s responded in %dms.", strings.ToUpper(testCfg.ActiveGenerator), latency),
	})
}

func (s *Server) handleTestJev(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	s.mu.RLock()
	testCfg := s.cfg.Jev
	s.mu.RUnlock()

	var reqBody struct {
		Jev      config.JevConfig `json:"jev"`
		Endpoint string           `json:"endpoint"`
		APIKey   string           `json:"api_key"`
		Model    string           `json:"model"`
	}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err == nil {
			if reqBody.Jev.Endpoint != "" || reqBody.Jev.APIKey != "" || reqBody.Jev.Model != "" {
				testCfg = reqBody.Jev
			} else {
				if reqBody.Endpoint != "" {
					testCfg.Endpoint = reqBody.Endpoint
				}
				if reqBody.APIKey != "" {
					testCfg.APIKey = reqBody.APIKey
				}
				if reqBody.Model != "" {
					testCfg.Model = reqBody.Model
				}
			}
		}
	}

	client := jev.NewHTTPClient(testCfg)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err := client.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         false,
			"latency_ms": latency,
			"error":      err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         true,
		"latency_ms": latency,
		"message":    fmt.Sprintf("Connected successfully: TypeSafe Jev Decision Engine responded in %dms.", latency),
	})
}

func (s *Server) handleTestEngine(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req struct {
		Slot     string `json:"slot"`
		Provider string `json:"provider"`
		Endpoint string `json:"endpoint"`
		Model    string `json:"model"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": fmt.Sprintf("invalid test request: %v", err),
		})
		return
	}

	providerName := strings.ToLower(strings.TrimSpace(req.Provider))
	if providerName == "none" || providerName == "disabled" {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         true,
			"slot":       req.Slot,
			"provider":   "none",
			"latency_ms": 0,
			"message":    "Decision Engine is disabled.",
		})
		return
	}

	s.mu.RLock()
	currentCfg := s.cfg
	s.mu.RUnlock()

	engCfg := config.DecisionEngineConfig{
		Provider: providerName,
		Endpoint: req.Endpoint,
		Model:    req.Model,
		APIKey:   req.APIKey,
	}

	engine, err := CreateEngine(engCfg, currentCfg)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":       false,
			"slot":     req.Slot,
			"provider": req.Provider,
			"error":    fmt.Sprintf("initialization failed: %v", err),
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	err = engine.Ping(ctx)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":         false,
			"slot":       req.Slot,
			"provider":   req.Provider,
			"latency_ms": latency,
			"error":      err.Error(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":         true,
		"slot":       req.Slot,
		"provider":   req.Provider,
		"model":      req.Model,
		"latency_ms": latency,
		"message":    fmt.Sprintf("Connected successfully: %s responded in %dms.", engine.Name(), latency),
	})
}

func (s *Server) handleGetDataset(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	tickets := s.tickets
	if tickets == nil {
		tickets = []*ticket.TriagedTicket{}
	}
	_ = json.NewEncoder(w).Encode(tickets)
}

func (s *Server) handleGenerateDataset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Count <= 0 {
		req.Count = 10
	}
	if req.Count < 1 {
		req.Count = 1
	}
	if req.Count > 99 {
		req.Count = 99
	}

	s.mu.RLock()
	gen := s.gen
	s.mu.RUnlock()

	genCtx, cancel := context.WithCancel(r.Context())
	s.mu.Lock()
	s.cancelGen = cancel
	s.mu.Unlock()

	defer func() {
		cancel()
		s.mu.Lock()
		s.cancelGen = nil
		s.mu.Unlock()
	}()

	var generatedItems []*ticket.TriagedTicket
	var lastGenErr error
	for i := 0; i < req.Count; i++ {
		if genCtx.Err() != nil {
			break
		}

		var t *ticket.Ticket
		var err error
		for attempt := 1; attempt <= 3; attempt++ {
			if genCtx.Err() != nil {
				break
			}
			t, err = gen.GenerateOne(genCtx)
			if err == nil {
				break
			}
			log.Printf("ticket %d/%d generation attempt %d failed: %v", i+1, req.Count, attempt, err)
			if attempt < 3 && genCtx.Err() == nil {
				select {
				case <-genCtx.Done():
				case <-time.After(time.Duration(attempt) * time.Second):
				}
			}
		}

		if err != nil {
			lastGenErr = err
			log.Printf("ticket %d/%d generation aborted after retries: %v", i+1, req.Count, err)
			if genCtx.Err() != nil {
				break
			}
			if len(generatedItems) == 0 {
				http.Error(w, fmt.Sprintf("generation failed: %v", err), http.StatusInternalServerError)
				return
			}
			break
		}

		item := &ticket.TriagedTicket{Ticket: *t}
		s.mu.Lock()
		s.tickets = append([]*ticket.TriagedTicket{item}, s.tickets...)
		s.mu.Unlock()

		s.broadcast(item)
		generatedItems = append(generatedItems, item)
	}

	s.mu.RLock()
	totalCount := len(s.tickets)
	s.mu.RUnlock()

	resp := map[string]any{
		"generated": len(generatedItems),
		"requested": req.Count,
		"total":     totalCount,
		"tickets":   generatedItems,
		"cancelled": genCtx.Err() != nil,
	}
	if lastGenErr != nil {
		resp["error"] = lastGenErr.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleCancelDataset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.cancelGen != nil {
		s.cancelGen()
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "cancelled",
	})
}

func (s *Server) handleGenerateSingle(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	gen := s.gen
	s.mu.RUnlock()

	t, err := gen.GenerateOne(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("generate ticket failed: %v", err), http.StatusInternalServerError)
		return
	}

	item := &ticket.TriagedTicket{Ticket: *t}
	s.mu.Lock()
	s.tickets = append([]*ticket.TriagedTicket{item}, s.tickets...)
	s.mu.Unlock()

	s.broadcast(item)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(item)
}

func (s *Server) handleSaveDataset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := req.Path
	if targetPath == "" {
		s.mu.RLock()
		targetPath = s.cfg.DatasetPath
		s.mu.RUnlock()
	}
	if targetPath == "" {
		targetPath = "dataset.json"
	}

	s.mu.RLock()
	rawTickets := make([]*ticket.Ticket, 0, len(s.tickets))
	for _, item := range s.tickets {
		t := item.Ticket
		rawTickets = append(rawTickets, &t)
	}
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		http.Error(w, fmt.Sprintf("create dataset directory: %v", err), http.StatusInternalServerError)
		return
	}

	f, err := os.Create(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("create dataset file: %v", err), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	if err := ticket.EncodeJSON(f, rawTickets); err != nil {
		http.Error(w, fmt.Sprintf("encode dataset json: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "saved",
		"path":   targetPath,
		"count":  len(rawTickets),
	})
}

func (s *Server) handleLoadDataset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := req.Path
	if targetPath == "" {
		s.mu.RLock()
		targetPath = s.cfg.DatasetPath
		s.mu.RUnlock()
	}
	if targetPath == "" {
		targetPath = "dataset.json"
	}

	f, err := os.Open(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("open dataset file: %v", err), http.StatusBadRequest)
		return
	}
	defer f.Close()

	rawTickets, err := ticket.DecodeJSON(f)
	if err != nil {
		http.Error(w, fmt.Sprintf("decode dataset json: %v", err), http.StatusBadRequest)
		return
	}

	items := make([]*ticket.TriagedTicket, 0, len(rawTickets))
	for _, t := range rawTickets {
		items = append(items, &ticket.TriagedTicket{Ticket: *t})
	}

	s.mu.Lock()
	s.tickets = items
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "loaded",
		"path":    targetPath,
		"count":   len(items),
		"tickets": items,
	})
}

func (s *Server) handleClearDataset(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.tickets = []*ticket.TriagedTicket{}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "cleared"})
}

func (s *Server) handleGetCriteria(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(s.questions)
}

func (s *Server) handleUpdateCriteria(w http.ResponseWriter, r *http.Request) {
	var updated []triage.Question
	if err := json.NewDecoder(r.Body).Decode(&updated); err != nil {
		http.Error(w, fmt.Sprintf("invalid criteria json: %v", err), http.StatusBadRequest)
		return
	}
	if len(updated) == 0 {
		http.Error(w, "at least one question is required", http.StatusBadRequest)
		return
	}
	for i, q := range updated {
		if strings.TrimSpace(q.ID) == "" {
			http.Error(w, fmt.Sprintf("question %d is missing id", i+1), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(q.Text) == "" {
			http.Error(w, fmt.Sprintf("question %d is missing text", i+1), http.StatusBadRequest)
			return
		}
	}
	s.mu.Lock()
	s.questions = updated
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "saved",
		"questions": updated,
	})
}

func (s *Server) handleResetCriteria(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	s.questions = triage.StandardQuestions()
	res := s.questions
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "reset",
		"questions": res,
	})
}

func (s *Server) handleSaveCriteria(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := strings.TrimSpace(req.Path)
	if targetPath == "" {
		targetPath = "questions.json"
	}

	s.mu.RLock()
	qs := s.questions
	s.mu.RUnlock()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		http.Error(w, fmt.Sprintf("create criteria directory: %v", err), http.StatusInternalServerError)
		return
	}

	data, err := json.MarshalIndent(qs, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("encode criteria json: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(targetPath, data, 0644); err != nil {
		http.Error(w, fmt.Sprintf("write criteria file: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "saved",
		"path":   targetPath,
		"count":  len(qs),
	})
}

func (s *Server) handleLoadCriteria(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	targetPath := strings.TrimSpace(req.Path)
	if targetPath == "" {
		targetPath = "questions.json"
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("read criteria file: %v", err), http.StatusBadRequest)
		return
	}

	var loaded []triage.Question
	if err := json.Unmarshal(data, &loaded); err != nil {
		http.Error(w, fmt.Sprintf("decode criteria json: %v", err), http.StatusBadRequest)
		return
	}

	if len(loaded) == 0 {
		http.Error(w, "criteria file contains no questions", http.StatusBadRequest)
		return
	}

	for i, q := range loaded {
		if strings.TrimSpace(q.ID) == "" {
			http.Error(w, fmt.Sprintf("question %d is missing id", i+1), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(q.Text) == "" {
			http.Error(w, fmt.Sprintf("question %d is missing text", i+1), http.StatusBadRequest)
			return
		}
	}

	s.mu.Lock()
	s.questions = loaded
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":    "loaded",
		"path":      targetPath,
		"count":     len(loaded),
		"questions": loaded,
	})
}

type triageRunStats struct {
	mu            sync.Mutex
	triagedCount  int
	totalLatencyA int64
	totalLatencyB int64
	countA        int
	countB        int
}

func (s *triageRunStats) recordA(latencyMs int64) {
	s.mu.Lock()
	s.totalLatencyA += latencyMs
	s.countA++
	s.mu.Unlock()
}

func (s *triageRunStats) recordB(latencyMs int64) {
	s.mu.Lock()
	s.totalLatencyB += latencyMs
	s.countB++
	s.mu.Unlock()
}

func (s *triageRunStats) recordCompleted() {
	s.mu.Lock()
	s.triagedCount++
	s.mu.Unlock()
}

func (s *Server) evaluateTicketInRun(
	ctx context.Context,
	item *ticket.TriagedTicket,
	questions []triage.Question,
	engineA, engineB triage.Engine,
	stats *triageRunStats,
) {
	s.mu.Lock()
	var startTicket *ticket.TriagedTicket
	for i, curr := range s.tickets {
		if curr.ID == item.ID {
			startTicket = &ticket.TriagedTicket{
				Ticket: curr.Ticket,
				Status: "evaluating",
			}
			if engineA == nil {
				startTicket.TriageA = curr.TriageA
			}
			if engineB == nil {
				startTicket.TriageB = curr.TriageB
			}
			s.tickets[i] = startTicket
			break
		}
	}
	s.mu.Unlock()

	if startTicket != nil {
		s.broadcast(startTicket)
	}

	var resA *ticket.TriageResult
	var resB *ticket.TriageResult

	var wg sync.WaitGroup
	if engineA != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, aErr := engineA.Classify(ctx, &item.Ticket, questions)
			if aErr != nil {
				log.Printf("engine A triage error for %s: %v", item.ID, aErr)
				return
			}
			resA = res
			stats.recordA(res.LatencyMs)

			s.mu.Lock()
			var intermediate *ticket.TriagedTicket
			for i, curr := range s.tickets {
				if curr.ID == item.ID {
					intermediate = &ticket.TriagedTicket{
						Ticket:  curr.Ticket,
						TriageA: resA,
						TriageB: curr.TriageB,
					}
					s.tickets[i] = intermediate
					break
				}
			}
			s.mu.Unlock()

			if intermediate != nil {
				s.broadcast(intermediate)
			}
		}()
	}

	if engineB != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, bErr := engineB.Classify(ctx, &item.Ticket, questions)
			if bErr != nil {
				log.Printf("engine B triage error for %s: %v", item.ID, bErr)
				return
			}
			resB = res
			stats.recordB(res.LatencyMs)

			s.mu.Lock()
			var intermediate *ticket.TriagedTicket
			for i, curr := range s.tickets {
				if curr.ID == item.ID {
					intermediate = &ticket.TriagedTicket{
						Ticket:  curr.Ticket,
						TriageA: curr.TriageA,
						TriageB: resB,
					}
					s.tickets[i] = intermediate
					break
				}
			}
			s.mu.Unlock()

			if intermediate != nil {
				s.broadcast(intermediate)
			}
		}()
	}

	wg.Wait()

	if resA != nil || resB != nil {
		stats.recordCompleted()
	} else if ctx.Err() != nil {
		s.mu.Lock()
		for i, curr := range s.tickets {
			if curr.ID == item.ID && curr.Status == "evaluating" {
				s.tickets[i] = &ticket.TriagedTicket{
					Ticket:  curr.Ticket,
					TriageA: curr.TriageA,
					TriageB: curr.TriageB,
					Status:  "",
				}
				s.broadcast(s.tickets[i])
				break
			}
		}
		s.mu.Unlock()
	}
}

type runTriageRequest struct {
	Concurrency int  `json:"concurrency,omitempty"`
	IntervalMs  *int `json:"interval_ms,omitempty"`
}

func (s *Server) handleRunTriage(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	engineA := s.engineA
	engineB := s.engineB
	evalMode := s.cfg.EvaluationMode
	items := make([]*ticket.TriagedTicket, len(s.tickets))
	copy(items, s.tickets)
	questions := make([]triage.Question, len(s.questions))
	copy(questions, s.questions)
	s.mu.RUnlock()

	if engineA == nil && engineB == nil {
		http.Error(w, "no decision engine is configured", http.StatusBadRequest)
		return
	}

	var req runTriageRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	maxConcurrent := 10
	interval := 1 * time.Second

	if strings.ToLower(evalMode) == "serial" {
		maxConcurrent = 1
		interval = 0
	}

	if req.Concurrency > 0 && req.Concurrency <= 50 {
		maxConcurrent = req.Concurrency
	} else if cStr := r.URL.Query().Get("concurrency"); cStr != "" {
		if c, err := strconv.Atoi(cStr); err == nil && c > 0 && c <= 50 {
			maxConcurrent = c
		}
	}

	if req.IntervalMs != nil && *req.IntervalMs >= 0 {
		interval = time.Duration(*req.IntervalMs) * time.Millisecond
	} else if intStr := r.URL.Query().Get("interval_ms"); intStr != "" {
		if ms, err := strconv.Atoi(intStr); err == nil && ms >= 0 {
			interval = time.Duration(ms) * time.Millisecond
		}
	}

	triageCtx, cancel := context.WithCancel(r.Context())
	s.mu.Lock()
	s.cancelTriage = cancel
	s.mu.Unlock()

	defer func() {
		cancel()
		s.mu.Lock()
		s.cancelTriage = nil
		s.mu.Unlock()
	}()

	sem := make(chan struct{}, maxConcurrent)
	var ticker *time.Ticker
	if interval > 0 {
		ticker = time.NewTicker(interval)
		defer ticker.Stop()
	}

	var allWg sync.WaitGroup
	stats := &triageRunStats{}

dispatchLoop:
	for i, item := range items {
		if triageCtx.Err() != nil {
			break dispatchLoop
		}

		// Rate pacing: first ticket (i == 0) starts immediately.
		// Subsequent tickets wait for ticker tick if interval > 0.
		if i > 0 && ticker != nil {
			select {
			case <-triageCtx.Done():
				break dispatchLoop
			case <-ticker.C:
			}
		}

		// Concurrency bounding: acquire a semaphore slot.
		// Blocks if maxConcurrent tickets are currently in flight.
		select {
		case <-triageCtx.Done():
			break dispatchLoop
		case sem <- struct{}{}:
		}

		if triageCtx.Err() != nil {
			<-sem
			break dispatchLoop
		}

		allWg.Add(1)
		go func(targetItem *ticket.TriagedTicket) {
			defer allWg.Done()
			defer func() { <-sem }()

			s.evaluateTicketInRun(triageCtx, targetItem, questions, engineA, engineB, stats)
		}(item)
	}

	allWg.Wait()

	stats.mu.Lock()
	triagedCount := stats.triagedCount
	var avgLatencyA int64
	if stats.countA > 0 {
		avgLatencyA = stats.totalLatencyA / int64(stats.countA)
	}
	var avgLatencyB int64
	if stats.countB > 0 {
		avgLatencyB = stats.totalLatencyB / int64(stats.countB)
	}
	stats.mu.Unlock()

	var nameA string
	if engineA != nil {
		nameA = engineA.Name()
	}
	var nameB string
	if engineB != nil {
		nameB = engineB.Name()
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"triaged_count":    triagedCount,
		"avg_latency_ms":   avgLatencyA,
		"avg_latency_b_ms": avgLatencyB,
		"engine_a_name":    nameA,
		"engine_b_name":    nameB,
		"has_engine_a":     engineA != nil,
		"has_engine_b":     engineB != nil,
		"cancelled":        triageCtx.Err() != nil,
	})
}

func (s *Server) handleCancelTriage(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	if s.cancelTriage != nil {
		s.cancelTriage()
	}
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "cancelled",
	})
}

func (s *Server) handleTriageSingle(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 3 {
			id = parts[2]
		}
	}

	s.mu.RLock()
	var target *ticket.TriagedTicket
	for _, item := range s.tickets {
		if item.ID == id || strings.TrimPrefix(item.ID, "#") == strings.TrimPrefix(id, "#") {
			target = item
			break
		}
	}
	engineA := s.engineA
	engineB := s.engineB
	questions := make([]triage.Question, len(s.questions))
	copy(questions, s.questions)
	s.mu.RUnlock()

	if target == nil {
		http.Error(w, "ticket not found", http.StatusNotFound)
		return
	}
	targetID := target.ID
	if engineA == nil && engineB == nil {
		http.Error(w, "no decision engine is configured", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	var startTicket *ticket.TriagedTicket
	for i, curr := range s.tickets {
		if curr.ID == targetID {
			startTicket = &ticket.TriagedTicket{
				Ticket: curr.Ticket,
				Status: "evaluating",
			}
			if engineA == nil {
				startTicket.TriageA = curr.TriageA
			}
			if engineB == nil {
				startTicket.TriageB = curr.TriageB
			}
			s.tickets[i] = startTicket
			break
		}
	}
	s.mu.Unlock()

	if startTicket != nil {
		s.broadcast(startTicket)
	}

	var resA *ticket.TriageResult
	var resB *ticket.TriageResult
	var errA error
	var errB error

	var wg sync.WaitGroup
	if engineA != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var aErr error
			res, aErr := engineA.Classify(r.Context(), &target.Ticket, questions)
			if aErr != nil {
				errA = aErr
				log.Printf("engine A triage failed for %s: %v", targetID, aErr)
				return
			}
			resA = res

			s.mu.Lock()
			var intermediate *ticket.TriagedTicket
			for i, curr := range s.tickets {
				if curr.ID == targetID {
					intermediate = &ticket.TriagedTicket{
						Ticket:  curr.Ticket,
						TriageA: resA,
						TriageB: curr.TriageB,
					}
					s.tickets[i] = intermediate
					break
				}
			}
			s.mu.Unlock()

			if intermediate != nil {
				s.broadcast(intermediate)
			}
		}()
	}

	if engineB != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var bErr error
			res, bErr := engineB.Classify(r.Context(), &target.Ticket, questions)
			if bErr != nil {
				errB = bErr
				log.Printf("engine B triage failed for %s: %v", targetID, bErr)
				return
			}
			resB = res

			s.mu.Lock()
			var intermediate *ticket.TriagedTicket
			for i, curr := range s.tickets {
				if curr.ID == targetID {
					intermediate = &ticket.TriagedTicket{
						Ticket:  curr.Ticket,
						TriageA: curr.TriageA,
						TriageB: resB,
					}
					s.tickets[i] = intermediate
					break
				}
			}
			s.mu.Unlock()

			if intermediate != nil {
				s.broadcast(intermediate)
			}
		}()
	}
	wg.Wait()

	if resA == nil && resB == nil {
		http.Error(w, fmt.Sprintf("triage failed: engine A error: %v, engine B error: %v", errA, errB), http.StatusInternalServerError)
		return
	}

	s.mu.RLock()
	var finalUpdated *ticket.TriagedTicket
	for _, curr := range s.tickets {
		if curr.ID == targetID {
			finalUpdated = curr
			break
		}
	}
	s.mu.RUnlock()

	if finalUpdated == nil {
		finalUpdated = &ticket.TriagedTicket{
			Ticket:  target.Ticket,
			TriageA: resA,
			TriageB: resB,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(finalUpdated)
}

func (s *Server) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan *ticket.TriagedTicket, 100)
	s.mu.Lock()
	s.listeners[ch] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.listeners, ch)
		s.mu.Unlock()
	}()

	// Flush initial connection comment immediately so client confirms open state
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
		case item := <-ch:
			data, err := json.Marshal(item)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}
