package triage_test

import (
	"testing"

	"github.com/MRKups/jev-usecase-1/internal/triage"
)

func TestStandardQuestions(t *testing.T) {
	questions := triage.StandardQuestions()
	if len(questions) != 6 {
		t.Fatalf("expected 6 standard questions, got %d", len(questions))
	}

	expectedIDs := []string{
		"ticket_type",
		"technical_domain",
		"operational_urgency",
		"security_incident",
		"target_resolution_group",
		"blast_radius",
	}

	for i, expectedID := range expectedIDs {
		q := questions[i]
		if q.ID != expectedID {
			t.Errorf("question %d: expected ID %q, got %q", i, expectedID, q.ID)
		}
		if len(q.Text) == 0 {
			t.Errorf("question %q: missing text", q.ID)
		}
		if len(q.Options) == 0 {
			t.Errorf("question %q: missing options", q.ID)
		}
	}
}
