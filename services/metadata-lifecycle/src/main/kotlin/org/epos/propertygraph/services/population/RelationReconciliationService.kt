package org.epos.propertygraph.services.population

import org.epos.propertygraph.client.PropertyGraphClient

data class RelationReconciliationResult(
    val dryRun: Boolean,
    val versionsScanned: Int,
    val missingEdgesDiscovered: Int,
    val edgesCreated: Int,
    val legacyEdgesScanned: Int,
    val legacyEdgesBackfilled: Int,
)

data class RelationAuditResult(
    val resourceId: String,
    val resolvedVersionId: String,
    val propertyCounts: Map<String, Int>,
    val relationCandidates: Map<String, List<String>>,
    val mixedConceptualAndVersionValues: Map<String, List<String>>,
)

class RelationReconciliationService(
    private val client: PropertyGraphClient,
) {
    fun auditResource(resourceId: String): RelationAuditResult? {
        val versionId = resolveTargetVersionId(resourceId) ?: return null
        val data = fetchVersionData(versionId) ?: return null

        val propertyCounts = linkedMapOf<String, Int>()
        val relationCandidates = linkedMapOf<String, List<String>>()
        val mixed = linkedMapOf<String, List<String>>()

        data.forEach { (key, rawValue) ->
            val propertyName = key?.toString() ?: return@forEach
            if (propertyName in STRUCTURAL_KEYS) return@forEach
            val values = toStringValues(rawValue)
            propertyCounts[propertyName] = values.size

            val resources = values.filter { looksLikeResourceId(it) }
            if (resources.isNotEmpty()) {
                relationCandidates[propertyName] = resources
                val normalized = resources.map { it.substringBeforeLast("/v", it) }
                if (normalized.toSet().size < resources.size) {
                    mixed[propertyName] = resources
                }
            }
        }

        return RelationAuditResult(
            resourceId = resourceId,
            resolvedVersionId = versionId,
            propertyCounts = propertyCounts,
            relationCandidates = relationCandidates,
            mixedConceptualAndVersionValues = mixed,
        )
    }

    fun reconcile(
        dryRun: Boolean = true,
        resourceId: String? = null,
        maxVersions: Int? = null,
        batchSize: Int? = null,
    ): RelationReconciliationResult {
        val versionIds =
            if (!resourceId.isNullOrBlank()) {
                listOfNotNull(resolveTargetVersionId(resourceId))
            } else {
                fetchAllVersionIds(
                    pageSize = (batchSize ?: 50).coerceIn(10, 200),
                    maxVersions = maxVersions?.coerceAtLeast(1),
                )
            }
        var missingEdgesDiscovered = 0
        var edgesCreated = 0

        versionIds.forEach { sourceId ->
            val data = fetchVersionData(sourceId) ?: return@forEach

            data.forEach { (key, rawValue) ->
                val propertyName = key?.toString() ?: return@forEach
                if (propertyName in STRUCTURAL_KEYS) return@forEach

                val values = toStringValues(rawValue)
                values.filter { looksLikeResourceId(it) }.forEach { targetId ->
                    val resolvedTargetId = resolveTargetVersionId(targetId) ?: return@forEach

                    val edgeLabel = sanitizeEdgeLabel(propertyName)
                    val hasEdge =
                        client.exists(
                            """
                            MATCH (s:Version {id: ${'$'}sourceId})-[e:$edgeLabel]->(t:Version {id: ${'$'}targetId})
                            RETURN t.id AS value LIMIT 1
                            """.trimIndent(),
                            mapOf("sourceId" to sourceId, "targetId" to resolvedTargetId),
                        )

                    if (!hasEdge) {
                        missingEdgesDiscovered++
                        if (!dryRun) {
                            client.runWrite(
                                """
                                MATCH (s:Version {id: ${'$'}sourceId}), (t:Version {id: ${'$'}targetId})
                                MERGE (s)-[r:$edgeLabel]->(t)
                                SET r.predicateKey = ${'$'}predicateKey
                                """.trimIndent(),
                                mapOf("sourceId" to sourceId, "targetId" to resolvedTargetId, "predicateKey" to propertyName),
                            )
                            edgesCreated++
                        }
                    }
                }
            }
        }

        val legacyEdgesScanned =
            client
                .runQuery(
                    """
                    MATCH ()-[e]->()
                    WHERE NOT type(e) IN ['HAS_VERSION','CURRENT_VERSION','REVISION_OF','SUPERSEDED_BY','AFFECTED']
                    RETURN count(e) AS value
                    """.trimIndent(),
                ).firstOrNull()
                ?.get("value")
                ?.toString()
                ?.toIntOrNull() ?: 0

        val legacyEdgesBackfilledCandidates =
            client
                .runQuery(
                    """
                    MATCH ()-[e]->()
                    WHERE startsWith(type(e), 'rel_')
                      AND (e.predicateKey IS NULL OR e.predicateIri IS NULL)
                    RETURN count(e) AS value
                    """.trimIndent(),
                ).firstOrNull()
                ?.get("value")
                ?.toString()
                ?.toIntOrNull() ?: 0

        val legacyEdgesBackfilled =
            if (dryRun) {
                legacyEdgesBackfilledCandidates
            } else {
                client
                    .runWriteReturning(
                        """
                        MATCH ()-[e]->()
                        WHERE startsWith(type(e), 'rel_') AND e.predicateKey IS NULL
                        SET e.predicateKey = replace(type(e), 'rel_', '')
                        RETURN count(e) AS value
                        """.trimIndent(),
                    ).firstOrNull()
                    ?.get("value")
                    ?.toString()
                    ?.toIntOrNull() ?: 0
            }

        return RelationReconciliationResult(
            dryRun = dryRun,
            versionsScanned = versionIds.size,
            missingEdgesDiscovered = missingEdgesDiscovered,
            edgesCreated = edgesCreated,
            legacyEdgesScanned = legacyEdgesScanned,
            legacyEdgesBackfilled = legacyEdgesBackfilled,
        )
    }

    private fun toStringValues(value: Any?): List<String> =
        when (value) {
            is List<*> -> value.mapNotNull { item -> item?.toString() }
            null -> emptyList()
            else -> listOf(value.toString())
        }

    private fun looksLikeResourceId(value: String): Boolean =
        value.startsWith("http://") || value.startsWith("https://") || value.startsWith("category:")

    private fun sanitizeEdgeLabel(label: String): String {
        val cleaned = label.trim().ifBlank { "relatedTo" }
        val normalized = cleaned.replace(Regex("[^A-Za-z0-9_]"), "_")
        return "rel_$normalized"
    }

    private fun resolveTargetVersionId(targetId: String): String? {
        val exact =
            client
                .runQuery(
                    "MATCH (v:Version {id: ${'$'}targetId}) RETURN v.id AS value LIMIT 1",
                    mapOf("targetId" to targetId),
                ).firstOrNull()
                ?.get("value")
                ?.toString()
        if (!exact.isNullOrBlank()) return exact

        return client
            .runQuery(
                """
                MATCH (:Conceptual {id: ${'$'}targetId})-[:CURRENT_VERSION]->(v:Version)
                RETURN v.id AS value LIMIT 1
                """.trimIndent(),
                mapOf("targetId" to targetId),
            ).firstOrNull()
            ?.get("value")
            ?.toString()
    }

    private fun fetchAllVersionIds(
        pageSize: Int = 50,
        maxVersions: Int? = null,
    ): List<String> {
        val ids = mutableListOf<String>()
        var offset = 0
        while (true) {
            val batch =
                client
                    .runQuery(
                        """
                        MATCH (v:Version)
                        RETURN v.id AS value
                        SKIP ${'$'}offset LIMIT ${'$'}limit
                        """.trimIndent(),
                        mapOf("offset" to offset, "limit" to pageSize),
                    ).mapNotNull { row -> row["value"]?.toString() }

            if (batch.isEmpty()) break
            ids.addAll(batch)
            if (maxVersions != null && ids.size >= maxVersions) {
                return ids.take(maxVersions)
            }
            offset += pageSize
            if (batch.size < pageSize) break
        }
        return ids
    }

    private fun fetchVersionData(versionId: String): Map<*, *>? {
        val row =
            client
                .runQuery(
                    """
                    MATCH (v:Version {id: ${'$'}versionId}) RETURN properties(v) AS value LIMIT 1
                    """.trimIndent(),
                    mapOf("versionId" to versionId),
                ).firstOrNull() ?: return null

        return (row["value"] as? Map<*, *>) ?: row as? Map<*, *>
    }

    companion object {
        private val STRUCTURAL_KEYS = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    }
}
