### Behavior: anonymous callers get 401 on group list.
GET {{baseUrl}}/v1/groups

HTTP 401
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.status" == 401
