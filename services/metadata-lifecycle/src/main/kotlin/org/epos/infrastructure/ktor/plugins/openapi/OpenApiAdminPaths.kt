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
          schema:
            type: string
      responses:
        '200':
          description: Entity repaired

  /api/v2/admin/populate:
    post:
      tags: [Admin]
      summary: Populate database from TTL files
      description: Triggers initial population from the configured INDEX_URL
      parameters:
        - name: X-Agent
          in: header
          schema:
            type: string
      responses:
        '200':
          description: Population completed
        '500':
          description: Population failed

  /api/v2/admin/populate/from-url:
    post:
      tags: [Admin]
      summary: Populate from custom URL
      description: Triggers population from a custom index URL
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
              required: [indexUrl]
              properties:
                indexUrl:
                  type: string
                  example: http://example.com/index.txt
      responses:
        '200':
          description: Population completed
        '400':
          description: Missing indexUrl
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

  /api/v2/admin/populate/file:
    post:
      tags: [Admin]
      summary: Populate from URL or local TTL file
      description: |
        Populates the database from a single TTL source.
        You can provide either:
        - `url`: an HTTP(S) URL to a TTL file
        - `path`: a local filesystem path to a TTL file available on the server
        Useful for loading individual TTL files from GitHub, HTTP endpoints, or mounted local volumes.

        **Private Repository Support:**
        Use the `X-Source-Authorization` header to access private GitHub repositories.
        The header value is forwarded as the `Authorization` header when fetching the file.

        Example URLs:
        - GitHub raw (public): https://raw.githubusercontent.com/org/repo/main/data.ttl
        - GitHub raw (private): Use X-Source-Authorization header with "token ghp_xxx"
        - Direct URLs: https://example.com/metadata.ttl
      parameters:
        - name: X-Agent
          in: header
          description: Agent URI for provenance tracking
          schema:
            type: string
        - name: X-Source-Authorization
          in: header
          description: |
            Authorization header to forward when fetching the URL.
            For GitHub private repos, use "token ghp_YOUR_PERSONAL_ACCESS_TOKEN"
          schema:
            type: string
          example: "token ghp_xxxxxxxxxxxx"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                url:
                  type: string
                  description: URL to a TTL file
                  example: https://raw.githubusercontent.com/EPOS-ERIC/metadatone/main/metadatone.ttl
                path:
                  type: string
                  description: Local filesystem path to a TTL file
                  example: /data/metadata/metadatone.ttl
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
                      sourceUrl:
                        type: string
                      subjectsProcessed:
                        type: integer
                      triplesInserted:
                        type: integer
                      agent:
                        type: string
        '400':
          description: Invalid request (missing/invalid source or both url and path provided)
        '500':
          description: Population failed

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
                  example: "@prefix dcat: <http://www.w3.org/ns/dcat#> .\\n<http://example.org/dataset1> a dcat:Dataset ."
                sourceName:
                  type: string
                  description: Optional name for provenance tracking
                  example: my-upload
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
          description: Population failed"""
}
