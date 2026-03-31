package org.epos.infrastructure.ktor.plugins

object OpenApiSystemPaths {
    val yaml = """
  # ============================================================
  # SYSTEM ENDPOINTS
  # ============================================================
  /health:
    get:
      tags: [System]
      summary: Health check
      description: Check the service status
      responses:
        '200':
          description: Service operational
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: ok
                  timestamp:
                    type: integer
                    format: int64

  /api/v2/resource-types:
    get:
      tags: [System]
      summary: List resource types
      description: Retrieve available resource types parsed from SHACL shapes
      parameters:
        - name: includeShapeDefinitions
          in: query
          description: Include resolvable shape definitions map keyed by shape IRI
          schema:
            type: boolean
            default: false
        - name: includeTransitiveShapeDefinitions
          in: query
          description: Include only shape definitions reachable from resource type properties via sh:node references
          schema:
            type: boolean
            default: false
        - name: includeResolvedRefs
          in: query
          description: Add cycle-safe recursive reference metadata per shape definition
          schema:
            type: boolean
            default: false
      responses:
        '200':
          description: List of resource types
          content:
            application/json:
              schema:
                oneOf:
                  - type: array
                    items:
                      type: object
                  - type: object
                    properties:
                      resourceTypes:
                        type: array
                        items:
                          type: object
                      shapeDefinitions:
                        type: object
                        additionalProperties:
                          type: object"""
}
