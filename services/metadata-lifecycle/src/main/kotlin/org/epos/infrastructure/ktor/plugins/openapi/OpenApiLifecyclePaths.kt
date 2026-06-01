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
      description: Move an entity from DRAFT to SUBMITTED. Path id may be a conceptual id, version id, or identifier, but it must resolve to a DRAFT version.
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
          description: Entity submitted
        '404':
          description: Entity not found
        '409':
          description: Resource resolves to non-DRAFT status

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
      description: Publish an entity from SUBMITTED to PUBLISHED. Path id may be a conceptual id, version id, or identifier, but it must resolve to a SUBMITTED version.
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
          description: Entity published
        '404':
          description: Entity not found
        '409':
          description: Resource resolves to non-SUBMITTED status

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
      description: Creates a new draft version from a published entity. Path id may be conceptual id, version id, or identifier and is resolved to conceptual resource. The new version number is auto-incremented from the current published version.
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
        '201':
          description: Draft created
        '404':
          description: Resource not found

  /api/v2/{resourceType}/{id}/discard:
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
      summary: Discard draft/submitted entity
      description: Move an entity from DRAFT or SUBMITTED to DISCARDED. Path id may be a conceptual id, version id, or identifier.
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
          description: Entity discarded
        '404':
          description: Entity not found
        '409':
          description: Resource resolves to non-discardable status"""
}
