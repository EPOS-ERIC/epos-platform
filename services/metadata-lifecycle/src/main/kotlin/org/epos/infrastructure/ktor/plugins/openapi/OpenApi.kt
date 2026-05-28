package org.epos.infrastructure.ktor.plugins

import org.epos.api.models.ResourceTypeInfo

object OpenApiSpec {
    private val baseYaml: String by lazy {
        listOf(
            OpenApiHeader.yaml,
            OpenApiPaths.System.yaml,
            OpenApiPaths.Search.yaml,
            OpenApiPaths.Access.yaml,
            OpenApiPaths.Sparql.yaml,
            OpenApiPaths.OaiPmh.yaml,
            OpenApiPaths.Resources.yaml,
            OpenApiPaths.Lifecycle.yaml,
            OpenApiPaths.Versioning.yaml,
            OpenApiPaths.Batch.yaml,
            OpenApiPaths.Admin.yaml,
            OpenApiComponents.yaml,
        ).joinToString("\n")
    }

    val yaml: String
        get() = baseYaml

    @Volatile
    private var runtimeYamlCache: String = baseYaml

    fun initializeRuntime(resourceTypes: List<ResourceTypeInfo>) {
        runtimeYamlCache = yaml(resourceTypes)
    }

    val runtimeYaml: String
        get() = runtimeYamlCache

    fun yaml(resourceTypes: List<ResourceTypeInfo>): String {
        val yamlWithRuntimeTypes = applyRuntimeResourceTypes(baseYaml, resourceTypes)
        val yamlWithEntityPaths = applyRuntimeEntityPaths(yamlWithRuntimeTypes, resourceTypes)
        val yamlWithEntitySchemas = applyRuntimeEntitySchemas(yamlWithEntityPaths, resourceTypes)
        val runtimeSection = buildRuntimeShapesExtension(resourceTypes)
        return "$yamlWithEntitySchemas\n$runtimeSection"
    }

    private fun applyRuntimeResourceTypes(
        base: String,
        resourceTypes: List<ResourceTypeInfo>,
    ): String {
        if (resourceTypes.isEmpty()) return base

        val dynamicTypes =
            resourceTypes
                .map { it.name.trim() }
                .filter { it.isNotBlank() }
                .distinct()
                .sorted()

        if (dynamicTypes.isEmpty()) return base

        val enumBlock =
            buildString {
                appendLine("      enum:")
                dynamicTypes.forEach { type ->
                    appendLine("        - ${escapeYaml(type)}")
                }
            }.trimEnd()

        val resourceTypeRegex =
            Regex(
                pattern = """(?m)^\s{4}ResourceType:\n\s{6}type:\s*string\n\s{6}enum:\n(?:\s{8}- [^\n]*\n)+""",
            )

        return base.replace(resourceTypeRegex) {
            "    ResourceType:\n      type: string\n$enumBlock\n"
        }
    }

    private fun buildRuntimeShapesExtension(resourceTypes: List<ResourceTypeInfo>): String {
        if (resourceTypes.isEmpty()) return "x-runtime-shapes: []"

        val lines = mutableListOf<String>()
        lines += "x-runtime-shapes:"
        resourceTypes.sortedBy { it.name }.forEach { type ->
            lines += "  - name: \"${escapeYaml(type.name)}\""
            lines += "    targetClass: \"${escapeYaml(type.targetClass)}\""
            type.description?.let { lines += "    description: \"${escapeYaml(it)}\"" }
            lines += "    filters:"
            if (type.properties.isEmpty()) {
                lines += "      []"
            } else {
                type.properties
                    .sortedBy { it.name }
                    .forEach { property ->
                        lines += "      - name: \"${escapeYaml(property.name)}\""
                        lines += "        path: \"${escapeYaml(property.path)}\""
                        property.datatype?.let { lines += "        datatype: \"${escapeYaml(it)}\"" }
                        property.classConstraint?.let { lines += "        classConstraint: \"${escapeYaml(it)}\"" }
                        property.required.takeIf { it }?.let { lines += "        required: true" }
                    }
            }
        }
        return lines.joinToString("\n")
    }

    private fun applyRuntimeEntityPaths(base: String, resourceTypes: List<ResourceTypeInfo>): String {
        if (resourceTypes.isEmpty()) return base

        val pathsBlock =
            buildString {
                resourceTypes
                    .map { it.name.trim().lowercase() }
                    .filter { it.isNotBlank() }
                    .distinct()
                    .sorted()
                    .forEach { typeName ->
                        val schemaName = runtimeEntitySchemaName(typeName)
                        appendLine("  /api/v2/entities/$typeName:")
                        appendLine("    get:")
                        appendLine("      tags: [Entities]")
                        appendLine("      summary: List $typeName entities")
                        appendLine("      parameters:")
                        appendLine("        - \$ref: '#/components/parameters/Page'")
                        appendLine("        - \$ref: '#/components/parameters/ResultsPerPage'")
                        appendLine("        - name: status")
                        appendLine("          in: query")
                        appendLine("          required: false")
                        appendLine("          schema:")
                        appendLine("            type: string")
                        appendLine("            default: published")
                        appendLine("      responses:")
                        appendLine("        '200':")
                        appendLine("          description: Entity list")
                        appendLine("          content:")
                        appendLine("            application/json:")
                        appendLine("              schema:")
                        appendLine("                \$ref: '#/components/schemas/$schemaName'")
                        appendLine("  /api/v2/entities/$typeName/{id}:")
                        appendLine("    get:")
                        appendLine("      tags: [Entities]")
                        appendLine("      summary: Get $typeName entity")
                        appendLine("      parameters:")
                        appendLine("        - name: id")
                        appendLine("          in: path")
                        appendLine("          required: true")
                        appendLine("          description: Conceptual or version resource identifier (not internal numeric graph id)")
                        appendLine("          schema:")
                        appendLine("            type: string")
                        appendLine("        - name: status")
                        appendLine("          in: query")
                        appendLine("          required: false")
                        appendLine("          schema:")
                        appendLine("            type: string")
                        appendLine("            default: published")
                        appendLine("        - name: format")
                        appendLine("          in: query")
                        appendLine("          required: false")
                        appendLine("          description: Response format override")
                        appendLine("          schema:")
                        appendLine("            type: string")
                        appendLine("            enum: [json, ttl, jsonld, rdfxml, nt]")
                        appendLine("      responses:")
                        appendLine("        '200':")
                        appendLine("          description: Entity detail")
                        appendLine("          content:")
                        appendLine("            application/json:")
                        appendLine("              schema:")
                        appendLine("                \$ref: '#/components/schemas/ResolvedEntityResponse_$schemaName'")
                        appendLine("            text/turtle:")
                        appendLine("              schema:")
                        appendLine("                type: string")
                        appendLine("            application/ld+json:")
                        appendLine("              schema:")
                        appendLine("                type: string")
                        appendLine("            application/rdf+xml:")
                        appendLine("              schema:")
                        appendLine("                type: string")
                        appendLine("            application/n-triples:")
                        appendLine("              schema:")
                        appendLine("                type: string")
                    }
            }.trimEnd()

        return base.replace("\ncomponents:\n", "\n$pathsBlock\ncomponents:\n")
    }

    private fun applyRuntimeEntitySchemas(base: String, resourceTypes: List<ResourceTypeInfo>): String {
        if (resourceTypes.isEmpty()) return base

        val uniqueTypesByLowerName =
            resourceTypes
                .map { it.name.trim().lowercase() to it }
                .filter { it.first.isNotBlank() }
                .groupBy { it.first }
                .mapValues { (_, items) -> items.first().second }

        val uniqueTypes =
            uniqueTypesByLowerName
                .toList()
                .groupBy { (typeName, _) -> runtimeEntitySchemaName(typeName) }
                .mapValues { (_, items) -> items.first() }
                .values

        val schemasBlock =
            buildString {
                uniqueTypes
                    .sortedBy { it.first }
                    .forEach { (typeName, type) ->
                        val schemaName = runtimeEntitySchemaName(typeName)
                        appendLine("    $schemaName:")
                        appendLine("      type: object")
                        appendLine("      properties:")
                        type.properties
                            .asSequence()
                            .filter { it.name.isNotBlank() }
                            .groupBy { it.name.trim().lowercase() }
                            .mapValues { (_, items) -> items.first() }
                            .toList()
                            .sortedBy { it.first }
                            .forEach { (lowerName, property) ->
                            val propKey = escapeYaml(lowerName)
                            append(buildPropertySchemaYaml(propKey, property))
                            }
                        appendLine("      additionalProperties: true")
                        appendLine("    ResolvedEntityResponse_$schemaName:")
                        appendLine("      allOf:")
                        appendLine("        - \$ref: '#/components/schemas/ResolvedEntityResponse'")
                        appendLine("        - type: object")
                        appendLine("          properties:")
                        appendLine("            entity:")
                        appendLine("              allOf:")
                        appendLine("                - \$ref: '#/components/schemas/EntityResponse'")
                        appendLine("                - type: object")
                        appendLine("                  properties:")
                        appendLine("                    properties:")
                        appendLine("                      \$ref: '#/components/schemas/$schemaName'")
                    }
            }.trimEnd()

        return base.replace("\n    ResourceTypesEnvelope:\n", "\n$schemasBlock\n\n    ResourceTypesEnvelope:\n")
    }

    private fun runtimeEntitySchemaName(typeName: String): String {
        val cleaned = typeName.replace(Regex("[^A-Za-z0-9_]"), "_")
        return "Entity_${cleaned}"
    }

    private fun buildPropertySchemaYaml(
        propertyKey: String,
        property: org.epos.api.shacl.PropertyInfo,
    ): String {
        val schemaType = inferOpenApiType(property)
        val format = inferOpenApiFormat(property)
        val enumValues = property.hasValueIn.orEmpty().map { it.trim() }.filter { it.isNotBlank() }.distinct().sorted()
        val isArray = (property.maxCount == null || property.maxCount > 1)
        val lines = mutableListOf<String>()
        lines += "        $propertyKey:"
        if (isArray) {
            lines += "          type: array"
            lines += "          items:"
            lines += "            type: $schemaType"
            if (!format.isNullOrBlank()) {
                lines += "            format: $format"
            }
            if (enumValues.isNotEmpty()) {
                lines += "            enum:"
                enumValues.forEach { value -> lines += "              - \"${escapeYaml(value)}\"" }
            }
            property.minCount?.let { lines += "          minItems: $it" }
            property.maxCount?.let { lines += "          maxItems: $it" }
        } else {
            lines += "          type: $schemaType"
            if (!format.isNullOrBlank()) {
                lines += "          format: $format"
            }
            if (enumValues.isNotEmpty()) {
                lines += "          enum:"
                enumValues.forEach { value -> lines += "            - \"${escapeYaml(value)}\"" }
            }
        }
        if (property.required) {
            lines += "          nullable: false"
        } else {
            lines += "          nullable: true"
        }
        if (property.description != null) {
            lines += "          description: \"${escapeYaml(property.description)}\""
        }
        return lines.joinToString("\n", postfix = "\n")
    }

    private fun inferOpenApiType(property: org.epos.api.shacl.PropertyInfo): String {
        val datatype = property.datatype?.lowercase().orEmpty()
        val datatypes = property.datatypes.orEmpty().map { it.lowercase() }
        val all = buildList {
            if (datatype.isNotBlank()) add(datatype)
            addAll(datatypes)
        }
        if (all.any { it.contains("boolean") }) return "boolean"
        if (all.any { it.contains("int") || it.contains("integer") || it.endsWith("#long") || it.endsWith("#short") }) return "integer"
        if (all.any { it.contains("decimal") || it.contains("double") || it.contains("float") }) return "number"
        return "string"
    }

    private fun inferOpenApiFormat(property: org.epos.api.shacl.PropertyInfo): String? {
        val datatype = property.datatype?.lowercase().orEmpty()
        val datatypes = property.datatypes.orEmpty().map { it.lowercase() }
        val all = buildList {
            if (datatype.isNotBlank()) add(datatype)
            addAll(datatypes)
        }
        if (property.classConstraint != null || !property.classConstraints.isNullOrEmpty()) return "uri"
        return when {
            all.any { it.contains("datetime") || it.endsWith("#datetime") } -> "date-time"
            all.any { it.endsWith("#date") || it.contains("/date") } -> "date"
            all.any { it.contains("anyuri") || it.endsWith("#uri") } -> "uri"
            else -> null
        }
    }

    private fun escapeYaml(value: String): String = value.replace("\\", "\\\\").replace("\"", "\\\"")
}

object OpenApiPaths {
    object System {
        val yaml = OpenApiSystemPaths.yaml
    }

    object Search {
        val yaml = OpenApiSearchPaths.yaml
    }

    object Access {
        val yaml = OpenApiAccessPaths.yaml
    }

    object Sparql {
        val yaml = OpenApiSparqlPaths.yaml
    }

    object OaiPmh {
        val yaml = OpenApiOaiPmhPaths.yaml
    }

    object Resources {
        val yaml = OpenApiResourcePaths.yaml
    }

    object Lifecycle {
        val yaml = OpenApiLifecyclePaths.yaml
    }

    object Versioning {
        val yaml = OpenApiVersioningPaths.yaml
    }

    object Batch {
        val yaml = OpenApiBatchPaths.yaml
    }

    object Admin {
        val yaml = OpenApiAdminPaths.yaml
    }
}
