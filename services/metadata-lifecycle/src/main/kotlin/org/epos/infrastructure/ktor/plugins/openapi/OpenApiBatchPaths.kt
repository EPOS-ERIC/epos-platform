package org.epos.infrastructure.ktor.plugins

object OpenApiBatchPaths {
    val yaml = """
  # ============================================================
  # BATCH ENDPOINTS
  # ============================================================
  /api/v2/batch/submit:
    post:
      tags: [Batch]
      summary: Batch submit entities
      description: Submit multiple entities in a single request
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
                entityUris:
                  type: array
                  items:
                    type: string
      responses:
        '200':
          description: Batch operation completed

  /api/v2/batch/publish:
    post:
      tags: [Batch]
      summary: Batch publish entities
      description: Publish multiple entities in a single request
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
                entityUris:
                  type: array
                  items:
                    type: string
      responses:
        '200':
          description: Batch operation completed"""
}
