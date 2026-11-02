# Semantic Aggregator

## Scope and design

- Keep ingestion and aggregation separate: ingestion maps a Turtle graph to Metadata Lifecycle resource inputs; aggregation builds versioned RDF graphs from published upstream data.
- Keep RDF parsing, graph traversal and version-specific mappings out of transport and persistence adapters. Use small, explicit mapping functions and separate input, domain and protobuf types.
- Flatten linked and blank-node structures deliberately; preserve identifiers, multiplicity, language, datatypes and missing values where the target permits. Report unsupported or ambiguous mappings rather than silently dropping data.
- Use an isolated in-memory graph per import. Keep persistent triplestore access behind a narrow adapter; do not assume a particular vendor in domain logic.
- Do not treat the legacy mapping ontology as the current resource schema. Consult the upstream protobuf contracts and validate against the target service.
- Aggregate only published records, account for pagination and failures, and publish a complete versioned graph atomically so a failed refresh does not replace a valid graph.
- Add packages when a concrete responsibility exists; prefer readable code and explicit errors to generic frameworks. Add dependencies only with approval.
- Expose only gRPC from this service. Public HTTP endpoints belong in Gateway; use the standard gRPC health protocol.
- Store versioned RDF paths in validated YAML mapping profiles. Keep target-field semantics and transformations explicit in code; reject unknown configuration and never silently discard unmapped input.
- Fuseki uses persistent volumes and the SPARQL Graph Store protocol. Do not commit credentials or use development defaults outside local environments. Graph Store tests use unique named graphs and must not alter unrelated graphs.

## Workflow

- Use Java 25 LTS, Maven and Apache Jena. Run `make build` and `make test` in this directory; `make lint` recompiles with Java compiler warnings enabled.
- Keep the README and stack configuration aligned with implemented behavior. Health currently indicates only that the process is running.
