// Package provider specifies the LLM provider interface and adapters for ticket generation.
package provider

import (
	"context"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

// LLMProvider generates structured ITSM tickets from prompt instructions.
type LLMProvider interface {
	// GenerateTicket invokes the model to author a single ITSM ticket.
	GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error)
	// Ping verifies connectivity and credentials with the provider by listing models.
	Ping(ctx context.Context) error
}
