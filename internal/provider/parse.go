package provider

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
)

var reasoningTagRegex = regexp.MustCompile(`(?is)<think>.*?</think>|<thought>.*?</thought>|<reasoning>.*?</reasoning>|<reflection>.*?</reflection>`)

// ExtractJSONObject scans raw LLM output, strips reasoning blocks, and extracts the outermost JSON object.
func ExtractJSONObject(raw string) string {
	cleaned := reasoningTagRegex.ReplaceAllString(raw, "")
	cleaned = strings.TrimSpace(cleaned)

	startIdx := strings.IndexByte(cleaned, '{')
	if startIdx == -1 {
		rawStart := strings.IndexByte(raw, '{')
		if rawStart != -1 {
			cleaned = raw
			startIdx = rawStart
		} else {
			return stripMarkdownFence(cleaned)
		}
	}

	depth := 0
	inString := false
	escaped := false
	endIdx := -1

	for i := startIdx; i < len(cleaned); i++ {
		c := cleaned[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if !inString {
			if c == '{' {
				depth++
			} else if c == '}' {
				depth--
				if depth == 0 {
					endIdx = i
					break
				}
			}
		}
	}

	if endIdx != -1 {
		return strings.TrimSpace(cleaned[startIdx : endIdx+1])
	}

	lastClose := strings.LastIndexByte(cleaned, '}')
	if lastClose > startIdx {
		return strings.TrimSpace(cleaned[startIdx : lastClose+1])
	}

	return stripMarkdownFence(cleaned)
}

func stripMarkdownFence(text string) string {
	clean := strings.TrimSpace(text)
	if strings.HasPrefix(clean, "```json") {
		clean = strings.TrimPrefix(clean, "```json")
		clean = strings.TrimSuffix(clean, "```")
	} else if strings.HasPrefix(clean, "```") {
		clean = strings.TrimPrefix(clean, "```")
		clean = strings.TrimSuffix(clean, "```")
	}
	return strings.TrimSpace(clean)
}

type rawTicketJSON struct {
	ReporterName    string `json:"reporter_name"`
	ReporterEmail   string `json:"reporter_email"`
	Department      string `json:"department"`
	Category        string `json:"category"`
	ReportedUrgency string `json:"reported_urgency"`
	Urgency         string `json:"urgency"`
	Summary         string `json:"summary"`
	Description     string `json:"description"`
	AffectedSystem  string `json:"affected_system"`
}

// parseTicketJSON extracts a structured Ticket from LLM JSON output.
func parseTicketJSON(rawText string, prompt string) (*ticket.Ticket, error) {
	cleanText := ExtractJSONObject(rawText)

	var parsed rawTicketJSON
	if err := json.Unmarshal([]byte(cleanText), &parsed); err != nil {
		return nil, fmt.Errorf("parse ticket json (%w): raw text: %s", err, cleanText)
	}

	b := make([]byte, 2)
	_, _ = rand.Read(b)
	ticketID := fmt.Sprintf("#%s", hex.EncodeToString(b))

	reportedUrgency := parsed.ReportedUrgency
	if reportedUrgency == "" {
		reportedUrgency = parsed.Urgency
	}
	if reportedUrgency == "" {
		reportedUrgency = "Medium"
	}

	category := parsed.Category
	if category == "" {
		category = "Software"
	}

	reporterName := strings.TrimSpace(parsed.ReporterName)
	rawEmail := strings.TrimSpace(parsed.ReporterEmail)

	var reporterEmail string
	if rawEmail != "" {
		if idx := strings.Index(rawEmail, "@"); idx != -1 {
			rawEmail = rawEmail[:idx]
		}
		rawEmail = strings.Trim(rawEmail, " ._")
		if rawEmail != "" {
			reporterEmail = strings.ToLower(rawEmail) + "@example.com"
		}
	}

	if reporterEmail == "" && reporterName != "" {
		parts := strings.Fields(strings.ToLower(reporterName))
		if len(parts) >= 2 {
			reporterEmail = fmt.Sprintf("%s.%s@example.com", parts[0], parts[len(parts)-1])
		} else if len(parts) == 1 {
			reporterEmail = fmt.Sprintf("%s@example.com", parts[0])
		}
	}
	if reporterEmail == "" {
		reporterEmail = "user@example.com"
	}

	// Infer reporter name from dot format username when not explicitly given
	if reporterName == "" {
		reporterName = inferReporterName(rawEmail)
	}

	return &ticket.Ticket{
		ID:              ticketID,
		CreatedAt:       time.Now().UTC(),
		ReporterName:    reporterName,
		ReporterEmail:   reporterEmail,
		Department:      parsed.Department,
		Category:        category,
		ReportedUrgency: reportedUrgency,
		Summary:         parsed.Summary,
		Description:     parsed.Description,
		AffectedSystem:  parsed.AffectedSystem,
		RawPrompt:       prompt,
	}, nil
}

// inferReporterName extracts and capitalizes a plaintext full name from a dot format string (e.g. "marcus.valerius" -> "Marcus Valerius").
func inferReporterName(raw string) string {
	raw = strings.TrimSpace(raw)
	if idx := strings.Index(raw, "@"); idx != -1 {
		raw = raw[:idx]
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '.' || r == '_' || r == '-' || r == ' '
	})
	if len(parts) == 0 {
		return "Anonymous User"
	}
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
		}
	}
	return strings.Join(parts, " ")
}
