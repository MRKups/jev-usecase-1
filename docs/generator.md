---
title: Ticket Generator
normative-for: [procedural prompt generation, ticket schema, export formats, streaming seam]
depends: [overview.md, architecture.md]
---

# Ticket Generator

The ticket generator procedurally creates realistic IT service management (ITSM) support tickets by combining varied scenario dimensions and querying an LLM provider.

## Ticket Schema

Every generated ticket adheres to the following structure:

| Field | Type | Description |
|---|---|---|
| `id` | string | Unique identifier (e.g. `#897d`) |
| `created_at` | string (ISO 8601) | Timestamp of generation |
| `reporter_name` | string | Full name of the fictitious employee |
| `reporter_email` | string | Corporate email address |
| `department` | string | Originating corporate department |
| `category` | string | Primary technical domain (Hardware, Software, Network, Access & Identity) |
| `reported_urgency` | string | Self-reported employee urgency (Low, Medium, High, Critical) |
| `summary` | string | Short issue headline |
| `description` | string | Detailed user narrative describing symptoms and business impact |
| `affected_system` | string | Generic enterprise service or asset (e.g. Internal CRM, Single Sign-On Portal, Network Printer) |

## Procedural Prompt Assembly

To generate diverse tickets without repetitive templates or prescriptive scenarios, the prompt engine combines orthogonal dimensions:

1. **Department and Worker Persona**: 10 hardcoded corporate departments (Engineering, Finance & Accounting, Sales & Commercial, Customer Support & Success, Human Resources & People, Legal & Compliance, Product & Design, Marketing & Communications, Operations & Workplace, Executive Office). Each department maps to realistic job titles without situational descriptions or abbreviations.
2. **Category**: Hardware, Software, Network, Access & Identity.
3. **Generic Affected Systems**: 30 enterprise systems using generic terminology without vendor trademarks or brand names (for example: Internal CRM, Telephony System, Single Sign-On Portal, Enterprise Resource Planning System, Video Conference System).
4. **Generic Problem Classes**: 24 generalized failure categories rather than specific pre-scripted stories (for example: Authentication Failure, Connection Timeout, Application Crash, Hardware Malfunction, Data Synchronization Error).
5. **Reporter Emotional Tone**: Calm and structured, Frustrated, Urgent, Confused, Polite, Concise.
6. **Reported Urgency**: Low, Medium, High, Critical.
7. **Communication Archetype**: 24 distinct user communication styles spanning language proficiency, technical literacy, and behavioral posture (such as Clear and Structured Professional, Non-Native English Speaker, Terse Mobile Reporter, Senior Engineer Stack Dump, Panicked Catastrophizing, Hardware Confuser, and Minimalist One-Liner).

The generator combines these sampled parameters into an instruction prompt that commands the LLM to write from the perspective of that worker persona facing that problem class on that system, adopting the assigned communication archetype and outputting structured JSON conforming to the ticket schema. To avoid collisions with real living individuals, the LLM generates fictitious Latin-rooted names with dot-formatted usernames (`first.last`), and the Go parser deterministically appends `@example.com`.

## Batch and Streaming Modes

The generator supports two modes of execution:

- **Batch Generation (`ticket-eval generate`)**: Synthesizes N tickets and persists them directly to JSON or CSV files for repeatable evaluation and archiving.
- **Continuous Stream (`ticket-eval serve` / SSE)**: Continuously produces tickets at a configured cadence and pushes them to connected web clients or directly into the triage queue.
