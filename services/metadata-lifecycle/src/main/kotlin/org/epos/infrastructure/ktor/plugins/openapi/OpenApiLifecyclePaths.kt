package org.epos.infrastructure.ktor.plugins

object OpenApiLifecyclePaths {
    val yaml = """
  # ============================================================
  # LIFECYCLE ENDPOINTS
  # ============================================================
  /api/v2/{resourceType}/{id}/submit:
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
    post:
      tags: [Lifecycle]
      summary: Submit for review
      description: Move an entity from DRAFT to SUBMITTED
      parameters:
        - name: X-Agent
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Entity submitted

  /api/v2/{resourceType}/{id}/publish:
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
    post:
      tags: [Lifecycle]
      summary: Publish entity
      description: Publish an entity from SUBMITTED to PUBLISHED
      parameters:
        - name: X-Agent
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Entity published

  /api/v2/{resourceType}/{id}/draft:
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
    post:
      tags: [Lifecycle]
      summary: Create new draft from published
      description: Creates a new draft version from a published entity
      parameters:
        - name: version
          in: query
          schema:
            type: string
            default: "2.0"
        - name: X-Agent
          in: header
          schema:
            type: string
      responses:
        '201':
          description: Draft created"""
}
