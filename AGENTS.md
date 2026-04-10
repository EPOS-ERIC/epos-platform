# AGENTS

## Scope

This file defines repository-level rules for the EPOS platform monorepo.

- Applies to all paths by default.
- If a service-level `AGENTS.md` exists, it may add stricter rules for that service.
- Service-level rules must not relax this root baseline.

## Core Philosophy

- Prefer simple, readable code over clever or over-engineered solutions.
- Keep changes minimal and incremental.
- Preserve existing behavior unless explicitly instructed to change it.
- Follow existing package structure and naming conventions.
- Avoid speculative refactors.
- Do not over-abstract.
- Do not extract every small functionality into a new function unless there is a clear reason (for example: reused more than once, meaningfully improves readability, or needs independent testing).
- If similar functionality already exists, extend it only when it matches the same logical responsibility; do not force reuse purely to make code look cleaner.
- Prefer early returns to reduce nesting and improve readability.

## Mandatory Rules

Agents must:

- Keep changes scoped and as small as possible.
- Ask for clarification when ambiguity could affect correctness, behavior, or structure.
- Get approval before introducing any new dependency (runtime or test).
- Ask for explicit approval before creating or modifying GitHub workflow files.
- Handle errors explicitly and gracefully.
- Use type-safe code by default.

Agents must not:

- Introduce new dependencies without approval.
- Perform broad cross-service refactors unless explicitly requested.
- Create or modify GitHub workflows without explicit approval.
- Add or modify integration/end-to-end tests unless explicitly requested.
- Ignore errors silently.
- Use untyped code unless it is the only practical option and the user explicitly approves it first.

## Testing Principles

- Do not over-test trivial behavior that is unlikely to change.
- Test behavior that meaningfully protects correctness.
- Keep tests deterministic and non-flaky.
- Avoid excessive mocking; if a test requires too much mocking, skip it or redesign for testability.

## Error Handling and Logging

- In backend services, add logging on useful operational and failure paths.
- Use standard logging levels: `info`, `warn`, `error`, `debug`.
- Do not over-log; prefer logs that are useful when reading a running instance.

## Monorepo Structure

Current top-level components:

- `metadata-lifecycle-service/` - service
- `orchestrator-service/` - service
- `graphdb/` - infrastructure component
- `docker-compose.yaml` - local stack orchestration

Conventions for new services:

- Use `<name>-service/` naming.
- Include a service-local `README.md`.

## Root Commands

- `make up` - starts the local Docker stack
- `make down` - stops the local Docker stack

## Workflow Discovery

- If workflow changes appear necessary, ask for explicit approval before making them.

## Git Hygiene

- Never revert or discard unrelated local changes.
- Avoid destructive git operations (hard reset, force push, history rewrites) unless explicitly requested or approved.

## Uncertainty Handling

If uncertain and the decision could affect behavior, correctness, or architecture:

1. State what is unclear.
2. Explain why it matters.
3. Present concrete options.
4. Wait for clarification before making functional changes.
