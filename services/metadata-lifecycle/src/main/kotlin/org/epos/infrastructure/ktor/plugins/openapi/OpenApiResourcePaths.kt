package org.epos.infrastructure.ktor.plugins

object OpenApiResourcePaths {
    val yaml = """
  # ============================================================
  # RESOURCE CRUD ENDPOINTS
  # ============================================================
  /api/v2/{resourceType}:
    parameters:
      - name: resourceType
        in: path
        required: true
        schema:
          type: string
        example: dataset
    get:
      tags: [Resources]
      summary: List entities
      description: Retrieve a paginated list of entities of a specific type
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
        - name: page
          in: query
          schema:
            type: integer
            default: 1
        - name: pageSize
          in: query
          schema:
            type: integer
            default: 20
      responses:
        '200':
          description: List of entities
    post:
      tags: [Resources]
      summary: Create new entity (DRAFT)
      description: Create a new entity in DRAFT state
      parameters:
        - name: X-Agent
          in: header
          schema:
            type: string
          example: mailto:user@example.org
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                type:
                  type: string
                properties:
                  type: object
      responses:
        '201':
          description: Entity created

  /api/v2/{resourceType}/{id}:
    parameters:
      - name: resourceType
        in: path
        required: true
        schema:
          type: string
      - name: id
        in: path
        required: true
        schema:
          type: string
        description: Full entity URI
    get:
      tags: [Resources]
      summary: Get entity
      description: Retrieve entity details. Supports multiple formats via Accept header or format parameter.
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
        - name: format
          in: query
          schema:
            type: string
            enum: [json, turtle, ttl, jsonld, rdfxml, ntriples]
      responses:
        '200':
          description: Entity details
        '404':
          description: Entity not found
    put:
      tags: [Resources]
      summary: Update entity (DRAFT only)
      parameters:
        - name: X-Agent
          in: header
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                properties:
                  type: object
      responses:
        '200':
          description: Entity updated
    delete:
      tags: [Resources]
      summary: Delete entity (DRAFT only)
      responses:
        '200':
          description: Entity deleted"""
}
