package org.epos.infrastructure.ktor.plugins

object OpenApiSearchPaths {
    val yaml = """
  # ============================================================
  # SEARCH AND DISCOVERY ENDPOINTS
  # ============================================================
  /api/v2/search:
    get:
      tags: [Search]
      summary: Search across all entities
      description: |
        Search across all entity types with filtering, pagination, and hierarchical facets.

        **Full GeoSPARQL Spatial Filtering Support:**
        - `bbox`: Simple bounding box filter
        - `geo` + `geoRelation`: Custom WKT geometry with spatial relation
        - `lat`, `lon`, `radius`: Point + radius filter (km)

        **Spatial Relations (geoRelation parameter):**
        - `intersects` (default): Entity intersects the query geometry
        - `within`: Entity is completely within the query geometry
        - `contains`: Entity completely contains the query geometry
        - `overlaps`: Entity overlaps the query geometry
        - `touches`: Entity touches the query geometry
        - `disjoint`: Entity does not touch the query geometry

        **Hierarchical Concept/Theme Filtering:**
        - `concept`: Filter by specific concept URI (dcat:theme)
        - `conceptScheme`: Filter by concept scheme (returns entities themed with any concept in that scheme)

        **Data Provider Filtering:**
        - `dataProvider`: Filter by data provider organization URI
        - `organization`: Filter by organization name (text search)

        **Hierarchical Facets (when facets=true):**
        - `conceptSchemes`: ConceptSchemes with child Concepts and entity counts
        - `dataProviders`: Organizations with resource type breakdown
      parameters:
        - name: q
          in: query
          description: Free text search query
          schema:
            type: string
          example: seismic data
        - name: type
          in: query
          description: Filter by resource type
          schema:
            type: string
          example: dataset
        - name: status
          in: query
          description: Lifecycle status filter
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
        - name: bbox
          in: query
          description: Bounding box filter (minLon,minLat,maxLon,maxLat). Uses GeoSPARQL for spatial filtering.
          schema:
            type: string
          example: "-10,35,30,70"
        - name: geo
          in: query
          description: WKT geometry for spatial filtering (e.g., POLYGON, POINT, LINESTRING)
          schema:
            type: string
          example: "POLYGON((10 40, 40 40, 40 10, 10 10, 10 40))"
        - name: geoRelation
          in: query
          description: Spatial relation to use with bbox or geo parameter
          schema:
            type: string
            enum: [within, intersects, contains, overlaps, touches, disjoint, crosses, equals]
            default: intersects
        - name: lat
          in: query
          description: Latitude for point + radius filter
          schema:
            type: number
            format: double
          example: 45.5
        - name: lon
          in: query
          description: Longitude for point + radius filter
          schema:
            type: number
            format: double
          example: 9.2
        - name: radius
          in: query
          description: Radius in kilometers for point + radius filter
          schema:
            type: number
            format: double
          example: 100
        - name: startDate
          in: query
          description: Temporal filter start date (ISO 8601)
          schema:
            type: string
            format: date-time
        - name: endDate
          in: query
          description: Temporal filter end date (ISO 8601)
          schema:
            type: string
            format: date-time
        - name: keywords
          in: query
          description: Comma-separated keywords filter
          schema:
            type: string
          example: "earthquake,seismology"
        - name: organization
          in: query
          description: Filter by organization URI or name (text search)
          schema:
            type: string
        - name: concept
          in: query
          description: Filter by concept URI (dcat:theme). Returns entities tagged with this specific concept.
          schema:
            type: string
          example: "https://epos-eu.org/concept/seismology"
        - name: conceptScheme
          in: query
          description: Filter by concept scheme URI. Returns entities tagged with any concept in that scheme.
          schema:
            type: string
          example: "https://epos-eu.org/conceptscheme/domains"
        - name: dataProvider
          in: query
          description: Filter by data provider organization URI (exact match)
          schema:
            type: string
          example: "https://epos-eu.org/organization/ingv"
        - name: page
          in: query
          schema:
            type: integer
            default: 1
            minimum: 1
        - name: pageSize
          in: query
          schema:
            type: integer
            default: 20
            minimum: 1
            maximum: 100
        - name: facets
          in: query
          description: Include facet counts in response
          schema:
            type: boolean
            default: false
        - name: facetstype
          in: query
          description: |
            How to group results when facets=true. Matches resources-service behavior.
            - categories: Group results hierarchically by ConceptScheme -> Concept
            - dataproviders: Group results by data provider organization
            - serviceproviders: Group results by service provider organization
            - none: Return flat list with facet counts only (default)
          schema:
            type: string
            enum: [none, categories, dataproviders, serviceproviders]
            default: none
      responses:
        '200':
          description: Search results
          content:
            application/json:
              schema:
                type: object
                properties:
                  results:
                    type: array
                    items:
                      type: object
                      properties:
                        id:
                          type: string
                        type:
                          type: string
                        title:
                          type: string
                        description:
                          type: string
                        href:
                          type: string
                        availableFormats:
                          type: array
                          items:
                            type: object
                            properties:
                              label:
                                type: string
                              format:
                                type: string
                              href:
                                type: string
                              type:
                                type: string
                  total:
                    type: integer
                  page:
                    type: integer
                  pageSize:
                    type: integer
                  totalPages:
                    type: integer
                  facets:
                    type: object
                    description: Hierarchical facets (only present when facets=true)
                    properties:
                      types:
                        type: array
                        description: Resource type facets with counts
                        items:
                          type: object
                          properties:
                            value:
                              type: string
                            label:
                              type: string
                            count:
                              type: integer
                      keywords:
                        type: array
                        description: Keyword facets with counts
                        items:
                          type: object
                          properties:
                            value:
                              type: string
                            label:
                              type: string
                            count:
                              type: integer
                      conceptSchemes:
                        type: array
                        description: Hierarchical concept scheme facets with child concepts
                        items:
                          type: object
                          properties:
                            id:
                              type: string
                              description: ConceptScheme URI
                            label:
                              type: string
                            count:
                              type: integer
                              description: Total entities tagged with concepts in this scheme
                            children:
                              type: array
                              description: Child concepts with their entity counts
                              items:
                                type: object
                                properties:
                                  value:
                                    type: string
                                    description: Concept URI
                                  label:
                                    type: string
                                  count:
                                    type: integer
                      dataProviders:
                        type: array
                        description: Data provider organization facets with resource type breakdown
                        items:
                          type: object
                          properties:
                            id:
                              type: string
                              description: Organization URI
                            label:
                              type: string
                              description: Organization name
                            count:
                              type: integer
                              description: Total resources from this provider
                            children:
                              type: array
                              description: Resource types with counts
                              items:
                                type: object
                                properties:
                                  value:
                                    type: string
                                    description: Resource type URI
                                  label:
                                    type: string
                                  count:
                                    type: integer

  /api/v2/details/{id}:
    parameters:
      - name: id
        in: path
        required: true
        description: Entity URI (URL encoded)
        schema:
          type: string
    get:
      tags: [Search]
      summary: Get entity details
      description: |
        Retrieve detailed information about a specific entity.

        **Standard Response (extended=false):**
        Returns EDM-mapped properties with relations as hrefs.

        **Extended/Aggregated Response (extended=true):**
        Returns fully nested response matching resources-service structure:
        - Nested relatedDataProducts with full DataProduct info
        - Nested relatedWebservices with Operations and ServiceParameters
        - Spatial and temporal extents as structured objects
        - Contact points with person info
        - Available formats for download/access
        - Categories as facet tree
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
        - name: format
          in: query
          description: Response format
          schema:
            type: string
            enum: [json, turtle, jsonld, rdfxml]
            default: json
        - name: extended
          in: query
          description: |
            If true, returns aggregated response with nested related entities
            (DataProducts, WebServices, Operations) matching resources-service structure.
          schema:
            type: boolean
            default: false
      responses:
        '200':
          description: Entity details
          content:
            application/json:
              schema:
                oneOf:
                  - description: Standard EDM response (extended=false)
                    type: object
                    properties:
                      id:
                        type: string
                      type:
                        type: string
                      title:
                        type: string
                      description:
                        type: string
                      relations:
                        type: object
                  - description: Aggregated response (extended=true)
                    type: object
                    properties:
                      id:
                        type: string
                      type:
                        type: string
                      title:
                        type: string
                      description:
                        type: string
                      href:
                        type: string
                      hrefExtended:
                        type: string
                      spatial:
                        type: object
                      temporalCoverage:
                        type: object
                      dataProvider:
                        type: array
                      relatedDataProducts:
                        type: array
                        description: Nested DataProduct objects
                      relatedWebservices:
                        type: array
                        description: Nested WebService objects with operations
        '404':
          description: Entity not found"""
}
