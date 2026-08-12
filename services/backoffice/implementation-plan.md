# Backoffice Implementation Roadmap

## Purpose

`lifecycle.md` is authoritative requirements source. This roadmap tracks outcome-based implementation slices. It is not a fixed task list: detail only active slice, update later slices when implementation exposes constraints or better boundaries.

Current implementation is undeployed. Replace abandoned baseline schema and behavior directly. Once deployed, freeze applied migrations and use forward migrations only.

## Workflow

For each slice:

1. Write short execution plan from `lifecycle.md` requirements.
2. Implement domain, persistence, API, and tests needed for slice.
3. Run slice verification.
4. Record decisions, deviations, and follow-up work here.
5. Mark complete only when acceptance criteria pass.

## Roadmap

| Slice                   | Status   | Outcome                                                                       | Depends on                                                |
| ----------------------- | -------- | ----------------------------------------------------------------------------- | --------------------------------------------------------- |
| Foundation              | Complete | Target baseline schema and generated persistence types.                       | None                                                      |
| Authorization kernel    | Complete | Stored-resource authorization using fixed roles, grants, and user state.      | Foundation                                                |
| Dataset lifecycle       | Active   | Complete dataset lifecycle as first usable vertical slice.                    | Authorization kernel                                      |
| Software lifecycle      | Pending  | Same lifecycle for applications and source code.                              | Dataset lifecycle                                         |
| Identity administration | Pending  | Membership/role requests, grants, blocking, admin safety.                     | Authorization kernel                                      |
| Canonical metadata      | Pending  | Group/category, organization, and contact management with confirmed deletion. | Foundation, authorization kernel                          |
| Import alignment        | Pending  | Admin import through canonical and draft resource operations.                 | Dataset lifecycle, software lifecycle, canonical metadata |
| Notifications           | Pending  | Subscription storage, eligibility, defaults, and cleanup.                     | Authorization kernel, identity administration             |
| Completion pass         | Pending  | Required scenarios, API cleanup, fresh-environment validation.                | All slices                                                |

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
- Cover stored-resource group lookup, inactive denial, role/grant decisions, and contact-point eligibility with focused tests.

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
- Focused authorization tests and `make test`, `make build`, and `make lint` pass.

## Active Slice: Dataset Lifecycle

### Goal

Deliver dataset lifecycle as first usable vertical slice using stored-resource authorization.

### Execution Plan

- Expose logical-dataset action routes with trusted `X-EPOS-User-ID`; no actor identity appears in request JSON.
- Persist complete draft documents atomically, including version-owned metadata and canonical references.
- Enforce lifecycle transitions, optimistic revision updates, atomic publication, cloning, copying, and discard semantics through stored-resource authorization.
- Allow anonymous reads of published datasets; return contact points only when contact-point eligibility passes.

### Implementation Record

- Added `POST /api/v1/datasets`, `GET /api/v1/datasets/{id}`, draft replacement, lifecycle action, draft-from-publication/archive, and copy routes. OpenAPI now documents this API instead of prototype routes.
- Commands require `X-EPOS-User-ID`. This header is trusted only because Backoffice is currently network-private; gateway-only transport protection is required before external exposure.
- Draft create/replace persist scalar metadata, identifiers, providers, contact points, categories, distributions, web services, web-service contacts, and parameters atomically. Canonical entities remain ID references.
- Public dataset reads redact dataset and web-service contact points. Authenticated reads include them only after active-membership or global-admin eligibility passes.
- `make generate`, `make test`, `make build`, and `make lint` pass. Fresh migration smoke remains pending until Tilt database is recreated for replaced baseline history; applying new baseline over current public schema correctly fails on existing types.

## Future Slice Completion Criteria

### Authorization Kernel

- Public, member, role, direct-grant, admin, inactive, and blocked decisions load ownership/group from stored resources.
- Caller-supplied ownership or group does not affect decision.
- Contact-point reads, including embedded resource contact points, require active membership in at least one group; public representations omit them.

### Dataset Lifecycle

- Create, edit, submit, withdraw, reject, reopen, publish, discard, unpublish, draft-from-publication, and copy satisfy lifecycle scenarios.
- Revision conflicts and atomic publication are tested.

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

- Every `lifecycle.md` required scenario has focused automated coverage.
- OpenAPI and gateway-generated client match implemented API.
- Fresh database and service restart smoke succeed.
