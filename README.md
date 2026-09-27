# jev-usecase-1: ITSM Ticket Triage Evaluation Harness

`jev-usecase-1` provides `ticket-eval`, an open-source evaluation harness and interactive web demonstrator for IT Service Management (ITSM) helpdesk ticket classification, automated labeling, and rapid triage.

It benchmarks specialized machine learning decision engines (such as TypeSafe Jev) against general-purpose large language models (OpenAI, Anthropic Claude, Google Gemini, OpenRouter, and local offline models via Ollama or LM Studio).

---

## What is This?

`ticket-eval` is a standalone Go application that couples a procedural ITSM ticket generation engine with a comparative dual-engine triage evaluation pipeline. It provides both a command-line interface (CLI) for batch processing and an interactive web dashboard for real-time demonstration.

The system is split into two decoupled operational halves:

1. **Procedural Ticket Generator**: Synthesizes authentic, high-variance IT helpdesk tickets by combining orthogonal scenario dimensions: corporate departments, worker roles, enterprise systems, failure modes, emotional urgency, and 24 distinct user communication archetypes (ranging from non-native English speakers to panicked catastrophizing and terse mobile reporters).
2. **Dual-Engine Decision Dashboard**: An interactive web application that ingests synthetic or imported ticket datasets, dispatches them concurrently to two decision engines (Engine A and Engine B), and visualizes triage decisions, confidence scores, and sub-second execution latency in real time.

---

## What is It For?

Modern enterprise IT helpdesks process high ticket volumes daily. Human triage creates bottlenecks, while general-purpose LLMs often introduce high token costs, multi-second inference latency, and schema variance.

This project demonstrates and evaluates:

- **Classification Accuracy & Consensus**: Evaluates how models categorize tickets against standard operational questions (Ticket Type, Technical Domain, Operational Urgency, Security Incident escalation, Target Resolution Group, Blast Radius).
- **Latency Benchmarking**: Measures wall-clock execution time to quantify the speed advantage of specialized decision models (often under 200 milliseconds) versus general-purpose LLMs (typically 1 to 5+ seconds).
- **Realistic Synthetic Corpora**: Generates realistic, reproducible test datasets on demand without exposing proprietary employee records or sensitive corporate incident history.
- **Custom Operational Rubrics**: Allows teams to customize triage questions, choices, and evaluation guidance in real time to model specific corporate IT workflows.

---

## Prerequisites

To build and run `ticket-eval`, you will need:

- **Go Compiler**: Go 1.24 or later (supports Go 1.24 through Go 1.27+). Pure Go with zero CGO required.
- **Operating System**: macOS (Apple Silicon or Intel), Linux (AMD64 or ARM64), or Windows (AMD64).
- **Web Browser**: Any modern web browser (Chrome, Firefox, Safari, Edge) to use the interactive dashboard.
- **Task Runner (Optional)**: [Task](https://taskfile.dev) for running predefined workflow tasks (`brew install go-task` or `go install github.com/go-task/task/v3/cmd/task@latest`).

### Model & API Credentials (Optional)

The harness can run entirely with local fixtures or local offline models, but supports external backends:

- **Synthetic Ticket Generation**: Requires an API key for OpenAI, Anthropic, or Google Gemini. Alternatively, point the base URL to a local OpenAI-compatible server (Ollama, LM Studio, vLLM) with no API key needed.
- **Decision Engine A / B**:
  - **TypeSafe Jev**: Requires TypeSafe API credentials and endpoint (defaults to `https://api.typesafe.ai`).
  - **LLM Providers**: Connects directly to OpenAI, Anthropic, Google Gemini, OpenRouter, or local OpenAI-compatible endpoints.

---

## How to Compile

`ticket-eval` compiles into a single, self-contained binary with zero external runtime dependencies and `CGO_ENABLED=0`.

### Standard Local Build

```sh
# Build binary into bin/ticket-eval
CGO_ENABLED=0 go build -o bin/ticket-eval ./cmd/ticket-eval
```

Using Task:

```sh
task build
```

### Cross-Compilation

Targeting Apple Silicon (macOS ARM64):

```sh
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -o bin/ticket-eval-darwin-arm64 ./cmd/ticket-eval
```

Targeting Linux (x86_64):

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -o bin/ticket-eval-linux-amd64 ./cmd/ticket-eval
```

Targeting Windows (x86_64):

```sh
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -o bin/ticket-eval.exe ./cmd/ticket-eval
```

---

## How to Run

### 1. Interactive Web Dashboard

Start the decision engine web server:

```sh
./bin/ticket-eval serve --addr=127.0.0.1:8080
# or using task:
task run:web
```

Open `http://127.0.0.1:8080` in your web browser. The dashboard includes four functional tabs:

1. **Configuration**: Set API credentials, base URLs, and active model providers for ticket generation and dual decision engines.
2. **Dataset Generator & Preview**: Generate N synthetic tickets, inspect generated ticket details, and save or load JSON datasets.
3. **Triage Criteria**: Review and customize the standard operational questions, choice options, and rubric guidance.
4. **Decision Engine**: Run batch evaluations, compare Engine A and Engine B side-by-side, inspect agreement/differs indicators, and view real-time latency and speedup metrics.

### 2. Batch Ticket Generation (CLI)

Generate synthetic ITSM tickets directly to disk:

```sh
# Generate 20 tickets to JSON
./bin/ticket-eval generate --count=20 --format=json --output=dataset.json

# Generate 20 tickets to CSV
./bin/ticket-eval generate --count=20 --format=csv --output=tickets.csv
```

### 3. Triage Evaluation (CLI)

Evaluate a ticket dataset against standard triage questions from the command line:

```sh
# Evaluate a generated dataset
./bin/ticket-eval triage --input=dataset.json

# Evaluate pre-packaged sample fixtures
./bin/ticket-eval triage --input=fixtures/tickets/sample_tickets.json
# or using task:
task triage:sample
```

### 4. CLI Command Surface

```text
ticket-eval serve [--addr=<address>]
  Starts the decision engine web server and live event stream. Default :8080.

ticket-eval generate [--count=<n>] [--format=json|csv] [--output=<path>]
  Procedurally generates ITSM tickets using the configured LLM provider.

ticket-eval triage [--input=<path>]
  Evaluates tickets against the standard operational questions.

ticket-eval version
  Prints the application version and build metadata.
```

---

## Testing & Verification

Run the automated test suite:

```sh
# Run all unit tests
task test
# or: go test -v ./...

# Run tests with the Go data race detector
task test:race
# or: go test -race -v ./...

# Run complete guard verification (formatting, vetting, race detector, CGO check)
task check
# or: ./scripts/check-guard.sh
```

---

## Repository Structure

```text
├── cmd/
│   └── ticket-eval/          # Main application process entry point
├── internal/
│   ├── cli/                  # CLI routing (serve, generate, triage) and flag parsing
│   ├── config/               # Runtime configuration and JSON persistence
│   ├── generator/            # Procedural prompt builder and dimension sampling
│   ├── jev/                  # TypeSafe Jev API client and benchmark timer
│   ├── provider/             # Native LLM adapters (OpenAI, Anthropic, Gemini, OpenRouter)
│   ├── ticket/               # Canonical ITSM ticket domain model and encoders
│   ├── triage/               # Standard triage questions, rubric models, and evaluation engine
│   └── webapp/               # Standard library HTTP server, templates, and SSE stream
├── fixtures/
│   └── tickets/              # Deterministic sample ticket corpora (JSON and CSV)
├── docs/                     # Specifications, architecture, and design records
├── scripts/                  # Verification and guard scripts
├── Taskfile.yml              # Predefined developer tasks
└── LICENSE                   # MIT License
```

---

## Documentation Index

| Topic | Document |
|---|---|
| Project Purpose & Principles | [docs/overview.md](docs/overview.md) |
| Architecture & Boundaries | [docs/architecture.md](docs/architecture.md) |
| Development Rules (DEV-01 to DEV-18) | [docs/development.md](docs/development.md) |
| Implementation Process | [docs/development-process.md](docs/development-process.md) |
| Physical Layout & Package Map | [docs/repository-map.md](docs/repository-map.md) |
| Ticket Schema & Generator Prompt Rules | [docs/generator.md](docs/generator.md) |
| Standard Triage Questions & Jev Contract | [docs/triage.md](docs/triage.md) |
| Web Application & Streaming Architecture | [docs/webapp.md](docs/webapp.md) |
| Architectural Decision Records (ADRs) | [docs/decisions.md](docs/decisions.md) |
| Documentation Voice & Style Guide | [docs/documentation-style.md](docs/documentation-style.md) |

For contributor constraints and engineering invariants, review [AGENTS.md](AGENTS.md).

---

## License

This project is licensed under the [MIT License](LICENSE).
