# EPOS Metadata Lifecycle Service

Kotlin service for managing metadata lifecycle in JanusGraph (Property Graph), with SHACL-driven resource typing, provenance tracking, versioning, search, and admin population tools.

## What this application does

- Manages lifecycle states for metadata resources: `draft -> submitted -> published -> archived`, with explicit `discarded` state for cancelled drafts/submissions
- Stores data as Property Graph nodes with embedded entity data and explicit relationships
- Tracks provenance with Activity nodes connected to Version nodes
- Parses SHACL shapes to expose resource types and constraints (`/api/v2/resource-types`)
- Provides search/discovery, Cypher passthrough, OAI-PMH harvesting, and admin population endpoints

## JanusGraph Property Graph Data Model

### Node Types

- `Conceptual` - stable identity for a resource (e.g., `https://epos-eu.org/id/dataset/uuid`)
- `Version` - a specific version with all entity data embedded in a `data` property
- `Activity` - provenance records for state transitions

### Relationships

- `(Conceptual)-[:HAS_VERSION]->(Version)` - all versions
- `(Conceptual)-[:CURRENT_VERSION]->(Version)` - points to current published
- `(Version)-[:REVISION_OF]->(Version)` - draft lineage
- `(Version)-[:SUPERSEDED_BY]->(Version)` - published replacement
- `(Activity)-[:AFFECTED]->(Version)` - provenance link

### Example

```
  (:Conceptual {id: "dataset/abc"})
       |
       +--[:CURRENT_VERSION]--> (:Version {id: "v2.0", status: "published", data: {...}})
       |
       +--[:HAS_VERSION]------> (:Version {id: "v1.0", status: "archived", data: {...}})
       |
       +--[:HAS_VERSION]------> (:Version {id: "v2.1-draft", status: "draft", data: {...}})
```

## Main entrypoint and runtime modes

The executable entrypoint is `org.epos.ApplicationKt`.

Run modes:

- `server` (default): start API server
- `populate`: run initial population only
- `populate-and-serve`: populate then start server
- `--help`: print usage

## Prerequisites

- JDK 21
- Gradle 9.0.0 via the included wrapper (`./gradlew`)
- JanusGraph + Cassandra + Elasticsearch

## Build and local run commands

Run commands from `services/metadata-lifecycle/`.

```bash
make build
make build-docker
make test
make fmt
make lint
```

Start JanusGraph stack for development:

```bash
docker compose -f docker-compose.janusgraph.yml up -d
```

## Configuration

Configuration is profile-based and loaded from YAML files under `src/main/resources/config/`:

- `application.yaml` (base)
- `application-dev.yaml`
- `application-test.yaml`
- `application-prod.yaml`

### Important environment variables

- `EPOS_JANUSGRAPH_HOST` (default `localhost`)
- `EPOS_JANUSGRAPH_PORT` (default `8182`)
- `EPOS_JANUSGRAPH_TRAVERSAL_SOURCE` (default `g`)
- `EPOS_JANUSGRAPH_USERNAME` (default empty)
- `EPOS_JANUSGRAPH_PASSWORD` (default empty)
- `EPOS_GRAPH_BOOTSTRAP_MODE` (set `skip-cypher` when JanusGraph has no Cypher plugin)
- `EPOS_INDEX_URL` (TTL index URL for population)
- `EPOS_BASE_URI` (default `https://epos-eu.org`)
- `PORT` (default `8080`)
- `HOST` (default `0.0.0.0`)
- `EPOS_PROFILE` (default `default`)

## API documentation

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI runtime YAML: `http://localhost:8080/openapi.runtime.yaml`
- Health: `http://localhost:8080/health`
- JanusGraph Visualizer UI: `http://localhost:3001`

### Required mutation headers

Mutating operations require both headers with absolute URIs:

- `X-Agent`: acting agent URI
- `X-Group`: acting group URI

## API groups (v2)

- System: `GET /health`, `GET /api/v2/resource-types`
- Resources: `GET|POST /api/v2/{resourceType}`, `GET|PUT|PATCH|DELETE /api/v2/{resourceType}/{id}`
- Lifecycle: `POST .../submit`, `POST .../publish`, `POST .../draft`, `POST .../discard`, `GET .../history`, `POST .../repair`
- Batch: `POST /api/v2/batch/submit`, `POST /api/v2/batch/publish`
- Search: `GET /api/v2/access/search`, `GET /api/v2/access/search/{shape}`
- Details: `GET /api/v2/dataset/{id}`, `GET /api/v2/distribution/{id}`, `GET /api/v2/software-application/{id}`, `GET /api/v2/category/{id}`
- Cypher: `POST /api/v2/cypher` (read-only)
- OAI-PMH: `GET /oai-pmh`
- Admin: `POST /api/v2/admin/populate`, `POST /api/v2/admin/import/revision/*`, `POST /api/v2/admin/relations/reconcile`

## Lifecycle and mutation behavior

- `{id}` in lifecycle/resource mutation endpoints accepts conceptual id, version id, or identifier URI.
- `PUT`, `PATCH`, `DELETE` operate on `draft` versions only.
- `submit` accepts `draft` only; `publish` accepts `submitted` only.
- `discard` accepts `draft` or `submitted` and moves to `discarded`.
- Invalid state transitions return `409 Conflict`.

## Populate endpoint behavior

- `POST /api/v2/admin/populate` with no body uses configured `EPOS_INDEX_URL`.
- `POST /api/v2/admin/populate` with body supports exactly one of:
  - `{"indexUrl": "http(s)://.../index.txt"}` for full index-based population
  - `{"url": "http(s)://.../file.ttl"}` for single TTL source
  - `{"path": "/absolute/path/file.ttl"}` for local TTL file

## Docker

### Start JanusGraph stack for development

```bash
docker compose -f docker-compose.janusgraph.yml up -d
```

### Build API image

```bash
make build-docker
```

### Run API container

```bash
docker run --rm -p 8080:8080 \
  -e EPOS_JANUSGRAPH_HOST=host.docker.internal \
  -e EPOS_JANUSGRAPH_PORT=8182 \
  -e EPOS_JANUSGRAPH_TRAVERSAL_SOURCE=g \
  epos-metadata-lifecycle:latest
```

## Project layout

```text
src/main/kotlin/org/epos/
  Application.kt
  api/
    controllers/
    models/
    shacl/
  propertygraph/
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
