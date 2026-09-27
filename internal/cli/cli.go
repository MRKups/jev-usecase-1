// Package cli handles command-line parsing, flag dispatch, and output formatting.
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/config"
	"github.com/MRKups/jev-usecase-1/internal/generator"
	"github.com/MRKups/jev-usecase-1/internal/jev"
	"github.com/MRKups/jev-usecase-1/internal/provider"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
	"github.com/MRKups/jev-usecase-1/internal/webapp"
)

const version = "0.2.0"

const usage = `Usage:
  ticket-eval [--help]
  ticket-eval serve [--addr=<address>]
  ticket-eval generate [--count=<n>] [--format=json|csv] [--output=<path>]
  ticket-eval triage [--input=<path>]
  ticket-eval version

Subcommands:
  serve       Start the interactive decision engine web server
  generate    Procedurally generate synthetic tickets via configured LLM
  triage      Evaluate tickets against standard questions using Jev
  version     Print version and build details
`

// RunWithInput parses arguments and executes the requested subcommand.
func RunWithInput(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, usage)
		return nil
	}

	cmd := args[0]
	subArgs := args[1:]

	switch cmd {
	case "serve":
		return runServe(ctx, subArgs, stdout)
	case "generate":
		return runGenerate(ctx, subArgs, stdout)
	case "triage":
		return runTriage(ctx, subArgs, stdout)
	case "version", "--version", "-v":
		fmt.Fprintf(stdout, "ticket-eval version %s\n", version)
		return nil
	case "help", "--help", "-h":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		return fmt.Errorf("unknown subcommand %q. Run 'ticket-eval --help' for usage", cmd)
	}
}

func setupLLMProvider(ctx context.Context, cfg config.Config) (provider.LLMProvider, error) {
	return provider.Factory(ctx, cfg)
}

func runServe(ctx context.Context, args []string, stdout io.Writer) error {
	cfg := config.LoadDefault()

	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", cfg.Addr, "HTTP listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg.Addr = *addr

	p, err := setupLLMProvider(ctx, cfg)
	if err != nil {
		fmt.Fprintf(stdout, "Notice: generator backend not configured (%v); configure providers in Web UI.\n", err)
	}
	gen := generator.NewGenerator(p, generator.DefaultMatrix())
	engineA, err := webapp.CreateEngine(cfg.EngineA, cfg)
	if err != nil {
		fmt.Fprintf(stdout, "Notice: decision engine A initialization notice (%v); falling back to default Jev.\n", err)
		engineA = jev.Factory(cfg.Jev)
	}
	engineB, _ := webapp.CreateEngine(cfg.EngineB, cfg)

	initial, source := loadInitialTickets(cfg.DatasetPath)
	if source != "" {
		fmt.Fprintf(stdout, "Loaded %d tickets from %s\n", len(initial), source)
	}

	initialQuestions, qSource := loadInitialQuestions("")
	if qSource != "" {
		fmt.Fprintf(stdout, "Loaded %d operational questions from %s\n", len(initialQuestions), qSource)
	}

	srv := webapp.NewServer(cfg, gen, engineA, engineB, initial, initialQuestions)
	httpServer := &http.Server{
		Addr:    *addr,
		Handler: srv.Handler(),
	}

	serverErr := make(chan error, 1)
	go func() {
		fmt.Fprintf(stdout, "Jev Usecase web server listening on http://%s\n", *addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-serverErr:
		return err
	}
}

// loadInitialTickets checks the configured dataset path and execution root for an existing
// data file. If found and non-empty, it loads it; otherwise it starts with an empty ticket dataset.
func loadInitialTickets(configuredPath string) ([]*ticket.Ticket, string) {
	var candidates []string
	if configuredPath != "" {
		candidates = append(candidates, configuredPath)
	}
	candidates = append(candidates, "dataset.json", "data.json", "tickets.json")

	seen := make(map[string]struct{})
	for _, p := range candidates {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		if data, err := os.ReadFile(p); err == nil {
			if loaded, err := ticket.DecodeJSON(strings.NewReader(string(data))); err == nil && len(loaded) > 0 {
				return loaded, p
			}
		}
	}

	// Fallback: check for any dataset*.json in the current working directory
	if matches, err := filepath.Glob("dataset*.json"); err == nil {
		for _, m := range matches {
			if _, ok := seen[m]; ok {
				continue
			}
			seen[m] = struct{}{}
			if data, err := os.ReadFile(m); err == nil {
				if loaded, err := ticket.DecodeJSON(strings.NewReader(string(data))); err == nil && len(loaded) > 0 {
					return loaded, m
				}
			}
		}
	}

	return nil, ""
}

// loadInitialQuestions checks the execution root for an existing questions file.
func loadInitialQuestions(customPath string) ([]triage.Question, string) {
	candidates := []string{
		"questions.json",
		"criteria.json",
	}
	if customPath != "" {
		candidates = append([]string{customPath}, candidates...)
	}
	for _, p := range candidates {
		if data, err := os.ReadFile(p); err == nil {
			var qs []triage.Question
			if err := json.Unmarshal(data, &qs); err == nil && len(qs) > 0 {
				return qs, p
			}
		}
	}
	return nil, ""
}

func runGenerate(ctx context.Context, args []string, stdout io.Writer) error {
	cfg := config.LoadDefault()

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	count := fs.Int("count", 10, "number of tickets to generate")
	format := fs.String("format", "json", "output format: json or csv")
	outputPath := fs.String("output", "", "destination file path, or stdout if empty")
	if err := fs.Parse(args); err != nil {
		return err
	}

	p, err := setupLLMProvider(ctx, cfg)
	if err != nil {
		return fmt.Errorf("generator not configured (%w): set active provider and API key in config.json or environment", err)
	}
	gen := generator.NewGenerator(p, generator.DefaultMatrix())

	tickets, err := gen.GenerateBatch(ctx, *count)
	if err != nil {
		return fmt.Errorf("generation failed: %w", err)
	}

	var out io.Writer = stdout
	if *outputPath != "" && *outputPath != "-" {
		if err := os.MkdirAll(filepath.Dir(*outputPath), 0755); err != nil {
			return fmt.Errorf("create output directory: %w", err)
		}
		f, err := os.Create(*outputPath)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer f.Close()
		out = f
	}

	switch strings.ToLower(*format) {
	case "csv":
		if err := ticket.EncodeCSV(out, tickets); err != nil {
			return fmt.Errorf("encode csv: %w", err)
		}
	case "json":
		if err := ticket.EncodeJSON(out, tickets); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
	default:
		return fmt.Errorf("unsupported format %q, use json or csv", *format)
	}

	if *outputPath != "" && *outputPath != "-" {
		fmt.Fprintf(stdout, "Generated %d tickets into %s\n", len(tickets), *outputPath)
	}

	return nil
}

func runTriage(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("triage", flag.ContinueOnError)
	inputPath := fs.String("input", "", "path to tickets file (JSON or CSV)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *inputPath == "" {
		return fmt.Errorf("missing required --input flag")
	}

	file, err := os.Open(*inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer file.Close()

	var tickets []*ticket.Ticket
	if strings.HasSuffix(strings.ToLower(*inputPath), ".csv") {
		tickets, err = ticket.DecodeCSV(file)
	} else {
		tickets, err = ticket.DecodeJSON(file)
	}
	if err != nil {
		return fmt.Errorf("decode tickets: %w", err)
	}

	cfg := config.LoadDefault()
	if cfg.Jev.Endpoint == "" {
		return fmt.Errorf("jev endpoint is not configured; set endpoint in config.json or JEV_ENDPOINT environment variable")
	}
	jevClient := jev.Factory(cfg.Jev)
	questions := triage.StandardQuestions()

	fmt.Fprintf(stdout, "Evaluating %d tickets with Jev model (%s)...\n\n", len(tickets), cfg.Jev.Endpoint)

	var totalLatency int64
	for i, t := range tickets {
		res, err := jevClient.Classify(ctx, t, questions)
		if err != nil {
			return fmt.Errorf("triage failed on %s: %w", t.ID, err)
		}
		totalLatency += res.LatencyMs

		fmt.Fprintf(stdout, "[%d/%d] %s: %s\n", i+1, len(tickets), t.ID, t.Summary)
		fmt.Fprintf(stdout, "  Latency: %d ms | Confidence: %.0f%%\n", res.LatencyMs, res.Confidence*100)
		for _, q := range questions {
			fmt.Fprintf(stdout, "  - %s: %s\n", q.Text, res.Answers[q.ID])
		}
		fmt.Fprintf(stdout, "  Action: %s\n\n", res.RecommendedAction)
	}

	if len(tickets) > 0 {
		avg := totalLatency / int64(len(tickets))
		fmt.Fprintf(stdout, "Summary: Evaluated %d tickets. Average latency: %d ms\n", len(tickets), avg)
	}

	return nil
}
