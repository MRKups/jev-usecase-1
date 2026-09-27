// Package jev defines the interface and adapters for communicating with the Jev ML model.
package jev

import (
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

// Client executes classification and labeling against the Jev model.
type Client interface {
	triage.Engine
}
