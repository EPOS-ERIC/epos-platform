GET {{baseUrl}}/v1/dataset/https%3A%2F%2Fexample.org%2Fmissing-dataset

HTTP 404
[Asserts]
header "Content-Type" contains "application/problem+json"
jsonpath "$.title" == "resource_not_found"
jsonpath "$.status" == 404
jsonpath "$.detail" == "dataset not found"
