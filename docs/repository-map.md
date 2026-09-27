---
title: Repository map
normative-for: [repository layout, package-to-document routing]
depends: [overview.md]
---

# Repository map

Where code and artifacts live on disk, and which document specifies each one.

## Entry points

| Path | Purpose |
|---|---|
| `README.md` | Product overview, build commands, and quickstart |
| `AGENTS.md` | Contributor instructions, engineering rules, and invariants |
| `Taskfile.yml` | Build, test, run, and ticket generation tasks |
| `cmd/ticket-eval/` | Process entry point and signal handling |
| `internal/` | Go packages and unit tests |
| `fixtures/` | Sample ticket corpora (JSON/CSV) and prompt templates |
| `docs/` | System specifications and design records |

## Specifications and Rules

| Path | Purpose |
|---|---|
| `docs/overview.md` | System thesis, principles, and high-level architecture |
| `docs/architecture.md` | Component authority, invariants, and data flows |
| `docs/development.md` | Numbered engineering rules (DEV-01 through DEV-18) |
| `docs/development-process.md` | Four-step implementation and verification loop |
| `docs/documentation-style.md` | Writing style, voice, and prose constraints |
| `docs/generator.md` | Procedural prompt generation and ticket schema |
| `docs/triage.md` | Standard triage questions and Jev contract |
| `docs/webapp.md` | Web application and streaming architecture |
| `docs/decisions.md` | Architectural decision records |

## Packages

| Package | Purpose | Specified in |
|---|---|---|
| `cmd/ticket-eval` | Command-line process entry point | [overview.md](overview.md) |
| `internal/cli` | Subcommand routing (`serve`, `generate`, `triage`) and flag parsing | [overview.md](overview.md) |
| `internal/ticket` | Canonical ITSM ticket model, JSON and CSV encoding/decoding | [generator.md](generator.md) |
| `internal/generator` | Procedural prompt builder, dimension sampling, and ticket generation pipeline | [generator.md](generator.md) |
| `internal/provider` | LLM provider interface and native SDK adapters (OpenAI, Anthropic, Gemini) | [generator.md](generator.md) |
| `internal/triage` | Standard triage questions, evaluation coordinator, rubric definitions | [triage.md](triage.md) |
| `internal/jev` | Jev model adapter interface, TypeSafe HTTP client, and benchmark timer | [triage.md](triage.md) |
| `internal/webapp` | Web server, REST API, SSE streaming, and HTML dashboard | [webapp.md](webapp.md) |
| `internal/config` | Environment and flag configuration parsing | [architecture.md](architecture.md) |

## Data artifacts

| Path | Purpose |
|---|---|
| `fixtures/tickets/sample_tickets.json` | Baseline JSON corpus for offline testing |
| `fixtures/tickets/sample_tickets.csv` | Baseline CSV corpus for offline testing |
| `dataset.json` | Default starter ticket corpus at root |
