---
title: Architecture
normative-for: [authority boundaries, invariants, composition, data flows]
depends: [overview.md]
---

# Architecture

This document specifies the authority boundaries, data flow, invariants, and component responsibilities of `jev-usecase-1`.

## Component Responsibilities

The system consists of two decoupled subsystems connected by an explicit ticket contract:

1. **Ticket Generation Subsystem (`internal/generator`, `internal/provider`)**
   - Owns the combinatorial prompt generation rules.
   - Combines domains, failure modes, user personas, and emotional urgency into structured LLM prompts.
   - Invokes the configured LLM provider to author realistic IT support tickets.
   - Emits structured tickets into batch files (JSON or CSV) or into an in-memory channel for live consumption.

2. **Triage and Decision Subsystem (`internal/triage`, `internal/jev`, `internal/webapp`)**
   - Ingests tickets from files or live streams.
   - Dispatches tickets along with a standard set of triage questions to the Jev model interface.
   - Collects answers, classification tags, confidence metrics, and execution latency.
   - Renders tickets and Jev triage results in an interactive web application.

## Authority

| Concern | Authoritative Component |
|---|---|
| ITSM Ticket Schema | `internal/ticket.Ticket` |
| Prompt Dimension Taxonomy | `internal/generator.DimensionMatrix` |
| Synthetic Ticket Generation | Configured `provider.LLMProvider` |
| Triage Question Definitions | `internal/triage.StandardQuestions` |
| Triage Answers and Latency | Configured `jev.Client` |
| Batch File Persistence | `internal/ticket` encoders (JSON/CSV) |
| Live Ingress and Presentation | `internal/webapp.Server` |

## Invariants

- **The ticket is an immutable value object.** Once generated, a ticket's fields (reporter, summary, description, reported urgency) do not mutate during triage. Triage outcomes are stored in an attached `TriageResult` record.
- **Providers sit behind interfaces.** Neither the generator nor the triage evaluator imports specific vendor SDKs directly into core logic. Adapters implement `provider.LLMProvider` and `jev.Client`.
- **Explicit provider configuration.** Generation uses configured LLM providers (OpenAI, Anthropic, Gemini) and Jev triage communicates with the TypeSafe Jev API endpoint. Missing configuration surfaces clear, actionable diagnostics.
- **Latency is strictly measured.** Triage latency is measured as wall-clock duration from the invocation of `jev.Client.Classify` until a response is received, recorded in milliseconds.
- **Errors are structured and non-fatal to streams.** A failure to generate or triage an individual ticket logs an error event and continues processing the remainder of the batch or stream.

## Non-goals

- Direct integration with proprietary ITSM databases (e.g. ServiceNow or Jira API syncing is outside the initial scope).
- Live retraining or fine-tuning of the Jev model within this harness.
- Multi-user authentication, billing, or tenant isolation for the web application.
