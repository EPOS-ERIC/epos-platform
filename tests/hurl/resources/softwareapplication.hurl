GET {{baseUrl}}/v1/softwareapplication/https%3A%2F%2Fepos-eu.org%2Fanthropogenic_hazards%2Fsoftware%2FCatalogToXls%2Fv1.0

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0"
jsonpath "$.name" == "Catalog to XLS converter"
jsonpath "$.download_url" == "https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0/download"
jsonpath "$.provider[0].href" == "v1/organization/ROR:048a87296/v1.0"
jsonpath "$.concepts[0].href" == "v1/concept/category:software_appl_conc_multi/v1.0"
