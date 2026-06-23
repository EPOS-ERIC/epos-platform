### Behavior: authenticated regular user can list groups.
GET {{baseUrl}}/v1/groups
Authorization: Bearer {{userBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
body contains "\"id\":\"{{groupAId}}\""
body contains "\"id\":\"{{groupBId}}\""

### Behavior: authenticated regular user can read group detail.
GET {{baseUrl}}/v1/groups/{{groupBId}}
Authorization: Bearer {{userBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupBId}}"
jsonpath "$.slug" == "{{groupBSlug}}"

### Behavior: regular user cannot create group.
POST {{baseUrl}}/v1/groups
Authorization: Bearer {{userBearerToken}}
Content-Type: application/json
{
  "slug": "forbidden-user-group",
  "name": "Forbidden User Group"
}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: regular user cannot patch group.
PATCH {{baseUrl}}/v1/groups/{{groupBId}}
Authorization: Bearer {{userBearerToken}}
Content-Type: application/json
{
  "name": "Forbidden Patch"
}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: regular user cannot delete group.
DELETE {{baseUrl}}/v1/groups/{{groupBId}}
Authorization: Bearer {{userBearerToken}}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: regular user cannot read full roster.
GET {{baseUrl}}/v1/groups/{{groupBId}}/memberships
Authorization: Bearer {{userBearerToken}}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: regular user can request membership and gets pending status.
POST {{baseUrl}}/v1/groups/{{groupBId}}/memberships
Authorization: Bearer {{userBearerToken}}
Content-Type: application/json
{
  "role": "VIEWER"
}

HTTP 201
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.user_id" == "{{userId}}"
jsonpath "$.group_id" == "{{groupBId}}"
jsonpath "$.role" == "VIEWER"
jsonpath "$.status" == "PENDING"

### Behavior: group detail shows only caller membership summary after request.
GET {{baseUrl}}/v1/groups/{{groupBId}}
Authorization: Bearer {{userBearerToken}}

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "{{groupBId}}"
jsonpath "$.membership.role" == "VIEWER"
jsonpath "$.membership.status" == "PENDING"

### Behavior: regular user cannot delete another user's membership.
DELETE {{baseUrl}}/v1/groups/{{groupAId}}/memberships/{{groupAdminUserId}}
Authorization: Bearer {{userBearerToken}}

HTTP 403
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 403

### Behavior: regular user can delete own pending membership.
DELETE {{baseUrl}}/v1/groups/{{groupBId}}/memberships/{{userId}}
Authorization: Bearer {{userBearerToken}}

HTTP 204
