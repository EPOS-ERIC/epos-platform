GET {{baseUrl}}/v1/concept

HTTP 200
[Asserts]
header "Content-Type" contains "application/json"
body contains "\"id\":\"category:software_jup/v1.0\""
body contains "\"name\":\"Jupyter Notebooks\""
body contains "\"href\":\"v1/concept/category:facets/software-theme/v1.0\""
body contains "\"name\":\"Software\""
