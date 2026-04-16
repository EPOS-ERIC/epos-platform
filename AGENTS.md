# AGENTS

## Scope

This file defines repo-level rules for EPOS platform monorepo.

- Apply to all paths by default.
- If app/service-level `AGENTS.md` exists, it may add stricter rules for that module.
- App/service-level rules must not relax this root baseline.

## Core Philosophy

- Prefer simple, readable code over clever or over-engineered solutions.
- Keep changes minimal, incremental.
- Preserve existing behavior unless explicitly told to change it.
- Follow existing package structure and naming conventions.
- Avoid speculative refactors.
- Do not over-abstract.
- Do not extract every small functionality into new function unless clear reason (for example: reused more than once, meaningfully improves readability, or needs independent testing).
- If similar functionality exists, extend only when it matches same logical responsibility; do not force reuse only to make code look cleaner.
- Prefer early returns to reduce nesting and improve readability.

## Mandatory Rules

Agents must:

- Keep changes scoped, as small as possible.
- Ask for clarification when ambiguity could affect correctness, behavior, or structure.
- Get approval before introducing new dependency (runtime or test).
- Ask for explicit approval before creating or modifying GitHub workflow files.
- Handle errors explicitly, gracefully.
- Use type-safe code by default.

Agents must not:

- Introduce new dependencies without approval.
- Perform broad cross-service refactors unless explicitly requested.
- Create or modify GitHub workflows without explicit approval.
- Add or modify integration/end-to-end tests unless explicitly requested.
- Ignore errors silently.
- Use untyped code unless only practical option and user explicitly approves first.

## Monorepo Structure and Ownership

Canonical top-level layout:

- `apps/` - frontend apps.
- `services/` - backend services.
- `packages/` - shared code and shared repository utilities.
- `cli/` - repository CLI tool.
- `docker/` - repository-level Docker assets (base images, compose files, related files).
- `helm/` - repository-level Helm charts.
- Root files: `AGENTS.md`, `Makefile`, `README.md`, `CHANGELOG.md`.

Conventions:

- App/service directories under `apps/` and `services/` use kebab-case names.
- Do not use a `-service` suffix inside `services/`.
- Service-specific infrastructure stays inside owning service (example: `services/metadata-lifecycle/graphdb/`).
- App/service internals may use module-specific conventions defined in module `AGENTS.md`.

## Required Files Per App/Service

Each app/service is expected to include:

- `AGENTS.md` (module-specific stricter rules).
- `Makefile`.
- `Dockerfile`.
- `README.md`.

`Makefile` target expectations:

- `build`
- `test`
- `lint` (must also run `fmt`)
- `fmt`

## Packages Conventions

- Language-specific shared libraries live under `packages/` (for example `packages/go/`, `packages/ts/`).
- Shared scripts live under `packages/scripts/`.
- Shared assets (logos and similar) live under `packages/assets/`.
- Apps/services use shared assets from `packages/assets/`; must not duplicate those assets inside app/service directories.

## Root Makefile Conventions

- Root `Makefile` provides repository-wide commands (for example local Docker development).
- Root `Makefile` also provides fan-out commands that run common targets across apps/services using consistent target names (`build`, `test`, `lint`).

## Changelog Policy

- Root `CHANGELOG.md` follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).
- When agent makes notable change, update `## Unreleased` with appropriate entry.
- Skip changelog updates for trivial internal noise.

## Migration Note

- Some conventions may not yet be fully implemented across modules.
- For new work, follow these conventions.
- For existing modules, align incrementally within scope of touched changes.
- Do not perform broad structural migrations unless explicitly requested.

## Testing Principles

- Do not over-test trivial behavior unlikely to change.
- Test behavior that meaningfully protects correctness.
- Keep tests deterministic, non-flaky.
- Avoid excessive mocking; if a test requires too much mocking, skip it or redesign for testability.

## Error Handling and Logging

- In backend services, add logging on useful operational and failure paths.
- Use standard logging levels: `info`, `warn`, `error`, `debug`.
- Do not over-log; prefer logs useful when reading running instance.

## Workflow Discovery

- If workflow changes appear necessary, ask for explicit approval before making them.

## Git Hygiene

- Never revert or discard unrelated local changes.
- Avoid destructive git operations (hard reset, force push, history rewrites) unless explicitly requested or approved.
