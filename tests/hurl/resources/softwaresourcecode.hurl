GET {{baseUrl}}/v1/softwaresourcecode/https%3A%2F%2Fdoi.org%2F10.5281%2Fzenodo.17598403

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
jsonpath "$.id" == "https://doi.org/10.5281/zenodo.17598403"
jsonpath "$.name" == "Discovery, automation/scripting and integration of EPOS data and services. Basic workflows in JupyterLab."
jsonpath "$.repository" == "https://raw.githubusercontent.com/daoane/epos-multidisciplinary-service-integration/refs/heads/main/Basic_workflows_service_integration.ipynb"
jsonpath "$.provider[0].href" == "v1/organization/ROR:048a87296/v1.0"
jsonpath "$.concepts[0].href" == "v1/concept/category:software_jup_conc_multi/v1.0"
