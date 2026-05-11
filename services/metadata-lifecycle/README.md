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
- `EPOS_SHACL_GRAPH` (named graph containing SHACL shapes, recommended in production)
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

### Runtime flow schema

```text
                                   +-----------------------------+
                                   |  SHACL schema / parser      |
                                   |  (resource types/constraints)|
                                   +--------------+--------------+
                                                  |
                                                  v
+----------------------+        +-----------------+-----------------+
| Client / UI / CLI    |<------>|  Ktor API routes + controllers    |
|                      |  JSON  |  (/api/v2/...)                    |
+----------+-----------+        +-----------------+-----------------+
           |                                       |
           | mutating endpoints require            | reads/search/details
           | X-Agent + X-Group                    |
           v                                       v
+----------+-----------+                 +---------+------------------+
| Lifecycle services   |                 | Query paths (graph-scoped)|
| (create/patch/submit |                 | status param or defaults   |
| /publish/import)     |                 |                            |
+----------+-----------+                 +----------------------------+
           |
           v
+----------+---------------------------------------------------------+
| GraphDB named graphs                                                |
|                                                                     |
|  [draft] --submit--> [submitted] --publish--> [published]           |
|     ^                            |                    |              |
|     |                            |                    +--> previous  |
|     +--------- reject -----------+                        version -> |
|                                                          [archived]  |
|                                                                     |
|  [conceptual] keeps current pointer: conceptual -> hasCurrentVersion|
|  [provenance] stores activities (agent + group + transitions)       |
+---------------------------------------------------------------------+

Revision import (same URIs already in published):
1) parse TTL,
2) create/reuse target drafts,
3) rewrite imported inter-resource links to draft URIs,
4) write into draft only,
5) submit + publish to promote and archive old published versions.
```

## API documentation

- Swagger UI: `http://localhost:8080/swagger`
- OpenAPI YAML: `http://localhost:8080/openapi.yaml`
- Health: `http://localhost:8080/health`

### Required mutation headers

Mutating operations require both headers with absolute URIs:

- `X-Agent`: acting agent URI
- `X-Group`: acting group URI

Example:

```text
X-Agent: mailto:user@example.org
X-Group: https://epos-eu.org/group/ingestion-team
```

## API groups (v2)

- System
  - `GET /health`
  - `GET /api/v2/resource-types`
- Resources
  - `GET|POST /api/v2/{resourceType}`
  - `GET|PUT|PATCH|DELETE /api/v2/{resourceType}/{id}`
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
    - optional query params: `extended=true`, `includeInbound=true`
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
- Admin revision import (draft-first)
  - `POST /api/v2/admin/import/revision/file`
  - `POST /api/v2/admin/import/revision/content`

## Endpoint flow schemas

### 1) System endpoints

```text
GET /health
  -> return { status: ok, timestamp }

GET /api/v2/resource-types
  -> if view=parsed (default):
       parse cached SHACL model -> build resource types -> optional shapeDefinitions -> JSON
  -> if view=shacl:
       read cached SHACL RDF model -> serialize by format/Accept -> RDF response
```

### 2) Resource CRUD endpoints

```text
POST /api/v2/{resourceType}
  -> validate X-Agent + X-Group (required absolute URIs)
  -> build new conceptual URI + v1.0-draft URI
  -> write entity triples in GRAPH_DRAFT
  -> write conceptual shell in GRAPH_CONCEPTUAL
  -> write provenance activity in GRAPH_PROVENANCE

GET /api/v2/{resourceType}
  -> read selected graph by status (default published)
  -> list/paginate entities

GET /api/v2/{resourceType}/{id}
  -> resolve graph by status
  -> serialize as JSON/Turtle/JSON-LD/RDFXML/N-Triples

PUT /api/v2/{resourceType}/{id} (draft only)
  -> validate X-Agent + X-Group
  -> replace semantics: remove non-core predicates, then add provided properties

PATCH /api/v2/{resourceType}/{id} (draft only)
  -> validate X-Agent + X-Group
  -> merge semantics: replace only provided predicates

DELETE /api/v2/{resourceType}/{id} (draft only)
  -> remove subject from draft graph
```

### 3) Lifecycle endpoints

```text
POST /api/v2/{resourceType}/{id}/submit
  -> validate X-Agent + X-Group
  -> move subject GRAPH_DRAFT -> GRAPH_SUBMITTED
  -> provenance transition DRAFT -> SUBMITTED

POST /api/v2/{resourceType}/{id}/publish
  -> validate X-Agent + X-Group
  -> if current published exists: move old published -> archived
  -> move submitted -> published (drop -draft suffix)
  -> update conceptual hasCurrentVersion pointer
  -> provenance transitions PUBLISHED -> ARCHIVED and SUBMITTED -> PUBLISHED

POST /api/v2/{resourceType}/{id}/draft
  -> validate X-Agent + X-Group
  -> copy current published to new {version}-draft
  -> provenance draft creation

GET /api/v2/{resourceType}/{id}/history
  -> query provenance graph for version history timeline

POST /api/v2/{resourceType}/{id}/repair
  -> validate X-Agent + X-Group
  -> rebuild conceptual pointer and missing provenance for published subject
```

### 4) Batch endpoints

```text
POST /api/v2/batch/submit
  -> validate X-Agent + X-Group
  -> iterate submit flow per URI
  -> return per-entity success/failure summary

POST /api/v2/batch/publish
  -> validate X-Agent + X-Group
  -> dependency-sort URIs
  -> iterate publish flow per URI
  -> return per-entity success/failure summary
```

### 5) Search and details endpoints

```text
GET /api/v2/search
  -> execute typed search query over published data
  -> return flat, heterogeneous result list with href links

GET /api/v2/details/{id}
  -> resolve typed detail query
  -> optional includeInbound=true and/or extended=true
  -> return JSON or RDF representation (format/Accept)
```

### 6) SPARQL and OAI-PMH endpoints

```text
POST /api/v2/sparql
  -> passthrough read/query execution against GraphDB repository
  -> return SPARQL JSON results

GET /oai-pmh
  -> process OAI-PMH verb (Identify/ListRecords/GetRecord/...)
  -> query metadata views and return OAI-PMH XML
```

### 7) Admin population endpoints

```text
POST /api/v2/admin/populate
POST /api/v2/admin/populate/from-url
POST /api/v2/admin/populate/file
POST /api/v2/admin/populate/content
  -> validate X-Agent + X-Group
  -> parse TTL sources
  -> normalize/skolemize/rewrite URIs to versioned form
  -> insert into GRAPH_PUBLISHED
  -> update GRAPH_CONCEPTUAL hasCurrentVersion pointers
  -> write initial provenance in GRAPH_PROVENANCE

GET /api/v2/admin/populate/status
  -> return effective index URL, endpoint, repository, graph names
```

### 8) Admin revision import endpoints (draft-first)

```text
POST /api/v2/admin/import/revision/file
POST /api/v2/admin/import/revision/content
  -> validate X-Agent + X-Group
  -> parse TTL import model
  -> for each conceptual resource: resolve published baseline and create/reuse target draft
  -> rewrite imported subject/object links to draft URIs (for imported peers)
  -> replace imported payload on draft subjects
  -> result can be dryRun (no writes) or applied writes to GRAPH_DRAFT
  -> manual promotion later via submit + publish endpoints
```

## Resource types endpoint for UI editors

`GET /api/v2/resource-types` supports two views:

- `view=parsed` (default): SHACL-derived resource definitions for UI/editor clients.
- `view=shacl`: original cached SHACL graph serialized as RDF.

Useful query parameters:

- `view=parsed|shacl`
  - default is `parsed`
- `format=turtle|ttl|jsonld|json-ld|rdfxml|rdf+xml|ntriples|n-triples|nt`
  - used when `view=shacl` (default `turtle`)
- `compact=true|false`
  - when `view=shacl`, `compact=true` returns a beautified compact representation for JSON-LD and Turtle
- `Accept` header
  - also supported for `view=shacl` content negotiation (`text/turtle`, `application/ld+json`, `application/rdf+xml`, `application/n-triples`)

- `includeShapeDefinitions=true`
  - include a `shapeDefinitions` map keyed by shape IRI (only for `view=parsed`)
- `includeTransitiveShapeDefinitions=true`
  - include only shape definitions reachable from resource properties via `sh:node` (only for `view=parsed`)
- `includeResolvedRefs=true`
  - add cycle-safe recursive references per shape (`resolvedRefs`) (only for `view=parsed`)

### Example

```bash
curl "http://localhost:8080/api/v2/resource-types?includeTransitiveShapeDefinitions=true&includeResolvedRefs=true"
```

### Raw SHACL export examples

```bash
# Original SHACL as Turtle
curl "http://localhost:8080/api/v2/resource-types?view=shacl&format=turtle"

# Original SHACL as JSON-LD
curl "http://localhost:8080/api/v2/resource-types?view=shacl&format=jsonld"

# Original SHACL as compact/beautified JSON-LD
curl "http://localhost:8080/api/v2/resource-types?view=shacl&format=jsonld&compact=true"

# Original SHACL as compact/beautified Turtle
curl "http://localhost:8080/api/v2/resource-types?view=shacl&format=turtle&compact=true"
```

Notes:

- Properties are aggregated by path (no duplicated `issued` lines with split min/max constraints)
- When multiple SHACL constraints exist for the same property/path, values are merged in the parsed view:
  - scalar/list dual fields expose 1-or-many values (`datatype` + `datatypes`, `classConstraint` + `classConstraints`, `severity` + `severities`)
  - cardinality is conservatively combined (`minCount` max, `maxCount` min)
- Scalar/list duplication is avoided:
  - single value -> `datatype` / `classConstraint` / `severity`
  - multiple values -> `datatypes` / `classConstraints` / `severities`
- `nodeShapeRef` can be resolved directly in `shapeDefinitions[<IRI>]`
- `shapeDefinitions` now also exposes node-level constraints for node-only shapes (e.g. `DateOrDateTimeDataType`), including `datatype(s)`, `classConstraint(s)` and `nodeShapeRef(s)`
- OWL schema definitions are parsed and used as fallback for richer constraints (e.g. `owl:ObjectProperty` with `rdfs:range` / `owl:unionOf` enriches `classConstraint(s)` when SHACL property constraints are implicit)

### Update semantics (`PUT` vs `PATCH`)

- `PUT /api/v2/{resourceType}/{id}` (draft only): full replace semantics for entity properties.
  Properties not included in the request are removed from the draft entity (except core lifecycle metadata such as type/version links).
- `PATCH /api/v2/{resourceType}/{id}` (draft only): merge semantics.
  Only provided properties are replaced; all other existing properties are preserved.

### Revision import workflow (draft -> submitted -> published)

- Bootstrap population endpoints (`/api/v2/admin/populate*`) insert directly into `published` and are intended for initial/technical loading.
- Revision import endpoints (`/api/v2/admin/import/revision/*`) are for metadata updates when URIs already exist in `published`.
- Revision import behavior:
  - creates/reuses target draft versions for existing conceptual resources,
  - rewrites links between imported resources to the corresponding draft URIs,
  - writes imported content into `draft` only (never directly to `published`).
- Promotion to publication remains explicit:
  1. `POST /api/v2/{resourceType}/{id}/submit`
  2. `POST /api/v2/{resourceType}/{id}/publish`
- During publish, the currently published version is automatically moved to `archived`.

Example revision import from inline TTL content:

```bash
curl -X POST "http://localhost:8080/api/v2/admin/import/revision/content" \
  -H "Content-Type: application/json" \
  -d '{
    "content": "@prefix dcat: <http://www.w3.org/ns/dcat#> .\n<https://epos-eu.org/id/distribution/abc/v1.0> a dcat:Distribution .",
    "versionMode": "auto",
    "replaceDraftIfExists": true,
    "dryRun": false
  }'
```

## Example curl commands

```bash
# Search
curl "http://localhost:8080/api/v2/search?q=seismic&page=1&pageSize=20"

# Get details as JSON
curl "http://localhost:8080/api/v2/details/https%3A%2F%2Fepos-eu.org%2Fid%2Fdataset%2F123"

# Get details as Turtle
curl "http://localhost:8080/api/v2/details/https%3A%2F%2Fepos-eu.org%2Fid%2Fdataset%2F123?format=turtle"

# Get details with inbound links
curl "http://localhost:8080/api/v2/details/https%3A%2F%2Fepos-eu.org%2Fid%2Fdataset%2F123?includeInbound=true"

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
