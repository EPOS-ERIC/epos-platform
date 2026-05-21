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
                            g.V().hasLabel('Version').has('id', ${gStr(sourceId)})
                              .outE(${gStr(edgeLabel)}).inV().has('id', ${gStr(resolvedTargetId)}).limit(1).toList()
                            """.trimIndent(),
                        )

                    if (!hasEdge) {
                        missingEdgesDiscovered++
                        if (!dryRun) {
                            client.runWrite(
                                """
                                def s = g.V().hasLabel('Version').has('id', ${gStr(sourceId)}).limit(1).next()
                                def t = g.V().hasLabel('Version').has('id', ${gStr(resolvedTargetId)}).limit(1).next()
                                g.V(s.id()).addE(${gStr(edgeLabel)}).to(t)
                                  .property('predicateKey', ${gStr(propertyName)})
                                  .iterate()
                                """.trimIndent(),
                            )
                            edgesCreated++
                        }
                    }
                }
            }
        }

        val legacyEdgesScanned =
            client.runQuery(
                """
                g.E().filter{ !['HAS_VERSION','CURRENT_VERSION','REVISION_OF','SUPERSEDED_BY','AFFECTED'].contains(it.get().label()) }
                  .count().toList()
                """.trimIndent(),
            ).firstOrNull()?.get("value")?.toString()?.toIntOrNull() ?: 0

        val legacyEdgesBackfilledCandidates =
            client.runQuery(
                """
                g.E().hasLabel(org.apache.tinkerpop.gremlin.process.traversal.TextP.startingWith('rel_'))
                  .filter{ !it.get().property('predicateKey').isPresent() || !it.get().property('predicateIri').isPresent() }
                  .count().toList()
                """.trimIndent(),
            ).firstOrNull()?.get("value")?.toString()?.toIntOrNull() ?: 0

        val legacyEdgesBackfilled =
            if (dryRun) {
                legacyEdgesBackfilledCandidates
            } else {
                client.runWriteReturning(
                    """
                    def updated = 0
                    g.E().hasLabel(org.apache.tinkerpop.gremlin.process.traversal.TextP.startingWith('rel_')).toList().each { e ->
                      def changed = false
                      if (!e.property('predicateKey').isPresent()) {
                        e.property('predicateKey', e.label().substring(4))
                        changed = true
                      }
                      if (changed) {
                        updated = updated + 1
                      }
                    }
                    updated
                    """.trimIndent(),
                ).firstOrNull()?.get("value")?.toString()?.toIntOrNull() ?: 0
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
        val normalized = cleaned.replace(Regex("[^A-Za-z0-9_\\-]"), "_")
        return "rel_$normalized"
    }

    private fun resolveTargetVersionId(targetId: String): String? {
        val exact =
            client.runQuery(
                "g.V().hasLabel('Version').has('id', ${gStr(targetId)}).values('id').limit(1).toList()",
            ).firstOrNull()?.get("value")?.toString()
        if (!exact.isNullOrBlank()) return exact

        return client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(targetId)})
              .out('CURRENT_VERSION').hasLabel('Version')
              .values('id').limit(1).toList()
            """.trimIndent(),
        ).firstOrNull()?.get("value")?.toString()
    }

    private fun fetchAllVersionIds(pageSize: Int = 50, maxVersions: Int? = null): List<String> {
        val ids = mutableListOf<String>()
        var offset = 0
        while (true) {
            val batch = client.runQuery(
                """
                g.V().hasLabel('Version').range($offset, ${offset + pageSize}).values('id').toList()
                """.trimIndent(),
            )
                .mapNotNull { row -> row["value"]?.toString() }

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
        val row = client.runQuery(
            """
            g.V().hasLabel('Version').has('id', ${gStr(versionId)}).valueMap().limit(1).toList()
            """.trimIndent(),
        ).firstOrNull() ?: return null

        return (row["value"] as? Map<*, *>) ?: row as? Map<*, *>
    }

    private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

    companion object {
        private val STRUCTURAL_KEYS = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    }
}
