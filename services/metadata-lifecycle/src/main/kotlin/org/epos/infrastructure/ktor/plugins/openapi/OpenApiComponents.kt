package org.epos.infrastructure.ktor.plugins

object OpenApiComponents {
    val yaml = """
components:

  # -------------------------------------------------------------------------
  # Parameters
  # -------------------------------------------------------------------------

  parameters:

    ResourceId:
      name: id
      in: path
      required: true
      description: Local identifier of the resource
      schema:
        type: string
        examples:
          - "123"

    Q:
      name: q
      in: query
      required: false
      description: Free-text search query
      schema:
        type: string
        examples:
          - "seismic stations"

    Page:
      name: page
      in: query
      required: false
      description: 1-based page number
      schema:
        type: integer
        minimum: 1
        default: 1
        examples:
          - 1

    ResultsPerPage:
      name: resultsPerPage
      in: query
      required: false
      description: Number of items to return per page
      schema:
        type: integer
        minimum: 1
        maximum: 100
        default: 10
        examples:
          - 10

    TypeFilter:
      name: type
      in: query
      required: false
      description: |
        Filter results to one or more resource types. When omitted, all types are returned.
        Repeat the parameter for multiple values: `?type=dataset&type=distribution`.
      schema:
        type: array
        items:
          ${'$'}ref: '#/components/schemas/ResourceType'
      style: form
      explode: true

    SpatialFilter:
      name: spatialFilter
      in: query
      required: false
      description: |
        Spatial filter expressed as a WKT polygon string (EPSG:4326).
        Resources whose spatial coverage intersects this polygon are returned.
        Example: `POLYGON((12.3 41.8, 14.5 41.8, 14.5 43.2, 12.3 43.2, 12.3 41.8))`
      schema:
        type: string
        examples:
          - "POLYGON((12.3 41.8, 14.5 41.8, 14.5 43.2, 12.3 43.2, 12.3 41.8))"

    DateFrom:
      name: dateFrom
      in: query
      required: false
      description: |
        Start of the temporal filter (inclusive). ISO 8601 date-time (UTC).
        Resources with temporal coverage overlapping this range are returned.
      schema:
        type: string
        format: date-time
        examples:
          - "2020-01-01T00:00:00Z"

    DateTo:
      name: dateTo
      in: query
      required: false
      description: |
        End of the temporal filter (inclusive). ISO 8601 date-time (UTC).
        Must be greater than or equal to `dateFrom` when both are supplied.
      schema:
        type: string
        format: date-time
        examples:
          - "2023-12-31T23:59:59Z"

    ServiceProviders:
      name: serviceProvider
      in: query
      required: false
      description: |
        Filter by one or more service provider names.
        Repeat the parameter for multiple values: `?serviceProvider=INGV&serviceProvider=UNINA`.
      schema:
        type: array
        items:
          type: string
      style: form
      explode: true

    VersioningStatusFilter:
      name: versioningStatus
      in: query
      required: false
      description: |
        Filter by one or more versioning statuses.
        When omitted, defaults to `PUBLISHED`.
        Repeat the parameter for multiple values: `?versioningStatus=PUBLISHED&versioningStatus=DRAFT`.
      schema:
        type: array
        items:
          ${'$'}ref: '#/components/schemas/VersioningStatus'
        default:
          - PUBLISHED
      style: form
      explode: true

    Keywords:
      name: keywords
      in: query
      required: false
      description: |
        Filter resources that carry all the supplied keywords.
        Repeat the parameter for multiple values: `?keywords=seismology&keywords=stations`.
      schema:
        type: array
        items:
          type: string
      style: form
      explode: true

  # -------------------------------------------------------------------------
  # Responses
  # -------------------------------------------------------------------------

  responses:

    BadRequest:
      description: Invalid request parameters
      content:
        application/json:
          schema:
            ${'$'}ref: '#/components/schemas/ErrorResponse'

    NotFound:
      description: Resource not found
      content:
        application/json:
          schema:
            ${'$'}ref: '#/components/schemas/ErrorResponse'

    InternalServerError:
      description: Internal server error
      content:
        application/json:
          schema:
            ${'$'}ref: '#/components/schemas/ErrorResponse'

  # -------------------------------------------------------------------------
  # Schemas
  # -------------------------------------------------------------------------

  schemas:

    # -----------------------------------------------------------------------
    # Shared enums
    # -----------------------------------------------------------------------

    ResourceType:
      type: string
      enum:
        - dataset
        - distribution
        - software-application
        - software-source-code
        - category

    VersioningStatus:
      type: string
      enum:
        - DRAFT
        - SUBMITTED
        - PUBLISHED
        - ARCHIVED

    # -----------------------------------------------------------------------
    # Search
    # -----------------------------------------------------------------------

    SearchResponse:
      type: object
      required:
        - total
        - items
      properties:
        total:
          type: integer
          minimum: 0
          description: Total number of matching resources across all pages
          examples:
            - 123
        items:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/SearchItem'

    SearchItem:
      oneOf:
        - ${'$'}ref: '#/components/schemas/DatasetSearchItem'
        - ${'$'}ref: '#/components/schemas/GenericSearchItem'
      discriminator:
        propertyName: type
        mapping:
          dataset: '#/components/schemas/DatasetSearchItem'
          distribution: '#/components/schemas/GenericSearchItem'
          software-application: '#/components/schemas/GenericSearchItem'
          software-source-code: '#/components/schemas/GenericSearchItem'
          category: '#/components/schemas/GenericSearchItem'

    BaseSearchItem:
      type: object
      required:
        - type
        - id
        - href
      properties:
        type:
          ${'$'}ref: '#/components/schemas/ResourceType'
        id:
          type: string
          examples:
            - "123"
        href:
          type: string
          description: Absolute URL of the typed detail endpoint. Follow this link; do not construct URLs.
          examples:
            - "https://metadata.epos-eu.org/api/v2/dataset/123"
        fields:
          type: object
          description: SHACL-selected property values returned as field-value pairs (includes `name`)
          additionalProperties: true

    DatasetSearchItem:
      allOf:
        - ${'$'}ref: '#/components/schemas/BaseSearchItem'
        - type: object
          required:
            - type
          properties:
            type:
              type: string
              const: dataset

    GenericSearchItem:
      allOf:
        - ${'$'}ref: '#/components/schemas/BaseSearchItem'
        - type: object
          required:
            - type
          properties:
            type:
              type: string
              enum:
                - distribution
                - software-application
                - software-source-code
                - category

    # -----------------------------------------------------------------------
    # Resource references (for hrefs)
    # -----------------------------------------------------------------------

    ResourceRef:
      type: object
      required:
        - id
        - href
      properties:
        id:
          type: string
        title:
          type: string
        href:
          type: string
          description: Absolute URL of the typed detail endpoint

    # -----------------------------------------------------------------------
    # Detail schemas
    # -----------------------------------------------------------------------

    TemporalExtent:
      type: object
      description: Temporal coverage of a resource
      properties:
        start:
          type: string
          format: date-time
          description: Start of the temporal coverage (inclusive), ISO 8601 UTC
          examples:
            - "2010-01-01T00:00:00Z"
        end:
          type: string
          format: date-time
          description: End of the temporal coverage (inclusive), ISO 8601 UTC. Omitted for open-ended coverage.
          examples:
            - "2023-12-31T23:59:59Z"

    Dataset:
      type: object
      required:
        - id
        - title
        - status
      properties:
        id:
          type: string
        title:
          type: string
        description:
          type: string
        keywords:
          type: array
          items:
            type: string
          examples:
            - ["seismology", "stations", "Irpinia"]
        categories:
          type: array
          description: Category labels assigned to this dataset
          items:
            type: string
          examples:
            - ["Seismology", "Strong Motion"]
        publisher:
          type: array
          description: Names of the organisations publishing this dataset
          items:
            type: string
          examples:
            - ["UNINA", "INGV"]
        spatial:
          type: string
          description: Spatial coverage expressed as a WKT polygon (EPSG:4326)
          examples:
            - "POLYGON((12.3 41.8, 14.5 41.8, 14.5 43.2, 12.3 43.2, 12.3 41.8))"
        temporal:
          ${'$'}ref: '#/components/schemas/TemporalExtent'
        status:
          ${'$'}ref: '#/components/schemas/VersioningStatus'
        distributions:
          type: array
          description: Distributions that provide access to this dataset
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'
        related:
          type: array
          description: Outbound related entities referenced by this dataset
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'
        relatedBy:
          type: array
          description: Inbound related entities that reference this dataset
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'

    Distribution:
      type: object
      required:
        - id
        - title
      properties:
        id:
          type: string
        title:
          type: string
        description:
          type: string
        accessUrl:
          type: string
          format: uri
          description: URL used to access the distribution
          examples:
            - "https://service.example.org/fdsnws/station/1/query"
        operation:
          type: string
          description: Operation ID describing the service operation exposed by this distribution
          examples:
            - "getFDSNStationXML"
        related:
          type: array
          description: Outbound related entities referenced by this distribution
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'
        relatedBy:
          type: array
          description: Inbound related entities that reference this distribution
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'

    SoftwareApplication:
      type: object
      required:
        - id
        - title
      properties:
        id:
          type: string
        title:
          type: string
        description:
          type: string
        keywords:
          type: array
          items:
            type: string
          examples:
            - ["portal", "seismology", "data access"]
        categories:
          type: array
          description: Category labels assigned to this application
          items:
            type: string
          examples:
            - ["Seismology", "Data Services"]
        downloadUrl:
          type: string
          format: uri
          description: URL to download or install the application
          examples:
            - "https://example.org/application/download"
        related:
          type: array
          description: Outbound related entities referenced by this application
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'
        relatedBy:
          type: array
          description: Inbound related entities that reference this application
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'

    SoftwareSourceCode:
      type: object
      required:
        - id
        - title
      properties:
        id:
          type: string
        title:
          type: string
        description:
          type: string
        programmingLanguage:
          type: string
          description: Primary programming language of the source code
          examples:
            - "Java"
        keywords:
          type: array
          items:
            type: string
          examples:
            - ["portal", "seismology", "open source"]
        categories:
          type: array
          description: Category labels assigned to this source code
          items:
            type: string
          examples:
            - ["Seismology", "Data Services"]
        repository:
          type: string
          format: uri
          description: URL of the source code repository
          examples:
            - "https://github.com/example/epos-portal"
        related:
          type: array
          description: Outbound related entities referenced by this source code
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'
        relatedBy:
          type: array
          description: Inbound related entities that reference this source code
          items:
            ${'$'}ref: '#/components/schemas/ResourceRef'

    # -----------------------------------------------------------------------
    # Category schemas
    # -----------------------------------------------------------------------

    Category:
      type: object
      required:
        - id
        - name
      properties:
        id:
          type: string
          examples:
            - "cat-seismology"
        name:
          type: string
          examples:
            - "Seismology"
        description:
          type: string
          examples:
            - "Resources related to seismic monitoring and analysis."
        children:
          type: array
          description: Nested child categories
          items:
            ${'$'}ref: '#/components/schemas/Category'

    CategoryScheme:
      type: object
      required:
        - id
        - label
        - type
      properties:
        id:
          type: string
          examples:
            - "scheme-solid-earth"
        label:
          type: string
          examples:
            - "Solid Earth"
        title:
          type: string
          examples:
            - "Solid Earth Science Categories"
        description:
          type: string
          examples:
            - "Top-level grouping for all solid earth science resources."
        logo:
          type: string
          format: uri
          examples:
            - "https://www.epos-eu.org/logos/solid-earth.png"
        homepage:
          type: string
          format: uri
          examples:
            - "https://www.epos-eu.org/solid-earth"
        color:
          type: string
          description: Hex colour code used for UI rendering
          examples:
            - "#3A7BD5"
        orderNumber:
          type: integer
          description: Display order among sibling category schemes
          examples:
            - 1
        type:
          type: string
          enum:
            - DATA
            - FACILITY
            - SOFTWARE
          examples:
            - "DATA"
        children:
          type: array
          description: Top-level categories within this scheme
          items:
            ${'$'}ref: '#/components/schemas/Category'

    CategoryTreeResponse:
      type: object
      required:
        - schemes
      properties:
        schemes:
          type: array
          description: Ordered list of category schemes, each carrying its full category subtree
          items:
            ${'$'}ref: '#/components/schemas/CategoryScheme'

    # -----------------------------------------------------------------------
    # Error response
    # -----------------------------------------------------------------------

    ErrorResponse:
      type: object
      required:
        - errorCode
        - errorMessage
      properties:
        errorCode:
          type: string
          examples:
            - "RESOURCE_NOT_FOUND"
        errorMessage:
          type: string
          examples:
            - "The requested resource was not found"

    EntityResponse:
      type: object
      required:
        - uri
        - type
        - status
        - properties
      properties:
        uri:
          type: string
        type:
          type: string
        status:
          type: string
        properties:
          type: object
          additionalProperties: true
        version:
          type: [string, 'null']
        conceptualUri:
          type: [string, 'null']

    ResolvedEntityResponse:
      type: object
      required:
        - resolvedId
        - entity
      properties:
        resolvedId:
          type: string
        entity:
          ${'$'}ref: '#/components/schemas/EntityResponse'

    # -----------------------------------------------------------------------
    # Legacy schemas (for backward compatibility with admin/resource endpoints)
    # -----------------------------------------------------------------------

    PropertyInfo:
      type: object
      properties:
        path:
          type: string
          format: uri
        name:
          type: string
        datatype:
          type: [string, 'null']
          format: uri
        datatypes:
          type: [array, 'null']
          items:
            type: string
            format: uri
        nodeKind:
          type: [string, 'null']
        nodeKinds:
          type: [array, 'null']
          items:
            type: string
        classConstraint:
          type: [string, 'null']
          format: uri
        classConstraints:
          type: [array, 'null']
          items:
            type: string
            format: uri
        nodeShapeRef:
          type: [string, 'null']
        nodeShapeRefs:
          type: [array, 'null']
          items:
            type: string
        minCount:
          type: [integer, 'null']
          format: int32
        maxCount:
          type: [integer, 'null']
          format: int32
        required:
          type: boolean
        recommended:
          type: boolean
        pattern:
          type: [string, 'null']
        minLength:
          type: [integer, 'null']
          format: int32
        maxLength:
          type: [integer, 'null']
          format: int32
        hasValueIn:
          type: [array, 'null']
          items:
            type: string
        severity:
          type: [string, 'null']
        severities:
          type: [array, 'null']
          items:
            type: string
        helpText:
          type: [string, 'null']
        description:
          type: [string, 'null']
      required: [path, name, required, recommended]

    ResourceTypeInfo:
      type: object
      properties:
        name:
          type: string
        targetClass:
          type: string
          format: uri
        properties:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/PropertyInfo'
        closed:
          type: boolean
        description:
          type: [string, 'null']
      required: [name, targetClass, properties, closed]

    ResourceTypeList:
      type: array
      description: Returned when shape definitions are not requested
      items:
        ${'$'}ref: '#/components/schemas/ResourceTypeInfo'

    ShapeDefinitionInfo:
      type: object
      properties:
        id:
          type: string
        targetClasses:
          type: array
          items:
            type: string
            format: uri
        properties:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/PropertyInfo'
        resolvedRefs:
          type: [array, 'null']
          items:
            type: string
        name:
          type: [string, 'null']
        datatype:
          type: [string, 'null']
          format: uri
        datatypes:
          type: [array, 'null']
          items:
            type: string
            format: uri
        nodeKind:
          type: [string, 'null']
        nodeKinds:
          type: [array, 'null']
          items:
            type: string
        classConstraint:
          type: [string, 'null']
          format: uri
        classConstraints:
          type: [array, 'null']
          items:
            type: string
            format: uri
        nodeShapeRef:
          type: [string, 'null']
        nodeShapeRefs:
          type: [array, 'null']
          items:
            type: string
        ontologyEquivalentClasses:
          type: [array, 'null']
          items:
            type: string
            format: uri
        ontologySubClassesOf:
          type: [array, 'null']
          items:
            type: string
            format: uri
        closed:
          type: boolean
        description:
          type: [string, 'null']
      required: [id, targetClasses, properties, closed]

    ResourceTypesEnvelope:
      type: object
      description: Returned when includeShapeDefinitions or includeTransitiveShapeDefinitions is true
      properties:
        resourceTypes:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/ResourceTypeInfo'
        shapeDefinitions:
          type: object
          description: Map keyed by shape IRI
          additionalProperties:
            ${'$'}ref: '#/components/schemas/ShapeDefinitionInfo'
      required: [resourceTypes]

    VersionInfo:
      type: object
      properties:
        uri:
          type: string
          format: uri
        version:
          type: string
        timestamp:
          type: string
        agent:
          type: [string, 'null']
        group:
          type: [string, 'null']
        status:
          type: string
          enum: [draft, submitted, published, archived, unknown]
      required: [uri, version, timestamp, status]

    VersionHistoryResponse:
      type: object
      properties:
        conceptualUri:
          type: string
          format: uri
        versions:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/VersionInfo'
      required: [conceptualUri, versions]

    RevisionImportFileRequest:
      type: object
      description: Revision import from a single TTL source (URL or local path)
      properties:
        url:
          type: string
          description: HTTP(S) URL to a TTL file
        path:
          type: string
          description: Local filesystem path to a TTL file
        versionMode:
          type: string
          enum: [auto, explicit]
          default: auto
        version:
          type: string
          description: Target version number, required when versionMode=explicit
          examples:
            - "2.0"
        replaceDraftIfExists:
          type: boolean
          default: true
        dryRun:
          type: boolean
          default: false

    RevisionImportContentRequest:
      type: object
      required: [content]
      description: Revision import from raw TTL content
      properties:
        content:
          type: string
        sourceName:
          type: string
          examples:
            - imported-batch-2026-04-24
        versionMode:
          type: string
          enum: [auto, explicit]
          default: auto
        version:
          type: string
          description: Target version number, required when versionMode=explicit
          examples:
            - "2.0"
        replaceDraftIfExists:
          type: boolean
          default: true
        dryRun:
          type: boolean
          default: false

    RevisionImportEntityResult:
      type: object
      properties:
        importedUri:
          type: string
        conceptualUri:
          type: string
        draftUri:
          type: string
        status:
          type: string
        message:
          type: [string, 'null']
      required: [importedUri, conceptualUri, draftUri, status]

    RevisionImportResponseData:
      type: object
      properties:
        source:
          type: string
        dryRun:
          type: boolean
        subjectsProcessed:
          type: integer
        triplesPrepared:
          type: integer
        triplesInserted:
          type: integer
        draftsCreated:
          type: integer
        draftsReused:
          type: integer
        linksRewritten:
          type: integer
        entities:
          type: array
          items:
            ${'$'}ref: '#/components/schemas/RevisionImportEntityResult'
        agent:
          type: string
        group:
          type: string
      required: [source, dryRun, subjectsProcessed, triplesPrepared, triplesInserted, draftsCreated, draftsReused, linksRewritten, entities, agent, group]

    RevisionImportApiResponse:
      type: object
      properties:
        success:
          type: boolean
        message:
          type: string
        data:
          ${'$'}ref: '#/components/schemas/RevisionImportResponseData'
      required: [success, message]
"""
}
