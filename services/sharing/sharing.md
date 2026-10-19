# Sharing

## Endpoints

| Method + URL                                  | Behavior                                                    |
| --------------------------------------------- | ----------------------------------------------------------- |
| `GET /v1/official-configurations`             | Public list of all official configurations                  |
| `POST /v1/configurations`                     | Create personal configuration owned by authenticated caller |
| `GET /v1/configurations`                      | List own configurations, including official ones            |
| `GET /v1/configurations?owner_id={id}`        | Manager-only list for specified creator                     |
| `GET /v1/configurations?scope=all`            | Manager-only list of every configuration                    |
| `GET /v1/configurations/{id}`                 | Unified detail endpoint; optional authentication            |
| `PUT /v1/configurations/{id}`                 | Replace configuration                                       |
| `DELETE /v1/configurations/{id}`              | Delete configuration                                        |
| `PUT /v1/configurations/{id}/official-status` | Manager-only promotion/demotion                             |

`owner_id` and `scope=all` together return `400`.

## Response shapes

All lists return **bare arrays of summaries**. No pagination, wrapper, or `total`.

```json
{
  "id": "uuid",
  "title": "Workspace",
  "description": null,
  "list_of_services": [],
  "is_official": false,
  "is_public": true,
  "created_at": "2026-10-09T10:00:00Z",
  "updated_at": "2026-10-09T10:00:00Z"
}
```

Detail adds `configuration`, containing opaque JSON.

Authenticated callers with `global.configuration.manage` additionally receive `owner_id`—**except public official list, which never exposes ownership**. Otherwise field is absent.

| Write          | Response                      |
| -------------- | ----------------------------- |
| Create         | `201` + detail; no `Location` |
| Replace        | `200` + updated detail        |
| Promote/demote | `200` + updated detail        |
| Delete         | `204`, no body                |

## Access and lifecycle

- Public configurations readable by anyone with ID.
- Private configurations readable only by creator or manager.
- Non-official configurations editable/deletable by creator or manager.
- Official configurations editable/deletable **only by managers**.
- Promotion/demotion preserves creator.
- Managers bypass ownership/visibility restrictions, not validation or creation quota.

Official-status body:

```json
{ "is_official": true }
```

Promotion forces public. Demotion keeps public and restores creator’s edit/delete access. Repeating current status succeeds without changing configuration.

Ordinary create/replace cannot set owner or official status. PUT:

- Non-official: absent/null/false `is_public` means private.
- Official: absent/null/true keeps public; explicit false returns `400`.

## Quota

Creation allowed only while creator has **fewer than 50 non-official configurations**. Public and private count; official ones don’t.

Demotion always allowed, even above quota. Creation remains blocked until count drops below 50. Concurrent creation must enforce limit atomically.

## Errors

Existing Problem format:

- `400`: invalid input/filter combination or official visibility violation.
- `401`: required authentication absent.
- `403`: authenticated caller lacks access to operation.
- `404`: missing record or inaccessible private configuration.
- `409`: creation quota reached.
