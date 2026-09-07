# Metadata Lifecycle and Authorization

## Scope

This document defines dataset and software versioning, linked-entity governance, group membership, category schemes, categories, authorization, and notification subscriptions for metadata lifecycle.

It does not define HTTP endpoints, UI layouts, migration order, or notification event delivery.

## Terms

| Term                    | Meaning                                                                                               |
| ----------------------- | ----------------------------------------------------------------------------------------------------- |
| Logical dataset         | Stable UUID shared by all versions and owned by one group.                                            |
| Dataset version         | Editable or historical metadata for one logical dataset.                                              |
| Logical software        | Stable UUID shared by all versions of one software application or source code and owned by one group. |
| Software version        | Editable or historical metadata for one logical software resource.                                    |
| Active change version   | Single `DRAFT`, `SUBMITTED`, or `REJECTED` version for one logical resource.                          |
| Group                   | Canonical category scheme, resource ownership, and authorization boundary stored in one `groups` row. |
| Category scheme         | Category-hierarchy aspect of a group; not a separate authorization resource.                          |
| Category                | Node inside category scheme. A subcategory is category with parent category, not separate type.       |
| Canonical category      | Shared live category persisted in its owning group's category scheme.                                 |
| Canonical organization  | Shared live organization managed directly in metadata lifecycle without lifecycle states.             |
| Canonical contact point | Shared live contact record managed directly in metadata lifecycle without lifecycle states.           |
| Identifier              | Value owned by exactly one resource version or organization; never shared.                            |
| Group member            | Active, unblocked user with `ACTIVE` membership in group and access to its category scheme.           |
| Direct grant            | Additive permission assigned outside a role.                                                          |

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

| Action                        | From                | To          | Required access                        | Rules and effects                                                                                       |
| ----------------------------- | ------------------- | ----------- | -------------------------------------- | ------------------------------------------------------------------------------------------------------- |
| Create empty draft            | None                | `DRAFT`     | `resource.create` in group             | Creates logical resource with a generated UUID and its first version.                                   |
| Create draft from publication | `PUBLISHED`         | New `DRAFT` | `resource.create` in resource group    | Clones version-owned metadata and canonical links.                                                      |
| Create draft after unpublish  | Latest `ARCHIVED`   | New `DRAFT` | `resource.create` in resource group    | Performs same clone when logical resource has no publication.                                           |
| Copy published resource       | `PUBLISHED`         | New `DRAFT` | `resource.create` in target group      | Creates independent logical resource with a generated UUID; clones content and links except categories. |
| Edit                          | `DRAFT`             | `DRAFT`     | `resource.edit` in resource group      | Any authorized group member may edit. Requires current revision.                                        |
| Submit                        | `DRAFT`             | `SUBMITTED` | `resource.submit` in resource group    | Version becomes immutable.                                                                              |
| Withdraw                      | `SUBMITTED`         | `DRAFT`     | `resource.withdraw` in resource group  | Removes submission from review.                                                                         |
| Reject                        | `SUBMITTED`         | `REJECTED`  | `resource.reject` in resource group    | Non-empty rejection note required. Latest note replaces previous note.                                  |
| Reopen                        | `REJECTED`          | `DRAFT`     | `resource.reopen` in resource group    | Makes version editable; latest rejection note remains visible.                                          |
| Publish                       | `SUBMITTED`         | `PUBLISHED` | `resource.publish` in resource group   | Self-publication allowed. Existing publication becomes `ARCHIVED`; rejection fields are cleared.        |
| Discard                       | `DRAFT`, `REJECTED` | Deleted     | `resource.discard` in resource group   | Permanently deletes active change version. Deletes logical resource too when no versions remain.        |
| Unpublish                     | `PUBLISHED`         | `ARCHIVED`  | `resource.unpublish` in resource group | No reason required. Logical resource remains available for a future draft.                              |

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

Draft creation for an existing logical resource fails with conflict when an active change version exists. `resource.create` is authorized against requested group before an empty logical resource exists; every later action uses stored logical-resource group. Latest archive means greatest `archived_at`, with version ID as deterministic tie-breaker.

Published resource may be copied into same or different group. Source is public, so source-group membership is not required; caller needs `resource.create` in target group. Copy creates new logical resource with a generated UUID and first `DRAFT`; original remains unchanged. Copy clones descriptive metadata, identifiers, and organization links. Dataset copies also clone distributions, web services, and contact-point links. Identifier rows receive new internal IDs and preserve type and value. Target draft has zero categories.

Published resource lists return current `PUBLISHED` versions grouped by logical resource group. Published versions, canonical organizations, groups, and canonical categories are publicly readable without authentication. Public group and category representations expose canonical metadata but never memberships, roles, grants, or role requests. Contact points are excluded from public resource representations because they contain email addresses. Any active group member may read every non-public version in that group. Global `ADMIN` may read everything.

Software application and software source code are independent resource types. Their versions contain type-specific metadata plus creator, identifier, contact-point, and category links; they do not reference each other, distributions, or web services.

## Metadata Ownership

| Entity                           | Editor use                                     | Editor create/propose | Management                 | Reference behavior                                                                                                                                        |
| -------------------------------- | ---------------------------------------------- | --------------------- | -------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Dataset metadata                 | Edit in `DRAFT`                                | Yes                   | Through lifecycle          | Copied per version.                                                                                                                                       |
| Distribution                     | Edit in `DRAFT`                                | Yes                   | Through lifecycle          | Owned by version.                                                                                                                                         |
| Web service                      | Edit in `DRAFT`                                | Yes                   | Through lifecycle          | Owned by version.                                                                                                                                         |
| Software application/source code | Edit in `DRAFT` and use own group's categories | Yes                   | Through resource lifecycle | Copied per version; creators, contact points, and categories are version-owned links. Identifiers are version-owned records cloned with new internal IDs. |
| Canonical organization           | Select existing organization                   | No                    | `organization.manage`      | Shared live reference.                                                                                                                                    |
| Canonical contact point          | Select existing contact point                  | No                    | `contact_point.manage`     | Shared live reference.                                                                                                                                    |
| Identifier                       | Edit through owning entity                     | Yes with owner        | Through owner              | Exactly one owner; cloned rather than shared.                                                                                                             |
| Category scheme                  | Use through group membership                   | No                    | `category_scheme.manage`   | Shared live metadata stored with group.                                                                                                                   |
| Canonical category               | Select from owning group's category scheme     | No                    | `category.manage` in group | Shared live reference; category cannot move to another group.                                                                                             |

Canonical organization, contact-point, group, and canonical-category changes appear immediately in every referencing version. No snapshots or history are required. Resource version category links retain only canonical category IDs; they never copy category fields.

Canonical organizations, contact points, categories, and groups may be deleted directly. The client is responsible for obtaining impact data, presenting any warning or preview to the user, and requesting deletion after confirmation. Metadata Lifecycle does not expose a separate preview or confirmation operation. Group deletion is blocked only while group owns logical dataset or software resource.

### Canonical Deletion

Canonical deletion is a direct atomic operation returning `204 No Content`. Clients manage any impact preview before requesting deletion. The metadata lifecycle unlinks every reference and deletes the canonical entity in one transaction. Deletion remains an explicit exception to version immutability and may change public and historical representations.

Category deletion removes selected category, complete descendant subtree, all hierarchy links, and all dataset/software links to any category in subtree. Organization deletion removes its resource and hierarchy links, deletes its owned identifiers, then deletes organization; related organizations remain. Contact-point deletion removes all organization, dataset-version, software-version, and web-service links, then deletes contact point.

Group deletion fails while any logical dataset, software application, or software source code belongs to group. Otherwise it deletes group/category scheme, canonical categories, memberships, group role assignments, grants, requests, and group notification subscriptions.

### Organizations

Organizations are canonical live entities managed directly in metadata lifecycle. They have no draft, submission, review, publication, archival, or rejection lifecycle.

| Action             | Required access       | Effect                                                         |
| ------------------ | --------------------- | -------------------------------------------------------------- |
| Read/select        | Public                | Uses current canonical organization.                           |
| Create/edit        | `organization.manage` | Change becomes visible immediately in every reference.         |
| Manage identifiers | `organization.manage` | Creates, edits, and deletes identifiers owned by organization. |
| Manage hierarchy   | `organization.manage` | Adds or removes parent-child organization links.               |
| Delete             | `organization.manage` | Requires destructive-deletion impact warning and confirmation. |

Dataset-version providers and software-version creators may reference multiple organizations. Organizations may reference multiple contact points. Version-owned web service may reference zero or one provider organization. All relations use local organization UUID foreign keys.

Organization hierarchy allows multiple parents. Direct self-links and indirect cycles are forbidden.

### Contact Points

Contact points are canonical live entities managed directly in metadata lifecycle. They have UUID, email, and role, and no lifecycle states.

| Action      | Required access        | Effect                                                                                |
| ----------- | ---------------------- | ------------------------------------------------------------------------------------- |
| Read/select | Active group member    | Requires active membership in at least one group; email is sensitive data.            |
| Create/edit | `contact_point.manage` | Change becomes visible in every dataset, software-version, and web-service reference. |
| Delete      | `contact_point.manage` | Requires destructive-deletion impact warning and confirmation.                        |

Dataset version, software version, and version-owned web service may each reference zero or more contact points. Organizations may also reference zero or more canonical contact points. Duplicate contact-point link on same owner is forbidden. Editors may change resource contact-point links inside draft but cannot create or edit canonical contact points. Contact points, including those referenced by otherwise public resources, are returned only to an active, unblocked user with active membership in at least one group. Global `ADMIN` may read them without membership.

### Identifiers

Identifier is owned value, not shared canonical entity. Dataset version, organization, software-application version, or software-source-code version may own zero or more identifiers. Each identifier has exactly one owner and owner never changes.

Creating draft clones published/archived version identifiers. Cross-group resource copy also clones identifiers. Every clone receives new internal ID while preserving type and value. Identical identifier content is allowed under different owners. Editing identifier affects only owner. Deleting owner deletes owned identifiers.

Future external organization management and migration are outside this document.

### Categories

Category scheme is a tree or forest. Each canonical category has zero or one parent in same group and cannot participate in cycle. Category names are compared after trimming whitespace and case folding. Name must be unique among siblings; top-level name must be unique among top-level categories. Same normalized name may exist in different branches.

Categories are managed outside resource lifecycle. An active group member with `category.manage` in category's group may create, edit, reparent, or delete canonical categories in that group. Editors may select only existing canonical categories from resource group's category scheme while editing a draft; they cannot create or propose categories through a resource. `category_scheme.manage` manages category-scheme metadata separately and does not grant category management.

Canonical category has stable UUID. Its group is immutable. Category changes take effect immediately for every dataset version and software resource that references category. Category deletion remains a confirmed destructive operation.

### Category Schema Requirements

The `categories` table stores `id` as stable UUID, `group_id` as direct owning-group foreign key, nullable `parent_category_id`, and `name`. `parent_category_id` must reference a category in same `group_id`; this represents a tree or forest without separate hierarchy-link table.

Resource-version category-reference tables store canonical `category_id` values only. They must enforce that referenced category's `group_id` equals owning resource's `group_id`. They do not copy category name, parent, or other category fields.

## Authorization Model

Authorization uses fixed composable roles plus additive direct grants. Default is deny. Explicit deny grants are not supported.

Permissions have one of these scopes:

| Scope             | Meaning                              | Example                   |
| ----------------- | ------------------------------------ | ------------------------- |
| Global role       | Applies across system                | `ADMIN`                   |
| Global permission | Applies across system                | `category_scheme.manage`  |
| Group role        | Applies within one active membership | `EDITOR`                  |
| Group permission  | Applies within one active membership | `group.membership.manage` |

### Fixed Roles

| Role          | Scope  | Included permissions                                                                                                                     |
| ------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `EDITOR`      | Group  | `resource.create`, `resource.edit`, `resource.submit`, `resource.withdraw`, `resource.reopen`, `resource.discard`                        |
| `REVIEWER`    | Group  | `resource.reject`, `resource.publish`                                                                                                    |
| `GROUP_ADMIN` | Group  | All `EDITOR` and `REVIEWER` permissions, `resource.unpublish`, `group.membership.manage`, `group.role.manage`, `group.permission.manage` |
| `ADMIN`       | Global | Full authorization bypass                                                                                                                |

Users may hold both `EDITOR` and `REVIEWER` in same group. Active membership itself grants read access, so no `VIEWER` role exists. Roles are fixed system definitions seeded by migrations and are not user-configurable. Only global `ADMIN` or a user with `global.role.manage` may assign/remove `GROUP_ADMIN`; users cannot request it and group role managers cannot grant it.

### Permission Catalog

| Permission                 | Scope  | Meaning                                                                                                                                                    |
| -------------------------- | ------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `resource.create`          | Group  | Create logical dataset or software resource, draft, or copy.                                                                                               |
| `resource.edit`            | Group  | Modify group resource draft.                                                                                                                               |
| `resource.submit`          | Group  | Submit group resource draft.                                                                                                                               |
| `resource.withdraw`        | Group  | Return submitted resource version to draft.                                                                                                                |
| `resource.reopen`          | Group  | Return rejected resource version to draft.                                                                                                                 |
| `resource.discard`         | Group  | Delete resource draft or rejected version.                                                                                                                 |
| `resource.reject`          | Group  | Reject submitted resource version.                                                                                                                         |
| `resource.publish`         | Group  | Publish submitted resource version.                                                                                                                        |
| `resource.unpublish`       | Group  | Archive current published resource version without replacement.                                                                                            |
| `group.membership.manage`  | Group  | View group membership roster and accept, reject, or revoke memberships.                                                                                    |
| `group.role.manage`        | Group  | View group membership roster; accept or reject role requests and assign or remove fixed group roles.                                                       |
| `group.permission.manage`  | Group  | View group membership roster and grant or revoke group-scoped direct permissions.                                                                          |
| `global.membership.manage` | Global | List and read users; view every group membership roster and accept, reject, or revoke memberships in every group.                                          |
| `global.role.manage`       | Global | List and read users; view every group membership roster; accept or reject role requests and assign or remove `EDITOR` and `REVIEWER` roles in every group. |
| `global.permission.manage` | Global | List and read users; grant or revoke globally scoped permissions.                                                                                          |
| `organization.manage`      | Global | Manage canonical organizations, identifiers, and hierarchy.                                                                                                |
| `contact_point.manage`     | Global | Manage canonical contact points.                                                                                                                           |
| `category_scheme.manage`   | Global | Manage groups' category-scheme metadata.                                                                                                                   |
| `category.manage`          | Group  | Manage canonical categories in owning group.                                                                                                               |

Global `ADMIN` grants and revokes global roles and `GROUP_ADMIN`. An active user with `global.permission.manage` or global `ADMIN` grants and revokes globally scoped permissions. An active member with `group.membership.manage`, `group.role.manage`, or `group.permission.manage` performs that action only in their group. An active user with the corresponding `global.membership.manage` or `global.role.manage` permission performs that action in every group without membership. Role managers may assign only `EDITOR` and `REVIEWER`; only global `ADMIN` or an active user with `global.role.manage` may assign or remove `GROUP_ADMIN`. Removing a role does not revoke membership; `GROUP_ADMIN` membership revocation requires global `ADMIN` or `global.membership.manage`. Group permission managers grant and revoke group-scoped direct permissions in their group. `GROUP_ADMIN` includes all three group management permissions. Users with `organization.manage`, `contact_point.manage`, or `category_scheme.manage` manage corresponding canonical entities or category-scheme metadata globally. User with `category.manage` manages canonical categories only in group where permission is granted. Group creation and deletion require global `ADMIN`. Global `ADMIN` may perform all operations.

Groups and categories are publicly readable without authentication. A group is also its canonical category scheme for authorization and API purposes. Public group metadata never includes memberships, roles, direct permissions, or role requests. Global `ADMIN` and every global management permission holder may list users and read any user detail; users may also read their own detail. Group membership roster access requires global `ADMIN`, global membership/role management permission, or an active membership with `group.membership.manage`, `group.role.manage`, or `group.permission.manage` in that group. Roster entries expose member ID, given name, family name, email, membership status, assigned roles, assigned direct group permissions, and pending role requests. A user may read the same assigned roles and direct group permissions for their own memberships.

Group roles and additive group permissions may be assigned only to active members of target group. Pending, rejected, and revoked memberships cannot hold dormant assignments.

### Decision Order

1. Allow public read without requiring user identity; blocked user has public-read access only.
2. Reject authenticated action for missing or blocked user.
3. Allow global `ADMIN` without membership or resource grants.
4. Evaluate additive global permission for global action or cross-group membership and role management.
5. Allow public reads of groups and categories without user identity.
6. For group action, require active membership in resource's stored group unless a matching global group-management permission grants cross-group access.
7. Allow baseline group reads for active member.
8. Combine permissions from all assigned group roles and additive group grants.
9. For resource category selection, require category `group_id` to equal resource `group_id`.
10. Otherwise deny.

Group, ownership, status, and category scheme association must be loaded from stored resource. Caller-provided ownership or group claims are not authorization inputs.

## Membership And Roles

Users request group membership with zero or more requested roles. Membership and role decisions are separate.

| Action                         | Required access                                                                                         | Result                                                                                                                                                                                  |
| ------------------------------ | ------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Read group or category         | Public                                                                                                  | Returns canonical metadata without authentication; excludes memberships, roles, direct permissions, and role requests.                                                                  |
| List users or read user detail | Global management permission                                                                            | Returns user identity, status, global access, and memberships. Users may read own detail. Global `ADMIN` may read all users.                                                            |
| Read group membership roster   | Group membership/role/permission management permission, or global membership/role management permission | Returns all membership states, member identity including email, assigned roles and direct permissions, and pending role requests. Global `ADMIN` may read any roster.                   |
| Read own membership access     | Requesting user                                                                                         | Returns own membership status, assigned roles, direct permissions, and pending role requests.                                                                                           |
| Request membership             | Active user                                                                                             | Creates pending membership and optional role requests.                                                                                                                                  |
| Withdraw membership request    | Requesting user                                                                                         | Deletes pending membership and its pending role requests.                                                                                                                               |
| Accept membership              | `group.membership.manage` or `global.membership.manage`                                                 | Membership becomes active with read-only access. Role requests remain pending.                                                                                                          |
| Reject membership              | `group.membership.manage` or `global.membership.manage`                                                 | Membership becomes rejected; system rejects pending role requests as cascade, without separate role permission.                                                                         |
| Request another role           | Active group member                                                                                     | Creates pending request for `EDITOR` or `REVIEWER`.                                                                                                                                     |
| Withdraw role request          | Requesting user                                                                                         | Deletes pending role request.                                                                                                                                                           |
| Accept/reject role             | `group.role.manage` or `global.role.manage`                                                             | Adds role assignment or rejects request.                                                                                                                                                |
| Assign/remove role directly    | `group.role.manage` or `global.role.manage`                                                             | Group role managers change `EDITOR` or `REVIEWER`; global role managers may also change `GROUP_ADMIN`.                                                                                  |
| Grant/revoke global permission | `global.permission.manage`                                                                              | Changes any global permission grant. Global `ADMIN` may do this.                                                                                                                        |
| Grant/revoke group permission  | `group.permission.manage`                                                                               | Changes any additive group grant in the group. Global `ADMIN` may do this in every group.                                                                                               |
| Revoke membership              | `group.membership.manage` or `global.membership.manage`                                                 | Permanently deletes membership, roles, grants, subscriptions, and all membership-related role requests. `GROUP_ADMIN` membership requires global `ADMIN` or `global.membership.manage`. |

Membership states are `PENDING`, `ACTIVE`, and `REJECTED`. Pending or active member cannot request duplicate membership. Rejected user may submit new request, resetting membership to `PENDING`. Revocation retains no record and has no restoration action; user may submit new membership request immediately.

Membership holding `GROUP_ADMIN` may be revoked only by global `ADMIN` or `global.membership.manage`; this prevents group membership managers from indirectly removing Group Admin assignment.

Manual role-request decisions require active membership. Membership rejection is system cascade exempt from this rule; revocation deletes pending requests. Rejected role may be requested again after membership is active. At most one matching role request may be pending. Direct assignment or removal is blocked while matching request is pending; manager must decide request first.

First created user receives global `ADMIN` role. Removing its legacy `users.is_super_admin` column leaves global role assignment as sole source of admin authority. Active admin means active, unblocked user assigned `ADMIN`. Admin eligibility changes must serialize and fail when resulting active-admin count would be zero.

### User Blocking

Global `ADMIN` may block user with required non-empty reason and may later unblock user. Block stores reason, actor, and timestamp. Memberships, roles, grants, and subscriptions remain stored but dormant. Unblocking restores their effect subject to normal active-membership checks.

Blocked identity remains attributable but has unauthenticated-level access: public reads remain allowed, while every authenticated action and request is denied due to blocked status. Blocking last active global `ADMIN` is forbidden. Attempt by blocked user to perform authenticated action emits attributed configured global security event; public reads do not.

## Metadata Import

Metadata import is restricted to global `ADMIN`. Import may create/update canonical organizations, contact points, categories, category schemes, and resource `DRAFT` content only. Imported categories and category-scheme changes are applied to specified group and must obey group ownership and hierarchy invariants. Imported resources must obey one-active-change rule and cannot enter `SUBMITTED`, `REJECTED`, `PUBLISHED`, or `ARCHIVED` directly. Import must obey revision, ownership, hierarchy, category-scheme, identifier, and atomicity invariants. Publication always uses normal submit/publish lifecycle.

## Notification Subscriptions

Notifications are separate from authorization. Permission answers whether user may act; subscription answers whether eligible user wants notification.

This design defines storage and eligibility only. It does not require specific events or delivery behavior.

| Subscription           | Eligibility                                                                     |
| ---------------------- | ------------------------------------------------------------------------------- |
| Global event           | Active global `ADMIN` only.                                                     |
| Group event            | Active membership in group.                                                     |
| Privileged group event | Active membership plus group-scoped `required_permission_key` defined by event. |

Users manage own eligible subscriptions. Global `ADMIN` may manage all subscriptions. Group subscriptions are opt-in. Assigning global `ADMIN` creates subscriptions to every existing global event by default; adding global event creates subscriptions for every existing global `ADMIN`; administrators may opt out by deleting rows. Removing global `ADMIN` deletes global subscription rows. Reassignment creates current defaults again.

Global events are administrator-only. Only explicitly configured security events are emitted for denied actions; ordinary authorization denials do not notify. Blocked-user authenticated attempt is configured security event. Delivery eligibility is rechecked when sending, so stale subscriptions never grant information access. Current channel is `EMAIL`.

## Conceptual Database Schema

### Lifecycle

| Table                                         | Key columns                                                                                                                                  | Constraints                                                                                                                     |
| --------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `groups`                                      | Group fields: `id`, `name`, `description`; category scheme fields: `concept`, `label`, `logo`, `page`, `color`, `position`; actor/timestamps | One row represents group and its category scheme; unique name; no special `all` group.                                          |
| `categories`                                  | `id` UUID, `group_id`, `parent_category_id`, `name`, normalized name                                                                         | Zero/one parent in same group; acyclic; immutable group; normalized name unique among siblings/top level.                       |
| `organizations`                               | `id`, `name`, address, logo, page, email, LEI fields                                                                                         | Local canonical organization; unique name.                                                                                      |
| `identifiers`                                 | `id`, `type`, `value`, one nullable owner FK per supported owner type                                                                        | Exactly one owner; immutable owner; unique owner plus trimmed case-sensitive `(type, value)`; owner deletion cascades.          |
| `organization_memberships`                    | `child_organization_id`, `parent_organization_id`                                                                                            | Multiple parents allowed; self-links/cycles forbidden; confirmed organization deletion removes links.                           |
| `contact_points`                              | `id`, `email`, `role`                                                                                                                        | Unique `(email, role)`; canonical shared record.                                                                                |
| `organization_contact_points`                 | `organization_id`, `contact_point_id`                                                                                                        | Unique link; organization or contact-point deletion cascades link.                                                              |
| `datasets`                                    | `id`, `group_id`, `created_by`                                                                                                               | Logical dataset; immutable UUID and group.                                                                                      |
| `software_applications`                       | `id`, `group_id`, `created_by`                                                                                                               | Logical software application; immutable UUID and group.                                                                         |
| `software_source_codes`                       | `id`, `group_id`, `created_by`                                                                                                               | Logical software source code; immutable UUID and group.                                                                         |
| `dataset_versions`                            | `id`, `dataset_id`, `status`, `revision`, metadata, source timestamps, actor/timestamps, latest rejection fields                             | One published and one active change version per dataset.                                                                        |
| `software_application_versions`               | `id`, `software_application_id`, `status`, `revision`, application metadata, actor/timestamps, latest rejection fields                       | One published and one active change version per software application.                                                           |
| `software_source_code_versions`               | `id`, `software_source_code_id`, `status`, `revision`, source-code metadata, actor/timestamps, latest rejection fields                       | One published and one active change version per software source code.                                                           |
| Version-owned metadata tables                 | `dataset_version_id`, entity fields                                                                                                          | Cascade when draft/rejected version is discarded.                                                                               |
| `dataset_version_providers`                   | `dataset_version_id`, `organization_id`                                                                                                      | Unique per version/organization; organization deletion cascades link.                                                           |
| Version-owned web services                    | `dataset_version_id`, optional `provider_organization_id`                                                                                    | Organization deletion clears provider link.                                                                                     |
| `software_application_version_creators`       | `software_application_version_id`, `organization_id`                                                                                         | Unique per version/organization; organization deletion cascades link.                                                           |
| `software_source_code_version_creators`       | `software_source_code_version_id`, `organization_id`                                                                                         | Unique per version/organization; organization deletion cascades link.                                                           |
| `software_application_version_contact_points` | `software_application_version_id`, `contact_point_id`                                                                                        | Unique link; contact-point deletion cascades link.                                                                              |
| `software_source_code_version_contact_points` | `software_source_code_version_id`, `contact_point_id`                                                                                        | Unique link; contact-point deletion cascades link.                                                                              |
| `dataset_version_contact_points`              | `dataset_version_id`, `contact_point_id`                                                                                                     | Unique link; contact-point deletion cascades link.                                                                              |
| `web_service_contact_points`                  | `web_service_id`, `contact_point_id`                                                                                                         | Unique link; contact-point deletion cascades link.                                                                              |
| `dataset_version_categories`                  | `dataset_version_id`, `group_id`, `category_id`                                                                                              | Stores canonical category IDs only; category `group_id` must equal dataset group; category deletion cascades link.              |
| `software_application_version_categories`     | `software_application_version_id`, `group_id`, `category_id`                                                                                 | Stores canonical category IDs only; category `group_id` must equal software application group; category deletion cascades link. |
| `software_source_code_version_categories`     | `software_source_code_version_id`, `group_id`, `category_id`                                                                                 | Stores canonical category IDs only; category `group_id` must equal software source code group; category deletion cascades link. |

All resource-version tables store latest `rejection_note`, `rejected_by`, and `rejected_at`; repeated rejection replaces these fields and successful publication clears them. Lifecycle actor and timestamp fields record submission, publication, archival, and latest rejection without a generic event-history table. Dataset versions also retain nullable `source_created_at` and `source_updated_at` from upstream metadata; these do not represent metadata lifecycle record creation or editing.

`groups` row is one canonical group/category-scheme object: it owns categories, defines resource ownership, and is the authorization boundary. Separate `category_schemes` table is removed; `categories.scheme_id` becomes `categories.group_id`. Categories have stable UUIDs and one nullable parent instead of many-to-many child table. Partial unique constraints enforce normalized sibling and top-level names. Resource versions link only canonical category IDs from own group and do not snapshot category fields.

Logical resource tables hold stable UUID identity and group. Their version tables hold mutable metadata and lifecycle state. A new version references the same logical row; a copy creates a new logical row with a new UUID. `based_on_version_id` is not stored.

`identifiers` owner columns are `dataset_version_id`, `organization_id`, `software_application_version_id`, and `software_source_code_version_id`; exactly-one check prevents sharing and orphan rows. Content may repeat across owners. Resource-version cloning creates new identifier IDs. UUID conflict with different owner fails rather than transferring ownership.

Canonical deletion uses cascading/nullable link behavior through the normal client-requested delete operation. The client owns impact preview and user confirmation; Metadata Lifecycle atomically unlinks references, deletes the canonical entity, and returns `204 No Content`. Direct repository/database deletion remains an implementation detail of that operation and must not be exposed as a separate bypass.

Group references from logical datasets, software applications, and software source codes restrict deletion. Group-owned categories, memberships, role assignments, grants, requests, and subscriptions cascade through group deletion operation.

### Authorization

| Table                      | Purpose                                                                          | Key                                                     |
| -------------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `users`                    | Identity/status plus nullable block reason, actor, and timestamp                 | `id`; block fields required when status is `BLOCKED`    |
| `permissions`              | Fixed permission catalog with key, UI label, description, and scope              | `key`                                                   |
| `roles`                    | Fixed `EDITOR`, `REVIEWER`, `GROUP_ADMIN`, and `ADMIN` definitions               | `key`                                                   |
| `role_permissions`         | Fixed role bundles                                                               | `(role_key, permission_key)`                            |
| `group_memberships`        | User relationship with `PENDING`, `ACTIVE`, or `REJECTED` status; no role column | `(user_id, group_id)`                                   |
| `group_membership_roles`   | Multiple fixed roles per membership                                              | `(user_id, group_id, role_key)`                         |
| `user_global_roles`        | Global role assignments                                                          | `(user_id, role_key)`                                   |
| `global_permission_grants` | Additive global permissions                                                      | `(user_id, permission_key)`                             |
| `group_permission_grants`  | Additive permissions in one group                                                | `(user_id, group_id, permission_key)`                   |
| `group_role_requests`      | `PENDING`, `ACCEPTED`, or `REJECTED` role requests with actor/timestamps         | `id`; at most one pending request per user, group, role |

Grant tables record `granted_by` and `granted_at`. Role-request decisions record decision actor and timestamp. Revoking membership deletes every role-request row for user/group regardless of status. Existing `group_permissions` does not model per-user grants and is replaced by scoped grant tables.

Database constraints must enforce scopes: only `ADMIN` may appear in `user_global_roles`; only `EDITOR`, `REVIEWER`, and `GROUP_ADMIN` may appear in `group_membership_roles`; global grant table accepts only global permissions; group grant table accepts only group permissions. Application policy permits only global `ADMIN` to change `GROUP_ADMIN` assignments.

Per-user category scheme access grants and `user_category_scheme_grants` are removed without replacement. Group metadata is public; active membership controls non-public resource access and group-scoped actions. Global `category_scheme.manage` manages group/category-scheme metadata; group-scoped `category.manage` is sole non-admin permission for managing canonical categories in that group.

### Notifications

| Table                               | Key                                        | Notes                                                                          |
| ----------------------------------- | ------------------------------------------ | ------------------------------------------------------------------------------ |
| `notification_event_types`          | `key`                                      | Fixed event definition, allowed scope, and nullable `required_permission_key`. |
| `global_notification_subscriptions` | `(user_id, event_type, channel)`           | Global-admin-only; created enabled by default and removable for opt-out.       |
| `group_notification_subscriptions`  | `(user_id, group_id, event_type, channel)` | Requires membership; removed on membership revocation.                         |

Notification preferences do not add columns to `users`.

## Required Scenarios

| Scenario                                                              | Result                                                                                                            |
| --------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| Editor and reviewer roles assigned to same user                       | User may submit and publish own submission.                                                                       |
| Two editors update same draft revision                                | First succeeds; second receives conflict.                                                                         |
| Editor copies published dataset to another group                      | New logical dataset/draft uses new UUID, copied identifier values in new rows, and zero categories.               |
| Editor creates software-application draft from publication            | New version has same logical application UUID and cloned metadata, creators, identifiers, and category links.     |
| Editor copies published software source code                          | New logical source code/draft has new UUID, cloned metadata, creators, and identifiers, and zero categories.      |
| Editor links category from another group's category scheme            | Denied even when editor belongs to both groups.                                                                   |
| Editor attempts to create category through dataset draft              | Denied; category management is outside dataset lifecycle.                                                         |
| Category manager creates category with duplicate sibling name         | Denied after normalized-name comparison.                                                                          |
| Client previews referenced category impact, then requests delete      | Full subtree and every dataset/software link are deleted atomically, including published/archived links.          |
| Client previews referenced organization impact, then requests delete  | Resource/hierarchy links and owned identifiers are removed, then organization is deleted.                         |
| Contact-point manager edits canonical contact                         | Change is visible through every dataset-version/web-service reference.                                            |
| Client previews referenced contact-point impact, then requests delete | All contact links are removed, then contact point is deleted.                                                     |
| Group owns no resource                                                | Category scheme, categories, memberships, access assignments, requests, and subscriptions are deleted with group. |
| Group Admin unpublishes resource                                      | Public version becomes archived; later draft may publish normally.                                                |
| Role manager tries to assign `GROUP_ADMIN`                            | Denied; only global `ADMIN` may assign it.                                                                        |
| Membership is revoked                                                 | Membership and related access/subscriptions/pending requests are deleted; user may request again.                 |
| Blocked user performs public read                                     | Allowed.                                                                                                          |
| Blocked user attempts authenticated action                            | Denied and configured global security event emitted; stored access remains dormant.                               |
| System Admin unblocks user                                            | Prior memberships, roles, grants, and subscriptions become effective again when otherwise valid.                  |
| New System Admin or global event is added                             | System Admin is subscribed to all global events by default and may opt out.                                       |
| Non-admin attempts metadata import                                    | Denied.                                                                                                           |
| Identifier is linked to second owner                                  | Denied; copy requires new identifier row.                                                                         |
| Attempt blocks/removes last active System Admin                       | Denied.                                                                                                           |
