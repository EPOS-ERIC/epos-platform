# Metadata Lifecycle and Authorization

## Scope

This document defines dataset and software versioning, linked-entity governance, group membership, categories, authorization, and notification subscriptions for metadata lifecycle.

It does not define HTTP endpoints, UI layouts, migration order, or notification event delivery.

## Terms

| Term                    | Meaning                                                                                               |
| ----------------------- | ----------------------------------------------------------------------------------------------------- |
| Logical dataset         | Stable UUID shared by all versions and owned by one group.                                            |
| Dataset version         | Editable or historical metadata for one logical dataset.                                              |
| Logical software        | Stable UUID shared by all versions of one software application or source code and owned by one group. |
| Software version        | Editable or historical metadata for one logical software resource.                                    |
| Active change version   | Single `DRAFT`, `SUBMITTED`, or `REJECTED` version for one logical resource.                          |
| Group                   | Category ownership, resource ownership, and authorization boundary stored in one `groups` row.        |
| Concept                 | Named classification containing groups; each group belongs to exactly one concept.                    |
| Category                | Node in a group's category hierarchy. A subcategory has a parent category.                            |
| Canonical category      | Shared live category persisted in its owning group.                                                   |
| Canonical organization  | Shared live organization managed directly in metadata lifecycle without lifecycle states.             |
| Canonical contact point | Shared live contact record managed directly in metadata lifecycle without lifecycle states.           |
| Identifier              | Value owned by exactly one resource version or organization; never shared.                            |
| Group member            | Active, unblocked user with `ACTIVE` membership in group and access to its categories.                |
| Granted permission      | Additive permission assigned outside a role.                                                          |

## Resource Lifecycle

Datasets, software applications, and software source codes use same lifecycle. A resource means one of those three types. Dataset-specific nested metadata is defined separately below.

### States

| State       | Mutable | Public | Meaning                                           |
| ----------- | ------: | -----: | ------------------------------------------------- |
| `DRAFT`     |     Yes |     No | Group editors are preparing resource changes.     |
| `SUBMITTED` |      No |     No | Waiting for reviewer decision.                    |
| `REJECTED`  |      No |     No | Reviewer rejected submission with feedback.       |
| `PUBLISHED` |      No |    Yes | Current public version.                           |
| `ARCHIVED`  |      No |     No | Replaced or administratively unpublished version. |

### Transitions

| Action                        | From                | To          | Required access                              | Rules and effects                                                                                       |
| ----------------------------- | ------------------- | ----------- | -------------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Create empty draft            | None                | `DRAFT`     | `group.resource.create` in group             | Creates logical resource with a generated UUID and its first version.                                   |
| Create draft from publication | `PUBLISHED`         | New `DRAFT` | `group.resource.create` in resource group    | Clones version-owned metadata and canonical links.                                                      |
| Create draft after unpublish  | Latest `ARCHIVED`   | New `DRAFT` | `group.resource.create` in resource group    | Performs same clone when logical resource has no publication.                                           |
| Copy published resource       | `PUBLISHED`         | New `DRAFT` | `group.resource.create` in target group      | Creates independent logical resource with a generated UUID; clones content and links except categories. |
| Edit                          | `DRAFT`             | `DRAFT`     | `group.resource.edit` in resource group      | Any authorized group member may edit. Requires current revision.                                        |
| Submit                        | `DRAFT`             | `SUBMITTED` | `group.resource.submit` in resource group    | Version becomes immutable.                                                                              |
| Withdraw                      | `SUBMITTED`         | `DRAFT`     | `group.resource.withdraw` in resource group  | Removes submission from review.                                                                         |
| Reject                        | `SUBMITTED`         | `REJECTED`  | `group.resource.reject` in resource group    | Non-empty rejection note required. Latest note replaces previous note.                                  |
| Reopen                        | `REJECTED`          | `DRAFT`     | `group.resource.reopen` in resource group    | Makes version editable; latest rejection note remains visible.                                          |
| Publish                       | `SUBMITTED`         | `PUBLISHED` | `group.resource.publish` in resource group   | Self-publication allowed. Existing publication becomes `ARCHIVED`; rejection fields are cleared.        |
| Discard                       | `DRAFT`, `REJECTED` | Deleted     | `group.resource.discard` in resource group   | Permanently deletes active change version. Deletes logical resource too when no versions remain.        |
| Unpublish                     | `PUBLISHED`         | `ARCHIVED`  | `group.resource.unpublish` in resource group | No reason required. Logical resource remains available for a future draft.                              |

### Invariants

| Rule                                               | Enforcement                                                                                              |
| -------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| One active change version per logical resource     | Unique constraint across `DRAFT`, `SUBMITTED`, and `REJECTED`.                                           |
| At most one published version per logical resource | Unique constraint on `PUBLISHED`.                                                                        |
| Group is immutable                                 | Stored on logical resource; copy creates new logical resource rather than moving original.               |
| Logical resource UUID is immutable                 | Stored on logical resource.                                                                              |
| Submitted and historical versions are immutable    | Lifecycle transitions may change status; confirmed canonical deletion may remove links from any version. |
| Group edits cannot silently overwrite each other   | Draft updates use optimistic revision checks; stale writes fail with conflict.                           |
| Publishing is atomic                               | Old-version archival and new publication commit together.                                                |

Draft creation for an existing logical resource fails with conflict when an active change version exists. `group.resource.create` is authorized against requested group before an empty logical resource exists; every later action uses stored logical-resource group. A matching global resource permission grants the same action in any group without membership. Latest archive means greatest `archived_at`, with version ID as deterministic tie-breaker.

Published resource may be copied into same or different group. Source is public, so source-group membership is not required; caller needs `group.resource.create` or `global.resource.create` in target group. Copy creates new logical resource with a generated UUID and first `DRAFT`; original remains unchanged. Copy clones descriptive metadata, identifiers, and organization links. Dataset copies also clone distributions, web services, and contact-point links. Identifier rows receive new internal IDs and preserve type and value. Target draft has zero categories.

Published resource lists return current `PUBLISHED` versions grouped by logical resource group. Published versions, canonical organizations, groups, and canonical categories are publicly readable without authentication. Public group and category representations expose canonical metadata but never memberships, roles, grants, or role requests. Contact points are excluded from public resource representations because they contain email addresses. Any active group member may read every non-public version in that group. Global `ADMIN` may read everything.

Software application and software source code are independent resource types. Their versions contain type-specific metadata plus creator, identifier, contact-point, and category links; they do not reference each other, distributions, or web services.

### Web-service defaults

Every web-service configuration must produce an executable default request without caller-supplied values. Required scalar parameters, including template children and body parameters, must have defaults; active templates must render using those defaults. Dynamic services also require defaults for longitude and latitude. Callers may supply values when accessing a service, subject to its parameter constraints.

## Metadata Ownership

| Entity                           | Editor use                                     | Editor create/propose | Management                                          | Reference behavior                                                                                                                                        |
| -------------------------------- | ---------------------------------------------- | --------------------- | --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Dataset metadata                 | Edit in `DRAFT`                                | Yes                   | Through lifecycle                                   | Copied per version.                                                                                                                                       |
| Distribution                     | Edit in `DRAFT`                                | Yes                   | Through lifecycle                                   | Owned by version.                                                                                                                                         |
| Web service                      | Edit in `DRAFT`                                | Yes                   | Through lifecycle                                   | Owned by version.                                                                                                                                         |
| Software application/source code | Edit in `DRAFT` and use own group's categories | Yes                   | Through resource lifecycle                          | Copied per version; creators, contact points, and categories are version-owned links. Identifiers are version-owned records cloned with new internal IDs. |
| Canonical organization           | Select existing organization                   | No                    | `global.organization.manage`                        | Shared live reference.                                                                                                                                    |
| Canonical contact point          | Select existing contact point                  | No                    | `global.contact_point.manage`                       | Shared live reference.                                                                                                                                    |
| Identifier                       | Edit through owning entity                     | Yes with owner        | Through owner                                       | Exactly one owner; cloned rather than shared.                                                                                                             |
| Group                            | Use through group membership                   | No                    | `global.group.manage`                               | Shared live group metadata.                                                                                                                               |
| Canonical category               | Select from owning group                       | No                    | `group.category.manage` or `global.category.manage` | Shared live reference; category cannot move to another group.                                                                                             |

Canonical organization, contact-point, group, and canonical-category changes appear immediately in every referencing version. No snapshots or history are required. Resource version category links retain only canonical category IDs; they never copy category fields.

Concepts are seeded as `Data`, `Facilities`, and `Software` and managed with `global.concept.manage`. Their UUID and name are independent of resource types. A concept cannot be deleted while any group references it.

Canonical organizations, contact points, categories, and groups may be deleted directly. The client is responsible for obtaining impact data, presenting any warning or preview to the user, and requesting deletion after confirmation. Metadata Lifecycle does not expose a separate preview or confirmation operation. Group deletion is blocked only while group owns logical dataset or software resource.

### Canonical Deletion

Canonical deletion is a direct atomic operation returning `204 No Content`. Clients manage any impact preview before requesting deletion. The metadata lifecycle unlinks every reference and deletes the canonical entity in one transaction. Deletion remains an explicit exception to version immutability and may change public and historical representations.

Category deletion removes selected category, complete descendant subtree, all hierarchy links, and all dataset/software links to any category in subtree. Organization deletion removes its resource and hierarchy links, deletes its owned identifiers, then deletes organization; related organizations remain. Contact-point deletion removes all dataset-version, software-version, and web-service links, then deletes contact point.

Group deletion fails while any logical dataset, software application, or software source code belongs to group. Otherwise it deletes the group, canonical categories, memberships, group role assignments, grants, requests, and group notification subscriptions.

### Organizations

Organizations are canonical live entities with no draft, submission, review, publication, archival, or rejection lifecycle. In system-managed mode, they are managed directly in metadata lifecycle.

| Action             | Required access              | Effect                                                         |
| ------------------ | ---------------------------- | -------------------------------------------------------------- |
| Read/select        | Public                       | Uses current canonical organization.                           |
| Create/edit        | `global.organization.manage` | Change becomes visible immediately in every reference.         |
| Manage identifiers | `global.organization.manage` | Creates, edits, and deletes identifiers owned by organization. |
| Manage hierarchy   | `global.organization.manage` | Adds or removes parent-child organization links.               |
| Delete             | `global.organization.manage` | Requires destructive-deletion impact warning and confirmation. |

Dataset-version providers and software-version creators may reference multiple organizations. Version-owned web service may reference zero or one provider organization. All relations use local organization UUID foreign keys.

Organization hierarchy allows multiple parents. Direct self-links and indirect cycles are forbidden.

In synced mode, Gateway periodically mirrors remote organizations by NID, preserving local UUIDs and replacing parent links. Gateway rejects organization create, edit, and delete requests. Applying a snapshot is transactional and deletes missing synced organizations and their links; failed fetches do not apply a snapshot.

### Contact Points

Contact points are canonical live entities managed directly in metadata lifecycle. They have UUID, email, and role, and no lifecycle states.

| Action      | Required access               | Effect                                                                                |
| ----------- | ----------------------------- | ------------------------------------------------------------------------------------- |
| Read/select | Active group member           | Requires active membership in at least one group; email is sensitive data.            |
| Create/edit | `global.contact_point.manage` | Change becomes visible in every dataset, software-version, and web-service reference. |
| Delete      | `global.contact_point.manage` | Requires destructive-deletion impact warning and confirmation.                        |

Dataset version, software version, and version-owned web service may each reference zero or more contact points. Duplicate contact-point link on same owner is forbidden. Editors may change resource contact-point links inside draft but cannot create or edit canonical contact points. Contact points, including those referenced by otherwise public resources, are returned only to an active, unblocked user with active membership in at least one group. Global `ADMIN` may read them without membership.

### Identifiers

Identifier is owned value, not shared canonical entity. Dataset version, organization, software-application version, or software-source-code version may own zero or more identifiers. Each identifier has exactly one owner and owner never changes.

Creating draft clones published/archived version identifiers. Cross-group resource copy also clones identifiers. Every clone receives new internal ID while preserving type and value. Identical identifier content is allowed under different owners. Editing identifier affects only owner. Deleting owner deletes owned identifiers.

### Categories

A group's category hierarchy is a tree or forest. Each canonical category has zero or one parent in same group and cannot participate in cycle. Category names are compared after trimming whitespace and case folding. Name must be unique among siblings; top-level name must be unique among top-level categories. Same normalized name may exist in different branches.

Categories are managed outside resource lifecycle. An active group member with `group.category.manage` in category's group may create, edit, reparent, or delete canonical categories in that group. A user with `global.category.manage` may perform the same actions in any group without membership. Editors may select only existing canonical categories from the resource's group while editing a draft; they cannot create or propose categories through a resource. `global.group.manage` manages groups and does not grant category management.

Canonical category has stable UUID. Its group is immutable. Category changes take effect immediately for every dataset version and software resource that references category. Category deletion remains a confirmed destructive operation.

### Category Schema Requirements

The `categories` table stores `id` as stable UUID, `group_id` as direct owning-group foreign key, nullable `parent_category_id`, and `name`. `parent_category_id` must reference a category in same `group_id`; this represents a tree or forest without separate hierarchy-link table.

Resource-version category-reference tables store canonical `category_id` values only. They must enforce that referenced category's `group_id` equals owning resource's `group_id`. They do not copy category name, parent, or other category fields.

## Authorization Model

Authorization uses fixed composable roles plus additive granted permissions. Default is deny. Explicit deny grants are not supported.

Permissions have one of these scopes:

| Scope             | Meaning                              | Example                  |
| ----------------- | ------------------------------------ | ------------------------ |
| Global role       | Applies across system                | `ADMIN`                  |
| Global permission | Applies across system                | `global.category.manage` |
| Group role        | Applies within one active membership | `EDITOR`                 |
| Group permission  | Applies within one active membership | `group.category.manage`  |

### Fixed Roles

| Role          | Scope  | Included permissions                                                                                                                                  |
| ------------- | ------ | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| `EDITOR`      | Group  | `group.resource.create`, `group.resource.edit`, `group.resource.submit`, `group.resource.withdraw`, `group.resource.reopen`, `group.resource.discard` |
| `REVIEWER`    | Group  | `group.resource.reject`, `group.resource.publish`                                                                                                     |
| `GROUP_ADMIN` | Group  | All `EDITOR` and `REVIEWER` permissions, `group.resource.unpublish`, `group.membership.manage`, `group.role.manage`, `group.permission.manage`        |
| `ADMIN`       | Global | All seeded global permissions through `role_permissions`; role-only exceptions are listed below.                                                      |

Users may hold both `EDITOR` and `REVIEWER` in same group. Active membership itself grants read access, so no `VIEWER` role exists. Roles are fixed system definitions seeded by migrations and are not user-configurable. A user with `global.role.manage` may assign/remove `GROUP_ADMIN`; users cannot request it and group role managers cannot grant it. Assigning or removing global `ADMIN` remains restricted to `AuthorizeAdmin`.

### Permission Catalog

| Permission                     | Scope  | Meaning                                                                                                                                             |
| ------------------------------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `group.resource.create`        | Group  | Create logical dataset or software resource, draft, or copy in member group.                                                                        |
| `global.resource.create`       | Global | Create logical dataset or software resource, draft, or copy in any group without membership.                                                        |
| `group.resource.edit`          | Group  | Modify group resource draft.                                                                                                                        |
| `global.resource.edit`         | Global | Modify resource drafts in any group without membership.                                                                                             |
| `group.resource.submit`        | Group  | Submit group resource draft.                                                                                                                        |
| `global.resource.submit`       | Global | Submit resource drafts in any group without membership.                                                                                             |
| `group.resource.withdraw`      | Group  | Return submitted resource version to draft.                                                                                                         |
| `global.resource.withdraw`     | Global | Return submitted resource versions to draft in any group without membership.                                                                        |
| `group.resource.reopen`        | Group  | Return rejected resource version to draft.                                                                                                          |
| `global.resource.reopen`       | Global | Return rejected resource versions to draft in any group without membership.                                                                         |
| `group.resource.discard`       | Group  | Delete resource draft or rejected version.                                                                                                          |
| `global.resource.discard`      | Global | Delete resource drafts or rejected versions in any group without membership.                                                                        |
| `group.resource.reject`        | Group  | Reject submitted resource version.                                                                                                                  |
| `global.resource.reject`       | Global | Reject submitted resource versions in any group without membership.                                                                                 |
| `group.resource.publish`       | Group  | Publish submitted resource version.                                                                                                                 |
| `global.resource.publish`      | Global | Publish submitted resource versions in any group without membership.                                                                                |
| `group.resource.unpublish`     | Group  | Archive current published resource version without replacement.                                                                                     |
| `global.resource.unpublish`    | Global | Archive published resource versions in any group without membership.                                                                                |
| `group.membership.manage`      | Group  | View group membership roster and accept, reject, or revoke memberships.                                                                             |
| `group.role.manage`            | Group  | View group membership roster; accept or reject role requests and assign or remove fixed group roles.                                                |
| `group.permission.manage`      | Group  | View group membership roster and grant or revoke group-scoped granted permissions.                                                                  |
| `global.membership.manage`     | Global | List and read users; view every group membership roster and accept, reject, or revoke memberships in every group.                                   |
| `global.role.manage`           | Global | List and read users; view every group membership roster; accept or reject role requests and assign or remove fixed group roles in every group.      |
| `global.permission.manage`     | Global | List and read users; grant or revoke globally scoped permissions.                                                                                   |
| `global.user_api_token.revoke` | Global | Revoke another user's API token.                                                                                                                    |
| `global.organization.manage`   | Global | Manage canonical organizations, identifiers, and hierarchy.                                                                                         |
| `global.contact_point.manage`  | Global | Manage canonical contact points.                                                                                                                    |
| `global.group.manage`          | Global | Create, edit, and delete groups and their metadata.                                                                                                 |
| `global.concept.manage`        | Global | Create, edit, and delete concepts.                                                                                                                  |
| `group.category.manage`        | Group  | Manage canonical categories in owning group.                                                                                                        |
| `global.category.manage`       | Global | Manage canonical categories in any group without membership.                                                                                        |
| `global.configuration.manage`  | Global | Manage official configurations and every user's personal configurations in the Sharing service. See [Sharing authorization](../sharing/sharing.md). |
| `global.monitoring.read`       | Global | Read published distribution monitoring targets.                                                                                                     |

An active user with `global.permission.manage` grants and revokes globally scoped permissions. An active member with a `group.*` permission performs that action only in their group. A user with the corresponding `global.*` permission performs that action in every group without membership. Group role managers may assign or remove `EDITOR` and `REVIEWER` only; a user with `global.role.manage` may also assign or remove `GROUP_ADMIN`. Removing a role does not revoke membership; `GROUP_ADMIN` membership revocation requires `global.membership.manage`. Group permission managers grant and revoke group-scoped granted permissions in their group. `GROUP_ADMIN` includes all three group management permissions. Users with `global.organization.manage`, `global.contact_point.manage`, `global.group.manage`, `global.concept.manage`, or `global.category.manage` manage corresponding canonical entities globally. A user with `group.category.manage` manages canonical categories only in the group where permission is granted. Global `ADMIN` assignment, user blocking, and non-public read/discovery retain explicit `AuthorizeAdmin` behavior for now; future permission-specific replacements are listed below.

`global.configuration.manage` is the permission for Sharing administration. It is granted to global `ADMIN` through `role_permissions` and may also be granted directly by a user with `global.permission.manage`. Its Sharing authorization flow is specified in [sharing.md](../sharing/sharing.md).

`global.monitoring.read` allows reading published distribution monitoring targets. It is granted to global `ADMIN` through `role_permissions` and may also be granted directly by a user with `global.permission.manage`.

Groups and categories are publicly readable without authentication. Public group metadata never includes memberships, roles, granted permissions, or role requests. A user with `global.membership.manage`, `global.role.manage`, `global.permission.manage`, or `global.notification.manage` may list users and read any user detail; users may also read their own detail. Group membership roster access requires a global membership/role management permission or an active membership with `group.membership.manage`, `group.role.manage`, or `group.permission.manage` in that group. Roster entries expose member ID, given name, family name, email, membership status, assigned roles, granted group permissions, role-derived group permissions, and pending role requests. A user may read the same assigned roles, granted group permissions, and role-derived group permissions for their own memberships.

User list and detail responses expose global `global_granted_permissions` and `global_role_permissions`. User membership summaries and group membership responses expose `granted_permissions` and `role_permissions`. These lists preserve their sources; a permission present in both sources appears in both lists. Group membership responses do not expose global roles or global permissions.

Group roles and additive group permissions may be assigned only to active members of target group. Pending, rejected, and revoked memberships cannot hold dormant assignments.

### User API Tokens

Each user may have one optional, non-expiring API token. Users start without one. An active user may issue a token when none exists, rotate an existing token, or revoke their own token without a special permission. Issuing when one exists conflicts; rotating without one returns not found. Rotation replaces the old token atomically. Revocation succeeds when the target user exists even if no token is present.

An active user with `global.user_api_token.revoke` may revoke another user's token, including a blocked user's token. `ADMIN` receives this permission through its seeded role bundle; it may also be granted directly. Authorization checks the permission, not the role name. The permission does not grant user-directory access or allow issuing or rotating another user's token.

Token management requires an OAuth bearer credential at Gateway. API tokens cannot manage tokens, but may perform any other operation allowed to their user. Every request uses current user status, permissions, memberships, and workflow rules; tokens do not bypass authorization. Gateway validates the token digest through the trusted Metadata Lifecycle boundary and loads the existing user without provisioning.

The plaintext token is returned only when issued or rotated; it cannot be revealed later. Storage retains only a unique SHA-256 digest. User responses and normal user reads do not expose the token or digest. Successful issue, rotation, and revocation produce structured audit logs with actor and target user IDs; logs and errors never contain token values or digests.

### Decision Order

1. Allow public read without requiring user identity; fully public operations ignore supplied credentials, while optional-auth operations validate supplied bearer tokens.
2. Reject missing or blocked identities for authenticated operations.
3. For explicit `AuthorizeAdmin` capabilities, require active global `ADMIN`.
4. Evaluate additive global permission for global action or cross-group membership and role management.
5. Allow public reads of groups and categories without user identity.
6. For group action, require active membership in resource's stored group unless a matching global permission grants cross-group access.
7. Allow baseline group reads for active member.
8. Combine permissions from all assigned group roles and additive group grants.
9. For resource category selection, require category `group_id` to equal resource `group_id`.
10. Otherwise deny.

Group, ownership, and status must be loaded from stored resource. Caller-provided ownership or group claims are not authorization inputs.

## Membership And Roles

Users request group membership with zero or more requested roles. Membership and role decisions are separate.

| Action                         | Required access                                                                                         | Result                                                                                                                                                                                                                                                              |
| ------------------------------ | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Read group or category         | Public                                                                                                  | Returns canonical metadata without authentication; excludes memberships, roles, granted or role-derived permissions, and role requests.                                                                                                                             |
| List users or read user detail | Global management permission                                                                            | Returns user identity, status, global granted and role-derived permissions, and memberships. Users may read own detail.                                                                                                                                             |
| Read group membership roster   | Group membership/role/permission management permission, or global membership/role management permission | Returns all membership states, member identity including email, assigned roles, granted and role-derived permissions, and pending role requests.                                                                                                                    |
| Read own membership access     | Requesting user                                                                                         | Returns own membership status, assigned roles, granted and role-derived permissions, and pending role requests.                                                                                                                                                     |
| Request membership             | Active user                                                                                             | Creates pending membership and optional role requests.                                                                                                                                                                                                              |
| Withdraw membership request    | Requesting user                                                                                         | Deletes pending membership and its pending role requests.                                                                                                                                                                                                           |
| Accept membership              | `group.membership.manage` or `global.membership.manage`                                                 | Membership becomes active with read-only access. Role requests remain pending.                                                                                                                                                                                      |
| Reject membership              | `group.membership.manage` or `global.membership.manage`                                                 | Membership becomes rejected; system rejects pending role requests as cascade, without separate role permission.                                                                                                                                                     |
| Request another role           | Active group member                                                                                     | Creates pending request for `EDITOR` or `REVIEWER`.                                                                                                                                                                                                                 |
| Withdraw role request          | Requesting user                                                                                         | Deletes pending role request.                                                                                                                                                                                                                                       |
| Accept/reject role             | `group.role.manage` or `global.role.manage`                                                             | Adds role assignment or rejects request.                                                                                                                                                                                                                            |
| Assign/remove role directly    | `group.role.manage` or `global.role.manage`                                                             | Group role managers change `EDITOR` or `REVIEWER`; global role managers may also change `GROUP_ADMIN`.                                                                                                                                                              |
| Grant/revoke global permission | `global.permission.manage`                                                                              | Changes any global permission grant. `ADMIN` receives this permission through `role_permissions`.                                                                                                                                                                   |
| Grant/revoke group permission  | `group.permission.manage` or `global.permission.manage`                                                 | Changes any additive group grant in the group. Revoking a permission supplied only by a role returns conflict; revoking a granted or absent permission is idempotent.                                                                                               |
| Assign/remove global `ADMIN`   | `AuthorizeAdmin`                                                                                        | Changes the global admin role while preserving the last-active-admin invariant.                                                                                                                                                                                     |
| Revoke membership              | Requesting user for own membership; otherwise `group.membership.manage` or `global.membership.manage`   | Permanently deletes membership, roles, grants, subscriptions, and all membership-related role requests. Users may revoke their own membership, including `GROUP_ADMIN`, while removing another user's `GROUP_ADMIN` membership requires `global.membership.manage`. |

Membership states are `PENDING`, `ACTIVE`, and `REJECTED`. Pending or active member cannot request duplicate membership. Rejected user may submit new request, resetting membership to `PENDING`. Revocation retains no record and has no restoration action; user may submit new membership request immediately.

Membership holding `GROUP_ADMIN` may be revoked by its owner or by `global.membership.manage`; this prevents group membership managers from indirectly removing another user's Group Admin assignment. Self-revocation permanently removes the user's membership and all cascaded membership access.

Manual role-request decisions require active membership. Membership rejection is system cascade exempt from this rule; revocation deletes pending requests. Rejected role may be requested again after membership is active. At most one matching role request may be pending. Direct assignment or removal is blocked while matching request is pending; manager must decide request first.

First created user receives global `ADMIN` role. Removing its legacy `users.is_super_admin` column leaves global role assignment as sole source of admin authority. Active admin means active, unblocked user assigned `ADMIN`. Admin eligibility changes must serialize and fail when resulting active-admin count would be zero.

### User Blocking

Global `ADMIN` may block user with required non-empty reason and may later unblock user. Block stores reason, actor, and timestamp. Memberships, roles, grants, and subscriptions remain stored but dormant. Blocking also revokes any API token in the same transaction; if the last-active-admin safeguard rejects the block, token revocation rolls back. Unblocking restores memberships, roles, grants, and subscriptions subject to normal active-membership checks, but does not restore the token. The user must issue a new token.

Blocked identity remains attributable. Operations requiring or optionally accepting authentication reject blocked bearer tokens; fully public operations ignore supplied credentials and remain readable. Requests without user identity retain public-read access. Blocking last active global `ADMIN` is forbidden. An attempted authenticated action by a blocked user is denied and remains attributable in authorization/audit telemetry; it does not produce an email notification topic.

### Future Permission Work

The following capabilities intentionally remain behind `AuthorizeAdmin` until explicit permissions are designed and implemented:

| Current capability               | Future permission candidate | Required alignment                                                                           |
| -------------------------------- | --------------------------- | -------------------------------------------------------------------------------------------- |
| Assign/remove global `ADMIN`     | `global.admin.manage`       | Preserve last-active-admin invariant and prevent privilege escalation.                       |
| Block/unblock users              | `global.user.block`         | Preserve reason, actor, timestamp, and security-event behavior.                              |
| Non-public read/discovery bypass | `global.read.private`       | Define exposure boundary for user, membership, contact-point, and unpublished-resource data. |

## Notifications

Notifications separate three concepts:

1. **Authorization** answers what a user may do.
2. **Event kind** answers what happened.
3. **Topic route** answers which audience may receive that event.

Authorization and subscription eligibility are independent. A permission does not automatically subscribe a user, and a subscription does not grant resource read or mutation access. Direct routes are internal mandatory delivery paths; only group and global routes are subscribable and exposed through the topic-listing API.

### Events And Topic Routes

Metadata emits one event kind per successful state change. Topic routes fan that event out to direct, group, and global audiences. A route key such as `global.resource.published` is a UI and subscription key; it is not a second domain event.

The MVP topic catalog is:

| Topic key                       | Event kind               | Route  | Subscription                        | Recipient or eligibility                                                                                                | Behavior                                                                                                                                                                |
| ------------------------------- | ------------------------ | ------ | ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `direct.membership.accepted`    | `membership.accepted`    | Direct | Internal mandatory delivery         | Membership applicant                                                                                                    | Sends one direct email.                                                                                                                                                 |
| `direct.membership.rejected`    | `membership.rejected`    | Direct | Internal mandatory delivery         | Membership applicant                                                                                                    | Sends one direct email. Pending role requests rejected by this operation do not produce additional emails.                                                              |
| `direct.role_request.accepted`  | `role_request.accepted`  | Direct | Internal mandatory delivery         | Role-request applicant                                                                                                  | Sends one direct email. Role assignment performed as part of acceptance does not produce a second `role.assigned` email.                                                |
| `direct.role_request.rejected`  | `role_request.rejected`  | Direct | Internal mandatory delivery         | Role-request applicant                                                                                                  | Sends one direct email for an explicit role-request rejection.                                                                                                          |
| `direct.role.assigned`          | `role.assigned`          | Direct | Internal mandatory delivery         | Assigned user                                                                                                           | Sends one email for an actual explicit group or global role assignment. No email is sent for a no-op assignment or an assignment performed by an accepted role request. |
| `direct.permission.assigned`    | `permission.assigned`    | Direct | Internal mandatory delivery         | Granted user                                                                                                            | Sends one email for an actual explicit group or global permission grant.                                                                                                |
| `direct.resource.published`     | `resource.published`     | Direct | Internal mandatory delivery         | Resource `submitted_by` user                                                                                            | Sends one publication result email.                                                                                                                                     |
| `direct.resource.rejected`      | `resource.rejected`      | Direct | Internal mandatory delivery         | Resource `submitted_by` user                                                                                            | Sends one rejection result email, including the rejection note.                                                                                                         |
| `direct.resource.unpublished`   | `resource.unpublished`   | Direct | Internal mandatory delivery         | Resource `submitted_by` user                                                                                            | Sends one unpublication result email.                                                                                                                                   |
| `group.membership.requested`    | `membership.requested`   | Group  | Subscribable                        | User with active membership and `group.membership.manage`                                                               | Sends membership-request emails only to eligible subscribers for that group.                                                                                            |
| `global.membership.requested`   | `membership.requested`   | Global | Subscribable; default-on for admins | User with `global.membership.manage`. Active global `ADMIN` users are subscribed by default.                            | Sends membership-request emails from every group to eligible subscribed users.                                                                                          |
| `group.role_request.requested`  | `role_request.requested` | Group  | Subscribable                        | User with active membership and `group.role.manage`                                                                     | Sends role-request emails only to eligible subscribers for that group. The requested role is included.                                                                  |
| `global.role_request.requested` | `role_request.requested` | Global | Subscribable; default-on for admins | User with `global.role.manage`. Active global `ADMIN` users are subscribed by default.                                  | Sends role-request emails from every group to eligible subscribed users. The requested role is included.                                                                |
| `group.resource.submitted`      | `resource.submitted`     | Group  | Subscribable                        | User with active membership and `group.resource.publish` OR `group.resource.reject`                                     | Sends submission emails only to users who explicitly subscribe to that group. Active membership alone does not subscribe a user.                                        |
| `group.resource.published`      | `resource.published`     | Group  | Subscribable                        | User with active membership in the resource group                                                                       | Sends publication emails only to users who explicitly subscribe to that group. Active membership alone does not subscribe a user.                                       |
| `global.resource.submitted`     | `resource.submitted`     | Global | Subscribable; default-on for admins | User with `global.resource.publish` OR `global.resource.reject`. Active global `ADMIN` users are subscribed by default. | Sends submission emails from every group to eligible subscribed users.                                                                                                  |
| `global.resource.published`     | `resource.published`     | Global | Subscribable; default-on for admins | Any active user may opt in. Active global `ADMIN` users are subscribed by default.                                      | Sends publication emails from every group to subscribed users.                                                                                                          |
| `global.conversion.requested`   | `conversion.requested`   | Global | Subscribable; default-on for admins | Active user with `global.converter.manage`.                                                                             | Sends conversion requests for draft web-service distributions to eligible subscribers.                                                                                  |

There are no group or global subscription topics for resource rejection or unpublication. Resource submission, publication, membership requests, and role requests are subscribable; rejection and unpublication remain direct-only. Request topics notify managers, not the requesting user; requesters receive the existing direct accepted/rejected topics. Submission notifications do not grant resource read access. Their payload must remain within the recipient's normal resource-read visibility. There are no email topics for membership revocation, blocked-user actions, or access removals caused by membership revocation.

The same event may route to multiple audiences. For example:

```text
resource.published
group_id = A
submitted_by = user-1
```

routes to `direct.resource.published` for `user-1`, `group.resource.published` for Group A subscribers, and `global.resource.published` for global subscribers. A user matching more than one route receives one email per event, user, and channel. Mandatory direct delivery wins during deduplication.

### Subscription Eligibility And Management

All current notifications use the `EMAIL` channel. Delivery checks user status and subscription eligibility again when work is built or sent:

| Subscription              | Create/delete eligibility                                                                                                                                                                                                                                                                                                  | Delivery eligibility                                                                                                                             |
| ------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Group membership request  | User may manage their own subscription when they are an active member with `group.membership.manage`. A user with `group.notification.manage` may manage subscriptions for other eligible members of that group. A user with `global.notification.manage` may manage subscriptions in any group.                           | Target user is active, has active membership in the group, and retains `group.membership.manage`.                                                |
| Group role request        | User may manage their own subscription when they are an active member with `group.role.manage`. A user with `group.notification.manage` may manage subscriptions for other eligible members of that group. A user with `global.notification.manage` may manage subscriptions in any group.                                 | Target user is active, has active membership in the group, and retains `group.role.manage`.                                                      |
| Group submission          | User may manage their own subscription when they are an active member with `group.resource.publish` OR `group.resource.reject`. A user with `group.notification.manage` may manage subscriptions for other eligible members of that group. A user with `global.notification.manage` may manage subscriptions in any group. | Target user is active, has active membership in the group, and retains `group.resource.publish` OR `group.resource.reject`.                      |
| Group publication         | User may manage their own subscription when they are an active member of the group. A user with `group.notification.manage` may manage subscriptions for other active members of that group. A user with `global.notification.manage` may manage subscriptions in any group.                                               | Target user is active and still has active membership in the group.                                                                              |
| Global membership request | User may manage their own subscription when active and holding `global.membership.manage`. A user with `global.notification.manage` may manage global subscriptions for other eligible users.                                                                                                                              | Target user is active and retains `global.membership.manage`; active global `ADMIN` users remain eligible by default.                            |
| Global role request       | User may manage their own subscription when active and holding `global.role.manage`. A user with `global.notification.manage` may manage global subscriptions for other eligible users.                                                                                                                                    | Target user is active and retains `global.role.manage`; active global `ADMIN` users remain eligible by default.                                  |
| Global submission         | User may manage their own subscription when active and holding `global.resource.publish` OR `global.resource.reject`. A user with `global.notification.manage` may manage global subscriptions for other eligible users.                                                                                                   | Target user is active and retains `global.resource.publish` OR `global.resource.reject`; active global `ADMIN` users remain eligible by default. |
| Global publication        | Any active user may manage their own subscription. A user with `global.notification.manage` may manage global subscriptions for other users.                                                                                                                                                                               | Target user is active. No global permission is required for publication notifications.                                                           |

Blocked users never receive notifications. Their subscriptions remain stored and dormant. Unblocking restores them when normal eligibility rules pass. Direct recipients do not need to remain group members; an active submitter still receives their direct resource result after membership loss.

Users manage their own subscriptions. `GROUP_ADMIN` includes `group.notification.manage`, which applies only to its group. `ADMIN` includes `global.notification.manage`, which applies to all global and group subscriptions. These are explicit notification-management permissions and are separate from `group.permission.manage` and `global.permission.manage`.

Global `ADMIN` users are subscribed to existing global topics by default. Deleting the row opts out. Assigning `ADMIN` recreates the default row if it is absent. Removing `ADMIN` does not delete an existing row; the user remains subscribed as an active user. A user who opted out and later receives `ADMIN` again is subscribed again by the default assignment behavior. Adding a new default-on global topic creates rows for active admins.

Group subscriptions are opt-in and are deleted with membership revocation. Rejoining a group does not restore the old row; the user must opt in again. Blocking does not delete either subscription type.

The topic catalog retains a `required_permissions` field for permission-gated topics. When populated, the list uses OR semantics: the user needs any one listed permission. Current group membership-request topics require `group.membership.manage`; current global membership-request topics require `global.membership.manage`. Current group role-request topics require `group.role.manage`; current global role-request topics require `global.role.manage`. Current submission topics use `group.resource.publish` OR `group.resource.reject` for group scope and `global.resource.publish` OR `global.resource.reject` for global scope. Current publication topics have no required permissions. Future group topics use group-scoped permissions; future global topics use global-scoped permissions.

### Notification Work And Delivery

Mandatory direct notifications have no subscription rows. Metadata builds transient notifications after the successful metadata mutation. No event or delivery record is persisted in the MVP.

Each work item contains enough data for the Gateway to send without another Metadata Lifecycle lookup:

| Field                          | Purpose                                                                     |
| ------------------------------ | --------------------------------------------------------------------------- |
| `event_kind`                   | Identifies what happened without audience-specific duplication.             |
| `group_id`                     | Identifies the affected group when applicable.                              |
| Resource or access identifiers | Identifies the resource, version, role, or permission involved.             |
| Target user ID and email       | Preserves the explicit direct target and gives Mailgun a recipient address. |
| Subject and body               | Metadata Lifecycle renders the complete email.                              |
| Channel                        | Currently `EMAIL`.                                                          |

The delivery flow is:

```text
metadata mutation commits
  -> metadata builds notifications
  -> gRPC response carries notifications
  -> Gateway starts detached, time-limited delivery
  -> Gateway calls Mailgun
```

Notifications must not use the request context after the response. Gateway uses an independent timeout context. Mailgun failures and notification-construction failures are logged and dropped. A successful metadata mutation remains successful when notifications cannot be built or delivered. Gateway crashes can lose notifications.

If a user qualifies for multiple routes, Metadata deduplicates before returning work. If a mutation causes several internal state changes, only the explicitly defined event is emitted; implicit cascade changes do not create extra notifications.

## Required Scenarios

| Scenario                                                                | Result                                                                                                                                                                                   |
| ----------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Editor and reviewer roles assigned to same user                         | User may submit and publish own submission.                                                                                                                                              |
| Two editors update same draft revision                                  | First succeeds; second receives conflict.                                                                                                                                                |
| Editor copies published dataset to another group                        | New logical dataset/draft uses new UUID, copied identifier values in new rows, and zero categories.                                                                                      |
| Editor creates software-application draft from publication              | New version has same logical application UUID and cloned metadata, creators, identifiers, and category links.                                                                            |
| Editor copies published software source code                            | New logical source code/draft has new UUID, cloned metadata, creators, and identifiers, and zero categories.                                                                             |
| Editor links category from another group                                | Denied even when editor belongs to both groups.                                                                                                                                          |
| Editor attempts to create category through dataset draft                | Denied; category management is outside dataset lifecycle.                                                                                                                                |
| Category manager creates category with duplicate sibling name           | Denied after normalized-name comparison.                                                                                                                                                 |
| Client previews referenced category impact, then requests delete        | Full subtree and every dataset/software link are deleted atomically, including published/archived links.                                                                                 |
| Client previews referenced organization impact, then requests delete    | Resource/hierarchy links and owned identifiers are removed, then organization is deleted.                                                                                                |
| Contact-point manager edits canonical contact                           | Change is visible through every dataset-version/web-service reference.                                                                                                                   |
| Client previews referenced contact-point impact, then requests delete   | All contact links are removed, then contact point is deleted.                                                                                                                            |
| Group owns no resource                                                  | Categories, memberships, access assignments, requests, and subscriptions are deleted with group.                                                                                         |
| Group Admin unpublishes resource                                        | Public version becomes archived; later draft may publish normally.                                                                                                                       |
| Group role manager tries to assign `GROUP_ADMIN`                        | Denied; a user with `global.role.manage` may assign it.                                                                                                                                  |
| Membership is revoked                                                   | Membership and related access/subscriptions/pending requests are deleted; user may request again.                                                                                        |
| Blocked user with bearer token performs fully public read               | Allowed; optional-auth public reads reject the blocked token.                                                                                                                            |
| Blocked user attempts authenticated action                              | Denied and attributable in authorization/audit telemetry; no email notification topic is emitted.                                                                                        |
| User issues a token when one exists or rotates without one              | Issue conflicts; rotation returns not found.                                                                                                                                             |
| User rotates or revokes a token                                         | Previous token is invalid immediately; revoking an absent token for an existing user succeeds.                                                                                           |
| User's permission is removed after issuing a token                      | Next token-authenticated request uses current permissions.                                                                                                                               |
| User revokes another user's token                                       | Requires `global.user_api_token.revoke` and OAuth authentication; an API token cannot manage tokens.                                                                                     |
| User is blocked and later unblocked                                     | Existing API token remains revoked; user may issue a new token after unblocking.                                                                                                         |
| Last active admin is blocked                                            | Block fails; any existing API token remains valid because revocation rolls back.                                                                                                         |
| System Admin unblocks user                                              | Prior memberships, roles, grants, and subscriptions become effective again when otherwise valid.                                                                                         |
| New System Admin or global topic is added                               | Active `ADMIN` receives the global subscription by default and may opt out; any otherwise eligible active user may opt in.                                                               |
| User subscribes to group publication notifications                      | User must be an active member; subscription is explicit and does not automatically include other group members.                                                                          |
| User requests group membership                                          | Eligible group subscribers with `group.membership.manage` receive the group request topic; eligible global subscribers with `global.membership.manage` receive the global request topic. |
| User requests a specific group role                                     | Eligible group subscribers with `group.role.manage` receive the group request topic; eligible global subscribers with `global.role.manage` receive the global request topic.             |
| User matches direct and broadcast routes for one event                  | Metadata returns one deduplicated email for that user and channel.                                                                                                                       |
| Membership rejection cascades to pending role requests                  | One membership-rejected email; cascaded role-request changes produce no additional email.                                                                                                |
| Role request is accepted                                                | One role-request-accepted email; implicit role assignment produces no second email.                                                                                                      |
| Membership revocation cascades to roles, permissions, and subscriptions | Access removals are silent; group subscription is deleted and must be recreated after rejoining.                                                                                         |
| Identifier is linked to second owner                                    | Denied; copy requires new identifier row.                                                                                                                                                |
| Attempt blocks/removes last active System Admin                         | Denied.                                                                                                                                                                                  |
