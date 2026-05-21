# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/):

- `Added` for new features.
- `Changed` for changes in existing functionality.
- `Deprecated` for soon-to-be removed features.
- `Fixed` for any bug fixes.
- `Improved` for performance improvements.

## Unreleased

### Changed

- metadata-lifecycle: switched infrastructure and runtime defaults from FalkorDB to JanusGraph with Cassandra and Elasticsearch. Updated service and repository Docker Compose stacks, JanusGraph environment keys (`EPOS_JANUSGRAPH_*`), and service configuration profiles to target Gremlin Server on port 8182 while preserving the Property Graph conceptual/version/activity model.
- metadata-lifecycle: migrated from Neo4j to FalkorDB as the graph database backend. All Neo4j driver references replaced with FalkorDB Java client. Configuration changed from `EPOS_NEO4J_*` to `EPOS_FALKORDB_*` environment variables (host, port, graph, password). Docker compose updated to use `falkordb/falkordb:edge` image on port 6379.
- metadata-lifecycle: migrated from GraphDB (triplestore/SPARQL) to Neo4j (Property Graph/Cypher). Lifecycle status and versioning are now managed via node properties and explicit relationships (`Conceptual`, `Version`, `Activity` nodes with `HAS_VERSION`, `CURRENT_VERSION`, `REVISION_OF`, `SUPERSEDED_BY`, `AFFECTED` edges). Entity data is embedded in the `data` property of Version nodes. SPARQL endpoint replaced with read-only Cypher endpoint (`POST /api/v2/cypher`).
- metadata-lifecycle: removed EDM mapping dependency from search/details responses and aligned type/property/relation naming to SHACL-derived metadata with URI-local fallback.
- docs: replaced placeholder root README with a practical project overview, repository layout, local Docker quick start, development commands, and service documentation links.
