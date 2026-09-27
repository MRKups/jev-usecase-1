---
title: Triage Evaluation
normative-for: [standard triage questions, Jev model contract, latency metrics]
depends: [overview.md, generator.md]
---

# Triage Evaluation

The triage subsystem evaluates the Jev model's ability to classify and label incoming ITSM tickets against a standardized set of operational questions.

## Standard Triage Questions

Every ticket submitted to Jev is evaluated against standardized operational questions (customizable through the web application):

1. **Ticket Type**: Is this ticket an Incident or a Service Request?
   - Allowed answers: `Incident`, `Service Request`
2. **Domain Classification**: What is the primary technical domain of this ticket?
   - Allowed answers: `Hardware`, `Software`, `Network`, `Access & Identity`
3. **Operational Urgency**: What is the verified operational urgency level based on technical impact?
   - Allowed answers: `Critical`, `High`, `Medium`, `Low`
4. **Security Incident**: Does this issue indicate a security breach, unauthorized access, or policy violation requiring immediate SecOps review?
   - Allowed answers: `Yes`, `No`
5. **Target Resolution Group**: Which support tier or engineering team should resolve this issue?
   - Allowed answers: `Service Desk`, `Network Operations`, `Identity & Access Management`, `Site Reliability Engineering`, `SecOps`
6. **Blast Radius**: Does this issue affect an isolated user or indicate a broader systemic outage?
   - Allowed answers: `Single User`, `Multiple Users`, `Company-wide Outage`

## Triage Result Contract

The output of a Jev evaluation includes:

| Field | Type | Description |
|---|---|---|
| `ticket_id` | string | ID of the evaluated ticket |
| `evaluated_at` | string (ISO 8601) | Timestamp of triage completion |
| `latency_ms` | int64 | End-to-end evaluation duration in milliseconds |
| `confidence` | float64 | Overall confidence score (0.0 to 1.0) |
| `answers` | map[string]string | Map of question ID or text to Jev's determined answer |
| `recommended_action`| string | One-sentence summary action for the helpdesk dispatcher |

## Latency Accounting

The harness benchmarks Jev's speed. In helpdesk environments, routing decisions must execute in fractions of a second to prevent queue backlogs.

The harness wraps model calls with monotonic nanosecond timers, converting elapsed duration to milliseconds before recording results.

## TypeSafe System One Mapping

Evaluations against the TypeSafe API target the System One endpoint (`POST /v1/systemone`) using model `jev-latest`.

The ticket object is submitted as structured JSON in the `state` field. Operational questions map to TypeSafe primitives:

- `ticket_type`: `choice` primitive with criteria distinguishing unplanned service disruptions from formal service requests.
- `technical_domain`: `choice` primitive with criteria mapping each allowed category.
- `operational_urgency`: `choice` primitive with criteria for Critical, High, Medium, and Low severity.
- `security_incident`: `noul` primitive calculating calibrated probability of a security breach. Values of 0.5 or greater map to "Yes".
- `target_resolution_group`: `choice` primitive mapping tiers from Service Desk to SecOps.
- `blast_radius`: `choice` primitive evaluating isolated versus systemic impact.

Endpoint health and authentication are verified via the models catalog endpoint (`GET /v1/models`). Operational questions, choices, and criteria can also be customized, added, or reset directly through the Triage Criteria screen.

## Dual Decision Engines

The evaluation harness supports concurrent execution of two independent decision engines, designated Decision Engine A and Decision Engine B.

Each engine slot implements the `triage.Engine` interface:

```go
type Engine interface {
	Classify(ctx context.Context, t *ticket.Ticket, questions []Question) (*ticket.TriageResult, error)
	Ping(ctx context.Context) error
	Name() string
}
```

Either slot can be bound to TypeSafe Jev or to any supported LLM provider (OpenAI, Anthropic, Gemini). This enables side-by-side benchmarking of specialized decision models against general-purpose LLMs across inference latency, question consensus, and routing recommendations.

