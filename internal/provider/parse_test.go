package provider

import (
	"strings"
	"testing"
)

func TestParseTicketJSONDotFormatEmail(t *testing.T) {
	jsonText := `{
		"reporter_name": "Marcus Valerius",
		"reporter_email": "marcus.valerius",
		"department": "Engineering",
		"category": "Hardware",
		"reported_urgency": "High",
		"summary": "Laptop trackpad bulge",
		"description": "Battery swelling observed.",
		"affected_system": "Laptop Workstation"
	}`

	ticket, err := parseTicketJSON(jsonText, "test-prompt")
	if err != nil {
		t.Fatalf("parseTicketJSON failed: %v", err)
	}

	if ticket.ReporterName != "Marcus Valerius" {
		t.Errorf("expected ReporterName 'Marcus Valerius', got %q", ticket.ReporterName)
	}

	if ticket.ReporterEmail != "marcus.valerius@example.com" {
		t.Errorf("expected ReporterEmail 'marcus.valerius@example.com', got %q", ticket.ReporterEmail)
	}

	if ticket.ReportedUrgency != "High" {
		t.Errorf("expected ReportedUrgency 'High', got %q", ticket.ReportedUrgency)
	}
}

func TestParseTicketJSONMarkdownFenceAndDomainStripping(t *testing.T) {
	jsonWithFence := "```json\n" + `{
		"reporter_name": "Claudia Octavia",
		"reporter_email": "claudia.octavia@otherdomain.com",
		"department": "Finance & Accounting",
		"category": "Software",
		"urgency": "Critical",
		"summary": "ERP sync failure",
		"description": "Sync error on financial batch.",
		"affected_system": "Enterprise Resource Planning System"
	}` + "\n```"

	ticket, err := parseTicketJSON(jsonWithFence, "test-prompt")
	if err != nil {
		t.Fatalf("parseTicketJSON failed: %v", err)
	}

	if ticket.ReporterEmail != "claudia.octavia@example.com" {
		t.Errorf("expected domain stripped to @example.com, got %q", ticket.ReporterEmail)
	}

	if ticket.ReportedUrgency != "Critical" {
		t.Errorf("expected legacy urgency mapped to ReportedUrgency 'Critical', got %q", ticket.ReportedUrgency)
	}

	if !strings.HasPrefix(ticket.ID, "#") || len(ticket.ID) != 5 {
		t.Errorf("expected ID with '#' prefix and 4 hex characters, got %q", ticket.ID)
	}
}

func TestParseTicketJSONInferReporterName(t *testing.T) {
	// JSON payload without reporter_name field
	jsonWithoutName := `{
		"reporter_email": "lucius.vorenus",
		"department": "Sales & Commercial",
		"category": "Network",
		"reported_urgency": "Medium",
		"summary": "VPN tunnel drops",
		"description": "Network tunnel disconnects frequently.",
		"affected_system": "Virtual Private Network Client"
	}`

	ticket, err := parseTicketJSON(jsonWithoutName, "test-prompt")
	if err != nil {
		t.Fatalf("parseTicketJSON failed: %v", err)
	}

	if ticket.ReporterName != "Lucius Vorenus" {
		t.Errorf("expected ReporterName inferred as 'Lucius Vorenus', got %q", ticket.ReporterName)
	}

	if ticket.ReporterEmail != "lucius.vorenus@example.com" {
		t.Errorf("expected ReporterEmail 'lucius.vorenus@example.com', got %q", ticket.ReporterEmail)
	}
}

func TestExtractJSONObjectWithReasoningAndPreambles(t *testing.T) {
	raw := "<think>\n" +
		"The user needs a support ticket for an ERP issue.\n" +
		"Let's make sure to include curly braces like {system} in thinking to test robust matching.\n" +
		"</think>\n" +
		"Here is the requested ticket:\n" +
		"```json\n" +
		"{\n" +
		"\t\"summary\": \"Database connectivity loss\",\n" +
		"\t\"description\": \"Error occurred in {cluster_prod_01} during batch job.\",\n" +
		"\t\"category\": \"Database\"\n" +
		"}\n" +
		"```\n" +
		"Let me know if you need more tickets!"

	extracted := ExtractJSONObject(raw)
	expectedPrefix := "{\n\t\"summary\": \"Database connectivity loss\""
	if !strings.HasPrefix(extracted, expectedPrefix) {
		t.Errorf("expected extracted JSON to start with %q, got %q", expectedPrefix, extracted)
	}
	if !strings.HasSuffix(extracted, "}") {
		t.Errorf("expected extracted JSON to end with '}', got %q", extracted)
	}
	if strings.Contains(extracted, "<think>") || strings.Contains(extracted, "Here is the requested") {
		t.Errorf("extracted text still contains reasoning or preamble: %q", extracted)
	}
}

func TestParseTicketJSONWithThinkTag(t *testing.T) {
	raw := `<think>
I need to generate a realistic ticket.
Reporter: Marcus Valerius.
Department: Finance.
</think>
{
	"reporter_name": "Marcus Valerius",
	"reporter_email": "marcus.valerius@example.com",
	"department": "Finance",
	"category": "Software",
	"reported_urgency": "High",
	"summary": "Payroll portal unresponsive",
	"description": "Timeout when loading monthly payroll records.",
	"affected_system": "Payroll System"
}`

	ticket, err := parseTicketJSON(raw, "test-prompt")
	if err != nil {
		t.Fatalf("parseTicketJSON failed with think tag: %v", err)
	}

	if ticket.ReporterName != "Marcus Valerius" {
		t.Errorf("expected ReporterName 'Marcus Valerius', got %q", ticket.ReporterName)
	}
	if ticket.Department != "Finance" {
		t.Errorf("expected Department 'Finance', got %q", ticket.Department)
	}
	if ticket.ReportedUrgency != "High" {
		t.Errorf("expected ReportedUrgency 'High', got %q", ticket.ReportedUrgency)
	}
}

func TestParseTicketJSONWithNestedBracesInStrings(t *testing.T) {
	raw := `{
		"reporter_name": "Marcus Valerius",
		"reporter_email": "marcus.valerius@example.com",
		"department": "Engineering",
		"category": "Software",
		"reported_urgency": "Medium",
		"summary": "Config file parse error on {template_id}",
		"description": "Line 42 has a syntax error in {config.json}: \"key\": \"value\".",
		"affected_system": "Config Service"
	}`

	ticket, err := parseTicketJSON(raw, "test-prompt")
	if err != nil {
		t.Fatalf("parseTicketJSON failed with braces in strings: %v", err)
	}

	if ticket.Summary != "Config file parse error on {template_id}" {
		t.Errorf("expected summary with preserved braces, got %q", ticket.Summary)
	}
}
