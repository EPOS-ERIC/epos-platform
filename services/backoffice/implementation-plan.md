# Backoffice Implementation Roadmap

## Purpose

`lifecycle.md` is authoritative requirements source. This roadmap tracks outcome-based implementation slices. It is not a fixed task list: detail only active slice, update later slices when implementation exposes constraints or better boundaries.

Current implementation is undeployed. Replace abandoned baseline schema and behavior directly. Once deployed, freeze applied migrations and use forward migrations only.

## Workflow

For each slice:

1. Write short execution plan from `lifecycle.md` requirements.
2. Implement domain, persistence, API needed for slice.
3. Run slice verification.
4. Record decisions, deviations, and follow-up work here.
5. Mark complete only when acceptance criteria pass.

## Roadmap

| Slice                   | Status   | Outcome                                                                       | Depends on                                                |
| ----------------------- | -------- | ----------------------------------------------------------------------------- | --------------------------------------------------------- |
| Foundation              | Complete | Target baseline schema and generated persistence types.                       | None                                                      |
| Authorization kernel    | Complete | Stored-resource authorization using fixed roles, grants, and user state.      | Foundation                                                |
| Dataset lifecycle       | Active   | Complete dataset lifecycle as first usable vertical slice.                    | Authorization kernel                                      |
| Software lifecycle      | Active   | Same lifecycle for applications and source code.                              | Dataset lifecycle                                         |
| Identity administration | Active   | Membership/role requests, grants, blocking, admin safety.                     | Authorization kernel                                      |
| Canonical metadata      | Pending  | Group/category, organization, and contact management with atomic deletion.    | Foundation, authorization kernel                          |
| Dataset import          | Active   | Admin Turtle import into canonical metadata and dataset drafts.               | Dataset lifecycle                                         |
| Import alignment        | Pending  | Extend import through software drafts and canonical management operations.    | Dataset import, software lifecycle, canonical metadata    |
| Notifications           | Pending  | Subscription storage, eligibility, defaults, and cleanup.                     | Authorization kernel, identity administration             |
| Completion pass         | Pending  | Required scenarios, API cleanup, fresh-environment validation.                | All slices                                                |

## Agreed API Shape: Canonical Metadata

The API design below is the current target for the canonical metadata slice. Lifecycle
requirements remain authoritative unless an explicit deviation is recorded here.

### General Conventions

- Collection responses are unpaginated arrays. No list envelope, offset, limit, or cursor is required for this slice.
- Create requests may omit `source_iri`. Backoffice generates an opaque immutable URI. Responses always include `source_iri`.
- Resource `id` values are server-generated UUIDs.
- `PUT` is full replacement. All mutable fields are required. Omitted relation entries are removed; an empty array clears all relations.
- Immutable `id`, `source_iri`, and owning `group_id` are response fields and cannot be changed.
- Canonical deletes are direct, atomic operations and return `200` with deleted-entity information and impact counts.
- Canonical organization responses expose contact-point IDs publicly. Contact-point email and role, and expanded contact-point records, require contact-point read eligibility.

### Organizations

| Method | Path | Behavior | Required access |
| ------ | ---- | -------- | --------------- |
| `GET` | `/api/v1/organizations` | List complete organization documents. | Public |
| `POST` | `/api/v1/organizations` | Create canonical organization. | `organization.manage` |
| `GET` | `/api/v1/organizations/{organization_id}` | Read complete organization document. | Public |
| `PUT` | `/api/v1/organizations/{organization_id}` | Replace organization fields, identifiers, parent organization IDs, and contact-point IDs atomically. | `organization.manage` |
| `DELETE` | `/api/v1/organizations/{organization_id}` | Delete organization, owned identifiers, hierarchy links, and organization links atomically. | `organization.manage` |

Organization documents use `address: {street, locality, postal_code, country}` and
include `logo_url`, `page_url`, `email`, `lei_code`, `parent_organization_ids`,
`contact_point_ids`, and embedded owned `identifiers`. A full `PUT` replaces every
identifier and relation list. Organization hierarchy cycles are rejected.

### Contact Points

| Method | Path | Behavior | Required access |
| ------ | ---- | -------- | --------------- |
| `GET` | `/api/v1/contact-points` | List complete contact-point documents, unpaginated. | Active member of any group or `ADMIN` |
| `POST` | `/api/v1/contact-points` | Create canonical contact point. | `contact_point.manage` |
| `GET` | `/api/v1/contact-points/{contact_point_id}` | Read contact point. | Active member of any group or `ADMIN` |
| `PUT` | `/api/v1/contact-points/{contact_point_id}` | Replace source IRI, email, and role fields. | `contact_point.manage` |
| `DELETE` | `/api/v1/contact-points/{contact_point_id}` | Delete contact point and all organization, resource-version, and web-service links atomically. | `contact_point.manage` |

### Categories

Groups are category schemes. There is no separate category-scheme resource.

| Method | Path | Behavior | Required access |
| ------ | ---- | -------- | --------------- |
| `GET` | `/api/v1/groups/{group_id}/categories` | List the group category forest as a flat list. | Public |
| `POST` | `/api/v1/groups/{group_id}/categories` | Create category with optional `parent_category_id`. | `category.manage` in group |
| `GET` | `/api/v1/groups/{group_id}/categories/{category_id}` | Read category. | Public |
| `PUT` | `/api/v1/groups/{group_id}/categories/{category_id}` | Replace category name and optional parent. | `category.manage` in group |
| `DELETE` | `/api/v1/groups/{group_id}/categories/{category_id}` | Delete category and complete descendant subtree, then unlink every affected resource reference atomically. | `category.manage` in group |

Category records contain `id`, `group_id`, `parent_category_id`, `source_iri`, and
`name`. The service enforces immutable group ownership and source IRI, same-group
parents, acyclic hierarchy, and normalized sibling-name uniqueness.

### Groups And Category-Scheme Metadata

| Method | Path | Behavior | Required access |
| ------ | ---- | -------- | --------------- |
| `POST` | `/api/v1/groups` | Create group and category scheme. | Global `ADMIN` |
| `PUT` | `/api/v1/groups/{group_id}/category-scheme` | Replace group name, description, and all category-scheme metadata. | `category_scheme.manage` |
| `DELETE` | `/api/v1/groups/{group_id}` | Delete group, category scheme, categories, memberships, assignments, grants, requests, and subscriptions when no logical resource belongs to it. | Global `ADMIN` |

Existing `GET /api/v1/groups` and `GET /api/v1/groups/{group_id}` responses remain
complete public group/category-scheme representations. Group deletion is blocked
while any logical dataset, software application, or software source code belongs
to the group.

### Delete Response

Canonical deletes return `200` with counts rather than affected-resource IDs. The
response includes the deleted entity type and ID plus counts such as deleted
descendant categories, unlinked dataset versions, unlinked software versions,
web services, organization links, and owned identifiers.

### Intentional Lifecycle Deviation

The current API design uses direct client-requested deletes. The client obtains
the data needed for its warning from list/detail and resource-reference endpoints;
the backoffice does not expose a preview token or require exact inventory echo.
Deletion still recomputes and applies all unlinking atomically in one transaction.
The destructive-deletion requirements in `lifecycle.md` lines 97-109 must be
updated before this slice is marked complete: remove backend preview, exact
confirmation, and stale-inventory conflict requirements, and retain atomic
unlinking/deletion.

## Known Gaps In Existing Endpoints

These mismatches must be fixed during identity administration or the completion pass.

- Membership requests currently require existing active membership. Lifecycle allows any active user to request membership in a group.
- Own user detail omits pending role requests. It must expose pending role requests for the user's memberships.
- Published-resource listing is missing. Public listings must return current published versions grouped by logical resource group.
- Notification subscription endpoints are missing. Users must manage eligible own subscriptions; `ADMIN` may manage all subscriptions.
- Canonical organization, contact-point, category, category-scheme, and group management routes are missing and must follow the API shape above.
- OpenAPI has no canonical-management, published-listing, global-role, or notification-subscription operations.

## Identity Administration: Membership Request Step

### Implementation Record

- Initial membership requests now require only an active, unblocked user; existing group membership is not required.
- Requested `EDITOR` and `REVIEWER` roles are inserted against the new pending membership atomically. Subsequent role requests remain restricted to active members.
- User detail responses now include pending role requests grouped under their memberships.
- User-directory reads now check actor state before self-read and allow active global membership, role, or permission managers to read other users.
- Global `ADMIN` assignment and removal now use serialized transactions and reject removal of the last active administrator.
- Gateway now exposes the documented authenticated `POST /groups/{groupId}/membership-requests` route and forwards role lists to Backoffice.
- User block/unblock now persists block reason, actor, and time, serializes active-admin safety with global role changes, and leaves dormant memberships, roles, grants, and subscriptions stored.
- Backoffice and Gateway builds pass. Gateway lint passes. Backoffice lint remains blocked by the pre-existing `identity.Store` godoclint error in `identity/service.go:28`; no tests were added or run per request.

## Completed Slice: Foundation

### Goal

Replace undeployed prototype data model with baseline required by `lifecycle.md`. No user-facing lifecycle commands yet.

### Scope

- Replace abandoned baseline migrations; do not add corrective migrations for abandoned schema.
- Model logical datasets, software applications, and software source codes plus their version tables.
- Add lifecycle status, revision, lifecycle actor/timestamps, rejection fields, and uniqueness constraints.
- Model version-owned identifiers and relation tables.
- Move category-scheme fields into `groups`; make categories group-owned with one nullable parent.
- Add fixed authorization, membership-role, grant, request, user-block, and notification schema.
- Remove schema structures contradicted by `lifecycle.md`, including legacy super-admin authority, special group behavior, category-scheme table, multi-parent category links, and shared identifier links.
- Update SQL query sources and regenerate SQLC. Generated `db/sqlc` files are never edited manually.

### Out Of Scope

- Lifecycle command handlers and HTTP routes.
- Authorization decision engine.
- Data migration from old model.
- Turtle import behavior.
- Gateway client/OpenAPI changes.

### Acceptance Criteria

- Fresh PostgreSQL/PostGIS database applies baseline migrations successfully.
- Schema encodes resource identity/version split, one active change, one publication, immutable ownership fields, exactly-one identifier owner, and same-group category links.
- Schema encodes category tree constraints and fixed role/permission scopes.
- `make generate` produces expected SQLC output.
- `make test`, `make build`, and `make lint` pass in `services/backoffice`.

### Verification

```sh
make generate
git diff -- db/sqlc
make test
make build
make lint
```

Use fresh disposable database for schema smoke. Do not remove existing local database volume without explicit approval.

### Implementation Record

- Replaced abandoned `0001` baseline with ordered dependency-focused migrations: foundation identity/groups, canonical metadata, logical resources/versions, version-owned metadata, authorization/notifications, and invariants/triggers.
- Database constraints and triggers enforce active/published version limits, category ownership/tree rules, fixed role and permission scopes, immutable ownership fields, and global logical-resource source IRI uniqueness.
- Removed prototype metadata import/read persistence operations. They cannot represent versioned resource semantics and return an explicit unavailable error until their scheduled lifecycle slices.
- Retained current identity HTTP/query surface as a temporary compatibility facade; it no longer persists `is_super_admin`, system groups, or a single membership role. Authorization kernel replaces that surface.
- `make generate`, `make test`, `make build`, and `make lint` pass in `services/backoffice`.
- Fresh migration validation applies all six baseline migrations and rolls them back inside a transaction. Local Tilt database must be recreated before it can apply this replaced migration history.

## Completed Slice: Authorization Kernel

### Execution Plan

- Remove prototype authorization, identity CRUD, import, and metadata-read routes.
- Provide internal-only authorization decisions. Existing resource decisions load group from stored logical resource; creation authorizes requested target group before resource exists.
- Apply active user, global `ADMIN`, global grants, active membership, fixed-role, and direct group-grant decision order.
- Provide contact-point eligibility decision for active members of any group or global `ADMIN`.
- Cover stored-resource group lookup, inactive denial, role/grant decisions, and contact-point eligibility

### Acceptance Criteria

- No HTTP authorization endpoint accepts caller-supplied group or ownership.
- Existing-resource decisions load group from stored logical resource.
- Global `ADMIN`, global grants, active membership, fixed roles, and direct grants follow `lifecycle.md` decision order.
- Inactive and blocked users are denied authenticated decisions.
- Contact-point eligibility requires active membership in at least one group unless user is global `ADMIN`.

### Implementation Record

- Removed prototype HTTP authorization and identity CRUD routes. Authorization is internal and enforced by future Backoffice resource handlers.
- Existing-resource decisions load ownership group from stored logical resource rows. `resource.create` alone authorizes caller-supplied target group before resource exists.
- Internal decisions distinguish global and group permission scopes, apply global `ADMIN` and direct global grants, combine fixed roles with direct group grants, and deny inactive or blocked users.
- Public reads allow published resources without identity; non-public reads require global `ADMIN` or active membership in stored resource group. Contact-point reads require global `ADMIN` or active membership in any group.
-  `make test`, `make build`, and `make lint` pass.

## Active Slice: Dataset Lifecycle

### Goal

Deliver dataset lifecycle as first usable vertical slice using stored-resource authorization.

### Execution Plan

- Expose logical-dataset action routes with trusted `X-EPOS-User-ID`; no actor identity appears in request JSON.
- Persist complete draft documents atomically, including version-owned metadata and canonical references.
- Enforce lifecycle transitions, optimistic revision updates, atomic publication, cloning, copying, and discard semantics through stored-resource authorization.
- Allow anonymous reads of published datasets; return contact points only when contact-point eligibility passes.

### Implementation Record

- Added shared `/api/v1/resources/{resource_type}` routes for dataset lifecycle actions. Resource type selects the existing typed handler and accepts `dataset`, `software-application`, or `software-source-code`.
- Commands require `X-EPOS-User-ID`. This header is trusted only because Backoffice is currently network-private; gateway-only transport protection is required before external exposure.
- Draft create/replace persist scalar metadata, identifiers, providers, contact points, categories, distributions, web services, web-service contacts, and parameters atomically. Canonical entities remain ID references.
- Public dataset reads redact dataset and web-service contact points. Authenticated reads include them only after active-membership or global-admin eligibility passes.
- Fixed draft-from-publication/archive routing, required absence of a current publication before cloning the latest archive, and made command responses reload complete persisted metadata.
- Dataset persistence now binds generated web services to their owning distributions by position and translates PostgreSQL constraint/trigger failures into lifecycle conflicts.
- Fixed clone queries for datasets and software resources: PostgreSQL data-modifying CTE results must be returned from their `inserted` CTE, not re-read from base table in same statement.
- Fresh Tilt PostgreSQL applied migrations `0001` through `0006`. HTTP smoke covered dataset, application, and source-code create, submit, publish, anonymous published read, draft-from-publication, and draft discard. `make generate`, `make test`, `make build`, and `make lint` pass.

### Remaining Verification

- Focused database-backed lifecycle acceptance scenarios are deferred while the API and model remain fluid.

## Active Slice: Dataset Import

### Goal

Allow global administrators to atomically import canonical metadata and dataset drafts from Turtle.

### Implementation Record

- Added global-admin-only `POST /api/v1/imports/turtle` with atomic canonical metadata and dataset draft persistence.
- Category schemes map to groups. Dataset categories must come from exactly one scheme, which determines immutable resource ownership.
- Reimport replaces existing `DRAFT` content by source IRI, creates a draft beside a publication when no active change exists, and conflicts with submitted or rejected changes. Import never publishes.
- Typed standalone category schemes, categories, organizations, and contact points are imported even when no resource references them.
- Organization `schema:memberOf` hierarchy is replaced atomically. Category parsing rejects multiple parents and hierarchy cycles before persistence.
- PostgreSQL constraint and invariant-trigger failures return import conflicts instead of internal failures.
- Software application and source-code drafts are imported atomically with canonical references and version-owned contact-point links.
- `make generate`, `make test`, `make build`, and `make lint` pass.
- Fresh Tilt PostgreSQL Turtle smoke imported `packages/assets/metadatone.ttl`: 5 groups, 8 categories, 6 organizations, 6 contact points, 3 dataset drafts, 1 application draft, and 1 source-code draft.

### Remaining Verification

- Focused database-backed import atomicity and lifecycle acceptance scenarios are deferred while the import model remains fluid.

## Active Slice: Software Lifecycle

### Goal

Apply the resource lifecycle to software applications and software source code using stored-resource authorization.

### Implementation Record

- Added typed application and source-code lifecycle services with create, exact-version read, draft replacement, submit, withdraw, reject, reopen, publish, discard, unpublish, draft-from-publication/archive, and copy operations.
- Existing-resource actions authorize against each logical software resource's stored group. Exact-version reads allow public `PUBLISHED` versions and restrict non-public versions to active group members or global administrators.
- Added atomic PostgreSQL persistence for lifecycle transitions, optimistic revisions, publication/archive replacement, cloning, copying, creators, identifiers, and canonical category links.
- Same-resource drafts clone metadata, creators, identifiers, and categories. Copies create independent logical resources with new source IRIs and zero categories.
- Added shared resource HTTP routes and OpenAPI schemas for applications and source code. Typed handlers remain separate behind the resource dispatcher.
- Software Turtle import now persists application and source-code drafts atomically.
- Fresh Tilt PostgreSQL lifecycle smoke covered application and source-code create, submit, publish, anonymous published read, draft-from-publication, and draft discard. `make generate`, `make test`, `make build`, and `make lint` pass.

### Remaining Verification

- Focused database-backed software lifecycle acceptance scenarios are deferred while the API and model remain fluid.

## Future Slice Completion Criteria

### Authorization Kernel

- Public, member, role, direct-grant, admin, inactive, and blocked decisions load ownership/group from stored resources.
- Caller-supplied ownership or group does not affect decision.
- Contact-point reads, including embedded organization and resource contact points, require active membership in at least one group; public representations return null contact-point fields.

### Dataset Lifecycle

- Create, edit, submit, withdraw, reject, reopen, publish, discard, unpublish, draft-from-publication, and copy satisfy lifecycle scenarios.

### Dataset Import

- Global `ADMIN` may atomically import Turtle category schemes, categories, organizations, contact points, and dataset drafts.
- Category schemes create or update groups. Each dataset must reference categories from exactly one imported scheme, which determines immutable ownership group.
- Existing `DRAFT` content is replaced by source IRI; an existing submitted or rejected change conflicts. Import never publishes resources.
- Software metadata imports use same one-active-change and draft-only rules as datasets.

### Software Lifecycle

- Applications and source code use same transitions and permissions.
- Draft/copy cloning handles metadata, creators, identifiers, and categories.

### Identity Administration

- Membership and role request flows, `GROUP_ADMIN` restrictions, blocking, and last-active-admin protection satisfy required scenarios.

### Canonical Metadata

- Category management enforces group tree and same-group references.
- Organization/contact/category deletion uses preview, exact confirmation, transaction-time recomputation, and atomic unlinking.

### Import Alignment

- Authenticated global `ADMIN` import creates or updates canonical data and resource drafts only.
- Import obeys lifecycle and ownership invariants atomically.

### Notifications

- Subscription eligibility, defaults, opt-out, cleanup, and send-time rechecks satisfy lifecycle requirements.

### Completion Pass

- OpenAPI and gateway-generated client match implemented API.
- Fresh database and service restart smoke succeed.
