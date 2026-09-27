# jev-usecase-1 Contributor Instructions

`jev-usecase-1` is an evaluation harness and web application for the Jev machine learning model and comparative decision engines. It evaluates Jev's ability to classify, label, and triage IT service management (ITSM) helpdesk tickets rapidly against standard operational questions.

## Read First

Before modifying architecture or contracts, read:

- `docs/overview.md`: Entry point, thesis, and subsystem routing.
- `docs/architecture.md`: Authority boundaries, invariants, and data flows.
- `docs/development.md`: Numbered development rules (DEV-01 through DEV-18).
- `docs/development-process.md`: Implementation and verification loop.
- `docs/documentation-style.md`: Writing style and rules for documentation.
- `docs/generator.md`: Procedural prompt generation and ticket schema.
- `docs/triage.md`: Standard triage questions and Jev contract.
- `docs/webapp.md`: Web application and streaming architecture.

Each concept has one normative home. Cross-reference it instead of introducing duplicated definitions.

## Repository Layout

```text
README.md                  project entry point, quickstart, build commands
AGENTS.md                  contributor constraints and design invariants
Taskfile.yml               build, test, generate, and run tasks
cmd/ticket-eval/           process entry point and CLI commands
internal/ticket/           canonical ITSM ticket domain model and encoders
internal/generator/        procedural prompt generation engine
internal/provider/         LLM provider interface and adapters
internal/triage/           standard triage questions and evaluation rubric
internal/jev/              Jev model client interface and HTTP client
internal/webapp/           HTTP server, templates, and SSE stream
internal/config/           runtime configuration
fixtures/                  deterministic ticket corpora and sample prompts
docs/                      normative specifications and decisions
```

## Design Invariants

- **The ticket is immutable.** Triage attaches a separate evaluation record without altering the original ticket fields.
- **Backends sit behind interfaces.** LLM generation and Jev classification both implement clean Go interfaces.
- **Zero CGO.** All builds target `CGO_ENABLED=0` for clean cross-platform binaries.
- **Standard library first.** The HTTP server and template rendering use Go's standard library.

## Documentation Rules

- Follow `docs/documentation-style.md` for all prose.
- **No em-dashes in prose.** Use commas, colons, periods, or new sentences.
- Keep statements direct and fact-based. Avoid conversational filler or promotional language.
