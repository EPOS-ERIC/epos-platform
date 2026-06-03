GET {{baseUrl}}/v1/search?type=concept

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "Unsupported search type: concept"

GET {{baseUrl}}/v1/search?type=dataset,distribution

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "Unsupported search type: distribution"

GET {{baseUrl}}/v1/search?pageSize=0

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "pageSize must be between 1 and 100"

GET {{baseUrl}}/v1/search?pageSize=101

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "pageSize must be between 1 and 100"

GET {{baseUrl}}/v1/search?pageNumber=0

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "pageNumber must be >= 1"

GET {{baseUrl}}/v1/search?dateFrom=2025-01-02T00:00:00Z&dateTo=2025-01-01T00:00:00Z

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "dateTo must be greater than or equal to dateFrom"

GET {{baseUrl}}/v1/search?dateFrom=not-a-date

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "dateFrom must be RFC3339"

GET {{baseUrl}}/v1/search?versioningStatus=bogus

HTTP 400
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "Invalid query parameter"
jsonpath "$.status" == 400
jsonpath "$.detail" == "versioningStatus must be one of DRAFT, SUBMITTED, PUBLISHED, ARCHIVED"
