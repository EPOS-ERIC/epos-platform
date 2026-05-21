package org.epos.api.controllers

import org.epos.api.models.*
import org.epos.api.shacl.ShaclParser
import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.EntityLifecycleService
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.utils.UriUtils
import com.fasterxml.jackson.databind.ObjectMapper
import org.apache.tinkerpop.gremlin.structure.T
import java.util.UUID

@Suppress("UNCHECKED_CAST")
class ResourceController(
    private val client: PropertyGraphClient,
    private val lifecycleService: EntityLifecycleService,
    private val provenanceService: ProvenanceService
) {
    private val mapper = ObjectMapper()
    var resourceTypes: List<ResourceTypeInfo> = listOf()

    fun setResourceType(resourceTypes: List<ResourceTypeInfo>) {
        this.resourceTypes = resourceTypes
    }

    fun getResourceType(): List<ResourceTypeInfo> {
        return resourceTypes
    }

    fun listEntities(
        resourceType: String,
        status: String = "published",
        page: Int = 1,
        pageSize: Int = 20
    ): EntityListResponse {
        val typeFilter = getTypeUri(resourceType)
        val typeCandidates = buildConceptualTypeCandidates(resourceType, typeFilter)

        val skip = (page - 1) * pageSize
        val typePredicate =
            if (typeCandidates.isNotEmpty()) {
                val values = typeCandidates.joinToString(",") { gStr(it) }
                ".has('type', within($values))"
            } else {
                ""
            }
        val results = client.runQuery(
            """
            g.V().hasLabel('Version').has('status', ${gStr(status)})
              .as('v')
              .in('HAS_VERSION').hasLabel('Conceptual')$typePredicate
              .as('c')
              .select('v','c')
              .by(valueMap(true))
              .by(valueMap(true))
              .range($skip, ${skip + pageSize})
              .toList()
            """.trimIndent()
        )

        val entities = results.mapNotNull { row ->
            val v = normalizePropertyMap(row["v"] as? Map<*, *> ?: return@mapNotNull null)
            val c = normalizePropertyMap(row["c"] as? Map<*, *> ?: return@mapNotNull null)

            val versionUri = propString(v["id"]) ?: return@mapNotNull null
            val conceptualUri = propString(c["id"])

            val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt", "_vertexId", "_vertexLabel", "label")
            val dataMap = v.filterKeys { it !in structuralKeys }
                .mapValues { (_, v2) -> normalizeValue(v2) ?: "" }

            EntityResponse(
                uri = versionUri,
                type = resourceType,
                status = status,
                properties = dataMap,
                conceptualUri = conceptualUri
            )
        }

        val countScript =
            "g.V().hasLabel('Version').has('status', ${gStr(status)}).in('HAS_VERSION').hasLabel('Conceptual')$typePredicate.count().next()"
        val total = client.runQuery(countScript).firstOrNull()?.get("value")?.toString()?.toIntOrNull() ?: 0

        return EntityListResponse(
            entities = entities,
            total = total,
            page = page,
            pageSize = pageSize
        )
    }

    private fun isManagedUri(value: String): Boolean {
        return value.startsWith("http://") || value.startsWith("https://")
    }

    private fun extractResourceTypeFromUri(uri: String): String? {
        val results = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(uri)}).valueMap(true).toList()")

        val v = normalizePropertyMap(results.firstOrNull()?.get("value") as? Map<*, *> ?: return null)
        val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
        val data = v.filterKeys { it !in structuralKeys }
        val typeValue = propString(data["type"]) ?: return null

        resourceTypes.forEach { resourceType ->
            if (resourceType.targetClass.equals(typeValue, ignoreCase = true)) {
                return resourceType.name
            }
        }

        return null
    }

    private fun enrichValue(value: Any): Any {
        return when (value) {
            is String -> {
                if (isManagedUri(value)) {
                    val resourceType = extractResourceTypeFromUri(value)
                    if (resourceType != null) {
                        EntityReference(
                            uri = value,
                            link = "/api/v2/$resourceType/${UriUtils.extractConceptualUri(value)}",
                            resourceType = resourceType
                        )
                    } else {
                        value
                    }
                } else {
                    value
                }
            }
            is List<*> -> value.map { if (it != null) enrichValue(it) else it }
            is Map<*, *> -> value.mapValues { if (it.value != null) enrichValue(it.value!!) else it.value }
            else -> value
        }
    }

    fun getEntity(entityUri: String, status: String = "published", enrichReferences: Boolean = true): EntityResponse? {
        val result =
            client.runQuery(
                "g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status', ${gStr(status)}).valueMap(true).limit(1).toList()",
            ).firstOrNull()
                ?: client.runQuery(
                    "g.V().hasLabel('Version').has('identifier', ${gStr(entityUri)}).has('status', ${gStr(status)}).valueMap(true).limit(1).toList()",
                ).firstOrNull()
                ?: return null

        val v = normalizePropertyMap(result["value"] as? Map<*, *> ?: return null)
        val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt", "_vertexId", "_vertexLabel", "label")
        val data = v.filterKeys { it !in structuralKeys }
            .mapValues { (_, v2) -> normalizeValue(v2) ?: "" }

        val enrichedProperties = if (enrichReferences) {
            data.mapValues { (_, value) -> enrichValue(value) }
        } else {
            data
        }

        val typeUri = propString(enrichedProperties["type"]) ?: ""
        val resourceType = extractResourceName(typeUri)

        val conceptualUri =
            client.runQuery(
                "g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).in('HAS_VERSION').values('id').limit(1).toList()",
            ).firstOrNull()?.get("value")?.toString()
                ?: client.runQuery(
                    "g.V().hasLabel('Version').has('identifier', ${gStr(entityUri)}).in('HAS_VERSION').values('id').limit(1).toList()",
                ).firstOrNull()?.get("value")?.toString()

        return EntityResponse(
            uri = entityUri,
            type = resourceType,
            status = status,
            properties = enrichedProperties,
            conceptualUri = conceptualUri
        )
    }

    fun getEntityByConceptualId(conceptualId: String, status: String = "published", enrichReferences: Boolean = true): EntityResponse? {
        val result = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualId)})
              .out('CURRENT_VERSION').hasLabel('Version').has('status', ${gStr(status)})
              .project('uri','data')
              .by(coalesce(values('identifier'), values('id'), constant('')))
              .by(valueMap())
              .limit(1).toList()
            """.trimIndent(),
        ).firstOrNull() ?: return null

        val versionUri = propString(result["uri"]) ?: conceptualId
        val v = normalizePropertyMap(result["data"] as? Map<*, *> ?: return null)
        val structuralKeys = setOf("status", "versionNumber", "createdAt", "updatedAt", "_vertexId", "_vertexLabel", "label")
        val data = v.filterKeys { it !in structuralKeys }
            .mapValues { (_, v2) -> normalizeValue(v2) ?: "" }

        val enrichedProperties = if (enrichReferences) {
            data.mapValues { (_, value) -> enrichValue(value) }
        } else {
            data
        }

        val typeUri = propString(enrichedProperties["type"]) ?: ""
        val resourceType = extractResourceName(typeUri)

        return EntityResponse(
            uri = versionUri,
            type = resourceType,
            status = status,
            properties = enrichedProperties,
            conceptualUri = conceptualId,
        )
    }

    fun createEntity(request: CreateEntityRequest, agent: String, group: String): ApiResponse {
        val entityId = UUID.randomUUID().toString()
        val conceptualUri = "${Config.ID_BASE}/${request.type}/$entityId"
        val draftUri = "$conceptualUri/1.0-draft"

        val typeUri = getTypeUri(request.type)
        val typeShortName = typeUri.substringAfterLast("/").substringAfterLast("#").lowercase()

        val data = request.properties.toMutableMap()
        data["type"] = typeUri

        val now = java.time.Instant.now().toString()
        val propScript = data.entries.joinToString("\n") { (k, v) ->
            "v.property(${gStr(sanitizeKey(k))}, ${gVal(v)})"
        }
        client.runWrite(
            """
            def c = g.addV('Conceptual').property('id', ${gStr(conceptualUri)}).property('type', ${gStr(typeShortName)}).property('createdAt', ${gStr(now)}).next()
            def v = g.addV('Version')
              .property('id', ${gStr(draftUri)})
              .property('status', 'draft')
              .property('versionNumber', '1.0')
              .property('createdAt', ${gStr(now)})
              .property('updatedAt', ${gStr(now)})
              .next()
            $propScript
            g.V(c.id()).addE('HAS_VERSION').to(V(v.id())).iterate()
            """.trimIndent()
        )

        provenanceService.recordDraftCreation(
            newVersionUri = draftUri,
            basedOnVersionUri = draftUri,
            conceptualUri = conceptualUri,
            agent = agent,
            group = group
        )

        return ApiResponse(
            success = true,
            message = "Entity created successfully",
            data = mapOf(
                "uri" to draftUri,
                "conceptualUri" to conceptualUri,
                "status" to "draft"
            )
        )
    }

    fun updateEntity(entityUri: String, request: UpdateEntityRequest, agent: String, group: String): ApiResponse {
        val exists = client.exists("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status','draft').limit(1).toList()")

        if (!exists) {
            return ApiResponse(success = false, message = "Entity not found in draft status")
        }

        val data = request.properties
        val now = java.time.Instant.now().toString()
        val setLines = data.entries.joinToString("\n") { (k, v) -> "v.property(${gStr(sanitizeKey(k))}, ${gVal(v)})" }
        client.runWrite("""
            def v = g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status','draft').next()
            $setLines
            v.property('updatedAt', ${gStr(now)})
        """.trimIndent())

        return ApiResponse(
            success = true,
            message = "Entity updated successfully",
            data = mapOf("uri" to entityUri)
        )
    }

    fun patchEntity(entityUri: String, request: UpdateEntityRequest, agent: String, group: String): ApiResponse {
        val exists = client.exists("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status','draft').limit(1).toList()")

        if (!exists) {
            return ApiResponse(success = false, message = "Entity not found in draft status")
        }

        val now = java.time.Instant.now().toString()
        val setLines = request.properties.entries.joinToString("\n") { (k, v) -> "v.property(${gStr(sanitizeKey(k))}, ${gVal(v)})" }
        client.runWrite("""
            def v = g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status','draft').next()
            $setLines
            v.property('updatedAt', ${gStr(now)})
        """.trimIndent())

        return ApiResponse(
            success = true,
            message = "Entity patched successfully",
            data = mapOf("uri" to entityUri)
        )
    }

    fun submitEntity(entityUri: String, agent: String, group: String): ApiResponse {
        lifecycleService.submitDraft(entityUri, agent, group)

        return ApiResponse(
            success = true,
            message = "Entity submitted for review",
            data = mapOf("uri" to entityUri, "status" to "submitted")
        )
    }

    fun publishEntity(entityUri: String, agent: String, group: String): ApiResponse {
        lifecycleService.publish(entityUri, agent, group)

        val finalUri = entityUri.removeSuffix("-draft")
        return ApiResponse(
            success = true,
            message = "Entity published successfully",
            data = mapOf("uri" to finalUri, "status" to "published")
        )
    }

    fun createNewDraft(conceptualUri: String, newVersion: String, agent: String, group: String): ApiResponse {
        try {
            val draftUri = lifecycleService.createDraft(
                conceptualUri = conceptualUri,
                newVersionNumber = newVersion,
                agent = agent,
                group = group
            )

            return ApiResponse(
                success = true,
                message = "New draft created successfully",
                data = mapOf(
                    "uri" to draftUri,
                    "conceptualUri" to conceptualUri,
                    "version" to newVersion,
                    "status" to "draft"
                )
            )
        } catch (e: Exception) {
            return ApiResponse(
                success = false,
                message = "Failed to create draft: ${e.message}"
            )
        }
    }

    fun deleteEntity(entityUri: String): ApiResponse {
        lifecycleService.deleteDraft(entityUri)

        return ApiResponse(
            success = true,
            message = "Entity deleted successfully"
        )
    }

    fun getVersionHistory(conceptualUri: String): VersionHistoryResponse {
        val history = provenanceService.getVersionHistory(conceptualUri)

        val versions = history.map { version ->
            val versionUri = version["version"] ?: ""
            VersionInfo(
                uri = versionUri,
                version = UriUtils.extractVersionNumber(versionUri),
                timestamp = version["timestamp"] ?: "",
                agent = version["agent"],
                group = version["group"],
                status = version["status"] ?: determineStatusForVersion(versionUri)
            )
        }

        return VersionHistoryResponse(
            conceptualUri = conceptualUri,
            versions = versions
        )
    }

    private fun determineStatusForVersion(versionUri: String): String {
        val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(versionUri)}).values('status').limit(1).toList()")
            .firstOrNull()?.get("value")?.toString()
        return result?.lowercase() ?: "unknown"
    }

    fun submitEntityBatch(entityUris: List<String>, agent: String, group: String): BatchOperationResponse {
        val results = entityUris.map { uri ->
            try {
                submitEntity(uri, agent, group)
            } catch (e: Exception) {
                ApiResponse(
                    success = false,
                    message = "Failed to submit $uri: ${e.message}",
                    data = mapOf("uri" to uri)
                )
            }
        }

        val successCount = results.count { it.success }
        val failureCount = results.count { !it.success }

        return BatchOperationResponse(
            success = failureCount == 0,
            message = "Batch submit completed: $successCount succeeded, $failureCount failed",
            results = results
        )
    }

    fun publishEntityBatch(entityUris: List<String>, agent: String, group: String): BatchOperationResponse {
        val sortedUris = sortByDependencies(entityUris)

        val results = sortedUris.map { uri ->
            try {
                publishEntity(uri, agent, group)
            } catch (e: Exception) {
                ApiResponse(
                    success = false,
                    message = "Failed to publish $uri: ${e.message}",
                    data = mapOf("uri" to uri)
                )
            }
        }

        val successCount = results.count { it.success }
        val failureCount = results.count { !it.success }

        return BatchOperationResponse(
            success = failureCount == 0,
            message = "Batch publish completed: $successCount succeeded, $failureCount failed",
            results = results,
            data = mapOf("executionOrder" to sortedUris)
        )
    }

    private fun sortByDependencies(entityUris: List<String>): List<String> {
        val dependencyGraph = mutableMapOf<String, MutableSet<String>>()

        entityUris.forEach { uri ->
            dependencyGraph[uri] = mutableSetOf()

            val results = client.runQuery("""
                g.V().hasLabel('Version').has('id', ${gStr(uri)})
                  .valueMap(true)
                  .toList()
            """.trimIndent())

            try {
                val deps = results.flatMap { row ->
                    val vm = row["value"] as? Map<*, *> ?: return@flatMap emptyList<String>()
                    vm.values.mapNotNull { propString(it) }
                }
                    .filter { it.startsWith("${Config.ID_BASE}/") }
                    .filter { it in entityUris }

                dependencyGraph[uri]?.addAll(deps)
            } catch (e: Exception) {
                println("Warning: Could not determine dependencies for $uri: ${e.message}")
            }
        }

        val sorted = mutableListOf<String>()
        val inDegree = entityUris.associateWith { uri ->
            dependencyGraph.values.count { it.contains(uri) }
        }.toMutableMap()

        val queue = ArrayDeque(inDegree.filter { it.value == 0 }.map { it.key })

        while (queue.isNotEmpty()) {
            val current = queue.removeFirst()
            sorted.add(current)

            dependencyGraph[current]?.forEach { dep ->
                inDegree[dep] = (inDegree[dep] ?: 0) - 1
                if (inDegree[dep] == 0) {
                    queue.add(dep)
                }
            }
        }

        val remaining = entityUris.filter { it !in sorted }
        sorted.addAll(remaining)

        return sorted
    }

    fun getTypeUri(resourceType: String): String {
        for (type in resourceTypes) {
            if (type.name.equals(resourceType, ignoreCase = true)) {
                return type.targetClass
            }
        }
        return ""
    }

    private fun extractResourceName(uri: String): String {
        val parts = uri.split("/", "#")
        return parts.lastOrNull()?.lowercase() ?: "resource"
    }

    private fun buildConceptualTypeCandidates(resourceType: String, targetClassUri: String): List<String> {
        val candidates = linkedSetOf<String>()
        val trimmedType = resourceType.trim()
        if (trimmedType.isNotBlank()) {
            candidates += trimmedType.lowercase()
        }

        if (targetClassUri.isNotBlank()) {
            candidates += targetClassUri
            val short = targetClassUri.substringAfterLast('/').substringAfterLast('#').trim().lowercase()
            if (short.isNotBlank()) {
                candidates += short
            }
        }

        return candidates.toList()
    }

    private fun sanitizeKey(key: String): String {
        val reserved = setOf("id", "label", "key", "value", "property")
        return if (key in reserved) "pg_$key" else key
    }

    private fun propString(value: Any?): String? {
        val normalized = normalizeValue(value) ?: return null
        return when (normalized) {
            is String -> normalized
            is List<*> -> normalized.firstOrNull()?.toString()
            else -> normalized.toString()
        }
    }

    private fun normalizeValue(value: Any?): Any? {
        return when (value) {
            null -> null
            is String -> {
                val trimmed = value.trim()
                if ((trimmed.startsWith("[") && trimmed.endsWith("]")) || (trimmed.startsWith("{") && trimmed.endsWith("}"))) {
                    runCatching { mapper.readValue(trimmed, Any::class.java) }
                        .getOrNull()
                        ?.let { parsed -> normalizeValue(parsed) }
                        ?: value
                } else {
                    value
                }
            }
            is List<*> -> {
                if (value.isEmpty()) return emptyList<Any?>()
                if (value.size == 1 && (value.first() is List<*> || value.first() is Map<*, *>)) {
                    normalizeValue(value.first())
                } else {
                    value.map { normalizeValue(it) }
                }
            }
            is Map<*, *> -> value.entries.associate { (k, v) -> (k?.toString() ?: "") to normalizeValue(v) }
            else -> value
        }
    }

    private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

    private fun gVal(value: Any): String = when (value) {
        is String -> gStr(value)
        is Number, is Boolean -> value.toString()
        is List<*>, is Map<*, *> -> gStr(mapper.writeValueAsString(value))
        else -> gStr(value.toString())
    }

    private fun normalizePropertyMap(raw: Map<*, *>): Map<String, Any?> {
        return raw.entries.associate { (key, value) ->
            val normalizedKey =
                when (key) {
                    is T ->
                        when (key) {
                            T.id -> "_vertexId"
                            T.label -> "_vertexLabel"
                            else -> key.name
                        }
                    else -> key?.toString() ?: ""
                }
            normalizedKey to value
        }
    }
}
