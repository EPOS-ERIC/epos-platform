package org.epos.infrastructure.ktor.plugins

object OpenApiAccessPaths {
    val yaml =
        """
  /api/v2/access/search:
    get:
      tags: [Access]
      summary: Search resources
      description: Returns a flat, typed list of catalogue resources matching the supplied criteria.
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
        '400':
          ${'$'}ref: '#/components/responses/BadRequest'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  /api/v2/access/search/{shape}:
    get:
      tags: [Access]
      summary: Search resources by shape
      description: Access search scoped to a single shape (equivalent to applying `type={shape}`).
      parameters:
        - name: shape
          in: path
          required: true
          schema:
            type: string
          description: Runtime SHACL shape/resource type name.
        - ${'$'}ref: '#/components/parameters/Q'
        - ${'$'}ref: '#/components/parameters/Page'
        - ${'$'}ref: '#/components/parameters/ResultsPerPage'
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
        '400':
          ${'$'}ref: '#/components/responses/BadRequest'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  # ============================================================
  # ACCESS ENDPOINT
  # ============================================================
  /api/v2/access/{shape}/{id}:
    get:
      tags: [Access]
      summary: Access shape resources and details
      description: |
        Unified access endpoint.
        - `id=all` returns all resources for the selected shape.
        - any other id returns the detailed property-value and property-relation structure.
      parameters:
        - name: shape
          in: path
          required: true
          schema:
            type: string
          description: Runtime SHACL shape/resource type name.
        - name: id
          in: path
          required: true
          schema:
            type: string
          description: Resource id or `all`.
      responses:
        '200':
          description: Access response returned successfully
        '400':
          ${'$'}ref: '#/components/responses/BadRequest'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'
"""
}
