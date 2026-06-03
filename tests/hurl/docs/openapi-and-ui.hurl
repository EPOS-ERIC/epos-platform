GET {{baseUrl}}/openapi.yaml

HTTP 200
[Asserts]
header "Content-Type" contains "application/yaml"
body contains "openapi: 3.0.3"
body contains "operationId: searchMetadata"

GET {{baseUrl}}/scalar

HTTP 301
[Asserts]
header "Location" == "/scalar/"

GET {{baseUrl}}/swagger

HTTP 301
[Asserts]
header "Location" == "/swagger/"
