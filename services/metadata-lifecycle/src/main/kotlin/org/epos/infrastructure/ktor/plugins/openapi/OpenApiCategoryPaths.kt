package org.epos.infrastructure.ktor.plugins

object OpenApiCategoryPaths {
    val yaml = """
  # ============================================================
  # CATEGORY ENDPOINTS
  # ============================================================
  /api/v2/category:
    get:
      tags: [Categories]
      summary: Get full category tree
      operationId: getCategoryTree
      description: |
        Returns the complete category tree as an ordered list of category schemes.
        Each scheme carries its full subtree of nested `Category` objects.
        Intended as the primary data source for building facet navigation in clients.
      responses:
        '200':
          description: Category tree returned successfully
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/CategoryTreeResponse'
              examples:
                tree:
                  summary: Category tree with one scheme
                  value:
                    schemes:
                      - id: "scheme-solid-earth"
                        label: "Solid Earth"
                        title: "Solid Earth Science Categories"
                        description: "Top-level grouping for solid earth science resources."
                        logo: "https://www.epos-eu.org/logos/solid-earth.png"
                        homepage: "https://www.epos-eu.org/solid-earth"
                        color: "#3A7BD5"
                        orderNumber: 1
                        type: "DATA"
                        children:
                          - id: "cat-seismology"
                            name: "Seismology"
                            description: "Resources related to seismic monitoring."
                            children:
                              - id: "cat-strong-motion"
                                name: "Strong Motion"
                                children: []
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'

  /api/v2/category/{id}:
    parameters:
      - ${'$'}ref: '#/components/parameters/ResourceId'
    get:
      tags: [Categories]
      summary: Get category by id
      operationId: getCategoryById
      description: Returns a single category node with its full subtree of children.
      responses:
        '200':
          description: Category found
          content:
            application/json:
              schema:
                ${'$'}ref: '#/components/schemas/Category'
        '404':
          ${'$'}ref: '#/components/responses/NotFound'
        '500':
          ${'$'}ref: '#/components/responses/InternalServerError'"""
}
