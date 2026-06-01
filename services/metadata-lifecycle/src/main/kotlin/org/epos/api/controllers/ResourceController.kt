package org.epos.api.controllers

import org.epos.api.models.*
import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.EntityLifecycleService
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.utils.UriUtils
import com.fasterxml.jackson.databind.ObjectMapper
import java.util.UUID

@Suppress("UNCHECKED_CAST")
class ResourceController(
    private val client: PropertyGraphClient,
    private val lifecycleService: EntityLifecycleService,
    private val provenanceService: ProvenanceService
) {
    private data class VersionState(
        val id: String,
        val status: String,
    )

    private sealed class DraftTargetResolution {
        data class Found(val versionUri: String) : DraftTargetResolution()
        data class Conflict(val status: String, val versionUri: String) : DraftTargetResolution()
        data object NotFound : DraftTargetResolution()
    }

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
        pageSize: Int = 20,
        agent: String? = null,
    ): EntityListResponse {
        val typeFilter = getTypeUri(resourceType)
        val typeCandidates = buildConceptualTypeCandidates(resourceType, typeFilter)
        val statusPredicate = statusWithinPredicate(status)

        val skip = (page - 1) * pageSize
        val statuses = statusCandidates(status)
        val params = mutableMapOf<String, Any>(
            "statuses" to statuses,
            "skip" to skip,
            "limit" to pageSize,
            "typeCandidates" to typeCandidates,
            "requireDraftAgent" to (status.equals("draft", ignoreCase = true) && !agent.isNullOrBlank()),
            "draftAgent" to (agent ?: ""),
        )

        val results = runCatching {
            client.runQuery(
                """
                MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version)
                WHERE toLower(v.status) IN ${'$'}statuses
                  AND (size(${ '$' }typeCandidates) = 0 OR toLower(c.type) IN ${'$'}typeCandidates)
                  AND (
                    ${'$'}requireDraftAgent = false OR EXISTS {
                      MATCH (a:Activity)-[:AFFECTED]->(v)
                      WHERE a.type = 'draft_creation' AND a.agent = ${'$'}draftAgent
                    }
                  )
                RETURN properties(v) AS v, properties(c) AS c
                SKIP ${'$'}skip LIMIT ${'$'}limit
                """.trimIndent(),
                params,
            )
        }.getOrDefault(emptyList())

        val entities = results.mapNotNull { row ->
            val v = normalizePropertyMap(row["v"] as? Map<*, *> ?: return@mapNotNull null)
            val c = normalizePropertyMap(row["c"] as? Map<*, *> ?: return@mapNotNull null)

            val versionUri = propString(v["identifier"])?.takeIf { it.isNotBlank() }
                ?: propString(v["id"])
                ?: return@mapNotNull null
            val conceptualUri = propString(c["id"])
            val resolvedStatus = propString(v["status"]) ?: status

            val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt", "_vertexId", "_vertexLabel", "label")
            val dataMap = v.filterKeys { it !in structuralKeys }
                .mapValues { (_, v2) -> normalizeValue(v2) ?: "" }
            val typeUri = resolveEntityTypeUri(versionProps = dataMap, conceptualProps = c)
            val resolvedType = extractResourceName(typeUri)

            EntityResponse(
                uri = versionUri,
                type = resolvedType,
                status = resolvedStatus,
                properties = dataMap,
                conceptualUri = conceptualUri
            )
        }

        val total = runCatching {
            client.runQuery(
                """
                MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version)
                WHERE toLower(v.status) IN ${'$'}statuses
                  AND (size(${ '$' }typeCandidates) = 0 OR toLower(c.type) IN ${'$'}typeCandidates)
                  AND (
                    ${'$'}requireDraftAgent = false OR EXISTS {
                      MATCH (a:Activity)-[:AFFECTED]->(v)
                      WHERE a.type = 'draft_creation' AND a.agent = ${'$'}draftAgent
                    }
                  )
                RETURN count(v) AS value
                """.trimIndent(),
                params,
            )
        }
            .getOrDefault(emptyList())
            .firstOrNull()
            ?.get("value")
            ?.toString()
            ?.toIntOrNull() ?: 0

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
        val results = runCatching {
            client.runQuery(
                "MATCH (v:Version {id: ${'$'}id}) RETURN properties(v) AS value LIMIT 1",
                mapOf("id" to uri),
            )
        }.getOrDefault(emptyList())

        val v = normalizePropertyMap(results.firstOrNull()?.get("value") as? Map<*, *> ?: return null)
        val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
        val data = v.filterKeys { it !in structuralKeys }
        val typeValue = resolveEntityTypeUri(versionProps = data, conceptualProps = null).ifBlank { return null }

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
        val result = runCatching {
            client.runQuery(
                "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) IN ${'$'}statuses RETURN properties(v) AS value LIMIT 1",
                mapOf("id" to entityUri, "statuses" to statusCandidates(status)),
            ).firstOrNull()
                ?: client.runQuery(
                    "MATCH (v:Version) WHERE v.identifier = ${'$'}id AND toLower(v.status) IN ${'$'}statuses RETURN properties(v) AS value LIMIT 1",
                    mapOf("id" to entityUri, "statuses" to statusCandidates(status)),
                ).firstOrNull()
        }.getOrNull() ?: return null

        val v = normalizePropertyMap(result["value"] as? Map<*, *> ?: return null)
        val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt", "_vertexId", "_vertexLabel", "label")
        val data = v.filterKeys { it !in structuralKeys }
            .mapValues { (_, v2) -> normalizeValue(v2) ?: "" }

        val enrichedProperties = if (enrichReferences) {
            data.mapValues { (_, value) -> enrichValue(value) }
        } else {
            data
        }

        val typeUri = resolveEntityTypeUri(versionProps = enrichedProperties, conceptualProps = null)
        val resourceType = extractResourceName(typeUri)

        val conceptualUri = runCatching {
            client.runQuery(
                "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {id: ${'$'}id}) RETURN c.id AS value LIMIT 1",
                mapOf("id" to entityUri),
            ).firstOrNull()?.get("value")?.toString()
                ?: client.runQuery(
                    "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version) WHERE v.identifier = ${'$'}id RETURN c.id AS value LIMIT 1",
                    mapOf("id" to entityUri),
                ).firstOrNull()?.get("value")?.toString()
        }.getOrNull()

        val resolvedUri = propString(v["identifier"])?.takeIf { it.isNotBlank() }
            ?: propString(v["id"])
            ?: entityUri
        val resolvedStatus = propString(v["status"]) ?: status

        return EntityResponse(
            uri = resolvedUri,
            type = resourceType,
            status = resolvedStatus,
            properties = enrichedProperties,
            conceptualUri = conceptualUri
        )
    }

    fun getEntityByConceptualId(conceptualId: String, status: String = "published", enrichReferences: Boolean = true): EntityResponse? {
        val statuses = statusCandidates(status)
        val result =
            client.runQuery(
                """
                MATCH (c:Conceptual {id: ${'$'}conceptualId})-[:HAS_VERSION]->(v:Version)
                WHERE toLower(v.status) IN ${'$'}statuses
                RETURN coalesce(v.identifier, v.id) AS uri, properties(v) AS data
                ORDER BY coalesce(v.updatedAt, v.createdAt, '') DESC
                LIMIT 1
                """.trimIndent(),
                mapOf("conceptualId" to conceptualId, "statuses" to statuses),
            ).firstOrNull()
                ?: client.runQuery(
                    """
                    MATCH (c:Conceptual {identifier: ${'$'}conceptualId})-[:HAS_VERSION]->(v:Version)
                    WHERE toLower(v.status) IN ${'$'}statuses
                    RETURN coalesce(v.identifier, v.id) AS uri, properties(v) AS data
                    ORDER BY coalesce(v.updatedAt, v.createdAt, '') DESC
                    LIMIT 1
                    """.trimIndent(),
                    mapOf("conceptualId" to conceptualId, "statuses" to statuses),
                ).firstOrNull()
                ?: return null

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
        val conceptualUri = resolveConceptualUriForCreate(request)
        val draftUri = "$conceptualUri/1.0-draft"

        val typeUri = getTypeUri(request.type)
        val typeShortName = typeUri.substringAfterLast("/").substringAfterLast("#").lowercase()

        val data = request.properties.toMutableMap()
        data["type"] = typeUri

        val now = java.time.Instant.now().toString()
        client.runWrite(
            """
            MERGE (c:Conceptual {id: ${'$'}conceptualUri})
            ON CREATE SET c.type = ${'$'}typeShortName, c.createdAt = ${'$'}now
            MERGE (v:Version {id: ${'$'}draftUri})
            ON CREATE SET v.status = 'draft', v.versionNumber = '1.0', v.createdAt = ${'$'}now
            SET v.updatedAt = ${'$'}now
            MERGE (c)-[:HAS_VERSION]->(v)
            """.trimIndent(),
            mapOf(
                "conceptualUri" to conceptualUri,
                "typeShortName" to typeShortName,
                "draftUri" to draftUri,
                "now" to now,
            ),
        )

        data.forEach { (k, v) ->
            val key = sanitizeCypherKey(sanitizeKey(k))
            client.runWrite(
                "MATCH (v:Version {id: ${'$'}id}) SET v.`$key` = ${'$'}value",
                mapOf("id" to draftUri, "value" to toPropertyValue(v)),
            )
        }

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

    private fun resolveConceptualUriForCreate(request: CreateEntityRequest): String {
        val provided = request.id?.trim().orEmpty()
        if (provided.isBlank()) {
            return "${Config.ID_BASE}/${request.type}/${UUID.randomUUID()}"
        }
        if (provided.startsWith("http://") || provided.startsWith("https://")) {
            return provided.removeSuffix("/1.0-draft").removeSuffix("/1.0")
        }
        return "${Config.ID_BASE}/${request.type}/$provided"
    }

    fun updateEntity(entityUri: String, request: UpdateEntityRequest): ApiResponse {
        val resolved = resolveDraftTarget(entityUri)
        val draftUri = when (resolved) {
            is DraftTargetResolution.Found -> resolved.versionUri
            is DraftTargetResolution.Conflict -> {
                return ApiResponse(
                    success = false,
                    message = "Entity is in '${resolved.status}' status at ${resolved.versionUri}. Create/use a draft before updating.",
                )
            }
            DraftTargetResolution.NotFound -> {
                return ApiResponse(success = false, message = "Entity not found")
            }
        }

        val data = request.properties
        val now = java.time.Instant.now().toString()
        data.forEach { (k, v) ->
            val key = sanitizeCypherKey(sanitizeKey(k))
            client.runWrite(
                "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) = 'draft' SET v.`$key` = ${'$'}value",
                mapOf("id" to draftUri, "value" to toPropertyValue(v)),
            )
        }
        client.runWrite(
            "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) = 'draft' SET v.updatedAt = ${'$'}now",
            mapOf("id" to draftUri, "now" to now),
        )

        return ApiResponse(
            success = true,
            message = "Entity updated successfully",
            data = mapOf("uri" to draftUri)
        )
    }

    fun patchEntity(entityUri: String, request: UpdateEntityRequest): ApiResponse {
        val resolved = resolveDraftTarget(entityUri)
        val draftUri = when (resolved) {
            is DraftTargetResolution.Found -> resolved.versionUri
            is DraftTargetResolution.Conflict -> {
                return ApiResponse(
                    success = false,
                    message = "Entity is in '${resolved.status}' status at ${resolved.versionUri}. Create/use a draft before patching.",
                )
            }
            DraftTargetResolution.NotFound -> {
                return ApiResponse(success = false, message = "Entity not found")
            }
        }

        val now = java.time.Instant.now().toString()
        request.properties.forEach { (k, v) ->
            val key = sanitizeCypherKey(sanitizeKey(k))
            client.runWrite(
                "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) = 'draft' SET v.`$key` = ${'$'}value",
                mapOf("id" to draftUri, "value" to toPropertyValue(v)),
            )
        }
        client.runWrite(
            "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) = 'draft' SET v.updatedAt = ${'$'}now",
            mapOf("id" to draftUri, "now" to now),
        )

        return ApiResponse(
            success = true,
            message = "Entity patched successfully",
            data = mapOf("uri" to draftUri)
        )
    }

    fun submitEntity(entityUri: String, agent: String, group: String): ApiResponse {
        val resolved = resolveDraftTarget(entityUri)
        val draftUri = when (resolved) {
            is DraftTargetResolution.Found -> resolved.versionUri
            is DraftTargetResolution.Conflict -> {
                return ApiResponse(
                    success = false,
                    message = "Entity is in '${resolved.status}' status at ${resolved.versionUri}. Only drafts can be submitted.",
                )
            }
            DraftTargetResolution.NotFound -> {
                return ApiResponse(success = false, message = "Entity not found")
            }
        }

        lifecycleService.submitDraft(draftUri, agent, group)

        return ApiResponse(
            success = true,
            message = "Entity submitted for review",
            data = mapOf("uri" to draftUri, "status" to "submitted")
        )
    }

    fun publishEntity(entityUri: String, agent: String, group: String): ApiResponse {
        val submitted = resolveSubmittedTarget(entityUri)
            ?: return ApiResponse(success = false, message = "Entity not found in submitted status")

        val finalUri = lifecycleService.publish(submitted.id, agent, group)
        return ApiResponse(
            success = true,
            message = "Entity published successfully",
            data = mapOf("uri" to finalUri, "status" to "published")
        )
    }

    fun discardEntity(entityUri: String, agent: String, group: String): ApiResponse {
        val version = resolveVersionTarget(entityUri) ?: resolveCurrentVersionFromConceptual(entityUri)
            ?: return ApiResponse(success = false, message = "Entity not found")

        val status = version.status.lowercase()
        if (status != "draft" && status != "submitted") {
            return ApiResponse(
                success = false,
                message = "Entity is in '${version.status}' status at ${version.id}. Only draft or submitted versions can be discarded.",
            )
        }

        lifecycleService.discard(version.id, status, agent, group)
        return ApiResponse(
            success = true,
            message = "Entity discarded successfully",
            data = mapOf("uri" to version.id, "status" to "discarded"),
        )
    }

    fun createNewDraft(conceptualUri: String, agent: String, group: String): ApiResponse {
        try {
            val resolvedConceptual = resolveConceptualTarget(conceptualUri)
                ?: return ApiResponse(success = false, message = "Conceptual resource not found")

            val draftUri = lifecycleService.createDraft(
                conceptualUri = resolvedConceptual,
                agent = agent,
                group = group
            )

            val createdVersion = draftUri.substringAfterLast('/').removeSuffix("-draft")

            return ApiResponse(
                success = true,
                message = "New draft created successfully",
                data = mapOf(
                    "uri" to draftUri,
                    "conceptualUri" to resolvedConceptual,
                    "version" to createdVersion,
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
        val resolved = resolveDraftTarget(entityUri)
        val draftUri = when (resolved) {
            is DraftTargetResolution.Found -> resolved.versionUri
            is DraftTargetResolution.Conflict -> {
                return ApiResponse(
                    success = false,
                    message = "Entity is in '${resolved.status}' status at ${resolved.versionUri}. Only drafts can be deleted.",
                )
            }
            DraftTargetResolution.NotFound -> {
                return ApiResponse(success = false, message = "Entity not found")
            }
        }

        lifecycleService.deleteDraft(draftUri)

        return ApiResponse(
            success = true,
            message = "Entity deleted successfully",
            data = mapOf("uri" to draftUri),
        )
    }

    private fun resolveDraftTarget(reference: String): DraftTargetResolution {
        val version = resolveVersionTarget(reference) ?: resolveCurrentVersionFromConceptual(reference)
        if (version == null) return DraftTargetResolution.NotFound
        return if (version.status.lowercase() == "draft") {
            DraftTargetResolution.Found(version.id)
        } else {
            DraftTargetResolution.Conflict(version.status, version.id)
        }
    }

    private fun resolveSubmittedTarget(reference: String): VersionState? {
        val version = resolveVersionTarget(reference) ?: resolveCurrentVersionFromConceptual(reference) ?: return null
        return version.takeIf { it.status.lowercase() == "submitted" }
    }

    private fun resolveConceptualTarget(reference: String): String? {
        val conceptualDirect = client.runQuery(
            "MATCH (c:Conceptual {id: ${'$'}ref}) RETURN c.id AS value LIMIT 1",
            mapOf("ref" to reference),
        ).firstOrNull()?.get("value")?.toString()
        if (!conceptualDirect.isNullOrBlank()) return conceptualDirect

        val conceptualByIdentifier = client.runQuery(
            "MATCH (c:Conceptual {identifier: ${'$'}ref}) RETURN c.id AS value LIMIT 1",
            mapOf("ref" to reference),
        ).firstOrNull()?.get("value")?.toString()
        if (!conceptualByIdentifier.isNullOrBlank()) return conceptualByIdentifier

        return client.runQuery(
            "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {id: ${'$'}ref}) RETURN c.id AS value LIMIT 1",
            mapOf("ref" to reference),
        ).firstOrNull()?.get("value")?.toString()
            ?: client.runQuery(
                "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {identifier: ${'$'}ref}) RETURN c.id AS value LIMIT 1",
                mapOf("ref" to reference),
            ).firstOrNull()?.get("value")?.toString()
    }

    private fun resolveVersionTarget(reference: String): VersionState? {
        val row =
            client.runQuery(
                """
                MATCH (v:Version {id: ${'$'}ref})
                RETURN v.id AS id, v.status AS status
                LIMIT 1
                """.trimIndent(),
                mapOf("ref" to reference),
            ).firstOrNull()
                ?: resolveVersionByIdentifierWithinConceptual(reference)
                ?: return null

        val id = row["id"]?.toString()?.trim().orEmpty()
        val status = row["status"]?.toString()?.trim().orEmpty()
        if (id.isBlank() || status.isBlank()) return null
        return VersionState(id = id, status = status)
    }

    private fun resolveVersionByIdentifierWithinConceptual(reference: String): Map<String, Any?>? {
        val conceptualCandidate = if (UriUtils.hasVersion(reference)) UriUtils.extractConceptualUri(reference) else return null

        return client.runQuery(
            """
            MATCH (c:Conceptual {id: ${'$'}conceptualCandidate})-[:HAS_VERSION]->(v:Version {identifier: ${'$'}reference})
            RETURN v.id AS id, v.status AS status
            LIMIT 1
            """.trimIndent(),
            mapOf("conceptualCandidate" to conceptualCandidate, "reference" to reference),
        ).firstOrNull()
    }

    private fun resolveCurrentVersionFromConceptual(reference: String): VersionState? {
        val conceptualByStructure = if (UriUtils.hasVersion(reference)) UriUtils.extractConceptualUri(reference) else null

        val row =
            client.runQuery(
                """
                MATCH (c:Conceptual {id: ${'$'}ref})-[:CURRENT_VERSION]->(v:Version)
                RETURN v.id AS id, v.status AS status
                LIMIT 1
                """.trimIndent(),
                mapOf("ref" to (conceptualByStructure ?: reference)),
            ).firstOrNull()
                ?: if (conceptualByStructure != null) {
                    null
                } else {
                    client.runQuery(
                        """
                        MATCH (c:Conceptual {identifier: ${'$'}ref})-[:CURRENT_VERSION]->(v:Version)
                        RETURN v.id AS id, v.status AS status
                        LIMIT 1
                        """.trimIndent(),
                        mapOf("ref" to reference),
                    ).firstOrNull()
                }
                ?: return null

        val id = row["id"]?.toString()?.trim().orEmpty()
        val status = row["status"]?.toString()?.trim().orEmpty()
        if (id.isBlank() || status.isBlank()) return null
        return VersionState(id = id, status = status)
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
        val result = client.runQuery(
            "MATCH (v:Version {id: ${'$'}id}) RETURN v.status AS value LIMIT 1",
            mapOf("id" to versionUri),
        )
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
                MATCH (v:Version {id: ${'$'}id})
                RETURN properties(v) AS value
                LIMIT 1
            """.trimIndent(), mapOf("id" to uri))

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

    private fun statusWithinPredicate(status: String): String {
        val normalized = status.trim().ifBlank { "published" }
        val candidates = linkedSetOf(normalized, normalized.lowercase(), normalized.uppercase())
        return "within(${candidates.joinToString(",") { gStr(it) }})"
    }

    private fun statusCandidates(status: String): List<String> {
        val normalized = status.trim().ifBlank { "published" }.lowercase()
        return listOf(normalized)
    }

    private fun sanitizeCypherKey(key: String): String = key.replace("`", "")

    private fun toPropertyValue(value: Any?): Any =
        when (value) {
            null -> ""
            is String, is Number, is Boolean -> value
            is List<*>, is Map<*, *> -> mapper.writeValueAsString(value)
            else -> value.toString()
        }

    private fun normalizePropertyMap(raw: Map<*, *>): Map<String, Any?> {
        return raw.entries.associate { (key, value) -> (key?.toString() ?: "") to value }
    }

    private fun resolveEntityTypeUri(
        versionProps: Map<String, Any?>,
        conceptualProps: Map<String, Any?>?,
    ): String {
        return propString(versionProps["type"])?.takeIf { it.isNotBlank() }
            ?: propString(versionProps["rdf:type"])?.takeIf { it.isNotBlank() }
            ?: conceptualProps?.let { propString(it["type"])?.takeIf { value -> value.isNotBlank() } }
            ?: ""
    }
}
