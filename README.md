# EPOS Platform

Monorepo for the EPOS Platform web application, backend services, shared Go packages, and local development tooling.

## Components

| Component                     | Purpose                                                        | Local URL                |
| ----------------------------- | -------------------------------------------------------------- | ------------------------ |
| `apps/platform`               | Angular web application                                        | `http://localhost:4200`  |
| `services/gateway`            | Public API, authentication, and backend aggregation            | `http://localhost:32000` |
| `services/metadata-lifecycle` | Metadata, users, groups, authorization, and resource lifecycle | `http://localhost:33000` |
| `services/data-retriever`     | Outbound retrieval, payload caching, and cached-text search    | `http://localhost:8100`  |
| `services/oss-aai-service`    | Local web login and OAuth2/OIDC provider                       | `http://localhost:8110`  |

The development stack also includes PostgreSQL databases for Metadata Lifecycle and Data Retriever, plus Grafana's OpenTelemetry LGTM stack for logs, metrics, traces, and profiles.

## Repository layout

```text
apps/                 frontend applications
services/             backend services
packages/go/          shared Go modules
packages/scripts/     development and test scripts
docker/               local Docker Compose configuration
tests/                Hurl and Schemathesis integration tests
cli/                  repository CLI tooling
```

## Local development

### Prerequisites

- Docker with Compose support
- [Tilt](https://docs.tilt.dev/install.html)
- GNU Make

Start the complete development stack from the repository root:

```bash
make up
```

Tilt uses `docker/compose.dev.yaml`, builds development images, and live-updates the application and service containers. For a Kubernetes-backed Tilt environment, select the context explicitly with `make up KUBE_CONTEXT=kind-epos`.

Stop the stack with `make down`. Pass the same `KUBE_CONTEXT` when stopping a Kubernetes-backed environment.

### Local endpoints

- Platform UI: `http://localhost:4200`
- Gateway API: `http://localhost:32000/v1`
- Gateway Scalar UI: `http://localhost:32000/scalar/`
- Gateway Swagger UI: `http://localhost:32000/swagger/`
- Metadata Lifecycle API: `http://localhost:33000/api/v1`
- Data Retriever API: `http://localhost:8100/api/v2`
- OSS AAI: `http://localhost:8110`
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Pyroscope: `http://localhost:4040`

The local PostgreSQL ports are `5433` for Data Retriever and `5434` for Metadata Lifecycle.

### Local authentication and sample data

The development identity provider seeds an administrator and OAuth client on first startup. Get an access token with:

```bash
make get-dev-token
```

The local-only defaults are documented in `services/oss-aai-service/README.md` and must not be used outside development. Populate the running stack with sample metadata using `make populate`.

## Build, test, and lint

The root targets fan out to the frontend and all four services:

```bash
make test
make fmt
make lint
```

Use each module's Makefile for builds, generated code, and module-specific commands. Integration suites run against an already-running development stack:

```bash
make test-hurl
make test-schemathesis
```

`test-schemathesis` obtains its own development token and fixture identifiers. Stateful API flows that require ordered setup are covered by Hurl rather than schema fuzzing.

## Generated code

- Gateway server code and its Metadata Lifecycle client: run `make generate` in `services/gateway`.
- Metadata Lifecycle SQLC code: run `make generate` in `services/metadata-lifecycle`.
- Frontend API types: run `make codegen` in `apps/platform`.

Do not edit generated files directly. Change the owning OpenAPI specification or SQL source and regenerate.

## CI

Workflows under `.github/workflows/` validate each application or service independently and run the Hurl and Schemathesis integration suites. Reusable workflow definitions are prefixed with `_`.

## Component documentation

- [Platform app](apps/platform/README.md)
- [Gateway](services/gateway/README.md)
- [Metadata Lifecycle](services/metadata-lifecycle/README.md)
- [Data Retriever](services/data-retriever/README.md)
- [OSS AAI](services/oss-aai-service/README.md)
