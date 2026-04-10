# AGENTS

## Scope

These rules apply to `services/metadata-lifecycle/` and add constraints on top of the repository root `AGENTS.md`.

## Build and Test Workflow

- Use module Makefile targets from this directory.
- `make build` builds the runnable shadow jar.
- `make test` runs unit and integration tests.
- `make fmt` formats Kotlin and Gradle Kotlin scripts.
- `make lint` must run `make fmt` first and then style checks.

## Service Conventions
