---
title: Overview
normative-for: [thesis, principles, high-level architecture, subsystem routing]
depends: []
---

# Overview

`jev-usecase-1` is an evaluation harness and web application for the Jev machine learning model. It evaluates Jev on ITSM helpdesk ticket classification, labeling, and rapid triage against standard operational questions.

The project consists of two operational halves:

1. **Ticket generation pipeline.** Procedurally generates diverse, realistic IT support prompts, dispatches them to an LLM provider, and outputs structured ITSM tickets. Output can be saved to batch CSV/JSON files or emitted continuously on the fly.
2. **Decision Engine web application.** Ingests the generated tickets, submits them to the Jev model and comparative engines, and displays triage classification, label answers, confidence scores, and sub-second evaluation latency in a real-time interface.

## Thesis

Demonstrating classification accuracy and latency requires high-variance, realistic test corpora that can be produced deterministically or on demand. Rather than relying on static, stale helpdesk datasets, this harness couples procedural ITSM prompt generation with an evaluation pipeline.

The harness separates ticket generation from triage evaluation. The LLM acts purely as a synthetic customer and helpdesk ticket author. The Jev model acts as the triage and classification specialist.

## Principles

1. **Explicit separation of concerns.** Ticket generation and ticket triage never share state. The ticket data model is the boundary.
2. **Pluggable model backends.** The LLM ticket author and the Jev triage model both sit behind clean Go interfaces for modular integration.
3. **Structured outputs over raw text.** Tickets conform to an explicit schema with typed fields. Triage results answer a fixed set of operational questions.
4. **Streaming and batch symmetry.** The generation engine supports batch output (JSON/CSV) and real-time event streaming over HTTP with identical ticket schemas.
5. **Measurable latency.** Every Jev evaluation records wall-clock inference time so speed improvements are transparent and verifiable.

## Architecture

```text
[Procedural Generator]
  Dimensions: Incident types, personas, hardware/software, urgency triggers
         │
         ▼
[LLM Provider] (OpenAI / Anthropic / Gemini)
         │
         ▼
   ITSM Ticket (ID, Summary, Description, Urgency, Reporter, Domain)
   ├── Batch Export: JSON / CSV files
   └── Continuous Stream: In-memory queue / SSE
         │
         ▼
[Webapp Decision & Triage Engine]
         │
         ▼
[Jev Model Interface] (TypeSafe Jev API Endpoint)
         │
         ▼
[Standard Triage Answers & Live UI]
  Ticket Type · Technical Domain · Urgency · Security Escalation · Routing Team · Blast Radius
```

## Subsystems

| Subsystem | Document | Owns |
|---|---|---|
| Ticket Schema | [generator.md](generator.md) | Canonical ticket fields, JSON and CSV serialization contracts |
| Prompt & Ticket Generation | [generator.md](generator.md) | Procedural prompt assembly, dimension matrices, batch & streaming generation |
| Triage Evaluation | [triage.md](triage.md) | Standard triage questions, Jev model contract, latency accounting |
| Web Application | [webapp.md](webapp.md) | HTTP server, REST endpoints, SSE stream, presentation UI |
| Architecture & Boundaries | [architecture.md](architecture.md) | Component authority, invariants, and ports |
| Development Rules | [development.md](development.md) | Numbered engineering rules (DEV-01 through DEV-18) |
| Development Process | [development-process.md](development-process.md) | Four-step implementation and proof loop |
| Physical Layout | [repository-map.md](repository-map.md) | Code packages and their specification files |
| Architectural Decisions | [decisions.md](decisions.md) | ADR records for settled design choices |
| Documentation Voice | [documentation-style.md](documentation-style.md) | Writing rules, voice, and prose constraints |
