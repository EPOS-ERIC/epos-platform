# EPOS Metadata Lifecycle Service

Kotlin service for managing RDF metadata lifecycle in GraphDB, with SHACL-driven resource typing, provenance tracking, versioning, search, and admin population tools.

## What this application does

- Manages lifecycle states for metadata resources: `draft -> submitted -> published -> archived`
- Stores data in dedicated named graphs (conceptual/data/provenance)
- Tracks provenance with PROV-O during create/update/submit/publish operations
- Parses SHACL shapes to expose resource types and constraints (`/api/v2/resource-types`)
- Supports UI-oriented SHACL output, including aggregated constraints and resolvable shape references
- Provides search/discovery, SPARQL passthrough, OAI-PMH harvesting, and admin population endpoints

## Main entrypoint and runtime modes

The executable entrypoint is `org.epos.ApplicationKt`.

Run modes:

- `server` (default): start API server
- `populate`: run initial population only
- `populate-and-serve`: populate then start server
- `--help`: print usage

Examples:

```bash
# API server
java -jar build/libs/metadata-service-1.0-SNAPSHOT.jar server

# Initial population only
java -jar build/libs/metadata-service-1.0-SNAPSHOT.jar populate

# Populate and then start API
java -jar build/libs/metadata-service-1.0-SNAPSHOT.jar populate-and-serve
```

## Prerequisites

- JDK 17
- Gradle 9.0.0 via the included wrapper (`./gradlew`)
- GraphDB (tested with 10.x)
- GraphDB repository (default: `epos`)

## Build and local run commands

Run commands from `services/metadata-lifecycle/`.

Preferred (repository standard):

```bash
make build
make test
make fmt
make lint
```

Equivalent Gradle commands:

```bash
# Build fat jar
./gradlew clean shadowJar

# Run tests
./gradlew test

# Format Kotlin and Gradle Kotlin scripts
./gradlew spotlessApply

# Check formatting
./gradlew spotlessCheck

# Run app via Gradle (server mode)
./gradlew run

# Run app via Gradle in populate mode
./gradlew run --args="populate"
```

## Configuration

Configuration is profile-based and loaded from YAML files under `src/main/resources/config/`:

- `application.yaml` (base)
- `application-dev.yaml`
- `application-test.yaml`
- `application-prod.yaml`

Override order (high -> low):

1. Environment variables
2. Profile file
3. Base file
4. Code defaults

Select profile with:

```bash
EPOS_PROFILE=dev java -jar build/libs/metadata-service-1.0-SNAPSHOT.jar
```

### Important environment variables

- `EPOS_GRAPHDB_ENDPOINT` (default `http://localhost:7200`)
- `EPOS_REPOSITORY_ID` (default `epos`)
- `EPOS_INDEX_URL` (TTL index URL for population)
- `EPOS_BASE_URI` (default `https://epos-eu.org`)
- `EPOS_SERVER_HOST` / `HOST` (default `0.0.0.0`)
- `EPOS_SERVER_PORT` / `PORT` (default `8080`)
- `EPOS_PROFILE` (default `default`)

## Named graph model

By default (resolved from `baseUri`):

- `/graphs/conceptual`
- `/graphs/data/draft`
- `/graphs/data/submitted`
- `/graphs/data/published`
- `/graphs/data/archived`
- `/graphs/provenance`

## API documentation

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI YAML: `http://localhost:8080/openapi.yaml`
- Health: `http://localhost:8080/health`

## API groups (v2)

- System
  - `GET /health`
  - `GET /api/v2/resource-types`
- Resources
  - `GET|POST /api/v2/{resourceType}`
  - `GET|PUT|DELETE /api/v2/{resourceType}/{id}`
- Lifecycle
  - `POST /api/v2/{resourceType}/{id}/submit`
  - `POST /api/v2/{resourceType}/{id}/publish`
  - `POST /api/v2/{resourceType}/{id}/draft`
  - `GET /api/v2/{resourceType}/{id}/history`
  - `POST /api/v2/{resourceType}/{id}/repair`
- Batch
  - `POST /api/v2/batch/submit`
  - `POST /api/v2/batch/publish`
- Search and details
  - `GET /api/v2/search`
  - `GET /api/v2/details/{id}`
- SPARQL
  - `POST /api/v2/sparql`
- OAI-PMH
  - `GET /oai-pmh`
- Admin population
  - `POST /api/v2/admin/populate`
  - `POST /api/v2/admin/populate/from-url`
  - `POST /api/v2/admin/populate/file`
  - `POST /api/v2/admin/populate/content`
  - `GET /api/v2/admin/populate/status`

## Resource types endpoint for UI editors

`GET /api/v2/resource-types` returns SHACL-derived resource definitions.

Useful query parameters:

- `includeShapeDefinitions=true`
  - include a `shapeDefinitions` map keyed by shape IRI
- `includeTransitiveShapeDefinitions=true`
  - include only shape definitions reachable from resource properties via `sh:node`
- `includeResolvedRefs=true`
  - add cycle-safe recursive references per shape (`resolvedRefs`)

### Example

```bash
curl "http://localhost:8080/api/v2/resource-types?includeTransitiveShapeDefinitions=true&includeResolvedRefs=true"
```

Notes:

- Properties are aggregated by path (no duplicated `issued` lines with split min/max constraints)
- Scalar/list duplication is avoided:
  - single value -> `datatype` / `classConstraint` / `severity`
  - multiple values -> `datatypes` / `classConstraints` / `severities`
- `nodeShapeRef` can be resolved directly in `shapeDefinitions[<IRI>]`
- `shapeDefinitions` now also exposes node-level constraints for node-only shapes (e.g. `DateOrDateTimeDataType`), including `datatype(s)`, `classConstraint(s)` and `nodeShapeRef(s)`
- OWL schema definitions are parsed and used as fallback for richer constraints (e.g. `owl:ObjectProperty` with `rdfs:range` / `owl:unionOf` enriches `classConstraint(s)` when SHACL property constraints are implicit)

## Example curl commands

```bash
# Search
curl "http://localhost:8080/api/v2/search?q=seismic&page=1&pageSize=20"

# Get details as JSON
curl "http://localhost:8080/api/v2/details/https%3A%2F%2Fepos-eu.org%2Fid%2Fdataset%2F123"

# Get details as Turtle
curl "http://localhost:8080/api/v2/details/https%3A%2F%2Fepos-eu.org%2Fid%2Fdataset%2F123?format=turtle"

# Populate from custom index URL
curl -X POST "http://localhost:8080/api/v2/admin/populate/from-url" \
  -H "Content-Type: application/json" \
  -d '{"indexUrl":"https://example.org/index.txt"}'
```

## Docker

### Repository compose stack

`docker/docker-compose.yaml` at repository root defines GraphDB, initialization, and this service.

```bash
docker compose -f docker/docker-compose.yaml up -d
docker compose -f docker/docker-compose.yaml logs -f graphdb
docker compose -f docker/docker-compose.yaml down
```

### Build API image

```bash
docker build -t epos-metadata-lifecycle:latest .
```

### Run API container

```bash
docker run --rm -p 8080:8080 \
  -e EPOS_GRAPHDB_ENDPOINT=http://host.docker.internal:7200 \
  -e EPOS_REPOSITORY_ID=epos \
  -e EPOS_PROFILE=prod \
  epos-metadata-lifecycle:latest
```

### Run population mode in container

```bash
docker run --rm \
  -e EPOS_GRAPHDB_ENDPOINT=http://host.docker.internal:7200 \
  -e EPOS_REPOSITORY_ID=epos \
  -e EPOS_INDEX_URL=https://example.org/index.txt \
  epos-metadata-lifecycle:latest populate
```

## Troubleshooting

- GraphDB unreachable:
  - verify `EPOS_GRAPHDB_ENDPOINT`
  - verify repository exists (`epos` by default)
- Empty or failing resource-types:
  - check SHACL URL connectivity (default EPOS SHACL fetched on startup)
- Test failures in local/CI without Docker:
  - integration tests rely on Testcontainers
  - run focused unit tests when Docker is unavailable

## Project layout (high level)

```text
src/main/kotlin/org/epos/
  Application.kt
  api/
    controllers/
    models/
    shacl/
  graphdb/
    client/
    services/
    utils/
  infrastructure/ktor/
    plugins/
    routes/
  services/
```

## License

GPLv3
