package generator

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

type testLLMProvider struct {
	counter int64
}

func (p *testLLMProvider) GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error) {
	n := atomic.AddInt64(&p.counter, 1)
	return &ticket.Ticket{
		ID:              fmt.Sprintf("INC-TEST-%03d", n),
		CreatedAt:       time.Now().UTC(),
		ReporterName:    "Test Reporter",
		ReporterEmail:   "reporter@example.com",
		Department:      "Engineering",
		Category:        "Hardware",
		ReportedUrgency: "High",
		Summary:         fmt.Sprintf("Hardware test issue %d", n),
		Description:     "Detailed issue description for testing",
		AffectedSystem:  "Laptop",
		RawPrompt:       prompt,
	}, nil
}

func (p *testLLMProvider) Ping(ctx context.Context) error {
	return nil
}

func TestProceduralPromptGeneration(t *testing.T) {
	matrix := DefaultMatrix()
	prompt := matrix.BuildPrompt()

	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}

	if !strings.Contains(prompt, "Communication Archetype:") {
		t.Errorf("prompt missing Communication Archetype parameter")
	}

	if !strings.Contains(prompt, "Fictitious Reporter Identity:") {
		t.Errorf("prompt missing Fictitious Reporter Identity instruction")
	}
}

func TestMatrixDimensions(t *testing.T) {
	m := DefaultMatrix()

	if len(m.Departments) < 10 {
		t.Errorf("expected at least 10 departments, got %d", len(m.Departments))
	}

	for _, dept := range m.Departments {
		roles := m.DepartmentRoles[dept]
		if len(roles) < 5 {
			t.Errorf("department %s has fewer than 5 job titles: %d", dept, len(roles))
		}
	}

	if len(m.Systems) < 20 {
		t.Errorf("expected at least 20 generic systems, got %d", len(m.Systems))
	}

	// Verify no brand or vendor product names in systems
	brandKeywords := []string{"MacBook", "Dell", "Cisco", "Okta", "AWS", "Zendesk", "Slack", "Google", "GitHub", "CrowdStrike", "Microsoft", "Apple"}
	for _, sys := range m.Systems {
		for _, brand := range brandKeywords {
			if strings.Contains(strings.ToLower(sys), strings.ToLower(brand)) {
				t.Errorf("system %q contains forbidden brand keyword %q", sys, brand)
			}
		}
	}

	if len(m.ProblemClasses) < 20 {
		t.Errorf("expected at least 20 problem classes, got %d", len(m.ProblemClasses))
	}

	if len(m.CommunicationArchetypes) < 20 {
		t.Errorf("expected at least 20 communication archetypes, got %d", len(m.CommunicationArchetypes))
	}
}

func TestGeneratorBatch(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := NewGenerator(testProv, DefaultMatrix())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	tickets, err := gen.GenerateBatch(ctx, 5)
	if err != nil {
		t.Fatalf("GenerateBatch failed: %v", err)
	}

	if len(tickets) != 5 {
		t.Fatalf("expected 5 tickets, got %d", len(tickets))
	}
	for i, tkt := range tickets {
		if tkt.ID == "" {
			t.Errorf("ticket %d missing ID", i)
		}
		if tkt.Summary == "" {
			t.Errorf("ticket %d missing Summary", i)
		}
	}
}

func TestGeneratorStream(t *testing.T) {
	testProv := &testLLMProvider{}
	gen := NewGenerator(testProv, DefaultMatrix())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch := gen.Stream(ctx, 10*time.Millisecond)
	count := 0
	for range ch {
		count++
		if count >= 3 {
			break
		}
	}

	if count < 1 {
		t.Errorf("expected at least 1 ticket from stream, got %d", count)
	}
}
