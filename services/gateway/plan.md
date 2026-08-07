# Backoffice Dataset Lifecycle Plan

## Scope

Backoffice becomes authoritative store and lifecycle service for `dataset` resources.

This milestone includes:

- Dataset detail reads.
- Create draft.
- Create draft from published dataset.
- Replace draft.
- Hard-delete draft.
- Submit draft.
- Publish submitted dataset.
- Discard draft or submitted dataset.
- Gateway routing, authentication context, authorization, error mapping, and tests.

This milestone does not include:

- Search migration. Keep current gateway search behavior unchanged.
- Migration of organization, distribution, webservice, software, concept, or concept scheme resources.
- Metadata-lifecycle retirement. Retire it only after all remaining consumers migrate.
- New queues, events, or asynchronous processing.
- GitHub workflow changes without explicit approval.

## Decisions

- Backoffice owns dataset data, versions, lifecycle state, and authorization checks.
- Gateway derives actor identity from bearer token. Client cannot override actor.
- Client selects group through required `X-Group` UUID header.
- Backoffice trusts gateway as internal caller, but revalidates actor, group, membership, role, ownership, and state.
- Client supplies conceptual dataset ID when creating a new dataset.
- Backoffice generates immutable UUID for every dataset version.
- Dataset has one active draft or submitted version at any time, globally per conceptual dataset.
- All active group members can read that dataset draft/submitted version.
- Group admins can read and manage every non-published version in their group.
- Super admins can operate across groups.
- Editors create, edit, submit, delete, and discard drafts.
- Reviewers publish submitted datasets and can discard submitted datasets.
- A stale submitted branch may publish. Publication archives whichever version is currently published.
- `DELETE` hard-deletes only `draft` versions.
- `discard` preserves snapshot and audit data, changing state to `discarded`.
- Published dataset reads remain publicly readable.
- Dataset search remains on current implementation for now.
- Store complete validated dataset payload as typed JSONB snapshot, with relational lifecycle columns.

## Current Gaps

- Gateway documents full lifecycle API, but only `POST .../{id}/draft` is routed.
- Gateway draft handler sends placeholder actor and group values.
- Backoffice has no dataset queries, service, handlers, routes, or dataset API.
- Existing backoffice dataset schema is mutable, dataset-centric, and cannot model immutable version IDs or shared draft uniqueness.
- Existing migration has enum casing drift: uppercase enum values and lowercase default.
- Existing schema uses `REJECTED`; gateway lifecycle uses `DISCARDED`.
- Gateway dataset detail reads from metadata-lifecycle.
- Gateway generated backoffice client is named `usergroupclient`.
- Gateway OpenAPI and runtime behavior have status/error mismatches.

## Phase 1: Freeze API Contracts

- [ ] Define internal backoffice dataset API under `/api/v2/datasets`.
- [ ] Add typed dataset request, response, lifecycle result, and error contracts.
- [ ] Define `X-Actor-ID` as gateway-derived internal actor header.
- [ ] Define `X-Group` as required UUID for dataset mutations and private reads.
- [ ] Remove `X-Agent` from gateway dataset write contract, or mark it ignored and document gateway derivation.
- [ ] Define conceptual ID versus generated version UUID behavior.
- [ ] Define ID resolution:
  - [ ] Version UUID resolves exact version.
  - [ ] Conceptual ID resolves current published version for public reads.
  - [ ] Conceptual ID resolves group dataset draft for private mutation/read operations.
  - [ ] Unknown or invisible private version returns `404`.
- [ ] Update gateway OpenAPI dataset schemas and lifecycle responses.
- [ ] Add missing `401`, `403`, `503`, and `504` responses.
- [ ] Decide final delete response, preferably `204 No Content`.
- [ ] Keep unsupported write types out of runtime until their backoffice models exist.

## Phase 2: Rebuild Backoffice Dataset Storage

Database can be reset, so correct the baseline migration directly before deployment.

- [ ] Replace dataset status enum with lowercase values:
  - [ ] `draft`
  - [ ] `submitted`
  - [ ] `published`
  - [ ] `archived`
  - [ ] `discarded`
- [ ] Remove `rejected` unless later contract adds explicit rejection behavior.
- [ ] Add `dataset_concepts`:
  - [ ] Stable conceptual text ID.
  - [ ] Creator user ID.
  - [ ] Creation timestamp.
- [ ] Add `dataset_versions`:
  - [ ] Generated UUID version ID.
  - [ ] Concept ID.
  - [ ] Base version ID.
  - [ ] Status.
  - [ ] Group ID.
  - [ ] Editor/owner user ID.
  - [ ] Complete JSONB snapshot.
  - [ ] Snapshot schema version.
  - [ ] Revision number.
  - [ ] Created/updated timestamps.
  - [ ] Submitted, published, archived, and discarded actor/timestamps.
- [ ] Add same-concept foreign key for `base_version_id`.
- [ ] Add JSON object, positive revision, and positive snapshot schema checks.
- [ ] Add unique active-version constraint:

  ```sql
  UNIQUE (concept_id)
  WHERE status IN ('draft', 'submitted')
  ```

- [ ] Add one-current-published constraint per concept.
- [ ] Make group and editor non-null for versions.
- [ ] Add lifecycle lookup indexes.
- [ ] Remove or stop using old mutable dataset association tables for version content.
- [ ] Keep independent tables only when required by later resource-type migrations.
- [ ] Add update timestamp trigger for versions.
- [ ] Regenerate sqlc models and queries.
- [ ] Verify fresh migration against PostgreSQL/PostGIS.

## Phase 3: Add Backoffice Transaction Boundary

- [ ] Add `PostgresDB.InTx` helper around `sqlc.Queries.WithTx`.
- [ ] Roll back on every service error.
- [ ] Preserve commit and rollback failures in returned errors.
- [ ] Lock concept row before version row for every lifecycle mutation.
- [ ] Use database constraints as race protection, not application checks alone.
- [ ] Convert unique/state constraint failures to typed `409` errors.

## Phase 4: Add Backoffice SQL Queries

Create focused dataset query file and regenerate sqlc.

- [ ] Create conceptual dataset.
- [ ] Get and lock concept.
- [ ] Insert dataset version.
- [ ] Get version by generated UUID.
- [ ] Resolve current published version by conceptual ID.
- [ ] Resolve the single shared active branch by concept.
- [ ] Get and lock version.
- [ ] Replace draft snapshot with expected revision.
- [ ] Transition draft to submitted.
- [ ] Transition draft/submitted to discarded.
- [ ] Archive current published version.
- [ ] Transition submitted to published.
- [ ] Hard-delete draft.
- [ ] Get dataset detail with visibility predicates.
- [ ] Add version history query for audit/debug support.

Queries must apply group/visibility constraints before returning private rows.

## Phase 5: Implement Backoffice Dataset Service

Add typed domain code under `services/backoffice/service`.

- [ ] Add `DatasetWrite`, `Dataset`, `DatasetVersion`, and `WriteResult` types.
- [ ] Add strict validation for dataset payload and nested fields.
- [ ] Normalize and validate actor/group context.
- [ ] Require active actor unless super admin policy explicitly allows otherwise.
- [ ] Require active membership for selected group unless super admin.
- [ ] Implement role checks:
  - [ ] Editor: create, replace own draft, submit own draft, delete/discard own draft.
  - [ ] Reviewer: read group drafts/submitted versions, publish submitted versions, discard submitted versions.
  - [ ] Admin: all operations within group.
  - [ ] Super admin: all groups.
- [ ] Enforce one shared active version per conceptual dataset, regardless of group.
- [ ] Ensure every active group member can read group draft/submitted version.
- [ ] Return `404` for private versions outside caller visibility.
- [ ] Add explicit sentinel errors for bad request, forbidden, not found, conflict, revision conflict, and dependency/storage failure.

## Phase 6: Implement Dataset Lifecycle Operations

### Create New Draft

- [ ] Validate conceptual ID and snapshot.
- [ ] Authorize editor/admin.
- [ ] Create concept.
- [ ] Insert generated version in `draft` state.
- [ ] Return generated version UUID and `draft`.
- [ ] Map duplicate concept or active branch to `409`.

### Create Draft From Published

- [ ] Resolve source version.
- [ ] Require source `published`.
- [ ] Authorize editor/admin in selected group.
- [ ] Lock concept.
- [ ] Reject any existing draft/submitted version for the concept with `409`, including one owned by another group.
- [ ] Copy complete snapshot.
- [ ] Set `base_version_id` to exact source version.
- [ ] Return new generated version UUID.

### Replace Draft

- [ ] Resolve the dataset's single shared draft.
- [ ] Require `draft` state.
- [ ] Require editor/admin membership in same group.
- [ ] Replace entire snapshot, removing omitted fields.
- [ ] Increment revision atomically.
- [ ] Support `If-Match` or equivalent optimistic concurrency check.
- [ ] Return `409` on stale revision or wrong lifecycle state.

### Delete Draft

- [ ] Resolve the dataset's single shared draft.
- [ ] Require `draft` state.
- [ ] Require editor/admin membership in same group.
- [ ] Hard-delete version.
- [ ] Delete concept if no versions remain.
- [ ] Reject submitted/published/archived/discarded versions with `409`.

### Submit

- [ ] Resolve the dataset's single shared draft.
- [ ] Require `draft` state.
- [ ] Require owning group editor/admin.
- [ ] Set `submitted`, actor, and timestamp atomically.
- [ ] Return `submitted` result.

### Publish

- [ ] Resolve exact submitted version.
- [ ] Require reviewer/admin membership in version group.
- [ ] Lock concept and target version.
- [ ] Archive current published version if present.
- [ ] Promote target to `published`.
- [ ] Record publishing actor and timestamp.
- [ ] Permit stale branch publication by design.
- [ ] Return `published` result.

### Discard

- [ ] Resolve the dataset's single shared draft/submitted version.
- [ ] Require owner, reviewer, admin, or super admin according to state.
- [ ] Require `draft` or `submitted`.
- [ ] Preserve snapshot and ancestry.
- [ ] Set `discarded`, actor, and timestamp.
- [ ] Return `discarded` result.

## Phase 7: Add Backoffice HTTP API

- [ ] Add dataset handlers.
- [ ] Parse `X-Actor-ID` and `X-Group`.
- [ ] Reject missing or malformed trusted context with `401`/`400`.
- [ ] Decode typed JSON bodies.
- [ ] Keep handlers thin.
- [ ] Map service errors consistently:
  - [ ] `400` validation.
  - [ ] `403` authorization.
  - [ ] `404` absent/invisible resource.
  - [ ] `409` lifecycle, uniqueness, or revision conflict.
  - [ ] `500` unexpected storage/invariant failure.
- [ ] Wire routes under `/api/v2/datasets`.
- [ ] Update embedded OpenAPI.
- [ ] Add backoffice module README and Dockerfile if required by repository standard.

## Phase 8: Regenerate and Refactor Gateway Client

- [ ] Generate `gen/backofficeclient` from backoffice OpenAPI.
- [ ] Do not edit generated files manually.
- [ ] Rename `NewUsergroup` to `NewBackoffice` or split identity and dataset adapters cleanly.
- [ ] Keep mutation caching disabled.
- [ ] Update gateway Makefile generation and clean targets.
- [ ] Update gateway dependency wiring in `main.go` and `server/server.go`.
- [ ] Keep auth/user provisioning behavior working while adding dataset client calls.

## Phase 9: Wire Gateway Dataset Routes

- [ ] Add create route and handler.
- [ ] Add replace route and handler.
- [ ] Add delete route and handler.
- [ ] Add submit route and handler.
- [ ] Add publish route and handler.
- [ ] Update existing draft-from-published route.
- [ ] Add discard route and handler.
- [ ] Require bearer authentication on every mutation.
- [ ] Read actor only from authenticated request context.
- [ ] Require and validate `X-Group`.
- [ ] Never forward client-provided actor identity.
- [ ] Map upstream backoffice statuses without collapsing all errors to `500`:
  - [ ] `400` -> gateway `400`.
  - [ ] `401` -> gateway `401`.
  - [ ] `403` -> gateway `403`.
  - [ ] `404` -> gateway `404`.
  - [ ] `409` -> gateway `409`.
  - [ ] timeout -> gateway `504`.
  - [ ] unavailable -> gateway `503`.
- [ ] Return canonical generated version ID, not requested conceptual ID.
- [ ] Ensure path type and body type both equal `dataset`.

## Phase 10: Move Dataset Detail Reads

- [ ] Add typed backoffice dataset detail adapter in gateway.
- [ ] Route only `dataset` detail reads to backoffice.
- [ ] Keep other resource detail reads on metadata-lifecycle.
- [ ] Map JSONB snapshot to gateway `api.Dataset`.
- [ ] Resolve conceptual ID to current published version for public reads.
- [ ] Resolve version UUID exactly for private reads.
- [ ] Enforce group-wide draft/submitted visibility.
- [ ] Keep anonymous published reads working.
- [ ] Remove dataset-specific metadata-lifecycle fan-out where snapshot already contains data.
- [ ] Do not alter `/v1/search` in this milestone.

## Phase 11: Tests

### Backoffice

- [ ] Typed payload validation.
- [ ] Role and membership matrix.
- [ ] Group-wide draft visibility.
- [ ] Admin cross-user access within group.
- [ ] Cross-group isolation.
- [ ] Super-admin access.
- [ ] One active branch per dataset globally.
- [ ] Concurrent branch creation race.
- [ ] Create and draft cloning.
- [ ] Full replace semantics.
- [ ] Revision conflict.
- [ ] Hard delete behavior.
- [ ] Submit/publish/archive transaction.
- [ ] Stale branch publish behavior.
- [ ] Discard audit retention.
- [ ] Every invalid state transition.
- [ ] Fresh database migration.
- [ ] Handler status and error mapping.

### Gateway

- [ ] All dataset lifecycle routes registered.
- [ ] Authentication required.
- [ ] Actor derived from token context.
- [ ] Group header required.
- [ ] Unsupported resource types rejected.
- [ ] Backoffice error/status propagation.
- [ ] Canonical generated version IDs returned.
- [ ] Dataset detail routes to backoffice only.
- [ ] Other resource details still route to metadata-lifecycle.
- [ ] Published anonymous read.
- [ ] Group-wide private read.
- [ ] Cross-group private read denied.
- [ ] Timeout/unavailable mapping.
- [ ] OpenAPI/runtime parity.

Run:

```text
services/backoffice: make generate
services/backoffice: make fmt
services/backoffice: make lint
services/backoffice: make test
services/backoffice: make build

services/gateway: make generate
services/gateway: make fmt
services/gateway: make lint
services/gateway: make test
services/gateway: make build

root: make fmt
root: make lint
root: make test
```

## Phase 12: Cutover and Retirement Gate

- [ ] Add backoffice to every required local/runtime deployment configuration.
- [ ] Verify gateway startup with backoffice URL configured.
- [ ] Verify dataset writes and reads without metadata-lifecycle.
- [ ] Keep metadata-lifecycle for non-dataset paths and search.
- [ ] Define migration/import strategy for any existing dataset data before production cutover.
- [ ] Migrate dataset search to backoffice in a later milestone.
- [ ] Migrate remaining resource types and reads in later milestones.
- [ ] Remove metadata-lifecycle search dependency.
- [ ] Remove metadata-lifecycle resource dependency.
- [ ] Remove metadata-lifecycle config, deployment, clients, fixtures, and tests.
- [ ] Retire metadata-lifecycle only after no runtime or test references remain.

## Implementation Order

1. Freeze API contracts and status semantics.
2. Correct and replace backoffice dataset schema.
3. Add transaction helper and SQL queries.
4. Implement typed dataset service and authorization.
5. Implement create/detail vertical slice.
6. Implement replace/delete.
7. Implement draft-from-published.
8. Implement submit/publish/discard.
9. Add backoffice HTTP routes and OpenAPI.
10. Regenerate gateway backoffice client.
11. Wire gateway lifecycle routes and error mapping.
12. Move gateway dataset detail reads.
13. Add deployment wiring and run verification.
14. Plan search migration.
15. Retire metadata-lifecycle after all consumers migrate.

## Key Files

Backoffice:

- `services/backoffice/db/migrations/0001_base_schema.sql`
- `services/backoffice/db/queries/datasets.sql`
- `services/backoffice/db/main.go`
- `services/backoffice/service/datasets.go`
- `services/backoffice/server/handlers/datasets.go`
- `services/backoffice/server/router.go`
- `services/backoffice/docs/openapi.yaml`

Gateway:

- `services/gateway/docs/openapi.yaml`
- `services/gateway/server/router.go`
- `services/gateway/server/handlers/lifecycle.go`
- `services/gateway/lifecycle/`
- `services/gateway/resources/dataset.go`
- `services/gateway/resources/service.go`
- `services/gateway/clients/`
- `services/gateway/server/server.go`
- `services/gateway/main.go`
- `services/gateway/Makefile`

Generated files must be regenerated, never hand-edited.
