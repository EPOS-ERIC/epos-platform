# EPOS Metadata Lifecycle Service

Kotlin service for managing metadata lifecycle in Neo4j Community (Property Graph), with SHACL-driven resource typing, provenance tracking, versioning, search, and admin population tools.

## What this application does

- Manages lifecycle states for metadata resources: `draft -> submitted -> published -> archived`, with explicit `discarded` state for cancelled drafts/submissions
- Stores data as Property Graph nodes with embedded entity data and explicit relationships
- Tracks provenance with Activity nodes connected to Version nodes
- Parses SHACL shapes to expose resource types and constraints (`/api/v2/resource-types`)
- Provides search/discovery, Cypher passthrough, OAI-PMH harvesting, and admin population endpoints

## Neo4j Property Graph Data Model

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
- Neo4j Community

## Build and local run commands

Run commands from `services/metadata-lifecycle/`.

```bash
make build
make build-docker
make test
make fmt
make lint
```

Start Neo4j stack for development:

```bash
docker compose -f docker-compose.neo4j.yml up -d
```

## Configuration

Configuration is profile-based and loaded from YAML files under `src/main/resources/config/`:

- `application.yaml` (base)
- `application-dev.yaml`
- `application-test.yaml`
- `application-prod.yaml`

### Important environment variables

- `EPOS_NEO4J_URI` (default `bolt://localhost:7687`)
- `EPOS_NEO4J_USERNAME` (default empty)
- `EPOS_NEO4J_PASSWORD` (default empty)
- `EPOS_NEO4J_DATABASE` (default `neo4j`)
- `EPOS_INDEX_URL` (TTL index URL for population)
- `EPOS_BASE_URI` (default `https://epos-eu.org`)
- `PORT` (default `8080`)
- `HOST` (default `0.0.0.0`)
- `EPOS_PROFILE` (default `default`)

## API documentation

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI runtime YAML: `http://localhost:8080/openapi.runtime.yaml`
- Health: `http://localhost:8080/health`
- Neo4j Browser: `http://localhost:7474`

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
- SPARQL: `POST /api/v2/sparql` (read-only)
- OAI-PMH: `GET /oai-pmh`
- Admin: `POST /api/v2/admin/populate`, `POST /api/v2/admin/import/revision/*`, `POST /api/v2/admin/relations/reconcile`

## Query examples

### SPARQL examples

Ask if any triple exists:

```bash
curl -X POST "http://localhost:8080/api/v2/sparql" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "query=ASK WHERE { ?s ?p ?o }"
```

Select triples (JSON results):

```bash
curl -X POST "http://localhost:8080/api/v2/sparql" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "query=SELECT * WHERE { ?s ?p ?o } LIMIT 5"
```

Select triples (XML results):

```bash
curl -X POST "http://localhost:8080/api/v2/sparql" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -H "Accept: application/sparql-results+xml" \
  --data-urlencode "query=SELECT * WHERE { ?s ?p ?o } LIMIT 5"
```

JSON request body:

```bash
curl -X POST "http://localhost:8080/api/v2/sparql" \
  -H "Content-Type: application/json" \
  -d '{"query":"SELECT * WHERE { ?s ?p ?o } LIMIT 5"}'
```

Write operation rejection (expected `403`):

```bash
curl -X POST "http://localhost:8080/api/v2/sparql" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "query=DELETE WHERE { ?s ?p ?o }"
```

### OAI-PMH examples

Identify repository:

```bash
curl "http://localhost:8080/oai-pmh?verb=Identify"
```

List supported metadata formats:

```bash
curl "http://localhost:8080/oai-pmh?verb=ListMetadataFormats"
```

List available sets:

```bash
curl "http://localhost:8080/oai-pmh?verb=ListSets"
```

List identifiers:

```bash
curl "http://localhost:8080/oai-pmh?verb=ListIdentifiers&metadataPrefix=oai_dc"
```

List records:

```bash
curl "http://localhost:8080/oai-pmh?verb=ListRecords&metadataPrefix=oai_dc"
```

Get one record:

```bash
curl "http://localhost:8080/oai-pmh?verb=GetRecord&metadataPrefix=oai_dc&identifier=http%3A%2F%2Fexample.org%2Fdataset%2Fabc"
```

POST form request:

```bash
curl -X POST "http://localhost:8080/oai-pmh" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "verb=Identify"
```

## Search and detail query guidelines

- Use URL-encoded IDs when calling detail/access endpoints (`{id}` can include `:`, `/`, `#`).
- Search defaults to published data only; include `versioningStatus` to broaden results.
- Keep `resultsPerPage` between `1` and `100` (server clamps values outside this range).
- For stable pagination, prefer `page` + `resultsPerPage`; use `offset` only for cursor-like clients.
- Use `type` filters when possible to reduce full-graph scans and improve response time.

Search endpoint examples:

```bash
# Full-text search in title/description/label
curl "http://localhost:8080/api/v2/access/search?q=volcano&resultsPerPage=10&page=1"

# Restrict to one type (dataset)
curl "http://localhost:8080/api/v2/access/search?type=dataset&q=seismic"

# Multiple types and non-default version statuses
curl "http://localhost:8080/api/v2/access/search?type=dataset&type=distribution&versioningStatus=published&versioningStatus=submitted"

# Include constrained fields in each item
curl "http://localhost:8080/api/v2/access/search?type=dataset&includeConstraints=true"

# Equivalent shape shortcut (redirects internally to type filter)
curl "http://localhost:8080/api/v2/access/search/dataset?q=climate"
```

Detail endpoint examples:

```bash
# Dataset detail (JSON default)
curl "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc"

# Request turtle via query param
curl "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc?format=ttl"

# Request JSON-LD via Accept header
curl "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc" \
  -H "Accept: application/ld+json"

# Read a non-default lifecycle version (if available)
curl "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc?status=submitted"
```

Access/detail-by-shape examples:

```bash
# List all resources for one shape
curl "http://localhost:8080/api/v2/access/dataset/all"

# Resolve one resource in access format
curl "http://localhost:8080/api/v2/access/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc"
```

## Lifecycle and mutation behavior

- `{id}` in lifecycle/resource mutation endpoints accepts conceptual id, version id, or identifier URI.
- `PUT`, `PATCH`, `DELETE` operate on `draft` versions only.
- `submit` accepts `draft` only; `publish` accepts `submitted` only.
- `discard` accepts `draft` or `submitted` and moves to `discarded`.
- Invalid state transitions return `409 Conflict`.

Lifecycle/mutation query guidelines:

- Always send `X-Agent` and `X-Group` headers with absolute URIs for mutating requests.
- Use URL-encoded IDs when an identifier contains reserved URL characters (`:`, `/`, `#`, `?`).
- Prefer conceptual IDs for client workflows; the service resolves to the current actionable version.
- Expect `409 Conflict` for valid requests in an invalid lifecycle state; do not retry blindly.
- Use `GET .../history` before state transitions when debugging lifecycle progression.

Lifecycle examples:

```bash
# Create draft from current published version
curl -X POST "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/draft" \
  -H "X-Agent: https://example.org/agent/cli" \
  -H "X-Group: https://example.org/group/metadata"

# Submit draft
curl -X POST "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/submit" \
  -H "X-Agent: https://example.org/agent/cli" \
  -H "X-Group: https://example.org/group/metadata"

# Publish submitted version
curl -X POST "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/publish" \
  -H "X-Agent: https://example.org/agent/cli" \
  -H "X-Group: https://example.org/group/metadata"

# Reject submitted version back to draft
curl -X POST "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/reject" \
  -H "X-Agent: https://example.org/agent/cli" \
  -H "X-Group: https://example.org/group/metadata"

# Discard draft or submitted version
curl -X POST "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/discard" \
  -H "X-Agent: https://example.org/agent/cli" \
  -H "X-Group: https://example.org/group/metadata"

# Inspect lifecycle history
curl "http://localhost:8080/api/v2/dataset/http%3A%2F%2Fexample.org%2Fdataset%2Fabc/history"
```

## Populate endpoint behavior

- `POST /api/v2/admin/populate` with no body uses configured `EPOS_INDEX_URL`.
- `POST /api/v2/admin/populate` with body supports exactly one of:
  - `{"indexUrl": "http(s)://.../index.txt"}` for full index-based population
  - `{"url": "http(s)://.../file.ttl"}` for single TTL source
  - `{"path": "/absolute/path/file.ttl"}` for local TTL file

## Docker

### Start Neo4j stack for development

```bash
docker compose -f docker-compose.neo4j.yml up -d
```

### Build API image

```bash
make build-docker
```

### Run API container

```bash
docker run --rm -p 8080:8080 \
  -e EPOS_NEO4J_URI=bolt://host.docker.internal:7687 \
  -e EPOS_NEO4J_USERNAME=neo4j \
  -e EPOS_NEO4J_PASSWORD=neo4jpassword \
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
