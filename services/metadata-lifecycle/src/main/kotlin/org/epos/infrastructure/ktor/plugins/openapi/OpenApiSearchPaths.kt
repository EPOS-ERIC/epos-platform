package org.epos.infrastructure.ktor.plugins

object OpenApiSearchPaths {
    val yaml = """
  # ============================================================
  # SEARCH ENDPOINT
  # ============================================================
  /api/v2/search:
    get:
      tags: [Search]
      summary: Search metadata resources
      operationId: searchMetadata
      description: |
        Returns a flat, typed list of catalogue resources matching the supplied criteria.
        Results may include datasets, distributions, software applications, software source
        codes, and categories. Each item carries an `href` pointing to its typed detail
        endpoint; clients should follow that link rather than construct URLs themselves.

        **Spatial Filtering:**
        - `spatialFilter`: WKT polygon string (EPSG:4326)

        **Temporal Filtering:**
        - `dateFrom`, `dateTo`: ISO 8601 date-time range

        **Multi-value Parameters:**
        Parameters like `type`, `keywords`, `serviceProvider`, and `versioningStatus` accept
        multiple values by repeating the parameter: `?type=dataset&type=distribution`

        **SHACL-driven property filters:**
        Any non-reserved query parameter is interpreted as a property filter when it matches
        a NodeShape property local name or full IRI for the selected type(s).
        You can also use `filter.<propertyName>=value`.
      parameters:
        - ${'$'}ref: '#/components/parameters/Q'
        - ${'$'}ref: '#/components/parameters/Page'
        - ${'$'}ref: '#/components/parameters/ResultsPerPage'
        - ${'$'}ref: '#/components/parameters/TypeFilter'
        - ${'$'}ref: '#/components/parameters/SpatialFilter'
        - ${'$'}ref: '#/components/parameters/DateFrom'
        - ${'$'}ref: '#/components/parameters/DateTo'
        - ${'$'}ref: '#/components/parameters/ServiceProviders'
        - ${'$'}ref: '#/components/parameters/VersioningStatusFilter'
        - ${'$'}ref: '#/components/parameters/Keywords'
      responses:
        '200':
          description: Search results returned successfully
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/SearchResponse'
              examples:
                mixedResults:
                  summary: Mixed resource search results
                  value:
                    total: 42
                    items:
                      - type: dataset
                        id: "123"
                        name: "IRPINIA Seismic Stations"
                        href: "https://metadata.epos-eu.org/api/v2/dataset/123"
                        serviceProviders: ["UNINA"]
                        categories:
                          - "Seismology"
                          - "Strong Motion"
                      - type: category
                        id: "cat-seismology"
                        name: "Seismology"
                        href: "https://metadata.epos-eu.org/api/v2/category/cat-seismology"
        '400':
          ${'$'}ref: '#/components/responses/BadRequest'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'"""
}
