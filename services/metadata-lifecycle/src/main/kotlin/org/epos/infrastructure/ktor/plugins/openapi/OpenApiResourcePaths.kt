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
      description: Retrieve a paginated list of entities of a specific type. For status=draft, X-Agent can scope results to drafts created by that agent.
      parameters:
        - name: X-Agent
          in: header
          required: false
          schema:
            type: string
          description: Optional agent URI used to filter draft results for this actor.
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived, discarded]
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
          required: true
          schema:
            type: string
          example: mailto:user@example.org
        - name: X-Group
          in: header
          required: true
          schema:
            type: string
          example: https://epos-eu.org/group/ingestion-team
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                id:
                  type: string
                  description: Optional client-provided conceptual id (absolute URI or local id segment). If omitted, server generates one.
                type:
                  type: string
                  description: Optional; ignored because resource type comes from path parameter.
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
        description: Resource identifier (conceptual id, version id, or identifier)
    get:
      tags: [Resources]
      summary: Get entity
      description: Retrieve entity details. Supports multiple formats via Accept header or format parameter.
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived, discarded]
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
      description: Full replace semantics. Properties not included in the request are removed from the draft entity. Request fails with 409 if the resolved resource is not in DRAFT status.
      parameters:
        - name: X-Agent
          in: header
          required: true
          schema:
            type: string
        - name: X-Group
          in: header
          required: true
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
        '404':
          description: Entity not found
        '409':
          description: Resolved resource is not in DRAFT status
    patch:
      tags: [Resources]
      summary: Patch entity (DRAFT only)
      description: Merge semantics. Only the provided properties are replaced; all other properties are preserved. Request fails with 409 if the resolved resource is not in DRAFT status.
      parameters:
        - name: X-Agent
          in: header
          required: true
          schema:
            type: string
        - name: X-Group
          in: header
          required: true
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
          description: Entity patched
        '404':
          description: Entity not found
        '409':
          description: Resolved resource is not in DRAFT status
    delete:
      tags: [Resources]
      summary: Delete entity (DRAFT only)
      description: Deletes a DRAFT version. Request fails with 409 if the resolved resource is not in DRAFT status.
      parameters:
        - name: X-Agent
          in: header
          required: true
          schema:
            type: string
        - name: X-Group
          in: header
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Entity deleted
        '404':
          description: Entity not found
        '409':
          description: Resolved resource is not in DRAFT status"""
}
