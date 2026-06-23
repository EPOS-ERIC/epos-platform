GET {{baseUrl}}/v1/resources/organization/ROR%3A048a87296%2Fv1.0

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "ROR:048a87296/v1.0"
jsonpath "$.name" == "Uppsala University"
jsonpath "$.url" == "http://www.uu.se"
jsonpath "$.identifiers[0].value" == "https://ror.org/048a87296"
