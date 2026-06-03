GET {{baseUrl}}/v1/search?type=softwareapplication

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.items[0].type" == "softwareapplication"
jsonpath "$.items[0].name" == "Catalog to XLS converter"
jsonpath "$.items[0].provider[0].name" == "Uppsala University"
jsonpath "$.items[0].href" == "v1/softwareapplication/https://epos-eu.org/anthropogenic_hazards/software/CatalogToXls/v1.0"

GET {{baseUrl}}/v1/search?type=softwaresourcecode

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.items[0].type" == "softwaresourcecode"
jsonpath "$.items[0].name" == "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab."
jsonpath "$.items[0].provider[0].name" == "Uppsala University"
