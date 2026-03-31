package org.epos.infrastructure.ktor.plugins

object OpenApiLegacyPaths {
    val yaml = """
  # ============================================================
  # LEGACY COMPATIBILITY ENDPOINTS
  # ============================================================
  /resources/search:
    get:
      tags: [Legacy]
      summary: Legacy search endpoint
      description: Backward-compatible search endpoint from resources-service
      parameters:
        - name: q
          in: query
          schema:
            type: string
        - name: startDate
          in: query
          schema:
            type: string
            format: date-time
        - name: endDate
          in: query
          schema:
            type: string
            format: date-time
        - name: bbox
          in: query
          schema:
            type: string
          example: "-10,35,30,70"
        - name: keywords
          in: query
          schema:
            type: string
        - name: sciencedomains
          in: query
          schema:
            type: string
        - name: servicetypes
          in: query
          schema:
            type: string
        - name: organisations
          in: query
          schema:
            type: string
        - name: facetstype
          in: query
          schema:
            type: string
            enum: [none, categories, dataproviders, serviceproviders, facilityproviders]
            default: none
        - name: facets
          in: query
          schema:
            type: boolean
            default: false
        - name: versioningStatus
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
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
      responses:
        '200':
          description: Search response
          content:
            application/json:
              schema:
                type: object

  /resources/details/{instance_id}:
    get:
      tags: [Legacy]
      summary: Legacy details endpoint
      parameters:
        - name: instance_id
          in: path
          required: true
          schema:
            type: string
        - name: extended
          in: query
          schema:
            type: boolean
            default: false
        - name: versioningStatus
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
      responses:
        '200':
          description: Entity details response

  /software/search:
    get:
      tags: [Legacy]
      summary: Legacy software search
      parameters:
        - name: q
          in: query
          schema:
            type: string
        - name: versioningStatus
          in: query
          schema:
            type: string
            enum: [draft, submitted, published, archived]
            default: published
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
      responses:
        '200':
          description: Search response

  /software/details/{instance_id}:
    get:
      tags: [Legacy]
      summary: Legacy software details
      parameters:
        - name: instance_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Software details response

  /facilities/search:
    get:
      tags: [Legacy]
      summary: Legacy facilities search
      parameters:
        - name: q
          in: query
          schema:
            type: string
        - name: bbox
          in: query
          schema:
            type: string
        - name: keywords
          in: query
          schema:
            type: string
        - name: facilitytypes
          in: query
          schema:
            type: string
        - name: equipmenttypes
          in: query
          schema:
            type: string
        - name: organisations
          in: query
          schema:
            type: string
        - name: facetstype
          in: query
          schema:
            type: string
            enum: [none, categories, facilityproviders, dataproviders]
            default: none
        - name: facets
          in: query
          schema:
            type: boolean
            default: false
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
      responses:
        '200':
          description: Search response

  /facilities/details/{instance_id}:
    get:
      tags: [Legacy]
      summary: Legacy facility details
      parameters:
        - name: instance_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Facility details response

  /equipments/{instance_id}:
    get:
      tags: [Legacy]
      summary: Legacy equipment details
      parameters:
        - name: instance_id
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: Equipment details response

  /resources/organizations:
    get:
      tags: [Legacy]
      summary: Legacy organizations endpoint
      parameters:
        - name: q
          in: query
          schema:
            type: string
        - name: country
          in: query
          schema:
            type: string
        - name: type
          in: query
          description: Provider type filter
          schema:
            type: string
            enum: [dataproviders, serviceproviders, facilitiesproviders]
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
      responses:
        '200':
          description: OrganizationBean list response
          headers:
            X-Total-Count:
              schema:
                type: integer
            X-Page:
              schema:
                type: integer
            X-Page-Size:
              schema:
                type: integer
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    id:
                      type: string
                    logo:
                      type: string
                    url:
                      type: string
                    name:
                      type: string
                    country:
                      type: string

  /resources/monitoring:
    get:
      tags: [Monitoring]
      summary: Legacy monitoring endpoint
      description: Returns graph-level monitoring summary in legacy-compatible format
      responses:
        '200':
          description: MonitoringBean list response
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    identifier:
                      type: string
                    name:
                      type: string
                    originalURL:
                      type: string
                    validationRules:
                      type: array
                      items:
                        type: object
                        properties:
                          type:
                            type: string
                          encodingFormat:
                            type: string
                          schemaVersion:
                            type: string
                    contactPoints:
                      type: array
                      items:
                        type: object
                        properties:
                          name:
                            type: string
                          role:
                            type: string
                          email:
                            type: array
                            items:
                              type: string
                    TCSGroup:
                      type: string
                    id:
                      type: string
                    uid:
                      type: string

  /api/v2/monitoring:
    get:
      tags: [Monitoring]
      summary: Monitoring summary endpoint
      description: Returns current triple counts and repository metadata
      responses:
        '200':
          description: Monitoring summary payload"""
}
