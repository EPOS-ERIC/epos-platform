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
                    description: Returned when shape definitions are not requested
                    items:
                      type: object
                      properties:
                        name:
                          type: string
                          example: dataset
                        targetClass:
                          type: string
                          format: uri
                          example: http://www.w3.org/ns/dcat#Dataset
                        properties:
                          type: array
                          items:
                            type: object
                            properties:
                              path:
                                type: string
                                format: uri
                              name:
                                type: string
                              datatype:
                                type: string
                                format: uri
                                nullable: true
                              datatypes:
                                type: array
                                nullable: true
                                items:
                                  type: string
                                  format: uri
                              nodeKind:
                                type: string
                                nullable: true
                              nodeKinds:
                                type: array
                                nullable: true
                                items:
                                  type: string
                              classConstraint:
                                type: string
                                format: uri
                                nullable: true
                              classConstraints:
                                type: array
                                nullable: true
                                items:
                                  type: string
                                  format: uri
                              nodeShapeRef:
                                type: string
                                nullable: true
                              nodeShapeRefs:
                                type: array
                                nullable: true
                                items:
                                  type: string
                              minCount:
                                type: integer
                                format: int32
                                nullable: true
                              maxCount:
                                type: integer
                                format: int32
                                nullable: true
                              required:
                                type: boolean
                              recommended:
                                type: boolean
                              pattern:
                                type: string
                                nullable: true
                              minLength:
                                type: integer
                                format: int32
                                nullable: true
                              maxLength:
                                type: integer
                                format: int32
                                nullable: true
                              hasValueIn:
                                type: array
                                nullable: true
                                items:
                                  type: string
                              severity:
                                type: string
                                nullable: true
                              severities:
                                type: array
                                nullable: true
                                items:
                                  type: string
                              helpText:
                                type: string
                                nullable: true
                              description:
                                type: string
                                nullable: true
                        closed:
                          type: boolean
                        description:
                          type: string
                          nullable: true
                      required: [name, targetClass, properties, closed]
                  - type: object
                    description: Returned when includeShapeDefinitions or includeTransitiveShapeDefinitions is true
                    properties:
                      resourceTypes:
                        type: array
                        items:
                          type: object
                          properties:
                            name:
                              type: string
                            targetClass:
                              type: string
                              format: uri
                            properties:
                              type: array
                              items:
                                type: object
                            closed:
                              type: boolean
                            description:
                              type: string
                              nullable: true
                          required: [name, targetClass, properties, closed]
                      shapeDefinitions:
                        type: object
                        description: Map keyed by shape IRI
                        additionalProperties:
                          type: object
                          properties:
                            id:
                              type: string
                              format: uri
                            targetClasses:
                              type: array
                              items:
                                type: string
                                format: uri
                            properties:
                              type: array
                              items:
                                type: object
                            resolvedRefs:
                              type: array
                              nullable: true
                              items:
                                type: string
                            name:
                              type: string
                              nullable: true
                            datatype:
                              type: string
                              format: uri
                              nullable: true
                            datatypes:
                              type: array
                              nullable: true
                              items:
                                type: string
                                format: uri
                            nodeKind:
                              type: string
                              nullable: true
                            nodeKinds:
                              type: array
                              nullable: true
                              items:
                                type: string
                            classConstraint:
                              type: string
                              format: uri
                              nullable: true
                            classConstraints:
                              type: array
                              nullable: true
                              items:
                                type: string
                                format: uri
                            nodeShapeRef:
                              type: string
                              nullable: true
                            nodeShapeRefs:
                              type: array
                              nullable: true
                              items:
                                type: string
                            ontologyEquivalentClasses:
                              type: array
                              nullable: true
                              items:
                                type: string
                                format: uri
                            ontologySubClassesOf:
                              type: array
                              nullable: true
                              items:
                                type: string
                                format: uri
                            closed:
                              type: boolean
                            description:
                              type: string
                              nullable: true
                          required: [id, targetClasses, properties, closed]"""
}
