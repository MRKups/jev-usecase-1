package triage

import (
	"context"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

// Engine evaluates tickets against operational triage questions.
type Engine interface {
	// Classify submits a ticket and triage questions to the engine.
	Classify(ctx context.Context, t *ticket.Ticket, questions []Question) (*ticket.TriageResult, error)
	// Ping verifies connectivity and credentials with the engine backend.
	Ping(ctx context.Context) error
	// Name returns the descriptive display name of the engine and model.
	Name() string
}
