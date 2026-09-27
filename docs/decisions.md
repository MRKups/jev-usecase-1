---
title: Decisions
normative-for: [architectural decision records]
depends: [overview.md, architecture.md]
---

# Decisions

Architectural decision records tracking choices, context, and consequences.

## D1 - Standard library HTTP stack with modular template composition

- **Context**: The web application is a decision engine dashboard demonstrating the Jev model's speed and capabilities.
- **Decision**: Use Go's standard library `net/http` and `html/template` with modular template partials (`templates/*.html`) and domain-separated static assets (`static/css/`, `static/js/`).
- **Consequences**: Zero external frontend dependencies, no npm or node build steps, instantaneous compilation, and maintainable modular files.

## D2 - Single binary for generator, triage engine, and web server

- **Context**: Users and evaluators need to run ticket generation, export to CSV/JSON, or launch the interactive UI from the command line.
- **Decision**: Maintain a single root CLI `cmd/ticket-eval` with subcommands (`serve`, `generate`, `triage`).
- **Consequences**: One binary to build, distribute, and cross-compile.

## D3 - Native providers and explicit endpoint configuration

- **Context**: The application demonstrates real model integrations without synthetic mock proxies.
- **Decision**: Remove mock providers. Connect directly to native LLM providers (OpenAI, Anthropic, Gemini) and the TypeSafe Jev decision engine endpoint.
- **Consequences**: No mock emulation layers in the codebase. Tests use standard library test doubles and HTTP test servers.

## D4 - Standard triage questions rubric and extensible criteria

- **Context**: Demonstrating Jev requires consistent, structured criteria rather than arbitrary open-ended text.
- **Decision**: Define six canonical operational questions (Ticket Type, Technical Domain, Urgency, Security Escalation, Target Resolution Group, Blast Radius) with closed categorical answer choices, while supporting runtime addition and customization of questions and rubrics.
- **Consequences**: Enables quantitative measurement of accuracy, consistent presentation in the web interface, and full domain extensibility.

## D5 - Native Go SDKs for OpenAI, Anthropic, and Gemini

- **Context**: Synthetic ticket generation requires support for the leading commercial LLM providers.
- **Decision**: Integrate official first-party Go SDKs (`github.com/openai/openai-go`, `github.com/anthropics/anthropic-sdk-go`, and `google.golang.org/genai`) encapsulated behind the single-method `provider.LLMProvider` interface.
- **Consequences**: First-class support for OpenAI, Anthropic, and Gemini with idiomatic client construction while preserving strict domain decoupling.

## D6 - Three separate root JSON files for state persistence

- **Context**: The application needs simple configuration, dataset management, and criteria customization for local demonstrations without database infrastructure.
- **Decision**: Persist application state across three independent JSON files at the root of execution: `config.json` (runtime configuration and provider credentials), `dataset.json` (synthetic tickets corpus), and `questions.json` (operational criteria and rubrics).
- **Consequences**: Clean separation of concerns. Datasets, prompts/criteria, and credentials can be shared, versioned, or swapped independently. Clear disclosure that API keys are retained in plaintext on disk.

## D7 - Four-page frontend architecture

- **Context**: Users need to configure providers, generate synthetic datasets, define triage criteria, and inspect decision engine outputs in an intuitive workflow.
- **Decision**: Organize the web interface into four focused tabs: (1) Configuration, (2) Dataset Generator & Preview, (3) Triage Criteria, and (4) Decision Engine.
- **Consequences**: Clean separation of administrative setup, ticket generation, operational rubric modeling, and comparative decision engine demonstration.

