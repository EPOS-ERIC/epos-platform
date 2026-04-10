package org.epos.infrastructure.ktor.plugins

object OpenApiComponents {
    val yaml = """
components:
  schemas:
    PropertyInfo:
      type: object
      properties:
        path:
          type: string
          format: uri
        name:
          type: string
        datatype:
          type: string
          format: uri
          nullable: true
        datatypes:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        nodeKind:
          type: string
          nullable: true
        nodeKinds:
          type: array
          nullable: true
          items:
            type: string
        classConstraint:
          type: string
          format: uri
          nullable: true
        classConstraints:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        nodeShapeRef:
          type: string
          nullable: true
        nodeShapeRefs:
          type: array
          nullable: true
          items:
            type: string
        minCount:
          type: integer
          format: int32
          nullable: true
        maxCount:
          type: integer
          format: int32
          nullable: true
        required:
          type: boolean
        recommended:
          type: boolean
        pattern:
          type: string
          nullable: true
        minLength:
          type: integer
          format: int32
          nullable: true
        maxLength:
          type: integer
          format: int32
          nullable: true
        hasValueIn:
          type: array
          nullable: true
          items:
            type: string
        severity:
          type: string
          nullable: true
        severities:
          type: array
          nullable: true
          items:
            type: string
        helpText:
          type: string
          nullable: true
        description:
          type: string
          nullable: true
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
          type: string
          nullable: true
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
          type: array
          nullable: true
          items:
            type: string
        name:
          type: string
          nullable: true
        datatype:
          type: string
          format: uri
          nullable: true
        datatypes:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        nodeKind:
          type: string
          nullable: true
        nodeKinds:
          type: array
          nullable: true
          items:
            type: string
        classConstraint:
          type: string
          format: uri
          nullable: true
        classConstraints:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        nodeShapeRef:
          type: string
          nullable: true
        nodeShapeRefs:
          type: array
          nullable: true
          items:
            type: string
        ontologyEquivalentClasses:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        ontologySubClassesOf:
          type: array
          nullable: true
          items:
            type: string
            format: uri
        closed:
          type: boolean
        description:
          type: string
          nullable: true
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
"""
}
