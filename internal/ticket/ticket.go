// Package ticket defines the domain model and serialization for ITSM tickets.
package ticket

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Ticket represents a synthetic IT service management support ticket.
type Ticket struct {
	ID              string    `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	ReporterName    string    `json:"reporter_name"`
	ReporterEmail   string    `json:"reporter_email"`
	Department      string    `json:"department"`
	Category        string    `json:"category"`
	ReportedUrgency string    `json:"reported_urgency"`
	Summary         string    `json:"summary"`
	Description     string    `json:"description"`
	AffectedSystem  string    `json:"affected_system"`
	RawPrompt       string    `json:"raw_prompt,omitempty"`
}

// TriageResult records the evaluation performed by a decision engine.
type TriageResult struct {
	TicketID          string            `json:"ticket_id"`
	EngineName        string            `json:"engine_name,omitempty"`
	Model             string            `json:"model,omitempty"`
	EvaluatedAt       time.Time         `json:"evaluated_at"`
	LatencyMs         int64             `json:"latency_ms"`
	Confidence        float64           `json:"confidence"`
	Answers           map[string]string `json:"answers"`
	RecommendedAction string            `json:"recommended_action"`
}

// TriagedTicket pairs a ticket with its evaluation results from Decision Engine A and Decision Engine B.
type TriagedTicket struct {
	Ticket
	TriageA *TriageResult `json:"triage_a,omitempty"`
	TriageB *TriageResult `json:"triage_b,omitempty"`
	Status  string        `json:"status,omitempty"`
}

// UnmarshalJSON deserializes a TriagedTicket, supporting legacy "triage" field.
func (t *TriagedTicket) UnmarshalJSON(data []byte) error {
	type alias TriagedTicket
	var aux struct {
		alias
		LegacyTriage *TriageResult `json:"triage"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*t = TriagedTicket(aux.alias)
	if t.TriageA == nil && aux.LegacyTriage != nil {
		t.TriageA = aux.LegacyTriage
	}
	return nil
}

// EncodeJSON writes tickets as formatted JSON to the writer.
func EncodeJSON(w io.Writer, tickets []*Ticket) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if tickets == nil {
		tickets = []*Ticket{}
	}
	return enc.Encode(tickets)
}

type rawJSONTicket struct {
	ID              string         `json:"id"`
	CreatedAt       time.Time      `json:"created_at"`
	ReporterName    string         `json:"reporter_name"`
	ReporterEmail   string         `json:"reporter_email"`
	Department      string         `json:"department"`
	Category        string         `json:"category"`
	ReportedUrgency string         `json:"reported_urgency"`
	UrgencyFallback string         `json:"urgency"`
	Summary         string         `json:"summary"`
	Description     string         `json:"description"`
	AffectedSystem  string         `json:"affected_system"`
	RawPrompt       string         `json:"raw_prompt,omitempty"`
	NestedTicket    *rawJSONTicket `json:"ticket,omitempty"`
}

func parseRawJSONTickets(raws []rawJSONTicket) []*Ticket {
	tickets := make([]*Ticket, 0, len(raws))
	for _, raw := range raws {
		r := raw
		if r.ID == "" && r.NestedTicket != nil {
			r = *r.NestedTicket
		}
		urgency := r.ReportedUrgency
		if urgency == "" {
			urgency = r.UrgencyFallback
		}
		tickets = append(tickets, &Ticket{
			ID:              r.ID,
			CreatedAt:       r.CreatedAt,
			ReporterName:    r.ReporterName,
			ReporterEmail:   r.ReporterEmail,
			Department:      r.Department,
			Category:        r.Category,
			ReportedUrgency: urgency,
			Summary:         r.Summary,
			Description:     r.Description,
			AffectedSystem:  r.AffectedSystem,
			RawPrompt:       r.RawPrompt,
		})
	}
	return tickets
}

// DecodeJSON reads tickets from a JSON reader.
func DecodeJSON(r io.Reader) ([]*Ticket, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("read tickets JSON: %w", err)
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return []*Ticket{}, nil
	}

	// Try unmarshaling as a flat ticket array first
	var raws []rawJSONTicket
	if err := json.Unmarshal(trimmed, &raws); err == nil {
		return parseRawJSONTickets(raws), nil
	}

	// Try unmarshaling as an object with tickets or dataset container
	var container struct {
		Tickets []rawJSONTicket `json:"tickets"`
		Dataset []rawJSONTicket `json:"dataset"`
	}
	if err := json.Unmarshal(trimmed, &container); err == nil {
		if len(container.Tickets) > 0 {
			return parseRawJSONTickets(container.Tickets), nil
		}
		if len(container.Dataset) > 0 {
			return parseRawJSONTickets(container.Dataset), nil
		}
		return []*Ticket{}, nil
	}

	return nil, fmt.Errorf("decode tickets JSON: unrecognized format")
}

// CSVHeaders defines standard columns for CSV export.
var CSVHeaders = []string{
	"ID",
	"CreatedAt",
	"ReporterName",
	"ReporterEmail",
	"Department",
	"Category",
	"ReportedUrgency",
	"Summary",
	"Description",
	"AffectedSystem",
}

// EncodeCSV exports tickets in CSV format.
func EncodeCSV(w io.Writer, tickets []*Ticket) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(CSVHeaders); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, t := range tickets {
		row := []string{
			t.ID,
			t.CreatedAt.Format(time.RFC3339),
			t.ReporterName,
			t.ReporterEmail,
			t.Department,
			t.Category,
			t.ReportedUrgency,
			t.Summary,
			strings.ReplaceAll(t.Description, "\n", " "),
			t.AffectedSystem,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	return writer.Error()
}

// DecodeCSV imports tickets from CSV data.
func DecodeCSV(r io.Reader) ([]*Ticket, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}

	if len(records) < 2 {
		return nil, nil
	}

	var tickets []*Ticket
	// Skip header row
	for i, row := range records[1:] {
		if len(row) < len(CSVHeaders) {
			return nil, fmt.Errorf("row %d has insufficient columns (%d expected %d)", i+2, len(row), len(CSVHeaders))
		}
		createdAt, err := time.Parse(time.RFC3339, row[1])
		if err != nil {
			createdAt = time.Now()
		}
		tickets = append(tickets, &Ticket{
			ID:              row[0],
			CreatedAt:       createdAt,
			ReporterName:    row[2],
			ReporterEmail:   row[3],
			Department:      row[4],
			Category:        row[5],
			ReportedUrgency: row[6],
			Summary:         row[7],
			Description:     row[8],
			AffectedSystem:  row[9],
		})
	}

	return tickets, nil
}
