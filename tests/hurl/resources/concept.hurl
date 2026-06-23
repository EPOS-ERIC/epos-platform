GET {{baseUrl}}/v1/resources/concept/category%3Asoftware_jup_conc_multi%2Fv1.0

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "category:software_jup_conc_multi/v1.0"
jsonpath "$.name" == "Multidisciplinary software"
jsonpath "$.in_scheme.href" == "v1/resources/conceptscheme/category:software_jup/v1.0"
