# EPOS Gateway

Public HTTP API for the EPOS Platform. The gateway exposes a stable `/v1` contract, validates bearer tokens through OSS AAI, and maps requests to the Metadata Lifecycle service. It also owns public error responses and API documentation.

## API and documentation

- API base path: `/v1`
- Health check: `GET /health`
- OpenAPI document: `GET /openapi.yaml`
- Scalar UI: `GET /scalar/`
- Swagger UI: `GET /swagger/`
- Source contract: `docs/openapi.yaml`

The OpenAPI contract is the source of truth for operations, request bodies, authentication requirements, and response schemas.

## Configuration

The gateway requires a YAML configuration file. Pass it with `-config`; the default path is `/etc/gateway/config.yaml`.

```bash
./gateway -config config/example.yaml
```

The configuration controls the server listen address and timeouts, CORS, upstream caching and timeouts, and the Metadata Lifecycle and authentication service URLs. See `config/example.yaml` for the complete schema. `docker/gateway-config.dev.yaml` contains the Tilt stack values.

`LOG_LEVEL` optionally sets shared structured logging to `DEBUG`, `INFO`, `WARN`, or `ERROR`. Standard OpenTelemetry SDK environment variables configure telemetry export.

## Build and checks

Run commands from `services/gateway`:

```bash
make build
make test
make fmt
make lint
```

`make test` enables the race detector. Build the container with `make docker`; set `IMAGE` to override the default tag.

## Generated code

```bash
make generate
```

This regenerates:

- `gen/gatewayapi` from `docs/openapi.yaml` using ogen;
- `gen/metadata-lifecycle` from `../metadata-lifecycle/docs/openapi.yaml` using OpenAPI Generator.

Do not edit either generated directory manually. Change the owning OpenAPI contract and regenerate.

## Upstream caching

Gateway clients share the cached outbound HTTP implementation in `upstream/`. Its timeout, stale-response, in-flight deduplication, and cache invalidation behavior is documented in [upstream/README.md](upstream/README.md).
