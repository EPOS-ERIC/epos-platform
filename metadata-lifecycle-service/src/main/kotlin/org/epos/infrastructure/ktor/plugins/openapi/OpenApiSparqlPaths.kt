package org.epos.infrastructure.ktor.plugins

object OpenApiSparqlPaths {
    val yaml = """
  # ============================================================
  # SPARQL ENDPOINT
  # ============================================================
  /api/v2/sparql:
    post:
      tags: [SPARQL]
      summary: Execute SPARQL query
      description: |
        Execute a SPARQL query against the triple store.
        Supports SELECT, ASK, CONSTRUCT, and DESCRIBE queries.
        UPDATE operations are not allowed.
      requestBody:
        required: true
        content:
          application/x-www-form-urlencoded:
            schema:
              type: object
              required: [query]
              properties:
                query:
                  type: string
                  description: SPARQL query string
          application/sparql-query:
            schema:
              type: string
          application/json:
            schema:
              type: object
              required: [query]
              properties:
                query:
                  type: string
      responses:
        '200':
          description: Query results
          content:
            application/sparql-results+json:
              schema:
                type: object
            application/sparql-results+xml:
              schema:
                type: string
            text/csv:
              schema:
                type: string
            application/n-triples:
              schema:
                type: string
            text/turtle:
              schema:
                type: string
        '400':
          description: Invalid query
        '403':
          description: UPDATE operations not allowed"""
}
