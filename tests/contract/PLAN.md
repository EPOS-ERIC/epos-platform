# Gateway Contract Tests

This document is the source of truth for deterministic Gateway contract-test
work. It records fixed architecture decisions, test quality rules, and the
process for planning each new flow. It does not pre-approve implementations.

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

## Current Baseline

Existing contract tests cover basic Gateway CRUD, one admin dataset lifecycle,
basic membership and role flows, canonical reads and writes, discovery of
imported drafts, and authorization reachability for all 56 current OpenAPI
operations. The authorization flow proves important allow/deny boundaries, but
it is not a substitute for functional tests of every lifecycle branch, filter,
validation rule, or cross-resource invariant.

New tests should strengthen those behavioral contracts instead of adding more
generic endpoint smoke calls.

## Authorization Coverage

The contract-test suite must cover authorization for every Gateway endpoint
whose access rules are defined by `lifecycle.md`.

- Derive the endpoint and permission matrix from the Gateway OpenAPI contract
  and `lifecycle.md`; do not infer authorization rules from test behavior.
- Test each endpoint with actors that should be allowed and denied under its
  defined access rules.
- Test public endpoints without authentication where `lifecycle.md` defines
  public access.
- Keep authorization coverage in a dedicated flow, initially
  `test_authorization.py`, rather than duplicating secondary-user setup in
  every endpoint flow.
- Each authorization test owns its users, memberships, roles, grants, groups,
  and resources. Do not depend on state created by business-flow tests.
- Endpoint-specific authorization regressions may remain in the dedicated
  authorization flow and must identify the caught bug in the test docstring.
- Authorization coverage is not complete until every applicable endpoint has
  an explicit test or is documented as public.

Authorization tests must use valid request bodies and valid lifecycle state so
  authorization is evaluated before validation or transition conflicts.

## Failure And Bug Handling

When a new test fails, do not weaken the test or alter expected behavior to
make it pass.

- Stop implementation and classify the failure as a test defect, environment
  failure, contract mismatch, or implementation bug.
- Compare actual behavior with the Gateway OpenAPI contract, `lifecycle.md`,
  and the relevant implementation before changing code or assertions.
- If the failure is an implementation bug, document the expected behavior,
  actual behavior, affected path, and impact, then ask whether product code
  should be fixed before continuing.
- Preserve the failing assertion when it captures a product bug. Add or update
  a regression test only after the expected behavior is agreed.
- Do not design setup or assertions around accidental current behavior.
- Regression-test docstrings must identify the bug being protected.

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

There is no shared fixture context at this stage. `client.py` and `auth.py` are
shared because every authenticated flow uses them. Keep flow-specific setup in
its flow file until the same setup is needed by at least two flows.

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

## Proposed Next Flows

These flows are ordered by regression value. They are proposals, not approval
to implement. Before each flow, use the planning gate below to confirm exact
scope, fixtures, actors, assertions, and exclusions.

### 1. `test_resource_lifecycle_branches.py`

Purpose: Lock lifecycle state transitions and version invariants beyond the
existing create/replace/submit/publish happy path.

OperationIds: `createResource`, `replaceResourceDraftVersion`,
`getResourceVersion`, `listResourceVersions`, `submitResourceVersion`,
`withdrawResourceVersion`, `rejectResourceVersion`, `reopenResourceVersion`,
`publishResourceVersion`, `unpublishResourceVersion`,
`discardResourceDraftVersion`, `draftFromPublication`, and `draftFromArchive`.

Flow: Establish unique dataset fixtures and drive valid branches through
`DRAFT -> SUBMITTED -> REJECTED -> DRAFT -> SUBMITTED -> PUBLISHED -> ARCHIVED`.
Use separate resources where one-active-change rules would otherwise block the
next transition.

Assertions: Verify statuses, stable logical IDs, version IDs and revisions,
rejection note/actor/timestamp persistence and clearing, publication archival,
historical-version immutability, active-change conflicts, draft discard, and
metadata preservation for drafts made from publication or archive.

Negative cases: Use valid requests for unauthorized actors and invalid
transitions. Do not assert a transition error until the expected state rule is
confirmed from the contract and implementation.

### 2. `test_resource_variants_and_copy.py`

Purpose: Protect resource-type mappings and copy semantics used by the three
resource implementations.

OperationIds: `createResource`, `copyResource`, `getResourceVersion`,
`replaceResourceDraftVersion`, and `draftFromPublication`.

Flow: Create or use one valid dataset, software application, and software
source-code fixture. Populate representative metadata, identifiers,
organizations, contact points, web services, and categories where each schema
supports them. Copy a published resource into another group and create a draft
from a publication.

Assertions: Verify discriminator-specific response shapes, new logical IDs for
copies, preserved cloned metadata and identifiers with new ownership rows,
preserved logical IDs for publication drafts, and category/link rules including
zero categories on copies where `lifecycle.md` requires it.

Negative cases: Cross-group category references, unauthorized target-group
copies, and resource-type/body mismatches when the request is otherwise valid.

### 3. `test_canonical_reference_integrity.py`

Purpose: Protect unlinking, ownership, and cascade behavior across canonical
metadata and resource versions.

OperationIds: `createOrganization`, `replaceOrganization`,
`deleteOrganization`, `createContactPoint`, `replaceContactPoint`,
`deleteContactPoint`, `createGroupCategory`, `deleteGroupCategory`, and
`deleteGroup`.

Flow: Create canonical organizations, contact points, a category subtree, and
resource versions that reference them. Exercise parent, identifier, provider,
contact-point, web-service, and category links before deleting canonical data.

Assertions: Verify resource links are removed or nulled according to the
documented rule, identifiers retain correct ownership, category subtree links
are removed from published and archived versions, and group deletion is blocked
while resources still reference the group.

Negative cases: Duplicate ownership, cross-group category links, hierarchy
cycles, self-links, and deletion conflicts must return schema-validated
business errors.

### 4. `test_discovery_filters.py`

Purpose: Lock search visibility and filter semantics so query/index changes do
not change the public catalogue contract.

OperationIds: `discoverResources` and the related
`getResourceVersion` read path.

Flow: Establish or identify unique records with distinct resource types,
statuses, groups, editors, providers, keywords, dates, and spatial values.
Exercise default search plus a compact matrix of positive and negative filters.

Assertions: Verify default published visibility, exact result identity for each
filter, mixed resource-type results, date-boundary behavior, and exclusion of
private records for anonymous callers and callers without group access.

Negative cases: Invalid filter combinations and invalid date ranges must retain
their documented validation behavior. Optional-auth requests must distinguish
anonymous, invalid-token, and blocked-token behavior.

### 5. `test_membership_invariants.py`

Purpose: Protect access cleanup and role-request state that basic membership
tests do not cover.

OperationIds: `requestGroupMembership`, `withdrawGroupMembershipRequest`,
`decideGroupMembership`, `deleteGroupMembership`, `requestGroupRole`,
`withdrawGroupRoleRequest`, `decideGroupRoleRequest`, `assignGroupRole`,
`removeGroupRole`, `grantGroupPermission`, and `revokeGroupPermission`.

Flow: Request membership with roles, reject it, verify role-request cascade,
retry and accept it, create pending role requests, revoke membership, and
request membership again. Include direct role and permission assignments.

Assertions: Verify pending/rejected/active state, role-request decisions,
effective permissions, removal of roles, grants, pending requests, and related
access after revocation, plus successful re-request after revocation.

Negative cases: Pending matching role requests block direct assignment; pending,
rejected, and revoked memberships cannot receive roles or permissions.

### 6. `test_authorization_edge_cases.py`

Purpose: Protect blocked-user restoration and global-admin safety boundaries.

OperationIds: `blockUser`, `unblockUser`, `getUser`, `discoverResources`,
`getResourceVersion`, `listContactPoints`, `getContactPoint`,
`removeGlobalRole`, and the relevant membership and resource operations.

Flow: Fully provision a user with active membership, roles, grants, and access;
block the user; exercise protected, optional-auth, and fully public operations;
unblock the user; verify access restoration. Exercise last-active-admin block
and role-removal safeguards.

Assertions: Verify blocked bearer rejection where authentication is required or
optional, token-agnostic behavior for fully public operations, dormant access
while blocked, restored access after unblock, and protection of the last active
administrator.

Negative cases: Invalid and missing tokens must use the same valid request/state
as the allowed cases. Do not add duplicate endpoint smoke tests already covered
by `test_authorization.py`.

### 7. `test_validation_errors.py`

Purpose: Lock representative HTTP and `Problem` error contracts at the Gateway
boundary.

OperationIds: `createGroup`, `createGroupCategory`, `replaceGroupCategory`,
`createOrganization`, `createContactPoint`, `replaceResourceDraftVersion`,
`rejectResourceVersion`, `requestGroupMembership`, `requestGroupRole`, and
`discoverResources`.

Cases: Wrong media type (`415`), missing required fields, unknown fields,
blank strings, malformed UUIDs, invalid enum values, invalid email/URI/date-time
values, invalid date ranges, empty rejection notes, duplicate data, and invalid
lifecycle transitions.

Assertions: Validate each response through `GatewayClient`, then assert only
stable status and `Problem` fields agreed by OpenAPI and `lifecycle.md`.
Resolve any OpenAPI-versus-implementation mismatch as a real contract decision;
do not silently encode whichever behavior the current server happens to return.

### 8. `test_import_semantics.py`

Purpose: Protect functional Turtle import behavior after authorization is
covered.

OperationId: `importTurtleMetadata`.

Flow: Submit one explicit valid Turtle document and one document containing
invalid entities or optional relations. Exercise repeated import against draft
state and a conflicting active state.

Assertions: Verify persisted counts, warning and diagnostic shapes, skipped
entities, omitted optional relations, draft-only persistence, and transaction
rollback on syntax or non-recoverable failure.

Negative cases: Non-admin denial remains in `test_authorization.py`; this flow
focuses on import behavior and must not use import as hidden fixture setup.

## Approved Flow Plans

### `test_resource_lifecycle_branches.py`

Status: approved
Purpose: Lock lifecycle state transitions and version invariants beyond the
existing create/replace/submit/publish happy path.

OperationIds: `createResource`, `replaceResourceDraftVersion`,
`getResourceVersion`, `listResourceVersions`, `submitResourceVersion`,
`withdrawResourceVersion`, `rejectResourceVersion`, `reopenResourceVersion`,
`publishResourceVersion`, `unpublishResourceVersion`,
`discardResourceDraftVersion`, `draftFromPublication`, and `draftFromArchive`.

Actors and permissions: Admin creates the unique group and may establish
fixtures. An active `EDITOR` performs create, edit, submit, withdraw, reopen,
discard, and draft creation. An active `REVIEWER` performs reject and publish;
the admin grants it the narrow group permission `resource.unpublish`.

Fixtures and ownership: The test owns one unique group, one editor, one
reviewer, and unique dataset resources. It leaves local data in place and does
not require cleanup.

HTTP flow: Drive one dataset through
`DRAFT -> SUBMITTED -> REJECTED -> DRAFT -> SUBMITTED -> PUBLISHED -> ARCHIVED`.
Create and discard a rejected-only resource and a draft-only resource. Create a
draft from publication, verify repeated draft creation conflicts, unpublish the
publication, and create a draft from the latest archive.

Assertions: Validate every Gateway response through `GatewayClient`; assert
statuses, stable logical IDs, version IDs and revisions, rejection note and
audit fields, rejection-field clearing on publication, publication archival,
historical-version immutability, active-change conflicts, draft discard, and
metadata preservation for drafts made from publication or archive.

Negative cases: Use valid requests and valid lifecycle state for unauthorized
actors and invalid transitions. Discarding a `REJECTED` version is valid and
must remain documented by OpenAPI, `lifecycle.md`, and the implementation.

Verification: Run `make generate` from `services/gateway`, then
`make test-contract`.

### `test_resource_variants_and_copy.py`

Status: approved
Purpose: Protect resource-type mappings and copy semantics used by the three
resource implementations.

OperationIds: `createResource`, `copyResource`, `getResourceVersion`,
`replaceResourceDraftVersion`, and `draftFromPublication`.

Actors and permissions: Admin creates canonical fixtures and publishes source
resources. An active `EDITOR` belongs to both source and target groups and
performs successful copies and publication drafts. A separate active source-
only user verifies target-group copy denial.

Fixtures and ownership: Create two unique groups, one source and one target;
three unique resources, one dataset, software application, and software
source-code; canonical organizations and contact points; and group categories.
The flow owns all mutable fixtures and leaves local data in place.

HTTP flow: Admin creates representative linked metadata for each resource and
publishes all three source resources. The editor copies each published resource
to the target group, replaces each copied draft once, and creates a draft from
each source publication. The source-only user attempts a copy into the target
group.

Assertions: Validate every Gateway response through `GatewayClient`; assert
discriminator-specific response shapes, new logical IDs for copies, stable
logical IDs for publication drafts, new version IDs and revisions, cloned
identifier values with new internal IDs, preserved supported links, zero
categories on copies, and preserved categories on publication drafts.

Negative cases: Cross-group category references, unauthorized target-group
copies, and resource-type/body mismatches use otherwise valid requests and
starting state. Cross-group category references and resource-type/body
mismatches return `400`; unauthorized target-group copies return `403`.
Assert stable schema-validated problem fields where the exact business error is
documented.

Exclusions: Do not add broad lifecycle transition coverage, cleanup, imported
fixtures, or new Go unit tests in this flow.

Verification: Run `make generate` from `services/gateway`, then
`make test-contract`.

### `test_canonical_reference_integrity.py`

Status: approved
Purpose: Protect canonical deletion, ownership, hierarchy, and cascade
behavior across resource versions.

OperationIds: `createOrganization`, `replaceOrganization`,
`deleteOrganization`, `createContactPoint`, `deleteContactPoint`,
`createGroupCategory`, `replaceGroupCategory`, `deleteGroupCategory`, and
`deleteGroup`. Resource creation and publication operations are setup steps for
the reference assertions: `createResource`, `submitResourceVersion`,
`publishResourceVersion`, `draftFromPublication`, and `getResourceVersion`.

Actors and permissions: The global admin creates and manages every fixture and
performs canonical and resource operations. No authorization boundary is under
test in this flow.

Fixtures and ownership: Create one unique group, parent and child canonical
organizations, one contact point, a root and child category, and one dataset,
software application, and software source-code resource. Each resource owns a
published and archived version with organization, contact-point, category, and
identifier references. The dataset also owns a web service with organization
and contact-point references. The flow leaves local data in place.

HTTP flow: Publish an initial version for each resource, create and publish a
second version so the first becomes archived, then delete the parent
organization, contact point, and category root. Verify each canonical entity is
deleted and all documented resource, hierarchy, web-service, and category links
are removed from both versions. Verify resource-owned identifier IDs and values
remain unchanged. Attempt group deletion while logical resources remain and
expect `409`.

Negative cases: Category self-links and indirect cycles, plus organization
self-links and indirect cycles, return `400` and leave the existing hierarchy
unchanged. Cross-group resource-category validation remains covered by
`test_resource_variants_and_copy.py` and is not duplicated here.

Assertions: Validate every Gateway response through `GatewayClient`; assert
only stable statuses, IDs, link presence/removal, identifier ownership, and
the group conflict problem fields documented by existing contract patterns.

Implementation alignment: Hierarchy trigger failures use PostgreSQL `23514`;
legacy `P0001` responses from an already-migrated local database map to the
same invalid domain errors.

Verification: Run `make generate` from `services/metadata-lifecycle`,
`make build`, `make test`, and `make test-contract`.

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

## Assertions

Use OpenAPI validation plus focused business assertions. Do not compare full
responses. Treat IDs, timestamps, audit fields, and ordering as dynamic unless
the flow plan identifies a specific invariant.

Every negative test must use a valid request and valid starting state when the
purpose is authorization. Validation tests must isolate the validation rule
being tested. Do not allow a malformed body or invalid lifecycle state to hide
an authorization failure.

For lifecycle flows, candidate business assertions include status changes,
stable logical resource IDs, new version IDs, revision conflicts, linked IDs,
rejection notes, publication archival, and deletion/unlinking effects. The
exact assertions belong in the approved flow plan.

## Verification And Scope

After implementing an approved flow:

```bash
make test-contract
```

Keep CI workflow changes, image pinning, environment cleanup, and automatic
database resets out of this phase. Update repository documentation only when
the implemented workflow or command usage changes.
