### Behavior: group admin can list roster for own group.
GET {{baseUrl}}/v1/groups/{{groupAId}}/memberships
Authorization: Bearer {{groupAdminBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
body contains "\"id\":\"{{groupAdminUserId}}\""
body contains "\"role\":\"ADMIN\""
body contains "\"status\":\"ACTIVE\""

### Behavior: regular user can create membership request for group-admin workflow.
POST {{baseUrl}}/v1/groups/{{groupAId}}/memberships
Authorization: Bearer {{userBearerToken}}
Content-Type: application/json
{
  "role": "VIEWER"
}

HTTP 201
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.user_id" == "{{userId}}"
jsonpath "$.group_id" == "{{groupAId}}"
jsonpath "$.status" == "PENDING"

### Behavior: group admin can patch member inside own group.
PATCH {{baseUrl}}/v1/groups/{{groupAId}}/memberships/{{userId}}
Authorization: Bearer {{groupAdminBearerToken}}
Content-Type: application/json
{
  "role": "REVIEWER",
  "status": "ACTIVE"
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.user_id" == "{{userId}}"
jsonpath "$.group_id" == "{{groupAId}}"
jsonpath "$.role" == "REVIEWER"
jsonpath "$.status" == "ACTIVE"

### Behavior: patched regular user sees updated membership summary.
GET {{baseUrl}}/v1/groups/{{groupAId}}
Authorization: Bearer {{userBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupAId}}"
jsonpath "$.membership.role" == "REVIEWER"
jsonpath "$.membership.status" == "ACTIVE"

### Behavior: group admin can patch own group.
PATCH {{baseUrl}}/v1/groups/{{groupAId}}
Authorization: Bearer {{groupAdminBearerToken}}
Content-Type: application/json
{
  "name": "Hurl Authz Group A Updated"
}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupAId}}"
jsonpath "$.name" == "Hurl Authz Group A Updated"

### Behavior: group admin can delete member inside own group.
DELETE {{baseUrl}}/v1/groups/{{groupAId}}/memberships/{{userId}}
Authorization: Bearer {{groupAdminBearerToken}}

HTTP 204

### Behavior: group admin cannot list roster for other group.
GET {{baseUrl}}/v1/groups/{{groupBId}}/memberships
Authorization: Bearer {{groupAdminBearerToken}}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: group admin cannot patch other group.
PATCH {{baseUrl}}/v1/groups/{{groupBId}}
Authorization: Bearer {{groupAdminBearerToken}}
Content-Type: application/json
{
  "name": "Forbidden Group Patch"
}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403
