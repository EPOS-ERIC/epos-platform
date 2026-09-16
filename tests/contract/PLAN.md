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

## Authorization Coverage

The completed contract-test suite must cover authorization for every Gateway
endpoint whose access rules are defined by `lifecycle.md`.

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
- Authorization coverage for every applicable Gateway endpoint.
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

### test_organizations_and_contact_points.py

Status: approved
Purpose: Verify canonical organization and contact-point reads, CRUD, links,
and deletion unlinking.
In scope: Read imported organization and contact-point records; create, read,
update, and delete one contact point and parent/child organization pair;
verify duplicate conflicts and link removal after canonical deletion.
Out of scope: Secondary-user authorization, unauthenticated contact-point
reads, software/application/web-service links, multiple parents, hierarchy
cycles, memberships, roles, direct grants, resources, and Metadatone mutation.
Actors and permissions: Admin actor only. Organization reads use public access;
contact-point reads and all writes use the admin actor.
Fixtures and ownership: Read imported INGV and Andrea contact-point records
without mutating them. Create unique contact point and parent/child
organizations owned by this flow. No cleanup is required.
Setup: Run against the existing stack after `make populate-metadatone`.
HTTP flow: Read imported organization and contact point; create and duplicate
contact point; create and duplicate parent organization; create child
organization; read child; replace contact point; replace child organization;
delete parent; verify child remains without parent; delete contact point;
verify child remains without contact point; delete child; verify all deleted
entities return `404`.
Assertions: Validate every Gateway response through `GatewayClient`; assert
imported read fields, stable IDs, organization parent/contact links, retained
links after replacement, exact duplicate conflict responses, and canonical
deletion unlinking.
Negative cases: Duplicate contact point and duplicate organization name return
schema-validated `409` responses with `title: "conflict"` and
`detail: "operation conflicts with existing data"`. Deleted entities return
schema-validated `404` responses.
Verification: `make test-contract`.

### test_users.py

Status: approved
Purpose: Verify a newly registered AAI user is provisioned and can read its
own Gateway profile.
In scope: Register one unique user, obtain its token, and call
`GET /users/self`; assert identity and initial access state.
Out of scope: User listing, reading other users, blocking, global roles,
global permissions, memberships, authorization matrix, and AAI implementation
details beyond successful registration and token issuance.
Actors and permissions: Newly registered user only.
Fixtures and ownership: Generate unique user credentials in memory. Leave the
registered user in the local environment; no cleanup is required.
Setup: Run against the existing stack with AAI and Gateway available.
HTTP flow: AAI `GET /register`; AAI `POST /register`; AAI
`POST /oauth/token`; Gateway `GET /users/{id}` with `id="self"`.
Assertions: Validate the Gateway response through `GatewayClient`; assert
`200`, a non-empty opaque string user ID, matching email, given name, and
family name, `ACTIVE` status, empty global roles, empty global permissions,
and empty memberships.
Negative cases: None in this flow.
Verification: `make test-contract`.

### test_memberships.py

Status: approved
Purpose: Verify membership requests, decisions, access states, and decision
authorization.
In scope: Create one group; register one applicant; request membership with no
roles; verify duplicate-request conflict, applicant self-decision denial,
admin rejection, re-request, admin acceptance, invalid repeated acceptance,
and the applicant's active membership view.
Out of scope: Role requests, membership roster reads, request withdrawal,
membership revocation, direct roles and permissions, and the full
authorization matrix.
Actors and permissions: Admin creates the group and makes membership
decisions. A newly registered applicant requests membership and cannot decide
its own request.
Fixtures and ownership: Create one unique group and applicant owned by this
flow. Use opaque user IDs returned by the API. Do not mutate Metadatone data.
No cleanup is required.
Setup: Run against the existing stack with AAI and Gateway available, including
the string `userId` Gateway fix.
HTTP flow: Admin `POST /groups`; AAI registration and token issuance; applicant
`POST /groups/{groupId}/membership-requests` with `{"roles": []}`; repeat the
request; applicant attempts `POST
/groups/{groupId}/memberships/{userId}/accept`; admin rejects; applicant
requests again; admin accepts; admin repeats acceptance; applicant reads
`GET /users/{id}` with `id="self"`.
Assertions: Validate every Gateway response through `GatewayClient`; assert
pending, rejected, and active membership states, applicant identity, empty
roles/permissions/role requests, stable group and user IDs, and the active
membership summary.
Negative cases: Duplicate request and repeated acceptance return
schema-validated `409` responses with `title: "conflict"` and
`detail: "operation conflicts with existing data"`. Applicant self-decision
returns schema-validated `403` with `title: "forbidden"` and
`detail: "operation is not permitted"`.
Verification: `make test-contract`.

### test_role_requests.py

Status: approved
Purpose: Verify role-request creation, rejection, re-request, acceptance, and
effective role permissions.
In scope: Create one group; register one applicant; accept membership; request
`EDITOR`; verify duplicate conflict, rejection, re-request, acceptance, and
the applicant's effective editor permissions.
Out of scope: Historical decided-request visibility, direct role assignment or
removal, `GROUP_ADMIN`, roster authorization, direct permissions, and the full
authorization matrix.
Actors and permissions: Admin creates the group and makes membership and role
decisions. An active applicant requests `EDITOR`.
Fixtures and ownership: Create one unique group and applicant owned by this
flow. Use opaque user IDs and the role-request UUID returned by the API. Do not
mutate Metadatone data. No cleanup is required.
Setup: Run against the existing stack with AAI and Gateway available,
including the string `userId` Gateway fix.
HTTP flow: Admin `POST /groups`; AAI registration and token issuance; applicant
`POST /groups/{groupId}/membership-requests` with `{"roles": []}`; admin accepts
membership; applicant `POST /groups/{groupId}/role-requests` with
`{"role": "EDITOR"}`; repeat the request; admin rejects the role request;
applicant requests `EDITOR` again; admin accepts it; applicant reads
`GET /users/{id}` with `id="self"`.
Assertions: Validate every Gateway response through `GatewayClient`; assert
pending role and membership states, duplicate conflict, no role after
rejection, `EDITOR` after acceptance, stable IDs, no pending role requests,
and effective editor permissions: `resource.create`, `resource.edit`,
`resource.submit`, `resource.withdraw`, `resource.reopen`, and
`resource.discard`.
Negative cases: Duplicate pending role request returns schema-validated `409`
with `title: "conflict"` and
`detail: "operation conflicts with existing data"`.
Verification: `make test-contract`.

### test_group_roles.py

Status: approved
Purpose: Verify direct group-role assignment, removal, idempotence, and the
`GROUP_ADMIN` authorization boundary.
In scope: Create one group; register target, group-role manager, and global-role
manager; establish memberships; assign and remove `EDITOR`; verify effective
permissions; deny `GROUP_ADMIN` to the group-role manager; allow the global
role manager to assign and remove `GROUP_ADMIN`.
Out of scope: Direct group permissions, role requests, roster coverage, and the
remaining authorization matrix.
Actors and permissions: Admin creates the group, accepts memberships, assigns
`GROUP_ADMIN` to the group-role manager, and grants `global.role.manage` to the
global-role manager. The group-role manager manages standard roles only. The
global-role manager manages `GROUP_ADMIN` without group membership.
Fixtures and ownership: Create one unique group and three unique users owned
by this flow. Use opaque user IDs returned by the API. Do not mutate Metadatone
data. No cleanup is required.
Setup: Run against the existing stack with AAI and Gateway available, including
the string `userId` Gateway fix and the corrected `GROUP_ADMIN` rule in
`lifecycle.md`.
HTTP flow: Admin `POST /groups`; AAI registration and token issuance for all
users; applicant membership requests and admin membership decisions for the
target and group-role manager; admin assigns `GROUP_ADMIN` to the group-role
manager; admin grants `global.role.manage` to the global-role manager; group
role manager assigns and removes `EDITOR` twice; group-role manager attempts
`GROUP_ADMIN`; global-role manager assigns and removes `GROUP_ADMIN`.
Read back each target state with `GET /users/{id}` using `id="self"`.
Assertions: Validate every Gateway response through `GatewayClient`; assert
stable IDs, active memberships, effective role permissions, idempotent `204`
responses, forbidden `GROUP_ADMIN` assignment by group role manager, and
successful `GROUP_ADMIN` assignment/removal by global role manager.
Negative cases: Group role manager assigning `GROUP_ADMIN` returns
schema-validated `403` with `title: "forbidden"` and
`detail: "operation is not permitted"`.
Verification: `make test-contract`.

### test_group_permissions.py

Status: approved
Purpose: Verify direct group-permission grants, revocations, idempotence, and
effective permission read-back.
In scope: Create one group; register manager and target; establish active
memberships; grant `group.permission.manage` to the manager; grant and revoke
`resource.create` on the target twice; verify direct permission visibility.
Out of scope: Roles, resources, Metadatone mutation, inactive targets, all
permission variants, generic authorization denials, and the full authorization
matrix.
Actors and permissions: Admin creates the group, accepts both memberships, and
grants the manager `group.permission.manage`. The manager grants and revokes
the target's `resource.create` permission.
Fixtures and ownership: Create one unique group and two unique users owned by
this flow. Use opaque user IDs returned by the API. Do not mutate Metadatone
data. No cleanup is required.
Setup: Run against the existing stack with AAI and Gateway available, including
the string `userId` Gateway fix.
HTTP flow: Admin `POST /groups`; AAI registration and token issuance for the
manager and target; both users request membership with `{"roles": []}`; admin
accepts both; admin grants the manager `group.permission.manage`; manager reads
`GET /users/{id}` with `id="self"`; manager grants target `resource.create`
twice; target reads its profile; manager revokes `resource.create` twice; target
reads its profile again.
Assertions: Validate every Gateway response through `GatewayClient`; assert
active memberships, no roles, manager direct permission, target direct grant
under membership `permissions` rather than `global_permissions`, and removal
after revocation.
Negative cases: Repeated grant and repeated revoke both return idempotent
`204` responses.
Verification: `make test-contract`.

### test_authorization.py

Status: approved
Purpose: Verify authorization for every Gateway operation defined by
`lifecycle.md`, including public, optional-auth, protected, global, group, and
resource access.
In scope: Cover all 56 OpenAPI operations with valid request bodies and valid
lifecycle state; use least-privilege allowed actors and active denied actors;
verify fully public operations ignore supplied credentials; verify optional-auth
operations allow no-token public access but reject invalid and blocked bearer
tokens; verify protected operations reject missing tokens and unauthorized
active actors.
Out of scope: New Go unit tests, Hurl changes, database cleanup, hidden Turtle
import setup, Schemathesis state-machine generation, and notification
subscription operations not exposed by Gateway.
Actors and permissions: Use global `ADMIN` only for admin-only operations.
Use the narrow global permission, group permission, or group role required by
each operation. Use active users without the required access as denied actors.
Use a blocked user only for optional-auth token rejection and fully public
token-agnostic read checks.
Fixtures and ownership: Each test method owns its users, memberships, grants,
groups, categories, organizations, contact points, and resources. Register
secondary users through AAI and keep credentials in memory. Use separate unique
dataset/version fixtures for resource operations requiring different lifecycle
states. Leave local data in place; no cleanup is required.
Setup: Run against the existing stack after `make populate-metadatone`; use
`GatewayClient` for every Gateway request and the existing AAI helpers for
secondary users.
HTTP flow: `test_public_and_optional_read_authorization` covers fully public
group/category/organization reads, anonymous discovery, active-member private
discovery and resource reads, and blocked/invalid optional bearer tokens.
`test_user_directory_authorization` covers list/read users, global permission
grant/revoke, blocking/unblocking, and global role assignment/removal.
`test_canonical_entity_authorization` covers group, category, organization,
and contact-point writes while public reads remain covered separately.
`test_membership_and_access_authorization` covers membership roster and request
operations, role requests and decisions, direct role changes, and direct group
permission changes. `test_resource_authorization` covers all dataset resource
creation, copying, draft editing/discarding, version reads/lists, lifecycle
transitions, and draft creation operations. `test_turtle_import_authorization`
covers admin success and non-admin denial for Turtle import.
Assertions: Validate every Gateway response through `GatewayClient`; assert
allowed operations succeed, denied operations return schema-validated `403`
with `title: "forbidden"` and `detail: "operation is not permitted"`, missing
tokens return schema-validated `401`, fully public reads succeed regardless of
supplied credentials, and optional-auth invalid/blocked tokens are rejected.
Use valid lifecycle state so authorization is evaluated before validation or
transition conflicts.
Negative cases: Use the same valid request and state for allowed and denied
actors. Include blocked-token rejection for optional-auth reads and
token-agnostic behavior for fully public reads.
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
