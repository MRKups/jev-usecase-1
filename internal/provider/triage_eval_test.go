package provider_test

import (
	"strings"
	"testing"

	"github.com/MRKups/jev-usecase-1/internal/provider"
	"github.com/MRKups/jev-usecase-1/internal/ticket"
	"github.com/MRKups/jev-usecase-1/internal/triage"
)

func TestBuildTriagePrompt(t *testing.T) {
	sampleTicket := &ticket.Ticket{
		ID:              "TCK-TEST-1",
		Summary:         "VPN connection dropping",
		Description:     "Cannot access internal network services.",
		Department:      "Sales",
		Category:        "Network",
		AffectedSystem:  "GlobalProtect",
		ReportedUrgency: "High",
	}

	questions := triage.StandardQuestions()
	prompt := provider.BuildTriagePrompt(sampleTicket, questions)

	if !strings.Contains(prompt, "TCK-TEST-1") {
		t.Errorf("prompt missing ticket ID")
	}
	if !strings.Contains(prompt, "VPN connection dropping") {
		t.Errorf("prompt missing ticket summary")
	}
	if !strings.Contains(prompt, "technical_domain") {
		t.Errorf("prompt missing question ID 'technical_domain'")
	}
}

func TestParseTriageJSON(t *testing.T) {
	questions := triage.StandardQuestions()

	raw := "```json\n" + `{
		"answers": {
			"technical_domain": "Network",
			"operational_urgency": "high",
			"security_incident": "no",
			"target_resolution_group": "NetOps Tier 2",
			"blast_radius": "single_user"
		},
		"confidence": 0.94,
		"recommended_action": "Route to NetOps Tier 2 for VPN gateway diagnostics."
	}` + "\n```"

	answers, conf, action, err := provider.ParseTriageJSON(raw, questions)
	if err != nil {
		t.Fatalf("unexpected error parsing triage JSON: %v", err)
	}

	if answers["technical_domain"] != "Network" {
		t.Errorf("expected 'Network', got %q", answers["technical_domain"])
	}
	// Case normalization check
	if answers["operational_urgency"] != "High" {
		t.Errorf("expected case-normalized 'High', got %q", answers["operational_urgency"])
	}
	if answers["security_incident"] != "No" {
		t.Errorf("expected case-normalized 'No', got %q", answers["security_incident"])
	}
	if answers["blast_radius"] != "Single User" {
		t.Errorf("expected normalized 'Single User', got %q", answers["blast_radius"])
	}
	if conf != 0.94 {
		t.Errorf("expected confidence 0.94, got %f", conf)
	}
	if !strings.Contains(action, "Route to NetOps") {
		t.Errorf("unexpected action: %q", action)
	}
}

func TestParseTriageJSONFallbackAction(t *testing.T) {
	questions := triage.StandardQuestions()

	raw := `{
		"answers": {
			"technical_domain": "Security",
			"operational_urgency": "Critical",
			"security_incident": "Yes",
			"target_resolution_group": "SecOps Incident Response",
			"blast_radius": "Organization-wide"
		},
		"confidence": 1.5,
		"recommended_action": ""
	}`

	answers, conf, action, err := provider.ParseTriageJSON(raw, questions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if conf != 1.0 {
		t.Errorf("expected clamped confidence 1.0, got %f", conf)
	}
	if answers["security_incident"] != "Yes" {
		t.Errorf("expected 'Yes', got %q", answers["security_incident"])
	}
	if !strings.Contains(action, "Escalate immediately to SecOps") {
		t.Errorf("expected security escalation fallback action, got %q", action)
	}

	// Non-security fallback should route to target group
	rawNonSec := `{
		"answers": {
			"technical_domain": "Hardware",
			"operational_urgency": "Low",
			"security_incident": "No",
			"target_resolution_group": "Service Desk",
			"blast_radius": "Single User"
		},
		"confidence": 0.88,
		"recommended_action": ""
	}`
	_, _, nonSecAction, err := provider.ParseTriageJSON(rawNonSec, questions)
	if err != nil {
		t.Fatalf("unexpected error for non-sec: %v", err)
	}
	if !strings.Contains(nonSecAction, "Route to Service Desk") {
		t.Errorf("expected routing action, got %q", nonSecAction)
	}
}

func TestParseTriageJSONWithReasoningAndPreambles(t *testing.T) {
	questions := triage.StandardQuestions()

	raw := "<think>\n" +
		"Evaluating operational questions for the incoming ticket:\n" +
		"1. technical_domain: looks like Network since DNS is failing.\n" +
		"2. operational_urgency: High because multiple users cannot connect.\n" +
		"3. security_incident: No evidence of compromise.\n" +
		"4. target_resolution_group: NetOps Tier 2.\n" +
		"5. blast_radius: Multi-user.\n" +
		"</think>\n" +
		"Here is the JSON evaluation:\n" +
		"```json\n" +
		"{\n" +
		"\t\"answers\": {\n" +
		"\t\t\"technical_domain\": \"Network\",\n" +
		"\t\t\"operational_urgency\": \"High\",\n" +
		"\t\t\"security_incident\": \"No\",\n" +
		"\t\t\"target_resolution_group\": \"NetOps Tier 2\",\n" +
		"\t\t\"blast_radius\": \"Multi-user\"\n" +
		"\t},\n" +
		"\t\"confidence\": 0.96,\n" +
		"\t\"recommended_action\": \"Route ticket to NetOps Tier 2 for DNS resolution.\"\n" +
		"}\n" +
		"```\n" +
		"All questions evaluated successfully."

	answers, conf, action, err := provider.ParseTriageJSON(raw, questions)
	if err != nil {
		t.Fatalf("unexpected error parsing triage JSON with reasoning: %v", err)
	}

	if answers["technical_domain"] != "Network" {
		t.Errorf("expected 'Network', got %q", answers["technical_domain"])
	}
	if answers["operational_urgency"] != "High" {
		t.Errorf("expected 'High', got %q", answers["operational_urgency"])
	}
	if conf != 0.96 {
		t.Errorf("expected confidence 0.96, got %f", conf)
	}
	if !strings.Contains(action, "Route ticket to NetOps") {
		t.Errorf("unexpected action: %q", action)
	}
}

func TestParseTriageJSONBooleanNormalization(t *testing.T) {
	questions := triage.StandardQuestions()

	testCases := []struct {
		name     string
		rawJSON  string
		expected string
	}{
		{
			name: "boolean literal false",
			rawJSON: `{
				"answers": { "security_incident": false },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
		{
			name: "string false",
			rawJSON: `{
				"answers": { "security_incident": "false" },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
		{
			name: "number zero",
			rawJSON: `{
				"answers": { "security_incident": 0 },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
		{
			name: "boolean literal true",
			rawJSON: `{
				"answers": { "security_incident": true },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "number one",
			rawJSON: `{
				"answers": { "security_incident": 1 },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "string yes",
			rawJSON: `{
				"answers": { "security_incident": "yes" },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "markdown bold and punctuation",
			rawJSON: `{
				"answers": { "security_incident": "**Yes**." },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "trailing dot false",
			rawJSON: `{
				"answers": { "security_incident": "false." },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
		{
			name: "float representation 1.0",
			rawJSON: `{
				"answers": { "security_incident": 1.0 },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "synonym negative",
			rawJSON: `{
				"answers": { "security_incident": "negative" },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
		{
			name: "synonym confirmed",
			rawJSON: `{
				"answers": { "security_incident": "confirmed" },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "Yes",
		},
		{
			name: "prefix in sentence",
			rawJSON: `{
				"answers": { "security_incident": "no, routine request" },
				"confidence": 0.9,
				"recommended_action": "Done"
			}`,
			expected: "No",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			answers, _, _, err := provider.ParseTriageJSON(tc.rawJSON, questions)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if answers["security_incident"] != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, answers["security_incident"])
			}
		})
	}
}
