GET {{baseUrl}}/v1/search?type=concept

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "unsupported search type: concept"

GET {{baseUrl}}/v1/search?type=dataset,distribution

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "unsupported search type: distribution"

GET {{baseUrl}}/v1/search?pageSize=0

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "page size must be >= 1"

GET {{baseUrl}}/v1/search?pageSize=101

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "page size must be <= 100"

GET {{baseUrl}}/v1/search?pageNumber=0

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "page number must be >= 1"

GET {{baseUrl}}/v1/search?dateFrom=2025-01-02T00:00:00Z&dateTo=2025-01-01T00:00:00Z

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "date to must be greater than or equal to date from"

GET {{baseUrl}}/v1/search?dateFrom=not-a-date

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "date from must be rfc3339"

GET {{baseUrl}}/v1/search?versioningStatus=bogus

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "invalid_query_parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "versioning status must be one of draft, submitted, published, archived"
