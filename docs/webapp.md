---
title: Decision Engine Web Application
normative-for: [HTTP API routes, user interface layout, 4-page navigation flow, live event streaming]
depends: [overview.md, generator.md, triage.md]
---

# Decision Engine Web Application

The web application provides an interactive four-page dashboard demonstrating configuration, synthetic dataset generation, operational criteria customization, and decision engine performance.

## Four-Page User Interface

The application interface is structured into four dedicated tabs:

1. **Page 1: Configuration (`#page-config`)**
   - **Ticket Generator Model (Card 1)**: Selects and configures the active LLM generator (one at a time):
     - `OpenAI`: Configures API key, model (e.g. `gpt-4o-mini`, `gpt-4o`), and optional base URL (supporting OpenRouter and local OpenAI-compatible proxies).
     - `Anthropic`: Configures API key, model (e.g. `claude-3-5-haiku-latest`, `claude-3-5-sonnet-latest`), and optional base URL.
     - `Google Gemini`: Configures API key and model (e.g. `gemini-2.5-flash`).
   - **Configuration Persistence (Card 2)**: Filesystem persistence for runtime configuration (`config.json`).
   - **Decision Engine A (Card 3)**: Configures primary decision engine slot (TypeSafe Jev or any supported LLM).
   - **Decision Engine B (Card 4)**: Configures optional comparative decision engine slot.

2. **Page 2: Dataset Generator & Preview (`#page-dataset`)**
   - **Generation Controls**: Allows specifying ticket count and generating synthetic tickets via the active generator backend.
   - **Dataset Preview**: Full-width structured data table displaying generated tickets with IDs, reported urgencies, departments, categories, affected systems, summaries, and reporters.
    - **Dataset Persistence**: Saves and loads synthetic datasets to and from a separate JSON file (`dataset.json` at root by default).

3. **Page 3: Triage Criteria (`#page-criteria`)**
   - **Operational Questions & Rubric**: Displays structured evaluation questions answered by Decision Engines for each ticket.
   - **Question Customization**: Edit question text, evaluation instructions, and per-option rubric guidance in real time. Add and remove multiple-choice options with discrete option tags.
   - **Question Management**: Add new custom operational questions or remove questions.
   - **Criteria Reset & Save**: Persist updated questions to the runtime engine, save/load custom question sets from `questions.json` at root, or reset to canonical defaults.

4. **Page 4: Decision Engine (`#page-showcase`)**
   - **KPI Summary Banner**: Real-time metrics tracking:
     - Total tickets evaluated vs total in dataset.
     - Engine A and Engine B latency.
     - Speedup multiplier and cross-engine consensus percentage.
   - **Batch Evaluation**: Evaluates all tickets concurrently against active operational criteria.
   - **Interactive Inspector**: Displays ticket narrative side-by-side with structured decisions from Engine A and Engine B:
     1. Ticket Type (Incident vs Service Request)
     2. Technical Domain
     3. Operational Urgency
     4. Security Incident (with prominent visual alert if detected)
     5. Target Resolution Group
     6. Blast Radius (and any additional custom criteria)
     - Agreement/differs indicators, confidence scores, and recommended action callouts.

## HTTP Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Serves the single-page HTML dashboard |
| `GET` | `/api/config` | Returns active runtime configuration |
| `POST` | `/api/config` | Updates configuration in-memory and reinitializes active generator |
| `POST` | `/api/config/save` | Persists current configuration to `config.json` |
| `POST` | `/api/config/load` | Reloads configuration from `config.json` |
| `POST` | `/api/config/test-provider` | Tests connectivity and credentials for the specified generator |
| `POST` | `/api/config/test-engine` | Tests connectivity and credentials for Decision Engine A or B |
| `POST` | `/api/config/test-jev` | Tests connectivity and credentials for TypeSafe Jev (compatibility alias) |
| `GET` | `/api/dataset` | Returns current ticket dataset in memory |
| `POST` | `/api/dataset/generate` | Generates N tickets using the active generator |
| `POST` | `/api/dataset/cancel` | Aborts active dataset generation |
| `POST` | `/api/dataset/save` | Persists current dataset to JSON file |
| `POST` | `/api/dataset/load` | Ingests tickets from JSON dataset file |
| `POST` | `/api/dataset/clear` | Clears in-memory dataset |
| `GET` | `/api/criteria` | Returns current operational triage questions and rubric |
| `POST` | `/api/criteria` | Updates operational triage questions and rubric in memory |
| `POST` | `/api/criteria/save` | Persists current triage questions to `questions.json` |
| `POST` | `/api/criteria/load` | Reloads triage questions from `questions.json` |
| `POST` | `/api/criteria/reset` | Resets triage questions to canonical defaults |
| `POST` | `/api/triage/run` | Concurrently evaluates all tickets using Decision Engines A and B |
| `POST` | `/api/triage/{id}` | Concurrently evaluates a single ticket using Decision Engines A and B |
| `GET` | `/api/stream` | Server-Sent Events (SSE) stream for live updates |
| `GET` | `/static/...` | Serves modular static CSS stylesheets and JavaScript assets |

## Implementation Constraints

The web application is self-contained:
- Pure Go standard library (`net/http`, `html/template`, and `embed`).
- Zero external frontend runtime frameworks or npm dependencies.
- Modular server-composed SPA architecture combining Go template partials (`templates/*.html`) with domain-separated static assets (`static/css/` and `static/js/`).
