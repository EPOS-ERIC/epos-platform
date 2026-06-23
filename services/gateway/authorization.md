# Authorization

## Actors

- Authenticated user
- Super admin
- Group admin

## Roles

- `super admin`: can manage any user, any group, and any membership.
- `group admin`: can manage only groups where they are active admin, including group memberships in those groups.
- `user`: can manage only self actions and memberships for self.

## Scope Rules

- `self` means authenticated user.
- `id=self` is valid on user read and patch endpoints.
- Group admin rights are checked per group and apply to group membership actions for that group.
- Roster access is explicit, not implicit.

## Capability Matrix

| Capability         | Endpoint                                        | Actor                                        | Conditions                                                                              |
| ------------------ | ----------------------------------------------- | -------------------------------------------- | --------------------------------------------------------------------------------------- |
| List users         | `GET /users`                                    | super admin                                  | none                                                                                    |
| Read user          | `GET /users/{id}`                               | authenticated user, super admin              | `id=self` for user (or user id); super admin may read any user                          |
| Patch user         | `PATCH /users/{id}`                             | authenticated user, super admin              | `id=self` for user (or user id); super admin may patch any user                         |
| Create group       | `POST /groups`                                  | super admin                                  | none                                                                                    |
| List groups        | `GET /groups`                                   | authenticated user                           | public to authenticated users                                                           |
| Read group         | `GET /groups/{groupId}`                         | authenticated user                           | caller sees only own membership summary                                                 |
| Patch group        | `PATCH /groups/{groupId}`                       | super admin, group admin                     | group admin must be active admin for target group                                       |
| Delete group       | `DELETE /groups/{groupId}`                      | super admin                                  | none                                                                                    |
| Request membership | `POST /groups/{groupId}/memberships`            | authenticated user                           | self-service access request for authenticated user only                                 |
| Update membership  | `PATCH /groups/{groupId}/memberships/{userId}`  | super admin, group admin                     | group admin must be active admin for target group                                       |
| Delete membership  | `DELETE /groups/{groupId}/memberships/{userId}` | authenticated user, super admin, group admin | self may leave/cancel own membership; group admin must be active admin for target group |
| List group roster  | `GET /groups/{groupId}/memberships`             | super admin, group admin                     | group admin must be active admin for target group                                       |

## Membership Flow

- Self-service request creates `PENDING` membership.
- Super admin request to join creates `ACTIVE` membership directly.
- Rejected membership may be requested again.
- `DELETE` on pending membership means cancel request.
- `DELETE` on active membership means leave group.

## Visibility Rules

- `GET /groups/{groupId}` must not expose full roster to regular users.
- Default group responses may include only caller membership summary.
- Full roster is only exposed by `GET /groups/{groupId}/memberships`.
- `GET /users/{id}` may expose memberships for `self` and super admin.

## Error Rules

- `401` when no valid token.
- `403` when authenticated but not allowed.
- `404` when target resource does not exist.
