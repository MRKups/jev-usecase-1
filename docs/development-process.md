---
title: Development process
normative-for: [implementation loop, proof workflow]
depends: [overview.md, architecture.md, development.md]
status: working guidance
---

# Development Process

This document defines the implementation and verification loop for all work in `jev-usecase-1`. `AGENTS.md` is the contributor contract; focused documents own behavior, and `docs/development.md` defines the numbered engineering rules (DEV-01 through DEV-18).

## The Implementation Loop

1. **Bound the outcome.**
   Read the focused subsystem document and relevant ADRs in [decisions.md](decisions.md). Define the smallest observable change and the evidence that will prove it. Record any new architectural decision in [decisions.md](decisions.md).

2. **Change the production path.**
   Follow the package boundary map in [repository-map.md](repository-map.md) and the dependency rules in [development.md](development.md#dev-05-strict-unidirectional-package-dependency-flow). Avoid temporary bypasses, unneeded abstractions, or parallel orchestration paths.

3. **Prove the outcome.**
   Use the narrowest meaningful proof: run unit tests (`task test`) or exercise the CLI command directly (`task generate:json`, `task triage:sample`, `task run:web`). Verify that all implementation paths adhere to the contract.

4. **Record only affected facts.**
   Update owning documentation, test evidence, decisions, or repository mappings only when the change alters that owner's subject. Keep planned and implemented facts separate: do not represent a planned path as implemented.
