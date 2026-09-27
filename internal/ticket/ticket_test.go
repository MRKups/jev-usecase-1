package ticket

import (
	"bytes"
	"testing"
	"time"
)

func sampleTicket() *Ticket {
	return &Ticket{
		ID:              "INC-001",
		CreatedAt:       time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC),
		ReporterName:    "Alice Smith",
		ReporterEmail:   "alice@example.com",
		Department:      "Engineering",
		Category:        "Hardware",
		ReportedUrgency: "High",
		Summary:         "Laptop battery swelling",
		Description:     "Trackpad is lifting due to swollen battery.",
		AffectedSystem:  "Laptop Workstation",
	}
}

func TestJSONEncoding(t *testing.T) {
	tickets := []*Ticket{sampleTicket()}
	var buf bytes.Buffer

	if err := EncodeJSON(&buf, tickets); err != nil {
		t.Fatalf("EncodeJSON failed: %v", err)
	}

	decoded, err := DecodeJSON(&buf)
	if err != nil {
		t.Fatalf("DecodeJSON failed: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(decoded))
	}
	if decoded[0].ID != "INC-001" {
		t.Errorf("expected ID INC-001, got %s", decoded[0].ID)
	}
	if decoded[0].ReportedUrgency != "High" {
		t.Errorf("expected ReportedUrgency High, got %s", decoded[0].ReportedUrgency)
	}
	if decoded[0].Summary != "Laptop battery swelling" {
		t.Errorf("expected Summary match, got %s", decoded[0].Summary)
	}
}

func TestLegacyUrgencyJSONCompatibility(t *testing.T) {
	legacyJSON := `[{"id":"INC-LEGACY","summary":"Old ticket","urgency":"Critical"}]`
	decoded, err := DecodeJSON(bytes.NewBufferString(legacyJSON))
	if err != nil {
		t.Fatalf("failed to decode legacy JSON: %v", err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(decoded))
	}
	if decoded[0].ReportedUrgency != "Critical" {
		t.Errorf("expected ReportedUrgency 'Critical' from legacy 'urgency', got %q", decoded[0].ReportedUrgency)
	}
}

func TestJSONEncodingNilOrEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := EncodeJSON(&buf, nil); err != nil {
		t.Fatalf("EncodeJSON(nil) failed: %v", err)
	}
	if got := buf.String(); got != "[]\n" {
		t.Errorf("expected '[]\\n', got %q", got)
	}
}

func TestJSONDecodeNullOrEmpty(t *testing.T) {
	cases := []string{"", "   ", "null", "[]"}
	for _, tc := range cases {
		decoded, err := DecodeJSON(bytes.NewBufferString(tc))
		if err != nil {
			t.Fatalf("DecodeJSON(%q) failed: %v", tc, err)
		}
		if decoded == nil || len(decoded) != 0 {
			t.Errorf("expected empty non-nil slice for %q, got %#v", tc, decoded)
		}
	}
}

func TestJSONDecodeWrappedAndNested(t *testing.T) {
	wrapped := `{"tickets":[{"id":"INC-WRAP","summary":"Wrapped ticket"}]}`
	decoded, err := DecodeJSON(bytes.NewBufferString(wrapped))
	if err != nil {
		t.Fatalf("DecodeJSON(wrapped) failed: %v", err)
	}
	if len(decoded) != 1 || decoded[0].ID != "INC-WRAP" {
		t.Errorf("unexpected decoded from wrapped: %#v", decoded)
	}

	nested := `[{"ticket":{"id":"INC-NEST","summary":"Nested ticket"}}]`
	decodedNested, err := DecodeJSON(bytes.NewBufferString(nested))
	if err != nil {
		t.Fatalf("DecodeJSON(nested) failed: %v", err)
	}
	if len(decodedNested) != 1 || decodedNested[0].ID != "INC-NEST" {
		t.Errorf("unexpected decoded from nested: %#v", decodedNested)
	}
}

func TestCSVEncoding(t *testing.T) {
	tickets := []*Ticket{sampleTicket()}
	var buf bytes.Buffer

	if err := EncodeCSV(&buf, tickets); err != nil {
		t.Fatalf("EncodeCSV failed: %v", err)
	}

	decoded, err := DecodeCSV(&buf)
	if err != nil {
		t.Fatalf("DecodeCSV failed: %v", err)
	}

	if len(decoded) != 1 {
		t.Fatalf("expected 1 ticket, got %d", len(decoded))
	}
	if decoded[0].ID != "INC-001" {
		t.Errorf("expected ID INC-001, got %s", decoded[0].ID)
	}
	if decoded[0].Department != "Engineering" {
		t.Errorf("expected Department Engineering, got %s", decoded[0].Department)
	}
}

func BenchmarkJSONEncoding(b *testing.B) {
	tickets := make([]*Ticket, 100)
	for i := range tickets {
		tickets[i] = sampleTicket()
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = EncodeJSON(&buf, tickets)
	}
}

func BenchmarkCSVEncoding(b *testing.B) {
	tickets := make([]*Ticket, 100)
	for i := range tickets {
		tickets[i] = sampleTicket()
	}
	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = EncodeCSV(&buf, tickets)
	}
}
