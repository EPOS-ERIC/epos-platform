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
        val runtimeSection = buildRuntimeShapesExtension(resourceTypes)
        return "$yamlWithRuntimeTypes\n$runtimeSection"
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

    private fun escapeYaml(value: String): String = value.replace("\\", "\\\\").replace("\"", "\\\"")
}

object OpenApiPaths {
    object System {
        val yaml = OpenApiSystemPaths.yaml
    }

    object Search {
        val yaml = OpenApiSearchPaths.yaml
    }

    object Categories {
        val yaml = OpenApiCategoryPaths.yaml
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
