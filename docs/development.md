---
title: Development rules
normative-for: [development invariants, architectural rules, code review standards]
depends: [documentation-style.md]
status: current reference. Immutable rule IDs DEV-01 through DEV-18.
---

# Development rules

These rules apply across `jev-usecase-1`. The priorities are simple code, clear boundaries, reliable behavior, and disciplined Go conventions. Project-specific boundaries, safety, and Git rules remain in [AGENTS.md](../AGENTS.md). The [documentation style guide](documentation-style.md) explains how focused subsystem documents capture contracts without restating code.

Rule IDs are permanent identifiers, not positions. Moving a rule does not change its ID; retired IDs remain reserved. New rules receive new IDs.

---

## Simple

### DEV-01: Solve the current problem
Make the smallest complete change. Avoid speculative features, premature abstractions, configurable plugin registries, and unneeded extension points. Write code for the capabilities the application needs today.

### DEV-02: Start concrete
Add an interface or abstraction only when it simplifies existing code, enables required test doubles, or enforces a clear subsystem boundary (such as the LLM provider or Jev model ports). Do not create 1-to-1 interface wrappers for internal helpers.

### DEV-03: Standard library first and justify dependencies
Prefer Go's standard library (`net/http`, `html/template`, `encoding/json`, `encoding/csv`, `sync`, `context`, `time`, `os`) where suitable. External dependencies require explicit justification and must maintain `CGO_ENABLED=0` compatibility.

### DEV-04: Keep performance work proportionate
Avoid unnecessary memory allocations in streaming and generation paths. Measure latency and allocations with benchmarks before introducing complexity for speed.

---

## Modular

### DEV-05: Strict unidirectional package dependency flow
Dependencies strictly flow downward toward domain primitives:
```text
cmd/ticket-eval -> internal/cli
internal/cli -> internal/config, internal/generator, internal/jev, internal/triage, internal/webapp
internal/webapp -> internal/generator, internal/jev, internal/ticket, internal/triage
internal/generator -> internal/provider, internal/ticket
internal/jev -> internal/ticket, internal/triage
internal/provider -> internal/ticket
internal/triage -> (leaf or ticket only)
internal/ticket -> (pure domain leaf, standard library only)
```
No package may introduce an upward or circular dependency. `internal/ticket` remains a pure domain model package free of networking, HTTP, and provider concerns.

### DEV-06: Small consumer-owned interfaces
Keep Go interfaces focused and narrow (1 to 2 methods):
- `provider.LLMProvider`: `GenerateTicket(ctx context.Context, prompt string) (*ticket.Ticket, error)`
- `jev.Client`: `Classify(ctx context.Context, t *ticket.Ticket, questions []triage.Question) (*ticket.TriageResult, error)`

Decouple policy from external backends so core logic can be tested independently of external network calls.

### DEV-07: Explicit ownership and concurrency safety
Give all shared state and resources an explicit owner:
- Shared in-memory data structures must be guarded by appropriate synchronization (`sync.RWMutex`).
- Never launch an unowned goroutine. Every goroutine must have an explicit owner, a termination condition, and a cleanup path.
- Channels must have an unambiguous sender that owns closing them. Receivers never close channels.

### DEV-08: Isolate external provider details
Keep vendor-specific REST payloads, HTTP headers, authentication tokens, and provider errors behind adapter boundaries (`internal/provider`, `internal/jev`). Domain objects and the web presentation layer must never import vendor SDKs or handle raw HTTP transport payloads.

---

## Reliable

### DEV-09: Propagate and respect context.Context
Every blocking, network, or background operation accepts `context.Context` as its first parameter. Operations must respect cancellation and timeouts immediately. A cancelled client request must stop all downstream provider generation or Jev evaluation work.

### DEV-10: Preserve error types and context
Use explicit error returns wrapped with `%w` for expected failures. Never panic in library or HTTP handler code. Never discard errors with `_ = err` on fallible operations; log or return them with actionable context and remediation guidance.

### DEV-11: Immutability of domain tickets
Once synthesized, an ITSM ticket is an immutable value object. Triage evaluation attaches a separate `TriageResult` record without mutating original ticket attributes (reporter, summary, description, reported urgency).

### DEV-12: Zero CGO and portable compilation
All packages and binaries must compile with `CGO_ENABLED=0` for clean cross-compilation across macOS (ARM64) and Linux (AMD64).

### DEV-13: Test observable behavior
Cover observable contracts, serialization roundtrips, and plausible failure paths. Core tests must execute host-safe and in milliseconds using in-memory test doubles and standard library HTTP test servers.

---

## Working Process

### DEV-14: Understand before changing
Read the owning specification under `docs/` and caller contracts before modifying code or architecture. Before a change with meaningful scope or risk, assess its effects on boundaries, concurrency, and contracts.

### DEV-15: Verify the change
Exercise the affected CLI, HTTP endpoint, or generator path for each slice. Run `task test` and verify that both test doubles and production paths satisfy requirements. State any verification that could not run.

### DEV-16: Do not hide failures
Fix root causes rather than weakening assertions, skipping tests, or silencing compiler/linter warnings to get green output.

### DEV-17: Maintain documentation with implementation
Update affected documentation under `docs/`, `README.md`, or `AGENTS.md` in the same commit as the code change that alters contracts. Planned and implemented behavior must stay clearly separated.

### DEV-18: Discrete atomic commits
Maintain one complete, reviewable intent per commit with explicit file staging. Include implementation, tests, and owning documentation together.
