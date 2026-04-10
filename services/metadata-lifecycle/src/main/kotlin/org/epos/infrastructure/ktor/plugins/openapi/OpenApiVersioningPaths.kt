package org.epos.infrastructure.ktor.plugins

object OpenApiVersioningPaths {
    val yaml = """
  # ============================================================
  # VERSIONING ENDPOINTS
  # ============================================================
  /api/v2/{resourceType}/{id}/history:
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
    get:
      tags: [Versioning]
      summary: Version history
      description: Retrieve all versions of a conceptual entity
      responses:
        '200':
          description: Version history"""
}
