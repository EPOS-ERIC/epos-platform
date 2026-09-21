# AGENTS

## Workflow

- Run against an already-running local stack.
- Import Metadatone manually with `make populate-metadatone`.
- Run tests with `make test-integration`.
- Do not reset databases, clean volumes, or add hidden import/setup steps.
- Do not add Python dependencies without approval.

## Test Conventions

- Use `unittest`. Send Gateway requests through `GatewayClient`; direct `requests` usage is limited to AAI helpers and GatewayClient unit tests.
- Group related behavior by flow in `test_*.py`, not by endpoint.
- Mutating flows create or locate the state they own. Read-only tests may use imported Metadatone fixtures.
- Never depend on test ordering or IDs created by another test. Shared `setUpClass` GatewayClient instances are transport/auth setup only; do not share mutable business fixtures across tests.
- Keep ordered lifecycle transitions inside one test method.
- Small helpers are allowed within one flow. Share fixture helpers across flows only after repeated use.
- Every test method must have a short docstring explaining what behavior it verifies.
- A regression-test docstring must also identify the bug it catches.
- Use unique mutable test data. No per-test cleanup is required.
- Treat imported Metadatone records as read-only. Use unique synthetic records for rich metadata, lifecycle, and destructive tests.
- Assert focused business invariants, not complete response bodies.
- Treat IDs, timestamps, audit fields, and response ordering as dynamic. Filter collections, compare sets, or use `assertCountEqual`; do not rely on server order.

## Gateway Boundary

- Send every Gateway request through `GatewayClient` in `client.py`.
- Pass OpenAPI path templates and URL-encode path parameters centrally.
- Support `GET`, `POST`, `PUT`, `PATCH`, and `DELETE` through one request path.
- Validate every Gateway response with `operation.validate_response(response)`.
- Do not call `raise_for_status()` for Gateway responses. Invalid transitions must return schema-validated `Problem` responses for assertions.

## AAI Boundary

- Keep token retrieval and local user registration in `auth.py`.
- AAI responses are outside the Gateway OpenAPI contract.
- Register secondary users through AAI's CSRF-protected `/register` flow.
- Create secondary users only in flows that need non-admin behavior; cache them within that flow when useful.
- Never log credentials, tokens, or secrets.

## Metadatone

- Use imported data directly for discovery, imported organization/contact reads, and access-context tests.
- Use synthetic unique records for rich metadata and lifecycle flows.
- Mutation is allowed when it simplifies a flow, but that flow owns the fixture and other tests must not depend on its original state.
- When fixing a bug, regression tests may extend `packages/assets/metadatone.ttl` with targeted cases required to reproduce and protect the bug.
- Create unique resources by default for lifecycle mutation tests.
- Draft discovery must use `GET /search?versioningStatus=DRAFT` without a type filter and find one record of each supported resource type.
- Missing required fixtures must fail clearly and mention `make populate-metadatone`.

## Runtime

- `make test-integration` runs the suite in the `ghcr.io/schemathesis/schemathesis:latest` Docker image.
- Runtime dependencies include `unittest`, `requests`, and `schemathesis`; do not add contract-local Python dependencies without approval.
- DELETE, block, revoke, and similar destructive operations are valid only against state created by the current test. Never reset the database or mutate imported baseline records.

## Scope

- Do not modify deleted Hurl tests or CI workflows in this phase.
- Do not edit Gateway OpenAPI or generated files from this directory.
