package provider

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

// BuildTriagePrompt formats a ticket and standard triage questions into an evaluation prompt.
func BuildTriagePrompt(t *ticket.Ticket, questions []triage.Question) string {
	var sb strings.Builder
	sb.WriteString("Evaluate the following IT service management support ticket and answer each operational triage question.\n\n")
	sb.WriteString("Ticket Information:\n")
	fmt.Fprintf(&sb, "- ID: %s\n", t.ID)
	fmt.Fprintf(&sb, "- Summary: %s\n", t.Summary)
	fmt.Fprintf(&sb, "- Description: %s\n", t.Description)
	fmt.Fprintf(&sb, "- Department: %s\n", t.Department)
	fmt.Fprintf(&sb, "- Category: %s\n", t.Category)
	fmt.Fprintf(&sb, "- Affected System: %s\n", t.AffectedSystem)
	fmt.Fprintf(&sb, "- Reported Urgency: %s\n\n", t.ReportedUrgency)

	sb.WriteString("Operational Questions to evaluate:\n")
	for i, q := range questions {
		fmt.Fprintf(&sb, "%d. %s (id: %s)\n", i+1, q.Text, q.ID)
		if len(q.Options) > 0 {
			fmt.Fprintf(&sb, "   Allowed options: %s\n", strings.Join(q.Options, ", "))
		}
		if len(q.Criteria) > 0 {
			posOpt, negOpt, isBin := findBinaryOptionTargets(q.Options)
			if !isBin && (q.Type == "noul" || q.Type == "bool" || q.Type == "boolean") {
				posOpt, negOpt, isBin = "Yes", "No", true
			}
			for opt, desc := range q.Criteria {
				displayOpt := opt
				if isBin {
					isPos, isNeg := parseBinaryPolarity(opt)
					if isPos && posOpt != "" {
						displayOpt = posOpt
					} else if isNeg && negOpt != "" {
						displayOpt = negOpt
					}
				}
				fmt.Fprintf(&sb, "   * %s: %s\n", displayOpt, desc)
			}
		}
	}

	sb.WriteString("\nOutput valid JSON adhering strictly to this schema:\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- For each question, select exactly one option from the \"Allowed options\" list verbatim.\n")
	sb.WriteString("- For yes/no questions, use \"Yes\" or \"No\" (never true, false, 1, or 0).\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"answers\": {\n")
	for i, q := range questions {
		comma := ","
		if i == len(questions)-1 {
			comma = ""
		}
		fmt.Fprintf(&sb, "    %q: \"<selected allowed option>\"%s\n", q.ID, comma)
	}
	sb.WriteString("  },\n")
	sb.WriteString("  \"confidence\": 0.95,\n")
	sb.WriteString("  \"recommended_action\": \"<concise one-sentence routing decision>\"\n")
	sb.WriteString("}\n")

	return sb.String()
}

type llmTriageResponse struct {
	Answers           map[string]any `json:"answers"`
	Confidence        float64        `json:"confidence"`
	RecommendedAction string         `json:"recommended_action"`
}

func cleanJSON(raw string) string {
	return ExtractJSONObject(raw)
}

func parseBinaryPolarity(val string) (isPositive bool, isNegative bool) {
	cleaned := strings.ToLower(strings.Trim(strings.TrimSpace(val), " \t\r\n`'\"*._,:;()[]{}"))
	cleaned = strings.TrimSuffix(cleaned, ".0")

	switch cleaned {
	case "yes", "true", "1", "y", "t", "positive", "confirmed", "pass", "on", "enabled":
		return true, false
	case "no", "false", "0", "n", "f", "negative", "unconfirmed", "fail", "off", "disabled", "none":
		return false, true
	}

	fields := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == ' ' || r == '-' || r == ',' || r == ':' || r == ';'
	})
	if len(fields) > 0 {
		switch fields[0] {
		case "yes", "true", "y", "t", "positive", "confirmed":
			return true, false
		case "no", "false", "n", "f", "negative", "none":
			return false, true
		}
	}

	return false, false
}

func findBinaryOptionTargets(options []string) (posOpt string, negOpt string, isBinary bool) {
	if len(options) != 2 {
		return "", "", false
	}
	isPos0, isNeg0 := parseBinaryPolarity(options[0])
	isPos1, isNeg1 := parseBinaryPolarity(options[1])

	if isPos0 && isNeg1 {
		return options[0], options[1], true
	}
	if isNeg0 && isPos1 {
		return options[1], options[0], true
	}
	return "", "", false
}

// ParseTriageJSON parses the LLM output into validated triage answers, confidence, and recommended action.
func ParseTriageJSON(raw string, questions []triage.Question) (map[string]string, float64, string, error) {
	cleaned := cleanJSON(raw)

	var parsed llmTriageResponse
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, 0, "", fmt.Errorf("unmarshal triage json: %w (content: %s)", err, raw)
	}

	answers := make(map[string]string, len(questions))
	for _, q := range questions {
		valAny, ok := parsed.Answers[q.ID]
		if !ok {
			continue
		}
		rawAns := strings.TrimSpace(fmt.Sprintf("%v", valAny))

		matched := false
		for _, opt := range q.Options {
			if strings.EqualFold(rawAns, opt) ||
				strings.EqualFold(strings.ReplaceAll(rawAns, "_", " "), strings.ReplaceAll(opt, "_", " ")) {
				answers[q.ID] = opt
				matched = true
				break
			}
		}

		if !matched {
			posOpt, negOpt, isBinary := findBinaryOptionTargets(q.Options)
			if !isBinary && (q.Type == "noul" || q.Type == "bool" || q.Type == "boolean") {
				posOpt, negOpt, isBinary = "Yes", "No", true
			}
			if isBinary {
				isPos, isNeg := parseBinaryPolarity(rawAns)
				if isPos {
					answers[q.ID] = posOpt
					matched = true
				} else if isNeg {
					answers[q.ID] = negOpt
					matched = true
				}
			}
		}

		if !matched {
			answers[q.ID] = rawAns
		}
	}

	conf := parsed.Confidence
	if conf <= 0 {
		conf = 0.90
	}
	if conf > 1.0 {
		conf = 1.0
	}

	action := strings.TrimSpace(parsed.RecommendedAction)
	if action == "" {
		action = fallbackAction(answers)
	}

	return answers, conf, action, nil
}

func fallbackAction(answers map[string]string) string {
	domain := answers["technical_domain"]
	isSecurity := strings.EqualFold(answers["security_incident"], "yes") || strings.EqualFold(answers["security_incident"], "true")
	targetGroup := answers["target_resolution_group"]
	if isSecurity {
		return "Escalate immediately to SecOps for security investigation."
	}
	if targetGroup != "" && domain != "" {
		return fmt.Sprintf("Route to %s for %s resolution.", targetGroup, domain)
	}
	if targetGroup != "" {
		return fmt.Sprintf("Route ticket to %s for standard triage.", targetGroup)
	}
	return "Assign to Service Desk for initial assessment."
}
