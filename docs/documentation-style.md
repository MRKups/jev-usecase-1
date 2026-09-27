---
title: Documentation style
normative-for: []
depends: []
status: guide, not a contract. Where it conflicts with being correct, be correct.
---

# Documentation style

How documentation in this repository should read. These guidelines keep technical writing concise, accurate, and maintainable.

## What to document

Documentation is a guide to the system, not the source code restated in English.

Document the features, capabilities, and subsystems: what each one is for, what goes in, what comes out, and what outcome it is meant to produce. Document the guarantees and, just as importantly, where they stop.

The test for any sentence: **would someone building a compatible client or pipeline in another language need this?** If yes, it is contract. If it only helps someone editing our Go, the source already says it better and will stay correct longer.

| Document this | Leave this to the source |
|---|---|
| What the subsystem is for | Type, function, and internal field names |
| Inputs, outputs, intended outcome | Exact error message strings |
| Guarantees, and where they stop | Order of private checks inside a function |
| Wire vocabulary: ticket schema, triage questions, provider errors | Mutex mechanics and internal buffers |
| Failure and degradation behaviour | Line references into source files |
| Where one subsystem's authority ends and another begins | Helper utility functions |

Source code is more authoritative than any document here. When the two disagree, the source is right and the document is a bug.

Prefer the shorter document. Cutting a sentence that restates its neighbour is always correct.

## Voice

These rules apply to everything a person reads: CLI help, README, error messages, and specifications under `docs/`.

**No em-dashes in prose.** Mid-sentence they read as machine-written. Use a comma, a colon, a period, or a new sentence.

Em-dashes are acceptable only where they are structural rather than grammatical: separating a heading from its subject or a label from its description in a table.

**Corporate-professional, friendly, down to earth.** Write like an experienced engineer explaining a system to a colleague who has to operate it.

**No AI-generated tone.** If it reads like a generic chatbot wrote it, rewrite it. Short, direct sentences that lead with the point.

**No defensive language.** State what this system does, not what other systems fail to do.

**No hedging where the answer is settled.** If the implementation decides it, state it plainly. If behaviour is intentionally uncommitted, explain why.

Avoid filler and buzzwords: "it is important to note", "simply", "essentially", "in order to", "leverage", "robust", "powerful", "seamless", "comprehensive", "it's worth mentioning".

## Audience

Two tiers, with different rules about jargon.

**User-facing.** CLI help text, error messages, and README. Written for someone running the application or testing the pipeline. Concepts are explained clearly without requiring familiarity with internal packages.

**Internal technical.** The specifications under `docs/`, architecture notes, and code comments. Precise technical language is correct here. Brevity and correctness come first.

## Errors

Every error tells the reader what happened and what to do next. "Failed to triage ticket" is not enough on its own. State the cause (e.g. missing API key, timeout, malformed payload) and remediation step.

## One home per concept

A concept has exactly one normative home. Everywhere else cross-references it. A summary provided for convenience is a second home that will drift, so do not add one.
