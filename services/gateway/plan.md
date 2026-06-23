# Gateway Authorization Alignment Plan

Current state: only `/v1/users` routes exist in gateway. Group and membership routes from `authorization.md` and updated OpenAPI are not wired yet.

## Shared groundwork
- [ ] Add gateway API DTOs for groups, memberships, and membership summaries.
- [ ] Add service helpers for group admin checks and caller membership lookup.
- [ ] Add router wiring for `/v1/groups` and membership routes.
- [ ] Keep route handlers thin: parse request, call service, map domain errors, write response.
- [ ] Update OpenAPI/docs to match final behavior, including user patch admin fields.

## 1. `GET /users`
- [x] Keep current super-admin-only list behavior.
- [x] Verify response shape matches `UserSummary`.
- [x] Keep regression tests for `200`, `401`, and `403`.

## 2. `GET /users/{id}`
- [x] Preserve `self` alias for authenticated user.
- [x] Allow authenticated user to read self and super admin to read any user.
- [x] Add memberships to `User` response for `self` and super admin.
- [x] Add tests for `self`, super admin, forbidden other-user access, and not found.

## 3. `PATCH /users/{id}`
- [x] Keep support for admin fields (`is_super_admin`, `status`) per current decision.
- [x] Allow self patch for own profile fields.
- [x] Allow super admin patch for any user.
- [x] Reject non-admin attempts to patch other users.
- [x] Add tests for self patch, super admin patch, forbidden access, and bad request payloads.
- [x] Update OpenAPI request schema so docs match kept behavior.

## 4. `POST /groups`
- [x] Add create-group route and handler.
- [x] Map request payload to gateway group DTO.
- [x] Enforce super-admin-only access.
- [x] Add tests for `201`, `401`, `403`, and request validation.

## 5. `GET /groups`
- [x] Add list-groups route and handler.
- [x] Fetch groups from upstream and return caller membership summary when present.
- [x] Keep full roster out of regular group responses.
- [x] Add tests for authenticated caller with and without membership.

## 6. `GET /groups/{groupId}`
- [x] Add group-detail route and handler.
- [x] Return group data plus only caller membership summary.
- [x] Do not expose full roster in this response.
- [x] Add tests for `200`, `401`, `404`, and visibility constraints.

## 7. `PATCH /groups/{groupId}`
- [x] Add patch-group route and handler.
- [x] Enforce super-admin or active group-admin access.
- [x] Reuse one group-admin authorization helper for this and membership routes.
- [x] Add tests for super admin, active group admin, forbidden caller, and not found.

## 8. `DELETE /groups/{groupId}`
- [x] Add delete-group route and handler.
- [x] Enforce super-admin-only access.
- [x] Add tests for `204`, `401`, `403`, and `404`.

## 9. `POST /groups/{groupId}/memberships`
- [x] Add request-membership route and handler.
- [x] Force target user to authenticated user.
- [x] Create `PENDING` membership for self-service request.
- [x] Super Admins memberships are created `ACTIVE` by default
- [x] Add tests for request creation, validation, and authorization.

## 10. `GET /groups/{groupId}/memberships`
- [x] Add roster route and handler.
- [x] Enforce super-admin or active group-admin access.
- [x] Return full roster only from this endpoint.
- [x] Add tests for `200`, `401`, `403`, and `404`.

## 11. `PATCH /groups/{groupId}/memberships/{userId}`
- [x] Add membership patch route and handler.
- [x] Read current membership, merge partial payload, then write full upstream update.
- [x] Enforce super-admin or active group-admin access.
- [x] Add tests for role-only patch, status-only patch, forbidden caller, and not found.

## 12. `DELETE /groups/{groupId}/memberships/{userId}`
- [x] Add membership delete route and handler.
- [x] Allow self leave/cancel for own membership.
- [x] Allow super admin and active group admin to delete any membership in target group.
- [x] Add tests for self delete, admin delete, forbidden caller, and not found.

## Suggested order
- [ ] Implement `GET /users/{id}` first.
- [ ] Implement `PATCH /users/{id}` next, then update OpenAPI docs.
- [ ] Add group CRUD endpoints.
- [ ] Add membership list and delete/update flows after group-admin helper exists.
- [ ] Finish with request-membership flow and roster visibility tests.
