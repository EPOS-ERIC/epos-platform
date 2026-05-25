package org.epos.infrastructure.ktor.plugins

object OpenApiDetailsPaths {
    val yaml = """
  # ============================================================
  # TYPED DETAIL ENDPOINTS
  # ============================================================
  /api/v2/dataset/{id}:
    parameters:
      - ${'$'}ref: '#/components/parameters/ResourceId'
    get:
      tags: [Datasets]
      summary: Get dataset by id
      operationId: getDatasetById
      description: |
        Returns detailed information about a specific dataset.
        Includes distributions as resource references with hrefs.
        Includes an immediate relation graph grouped by predicate and direction.
        Inbound relations (entities referencing this dataset) are always included when present.
      responses:
        '200':
          description: Dataset found
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/Dataset'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  /api/v2/distribution/{id}:
    parameters:
      - ${'$'}ref: '#/components/parameters/ResourceId'
    get:
      tags: [Distributions]
      summary: Get distribution by id
      operationId: getDistributionById
      description: |
        Returns detailed information about a specific distribution.
        Includes an immediate relation graph grouped by predicate and direction.
        Inbound relations (entities referencing this distribution) are always included when present.
      responses:
        '200':
          description: Distribution found
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/Distribution'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  /api/v2/softwareapplication/{id}:
    parameters:
      - ${'$'}ref: '#/components/parameters/ResourceId'
    get:
      tags: [Software Applications]
      summary: Get software application by id
      operationId: getSoftwareApplicationById
      description: |
        Returns detailed information about a specific software application.
        Includes an immediate relation graph grouped by predicate and direction.
        Inbound relations (entities referencing this application) are always included when present.
      responses:
        '200':
          description: Software application found
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/SoftwareApplication'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  /api/v2/softwaresourcecode/{id}:
    parameters:
      - ${'$'}ref: '#/components/parameters/ResourceId'
    get:
      tags: [Software Source Codes]
      summary: Get software source code by id
      operationId: getSoftwareSourceCodeById
      description: |
        Returns detailed information about a specific software source code.
        Includes an immediate relation graph grouped by predicate and direction.
        Inbound relations (entities referencing this source code) are always included when present.
      responses:
        '200':
          description: Software source code found
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/SoftwareSourceCode'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'"""
}
