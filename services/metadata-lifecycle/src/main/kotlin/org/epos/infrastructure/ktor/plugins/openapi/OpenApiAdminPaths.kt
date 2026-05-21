package org.epos.infrastructure.ktor.plugins

object OpenApiAdminPaths {
    val yaml = """
  # ============================================================
  # ADMIN ENDPOINTS
  # ============================================================
  /api/v2/{resourceType}/{id}/repair:
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
      tags: [Admin]
      summary: Repair entity metadata
      description: Repair missing conceptual graph entries or provenance records
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
          description: Entity repaired

  /api/v2/admin/populate:
    post:
      tags: [Admin]
      summary: Populate database
      description: |
        Population entrypoint.
        - No body: populates from configured INDEX_URL.
        - Body with `url` or `path`: populates from a single TTL source.
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
        - name: X-Source-Authorization
          in: header
          required: false
          schema:
            type: string
          description: Optional auth header forwarded when fetching `url`.
      requestBody:
        required: false
        content:
          application/json:
            schema:
              type: object
              properties:
                url:
                  type: string
                path:
                  type: string
      responses:
        '200':
          description: Population completed
        '500':
          description: Population failed

  /api/v2/admin/populate/status:
    get:
      tags: [Admin]
      summary: Get population configuration
      description: Returns current population configuration
      responses:
        '200':
          description: Population configuration
          content:
            application/json:
              schema:
                type: object
                properties:
                  indexUrl:
                    type: string
                  graphdbEndpoint:
                    type: string
                  repositoryId:
                    type: string
                  graphs:
                    type: object

  /api/v2/admin/populate/content:
    post:
      tags: [Admin]
      summary: Populate from TTL content
      description: |
        Populates the database from TTL content uploaded directly in the request body.
        Useful for uploading small TTL snippets or programmatic population.
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
              required: [content]
              properties:
                content:
                  type: string
                  description: TTL content as a string
                  examples:
                    - "@prefix dcat: <http://www.w3.org/ns/dcat#> .\\n<http://example.org/dataset1> a dcat:Dataset ."
                sourceName:
                  type: string
                  description: Optional name for provenance tracking
                  examples:
                    - my-upload
      responses:
        '200':
          description: Population completed
          content:
            application/json:
              schema:
                type: object
                properties:
                  success:
                    type: boolean
                  message:
                    type: string
                  data:
                    type: object
                    properties:
                      sourceName:
                        type: string
                      subjectsProcessed:
                        type: integer
                      triplesInserted:
                        type: integer
                      agent:
                        type: string
        '400':
          description: Missing content
        '500':
          description: Population failed

  /api/v2/admin/import/revision/file:
    post:
      tags: [Admin]
      summary: Import metadata revisions from URL or local TTL file
      description: |
        Imports TTL metadata into DRAFT revisions for existing published entities.
        Existing links between imported resources are rewritten to their corresponding draft URIs.
        This endpoint does not write directly to PUBLISHED.
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
              ${'$'}ref: '#/components/schemas/RevisionImportFileRequest'
      responses:
        '200':
          description: Revision import completed
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/RevisionImportApiResponse'
        '400':
          description: Invalid request
        '500':
          description: Revision import failed

  /api/v2/admin/import/revision/content:
    post:
      tags: [Admin]
      summary: Import metadata revisions from TTL content
      description: |
        Imports TTL metadata into DRAFT revisions for existing published entities.
        Existing links between imported resources are rewritten to their corresponding draft URIs.
        This endpoint does not write directly to PUBLISHED.
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
              ${'$'}ref: '#/components/schemas/RevisionImportContentRequest'
      responses:
        '200':
          description: Revision import completed
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/RevisionImportApiResponse'
        '400':
          description: Invalid request
        '500':
          description: Revision import failed"""
}
