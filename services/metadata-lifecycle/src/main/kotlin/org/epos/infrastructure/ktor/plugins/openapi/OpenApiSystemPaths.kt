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
      description: |
        Retrieve available resource types parsed from SHACL shapes (default view),
        or export the original cached SHACL graph in RDF formats.
      parameters:
        - name: view
          in: query
          description: Response view. `parsed` returns JSON resource types; `shacl` returns raw SHACL RDF.
          schema:
            type: string
            enum: [parsed, shacl]
            default: parsed
        - name: format
          in: query
          description: RDF serialization format used when `view=shacl`
          schema:
            type: string
            enum: [turtle, ttl, jsonld, json-ld, rdfxml, rdf+xml, ntriples, n-triples, nt]
        - name: compact
          in: query
          description: When `true` and `view=shacl`, returns compact/beautified output for JSON-LD and Turtle
          schema:
            type: boolean
            default: false
        - name: includeShapeDefinitions
          in: query
          description: Include resolvable shape definitions map keyed by shape IRI (only for `view=parsed`)
          schema:
            type: boolean
            default: false
        - name: includeTransitiveShapeDefinitions
          in: query
          description: Include only shape definitions reachable from resource type properties via sh:node references (only for `view=parsed`)
          schema:
            type: boolean
            default: false
        - name: includeResolvedRefs
          in: query
          description: Add cycle-safe recursive reference metadata per shape definition (only for `view=parsed`)
          schema:
            type: boolean
            default: false
      responses:
        '200':
          description: Parsed resource types or raw SHACL RDF, depending on `view`
          content:
            application/json:
              schema:
                oneOf:
                  - ${'$'}ref: '#/components/schemas/ResourceTypeList'
                  - ${'$'}ref: '#/components/schemas/ResourceTypesEnvelope'
            text/turtle:
              schema:
                type: string
            application/ld+json:
              schema:
                type: string
            application/rdf+xml:
              schema:
                type: string
            application/n-triples:
              schema:
                type: string"""
}
