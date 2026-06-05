# AGENTS

- Apply by default to every path.
- If app/service-level `AGENTS.md` exists, may add stricter rules for that module.
- App/service-level rules must not relax this root baseline.
- Module `AGENTS.md` files: focus on exceptions, stricter constraints, or module-specific architecture. Do not repeat root rules unless local behavior differs.

## Core Principles

- Keep changes small, scoped, incremental.
- Preserve existing behavior unless explicitly told to change it.
- Prefer simple, readable code over clever or over-engineered solutions.
- Follow existing package structure, naming, and local patterns.
- Avoid speculative refactors and broad structural cleanup.
- Do not over-abstract.
- Do not extract every small step into new function unless reused, materially improves readability, or needs independent testing.
- If similar functionality exists, extend it only when responsibility matches. Do not force reuse only for visual consistency.
- Prefer early returns to reduce nesting and keep control flow obvious.

## Repo Workflow

- Use `make populate` when local dev data/bootstrap population needed.
- Use root `make test`, `make fmt`, and `make lint` for repo-wide verification across current services.
- Use root `make test-schemathesis` for gateway API integration tests under `tests/schemathesis/`.
- For targeted work inside one service, run that service's local `make build`, `make test`, `make fmt`, and `make lint` from its directory.
- Do not assume root `Makefile` has per-service build targets. Service-specific build and Docker targets live in each module `Makefile`.
- Tilt is default local dev orchestrator for this repo. Prefer existing `Tiltfile` and `docker/compose.dev.yaml` workflow over ad hoc local startup commands.
- If adding or removing service from active local development, keep root `Makefile`, `Tiltfile`, and related dev compose wiring aligned.

## Mandatory Rules

Agents must:

- Keep changes as small as possible.
- Ask for clarification when ambiguity could affect correctness, behavior, or structure.
- Get approval before introducing new dependency, runtime or test.
- Ask for explicit approval before creating or modifying GitHub workflow files, even when adding a new app or service.
- Handle errors explicitly and gracefully.
- Keep code strongly typed by default.

Agents must not:

- Introduce new dependencies without approval.
- Perform broad cross-service refactors unless explicitly requested.
- Create or modify GitHub workflows without explicit approval.
- Add or modify integration or end-to-end tests unless explicitly requested.
- Ignore errors silently.
- Use untyped code unless only practical option and user explicitly approves first.

## Monorepo Structure

Canonical top-level layout:

- `apps/` - frontend apps.
- `services/` - backend services.
- `packages/` - shared code and repository utilities.
- `cli/` - repository CLI tool.
- `docker/` - repository-level Docker assets, including local dev compose.
- `helm/` - repository-level Helm charts.
- Root files include `AGENTS.md`, `Makefile`, `Tiltfile`, `README.md`, `CHANGELOG.md`.

Conventions:

- App/service directories under `apps/` and `services/` use kebab-case names.
- Do not use `-service` suffix in directory names under `services/`.
- Service-specific infrastructure should stay inside owning service when it belongs only to that service.
- Shared code should live under `packages/` instead of being duplicated across services.
- App/service internals may define stricter module conventions in local `AGENTS.md`.

## App/Service Expectations

Each app/service should include:

- `AGENTS.md`
- `Makefile`
- `Dockerfile`
- `README.md`

Standard `Makefile` targets per app/service:

- `build`
- `test`
- `fmt`
- `lint` and it must run `fmt` first

If module behavior differs meaningfully from this standard, document only that difference in module `AGENTS.md`.

## CI/CD And Deployment Manifests

When adding a new app or service under `apps/` or `services/`, also check whether repository automation and deployment assets need to be created or updated.

For each new app/service, consider:

- GitHub Actions workflow under `.github/workflows/`.
- Helm chart or chart values under `helm/`.
- Root `Makefile` targets if repo-wide commands need to include the new module.
- `Tiltfile` and `docker/compose.dev.yaml` if the module participates in local development.
- Module-level `README.md` and `AGENTS.md`.

GitHub workflow files must not be created or modified without explicit user approval.

Before creating or updating a GitHub workflow, ask the user for approval and clarify:

- Workflow name.
- Trigger policy, such as pull request, push to main, tag, or manual dispatch.
- Required jobs, such as build, test, lint, Docker image build, Helm lint, or deploy.
- Target environments, such as dev, staging, or production.
- Required secrets or repository variables.

When adding or changing Helm charts:

- Prefer extending the existing chart structure under `helm/`.
- Keep chart names aligned with app/service directory names.
- Add or update values for image repository, tag, ports, environment variables, probes, resources, ingress, and service account settings as needed.
- Avoid duplicating chart templates if an existing shared pattern can be extended cleanly.
- Run Helm validation where practical, such as `helm lint`, and document if validation could not be run.
- Keep local development wiring, Helm values, and service ports consistent.

## Packages Conventions

- Language-specific shared libraries live under `packages/` such as `packages/go/`.
- Shared scripts live under `packages/scripts/`.
- Shared assets live under `packages/assets/`.
- Apps/services should use shared assets from `packages/assets/` instead of duplicating them locally.

## Testing Principles

- Test behavior that meaningfully protects correctness.
- Do not over-test trivial behavior unlikely to change.
- Keep tests deterministic and non-flaky.
- Avoid real sleeps, randomness, and unnecessary network dependency in tests.
- Avoid excessive mocking. If test setup becomes mock-heavy, simplify design or skip low-value coverage.

## Error Handling and Logging

- Handle returned errors explicitly.
- In backend services, log useful operational and failure paths.
- Use standard logging levels: `debug`, `info`, `warn`, `error`.
- Do not over-log. Prefer logs that help operators understand boundaries, failures, and major state changes.

## Changelog Policy

- Root `CHANGELOG.md` follows [Keep a Changelog 1.1.0](https://keepachangelog.com/en/1.1.0/).
- When change is notable to repo users or developers, update `## Unreleased` with appropriate entry.
- Skip changelog updates for trivial internal noise.

## AGENTS Maintenance

- Keep `AGENTS.md` files aligned with current repo reality.
- If change updates repo-wide workflow, structure, conventions, standard targets, or shared patterns and makes current `AGENTS.md` guidance inaccurate, update relevant `AGENTS.md` file in same work.
- Update root `AGENTS.md` for repo-wide changes. Update module `AGENTS.md` only for module-specific changes or exceptions.
- Do not leave `AGENTS.md` guidance knowingly stale after changing source-of-truth files such as root `Makefile`, `Tiltfile`, `docker/compose.dev.yaml`, shared package conventions, or standard module layout.

## Git Hygiene

- Never revert or discard unrelated local changes.
- Avoid destructive git operations such as hard reset, force push, or history rewrites unless explicitly requested or approved.
