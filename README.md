# jev-usecase-1

A small demonstrator project showing how [TypeSafe Jev](https://typesafe.ai) can be used to classify and triage IT helpdesk tickets, comparing its answers and latency side-by-side with general LLMs.

## What it does

- **Synthetic ticket generator**: creates fake IT support tickets with varying user tones and technical issues (using OpenAI, Claude, Gemini, OpenRouter, or local models via Ollama/LM Studio).
- **Triage evaluator**: runs tickets through standard IT triage questions (ticket type, urgency, category, routing group) using Jev and an LLM.
- **Web UI**: a local dashboard to generate tickets, edit triage criteria, and compare decisions and response times side-by-side.
- **CLI**: straightforward commands to generate tickets or run triage from your terminal.

## Quickstart

### Prerequisites

- Go 1.24+
- A modern web browser
- (Optional) API keys for LLM providers or Jev, or a local OpenAI-compatible endpoint like Ollama or LM Studio.

### Run the web app

```sh
go run ./cmd/ticket-eval serve
```

Then open `http://127.0.0.1:8080` in your browser.

To build a standalone binary instead:

```sh
go build -o bin/ticket-eval ./cmd/ticket-eval
./bin/ticket-eval serve
```

### CLI usage

Generate 20 test tickets to a JSON file:

```sh
./bin/ticket-eval generate --count=20 --output=dataset.json
```

Run triage evaluation on a dataset:

```sh
./bin/ticket-eval triage --input=dataset.json
```

Or run against the included sample tickets:

```sh
./bin/ticket-eval triage --input=fixtures/tickets/sample_tickets.json
```

## Project layout

- `cmd/ticket-eval/`: Main CLI and server entry point.
- `internal/generator/`: Synthetic ticket generation logic.
- `internal/jev/`: Jev API client.
- `internal/provider/`: LLM adapters (OpenAI, Anthropic, Gemini, OpenRouter).
- `internal/triage/`: Triage questions and scoring rubric.
- `internal/webapp/`: Local web dashboard and live event stream.
- `fixtures/`: Sample tickets and prompt templates.
- `docs/`: Technical notes and architecture documentation.

## Testing

```sh
go test ./...
```

Or run the full check script:

```sh
./scripts/check-guard.sh
```

## License

[MIT](LICENSE)
