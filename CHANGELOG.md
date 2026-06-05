# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/):

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Fixed` for any bug fixes.
- `Improved` for performance improvements.

## Unreleased

### Added

- helm: added an initial `epos-platform` chart for gateway, metadata-lifecycle, user-group, data-retriever, and optional development datastores.
- ci: added GitHub Actions pipelines for service build/test/lint, integration tests, Docker image publishing, and Helm chart packaging/release artifacts.
- data-retriever: added a new backend service under `services/data-retriever` with `/api/v2` retrieval and search endpoints, strict URL placeholder validation, optional bearer authentication for outbound requests, PostgreSQL persistence, and Swagger/OpenAPI documentation endpoints.
- user-group: added a new backend service scaffold under `services/user-group` with `/api/v2` API contract, initial PostgreSQL migrations for users/groups/memberships/permissions, and baseline build/test/lint integration in the root Makefile.
- user-group: added built-in API documentation endpoints (`/openapi.yaml` and `/swagger`) to expose the OpenAPI contract and interactive Swagger UI.
- user-group: added in-memory `/api/v2/groups` management endpoints with full CRUD plus PATCH semantics.
- user-group: added in-memory `/api/v2/users` management endpoints with full CRUD plus PATCH semantics, including user lookup by auth identifier.
- user-group: wired users and groups CRUD/PATCH endpoints to PostgreSQL-backed repositories and added database connection configuration via `USER_GROUP_DB_*` environment variables.
- packages/go: added shared `postgres` and `migrations` Go modules under `packages/go/` to centralize PostgreSQL connection bootstrap and SQL migration execution logic.

### Changed

- metadata-lifecycle: completed migration to Neo4j Community for runtime and local infrastructure. Replaced JanusGraph/Gremlin endpoint query paths with Cypher across CRUD, lifecycle, access/search, OAI-PMH, population, revision import, and relation reconciliation flows. Updated Tilt/Docker local stack to run Neo4j and switched service configuration to `EPOS_NEO4J_*` variables.
- metadata-lifecycle: switched infrastructure and runtime defaults from FalkorDB to JanusGraph with Cassandra and Elasticsearch. Updated service and repository Docker Compose stacks, JanusGraph environment keys (`EPOS_JANUSGRAPH_*`), and service configuration profiles to target Gremlin Server on port 8182 while preserving the Property Graph conceptual/version/activity model.
- metadata-lifecycle: migrated from Neo4j to FalkorDB as the graph database backend. All Neo4j driver references replaced with FalkorDB Java client. Configuration changed from `EPOS_NEO4J_*` to `EPOS_FALKORDB_*` environment variables (host, port, graph, password). Docker compose updated to use `falkordb/falkordb:edge` image on port 6379.
- metadata-lifecycle: migrated from GraphDB (triplestore/SPARQL) to Neo4j (Property Graph/Cypher). Lifecycle status and versioning are now managed via node properties and explicit relationships (`Conceptual`, `Version`, `Activity` nodes with `HAS_VERSION`, `CURRENT_VERSION`, `REVISION_OF`, `SUPERSEDED_BY`, `AFFECTED` edges). Entity data is embedded in the `data` property of Version nodes. SPARQL endpoint replaced with read-only Cypher endpoint (`POST /api/v2/cypher`).
- metadata-lifecycle: removed EDM mapping dependency from search/details responses and aligned type/property/relation naming to SHACL-derived metadata with URI-local fallback.
- docs: replaced placeholder root README with a practical project overview, repository layout, local Docker quick start, development commands, and service documentation links.
- metadata-lifecycle: consolidated admin population API into `POST /api/v2/admin/populate`, and added request-level `indexUrl` support for index-based population without restarting service.
- metadata-lifecycle: expanded lifecycle transitions with `discarded` state and `POST /api/v2/{resourceType}/{id}/discard` endpoint.
- metadata-lifecycle: normalized lifecycle/resource mutation ID resolution so conceptual/version/identifier URIs resolve consistently across submit/publish/draft/discard and PUT/PATCH/DELETE flows.
- data-retriever: switched to shared structured logging package usage across service startup and HTTP handlers, and aligned `/health` to include database reachability checks.
- user-group: switched to shared structured logging package usage across service startup and handlers, and aligned `/health` to include database reachability checks.
- data-retriever, user-group: migrated storage internals to shared `packages/go/postgres` and `packages/go/migrations` modules and updated development Docker/Tilt wiring for shared local package paths.

### Fixed

- metadata-lifecycle: fixed Cypher search filtering and text matching regressions after migration (`CONTAINS` operator usage and robust type matching from conceptual and version properties, including `rdf:type` payload keys).
- metadata-lifecycle: fixed RDF/entity responses leaking internal JanusGraph numeric ids by preferring URI identifiers in entity and relation payloads.
- metadata-lifecycle: fixed list/detail payload normalization so URI fields are not serialized as bracketed strings and internal vertex metadata is filtered out.
- metadata-lifecycle: fixed multiple JanusGraph serializer/timeout edge cases in lifecycle and mutation operations by avoiding relation-returning writes and replacing full-scan identifier lookups with constrained traversals.
- metadata-lifecycle: fixed access and dataset detail resolution for conceptual/version identifiers by adding robust `HAS_VERSION` fallbacks and identifier-aware lookups.

### Improved

- metadata-lifecycle: improved operator log tracking by removing the raw startup banner, making request logs uncolored, and adding structured population event fields.
- metadata-lifecycle: improved cold-start and early-request latency with startup warmup queries.
- metadata-lifecycle: improved search/discovery performance by adding JanusGraph composite indexes for `Conceptual.type`, `Conceptual.identifier`, and `Version.identifier` and reducing broad `valueMap(true)` projections on hot access-list paths.
