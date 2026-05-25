# AGENTS

## Scope

These rules apply to `services/user-group/` and add constraints on top of the repository root `AGENTS.md`.

## Build and Test Workflow

- Use module Makefile targets from this directory.
- `make build` builds the service binary.
- `make test` runs unit tests.
- `make fmt` formats Go sources.
- `make lint` must run `make fmt` first and then static checks.

## Service Conventions

- Keep business logic in domain-oriented packages.
- Keep transport layer thin and validation explicit.
- Keep authorization decisions deterministic and explainable.
