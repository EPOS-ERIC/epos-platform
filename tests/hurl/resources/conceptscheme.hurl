GET {{baseUrl}}/v1/conceptscheme/category%3Asoftware_jup%2Fv1.0

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "category:software_jup/v1.0"
jsonpath "$.label" == "Jupyter Notebooks"
jsonpath "$.top_concepts[0].href" == "v1/concept/category:facets/software-theme/v1.0"
