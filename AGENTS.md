# Repository Guidance

## Scope And Delegation

- These rules apply throughout the repository.
- A module `AGENTS.md` may add stricter, module-specific rules but must not relax this guidance.
- Use module `AGENTS.md` for module architecture and workflows. Load the matching available skill before specialized work such as Go, SQLC, frontend, documentation, or presentation tasks.
- Do not copy skill guidance into `AGENTS.md`; keep each source of truth focused.

## Repository Structure

- `apps/` contains frontend applications.
- `services/` contains backend services.
- `packages/` contains shared libraries, scripts, and assets.
- `cli/` contains repository CLI tooling.
- `docker/` contains repository-level local development assets.
- `helm/` contains deployment charts and values.
- Use kebab-case for app and service directories. Do not add a `-service` suffix to service directory names.
- Keep service-specific code and infrastructure in its owning module. Put code shared across modules in `packages/`.

## Change Rules

- Keep changes small, scoped, and incremental. Preserve existing behavior unless the requested change requires otherwise.
- Extend existing code when responsibility matches. Avoid speculative refactors and broad cross-module changes.
- Handle errors explicitly. Do not ignore failures or hide them behind untyped workarounds.
- Get approval before adding runtime, build, or test dependencies.
- Never commit, discard, or rewrite work without explicit user approval. Do not alter unrelated changes.
- Never expose credentials, tokens, private keys, or sensitive local data in source, logs, commits, or responses.

## Workflow And Integration

- Use module-local Makefiles and module guidance for build, format, lint, and test commands.
- Use root Makefile targets for cross-repository operations; inspect available targets instead of assuming root and module commands are interchangeable.
- When adding or changing an app or service, inspect affected root Makefile, `Tiltfile`, Docker Compose, Helm, workflow, documentation, and shared-asset wiring. Report required follow-up changes.
- Before modifying a GitHub workflow, inspect related workflows and request explicit approval. Do not edit workflow files without that approval.

## Local Development

- Treat `Tiltfile` and its referenced Docker Compose configuration as the source of truth for the local development stack.
- Expect the user to run `tilt up` with the development stack deployed. Code changes should be reflected in the running environment through the existing development workflow.
- Agents may inspect container logs, execute commands in local containers, and read or edit local development database tables when required for the task.
- Local development data and containers are volatile and must not be treated as persistent or production data.
- Use `make get-dev-token` to get admin token to use the local tilt environment.
- Ask the user before cleaning or deleting the local environment. Cleanup requires stopping Tilt, running the appropriate clean operation, and restarting Tilt; do not perform those steps autonomously.

## Observability

- Use the configured Grafana MCP when observability evidence is needed. Inspect logs, traces, spans, metrics, dashboards, alerts, and related telemetry before guessing at runtime behavior.
- Treat Grafana data from the local development stack as local diagnostic data, consistent with the volatile environment rules above.
- Use these local datasource UIDs:
  - `prometheus` (`Prometheus`) for metrics; default datasource.
  - `loki` (`Loki`) for logs.
  - `tempo` (`Tempo`) for traces and spans.
  - `pyroscope` (`Pyroscope`) for profiles.
- Discover metric names, labels, and log labels before running targeted queries. Use precise time ranges and service labels, and correlate logs, traces, and metrics when identifiers allow it.
- Treat datasource availability and provisioning as runtime configuration. Verify them through Grafana if a documented datasource is unavailable.

## Documentation And Guidance

- Keep user-facing and developer-facing documentation aligned with behavior and workflows. Update the relevant README, API specification, changelog, or other source documentation when needed.
- Keep affected `AGENTS.md` files aligned with current structure, workflows, and rules. Remove guidance duplicated by an available skill.
