### Behavior: super-admin can create group.
POST {{baseUrl}}/v1/groups
Authorization: Bearer {{adminBearerToken}}
Content-Type: application/json
{
  "slug": "{{adminLifecycleGroupSlug}}",
  "name": "Hurl Admin Lifecycle Group",
  "description": "Group created by Hurl group authz tests",
  "is_system": false
}

HTTP 201
[Captures]
groupId: jsonpath "$.id"
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.slug" == "{{adminLifecycleGroupSlug}}"
jsonpath "$.name" == "Hurl Admin Lifecycle Group"
jsonpath "$.description" == "Group created by Hurl group authz tests"
jsonpath "$.is_system" == false

### Behavior: group list includes created group.
GET {{baseUrl}}/v1/groups
Authorization: Bearer {{adminBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
body contains "\"id\":\"{{groupId}}\""
body contains "\"slug\":\"{{adminLifecycleGroupSlug}}\""
body contains "\"name\":\"Hurl Admin Lifecycle Group\""

### Behavior: group detail returns created group.
GET {{baseUrl}}/v1/groups/{{groupId}}
Authorization: Bearer {{adminBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupId}}"
jsonpath "$.slug" == "{{adminLifecycleGroupSlug}}"
jsonpath "$.name" == "Hurl Admin Lifecycle Group"

### Behavior: super-admin direct membership request becomes active.
POST {{baseUrl}}/v1/groups/{{groupId}}/memberships
Authorization: Bearer {{adminBearerToken}}
Content-Type: application/json
{
  "role": "VIEWER"
}

HTTP 201
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.user_id" == "{{adminUserId}}"
jsonpath "$.group_id" == "{{groupId}}"
jsonpath "$.role" == "VIEWER"
jsonpath "$.status" == "ACTIVE"

### Behavior: group detail exposes caller membership summary.
GET {{baseUrl}}/v1/groups/{{groupId}}
Authorization: Bearer {{adminBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupId}}"
jsonpath "$.membership.role" == "VIEWER"
jsonpath "$.membership.status" == "ACTIVE"

### Behavior: roster endpoint returns full membership summary.
GET {{baseUrl}}/v1/groups/{{groupId}}/memberships
Authorization: Bearer {{adminBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$[0].user.id" == "{{adminUserId}}"
jsonpath "$[0].user.href" == "v1/users/{{adminUserId}}"
jsonpath "$[0].role" == "VIEWER"
jsonpath "$[0].status" == "ACTIVE"

### Behavior: membership patch updates role and status.
PATCH {{baseUrl}}/v1/groups/{{groupId}}/memberships/{{adminUserId}}
Authorization: Bearer {{adminBearerToken}}
Content-Type: application/json
{
  "role": "REVIEWER",
  "status": "REVOKED"
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.user_id" == "{{adminUserId}}"
jsonpath "$.group_id" == "{{groupId}}"
jsonpath "$.role" == "REVIEWER"
jsonpath "$.status" == "REVOKED"

### Behavior: group patch updates mutable fields.
PATCH {{baseUrl}}/v1/groups/{{groupId}}
Authorization: Bearer {{adminBearerToken}}
Content-Type: application/json
{
  "name": "Hurl Admin Lifecycle Group Updated",
  "description": "Updated by Hurl group authz tests"
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupId}}"
jsonpath "$.name" == "Hurl Admin Lifecycle Group Updated"
jsonpath "$.description" == "Updated by Hurl group authz tests"

### Behavior: super-admin can delete own membership.
DELETE {{baseUrl}}/v1/groups/{{groupId}}/memberships/{{adminUserId}}
Authorization: Bearer {{adminBearerToken}}

HTTP 204

### Behavior: super-admin can delete group.
DELETE {{baseUrl}}/v1/groups/{{groupId}}
Authorization: Bearer {{adminBearerToken}}

HTTP 204

### Behavior: deleted group no longer resolves.
GET {{baseUrl}}/v1/groups/{{groupId}}
Authorization: Bearer {{adminBearerToken}}

HTTP 404
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 404
