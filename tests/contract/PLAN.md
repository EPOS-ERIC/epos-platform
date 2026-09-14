# Gateway Contract Tests

This document is the source of truth for deterministic Gateway contract-test
work. It records fixed architecture decisions and the process for planning
each flow. It does not pre-approve implementations for future test files.

## Fixed Decisions

- Use Python `unittest` and `requests`.
- Use Schemathesis only for Gateway response validation.
- Keep broad stateless fuzzing in the Schemathesis CLI suite.
- Do not use automatic Schemathesis state-machine generation initially.
- Do not add Python dependencies.
- Do not add response golden files.
- Keep Hurl deletions untouched; Hurl is not the primary workflow-test layer.
- Use `make test-contract` as the contract-test target.
- Defer CI integration until local coverage is stable.
- Every test method must have a short docstring explaining the behavior it
  verifies. Regression-test docstrings must also identify the bug they catch.

## Runtime Setup

Tests run against an already-running local stack. The user must import
Metadatone manually before running the suite:

```bash
make populate-metadatone
make test-contract
```

Tests must not import the Turtle file, call `/imports/turtle` as hidden setup,
reset databases, remove volumes, or clean test data. Local test data may
remain. CI will use fresh volumes later.

The existing Schemathesis Docker image and `test-contract` target remain the
runtime foundation. Pin image versions only as part of deferred CI work.

## Directory Structure

Shared support belongs in:

```text
tests/contract/
  PLAN.md
  AGENTS.md
  auth.py
  client.py
  test_client.py
```

Each implemented integration flow belongs in its own `test_*.py` file. Group
related behavior by flow, not by individual endpoint. Do not create empty
future test files.

## Gateway Client

`client.py` owns all Gateway HTTP calls.

`GatewayClient` must provide one request implementation plus thin wrappers for
`GET`, `POST`, `PUT`, `PATCH`, and `DELETE`. It must:

- Resolve OpenAPI operations from path templates and upper-case methods.
- Fail clearly for unknown paths, unsupported methods, and missing path values.
- URL-encode path parameters centrally with `quote(value, safe="")`.
- Support query parameters, JSON bodies, raw bodies, and headers.
- Apply a timeout and bearer authentication through `requests.Session`.
- Call `operation.validate_response(response)` for every Gateway response.

Do not call `raise_for_status()` for Gateway responses. Negative tests expect
validated `Problem` responses and must assert their status and business fields
after schema validation.

Flow tests must pass OpenAPI path templates to the client and must not
hand-build encoded URLs.

## AAI Authentication

`auth.py` owns AAI calls. AAI responses are outside the Gateway OpenAPI
contract and must not go through Schemathesis response validation.

Required support:

- `get_token(...)` obtains a password-grant token and fails clearly when the
  request fails or `access_token` is absent.
- `register_user(...)` creates a local user through AAI's existing
  CSRF-protected `/register` HTML flow.
- Flows may call `register_user(...)` when they need a secondary user and may
  cache that user within the flow.
- Credentials, tokens, and secrets must never be logged or persisted.

Use the existing `AUTH_*` environment variables and local defaults for the
admin. Do not silently use the admin actor where a non-admin behavior is under
test.

## Setup And Data

There is no shared fixture context at this stage. `client.py` and `auth.py`
are shared because every authenticated flow uses them. Keep flow-specific
setup in its flow file until the same setup is needed by at least two flows.

When real duplication appears, add the smallest shared helper that removes it
without hiding mutable state. Discuss that extraction with the user before
adding it.

Importing a test module must not make network requests. Missing services,
fixtures, or invalid environment state must fail with actionable messages.

Each test method establishes its own mutable starting state. Tests must not
depend on unittest ordering or IDs produced by another test. Ordered state
transitions belong inside one test method. Use unique mutable data and do not
require per-test cleanup.

## Metadatone Policy

Metadatone is intentionally rich and may be used for datasets, software,
organizations, contact points, categories, web services, downloadable files,
and other linked examples.

- Use imported records directly when they simplify read-oriented tests.
- Imported records may be mutated when that simplifies a flow.
- A flow that mutates an imported record owns it; other flows must not depend
  on its original state.
- Create unique resources by default for lifecycle mutation tests.
- Regression tests may extend `packages/assets/metadatone.ttl` with targeted
  cases when existing data cannot reproduce the bug.
- Draft discovery must use `GET /search?versioningStatus=DRAFT` without a type
  filter and identify one record of each supported resource type.
- Do not rely on API response ordering when selecting fixtures.

## Coverage Backlog

Coverage is implemented flow by flow after the planning gate below. The
current backlog includes:

- Resource lifecycle for datasets, software applications, and software source
  code.
- Groups and categories.
- Organizations and contact points.
- Users and secondary-user behavior.
- Membership requests and membership decisions.
- Role requests and role management.
- Permission grants and revocations.
- Turtle import.
- Discovery filters and public/private access.

The exact endpoints, starting state, actors, data, assertions, negative cases,
and file boundaries are decided with the user before each file is implemented.

## Per-Flow Planning Gate

Before implementing a new flow file, the agent must stop and work with the
user to define that file's scope.

1. Read the relevant `lifecycle.md`, Gateway OpenAPI paths and schemas, service
   behavior, and existing contract-test support.
2. Identify facts already made explicit by those sources.
3. Identify ambiguity, contradictions, missing preconditions, authorization
   choices, and useful additional cases.
4. Ask the user focused questions about those points before writing test code.
5. Propose a concrete flow plan covering scope, exclusions, actors, fixtures,
   setup sequence, endpoint sequence, assertions, invalid cases, fixture
   ownership, and verification.
6. Record the agreed plan in this document under `Approved Flow Plans`.
7. Wait for the user's approval of that flow plan before implementing the test
   file.
8. Implement only the approved scope, then run the contract test target.

Questions should cover, when relevant:

- Which endpoints and behaviors belong in the file?
- Which behaviors are explicitly out of scope?
- Should the flow reuse, mutate, or create Metadatone data?
- Which actor and permissions are required at each step?
- What starting lifecycle state must be established?
- Which success, invalid-transition, validation, and authorization cases are
  required?
- Which linked IDs, revisions, rejection notes, audit fields, and publication
  effects are business requirements?
- Does a regression need a new Turtle fixture?
- How should mutable data remain isolated from other flows?
- Are any OpenAPI, `lifecycle.md`, or current-implementation details
  inconsistent?

Do not guess answers that are not explicit. Surface the inconsistency and ask.

## Approved Flow Plans

### test_resource_lifecycle.py

Status: approved
Purpose: Verify the admin dataset lifecycle from creation through publication.
In scope: Create one unique dataset, inspect its draft, replace it, reject a
stale replacement revision, submit it, and publish it.
Out of scope: Withdraw, reject, reopen, archive/unpublish, delete, copy,
draft-from-publication, other resource types, and authorization tests.
Actors and permissions: Admin actor only.
Fixtures and ownership: Reuse imported `Seismology` group by name without
mutating it. Create one unique dataset owned by this flow. Leave local data in
place; no cleanup.
Setup: Run against the existing stack after `make populate-metadatone`.
HTTP flow: `GET /groups`; `POST /groups/{groupId}/resources`; list and fetch
the draft version; `PUT /resources/{id}/versions/{versionId}/draft`; retry
with a stale revision; re-list and fetch the current draft; submit; re-list
and fetch the submitted version; publish; re-list and fetch the published
version.
Assertions: Validate every Gateway response through `GatewayClient`; assert
creation and transition statuses, dataset type, stable logical resource ID
and owning group, draft revision increment, submission and publication audit
fields, and published status.
Negative cases: A stale draft revision returns schema-validated `409` with
`title: "conflict"` and `detail: "operation conflicts with existing data"`.
Verification: `make test-contract`.

### test_groups_and_categories.py

Status: approved
Purpose: Verify admin group and category CRUD, hierarchy, normalization, and
cascade behavior.
In scope: Create, read, update, and delete one unique group; create, read,
update, and delete a root and child category; verify normalized duplicate-name
conflict and root-category cascade deletion.
Out of scope: Secondary-user authorization, unauthenticated reads,
cross-group parents, hierarchy cycles, group deletion conflicts, memberships,
roles, direct grants, organizations, resources, and Metadatone mutation.
Actors and permissions: Admin actor only.
Fixtures and ownership: Create unique group, root category, and child category
owned by this flow. Do not mutate imported groups or categories. No cleanup is
required.
Setup: Run against the existing stack with the admin credentials configured.
HTTP flow: `POST /groups`; `GET /groups` and `GET /groups/{groupId}`; `PUT
/groups/{groupId}`; create root and child categories; list and fetch both;
replace the child; create a case-normalized duplicate root; delete the root;
verify the root and child are gone; delete the group; verify the group is gone.
Assertions: Validate every Gateway response through `GatewayClient`; assert
stable IDs, group ownership, parent relationships, updated metadata, empty
category listing after cascade deletion, and `404` responses after deletion.
Negative cases: A case-normalized duplicate root category returns
schema-validated `409` with `title: "conflict"` and
`detail: "operation conflicts with existing data"`.
Verification: `make test-contract`.

Add one concise subsection here before implementing each additional flow:

```text
### test_<flow>.py

Status: approved
Purpose:
In scope:
Out of scope:
Actors and permissions:
Fixtures and ownership:
Setup:
HTTP flow:
Assertions:
Negative cases:
Verification:
```

An approved subsection is the implementation boundary for that file. Update
it if scope changes.

## Assertions

Use OpenAPI validation plus focused business assertions. Do not compare full
responses. Treat IDs, timestamps, audit fields, and ordering as dynamic unless
the flow plan identifies a specific invariant.

For lifecycle flows, candidate business assertions include status changes,
stable logical resource IDs, new version IDs, revision conflicts, linked IDs,
rejection notes, and publication archival. The exact assertions belong in the
approved flow plan.

## Verification And Scope

After implementing an approved flow:

```bash
make test-contract
```

Keep CI workflow changes, image pinning, environment cleanup, and automatic
database resets out of this phase. Update repository documentation only when
the implemented workflow or command usage changes.
