package org.epos.propertygraph.services

import com.fasterxml.jackson.databind.ObjectMapper
import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils
import org.epos.propertygraph.utils.VersionRecord
import org.slf4j.LoggerFactory

class EntityLifecycleService(
    private val client: PropertyGraphClient,
    private val provenanceService: ProvenanceService,
) {
    private val mapper = ObjectMapper()
    private val logger = LoggerFactory.getLogger(javaClass)

    fun getCurrentPublishedVersion(conceptualUri: String): VersionRecord? {
        val current =
            client
                .runQuery(
                    """
                    MATCH (:Conceptual {id: ${'$'}conceptualUri})-[:CURRENT_VERSION]->(v:Version)
                    WHERE toLower(v.status) = 'published'
                    RETURN properties(v) AS value
                    LIMIT 1
                    """.trimIndent(),
                    mapOf("conceptualUri" to conceptualUri),
                ).firstOrNull()

        if (current != null) {
            val currentMap = (current["value"] as? Map<*, *>) ?: (current as? Map<*, *>) ?: return null
            return toVersionRecord(currentMap)
        }

        val fallback =
            client
                .runQuery(
                    """
                    MATCH (:Conceptual {id: ${'$'}conceptualUri})-[:HAS_VERSION]->(v:Version)
                    WHERE toLower(v.status) = 'published'
                    RETURN properties(v) AS value
                    ORDER BY coalesce(v.updatedAt, v.createdAt, '') DESC
                    LIMIT 1
                    """.trimIndent(),
                    mapOf("conceptualUri" to conceptualUri),
                ).firstOrNull() ?: return null

        val fallbackMap = (fallback["value"] as? Map<*, *>) ?: (fallback as? Map<*, *>) ?: return null
        return toVersionRecord(fallbackMap)
    }

    fun createDraft(
        conceptualUri: String,
        newVersionNumber: String? = null,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ): String {
        val currentVersion =
            getCurrentPublishedVersion(conceptualUri)
                ?: throw IllegalStateException("No published version found for $conceptualUri")

        val resolvedVersionNumber =
            newVersionNumber?.trim().takeUnless { it.isNullOrBlank() }
                ?: nextVersionFrom(currentVersion.id)

        val newDraftUri = "$conceptualUri/$resolvedVersionNumber-draft"

        val draftData = currentVersion.data.toMutableMap()
        draftData["versionNumber"] = resolvedVersionNumber

        val timestamp =
            java.time.Instant
                .now()
                .toString()

        client.runWrite(
            """
            MATCH (c:Conceptual {id: ${'$'}conceptualUri})-[:CURRENT_VERSION]->(published:Version)
            MERGE (draft:Version {id: ${'$'}newDraftUri})
            ON CREATE SET draft.status = 'draft', draft.versionNumber = ${'$'}resolvedVersionNumber, draft.createdAt = ${'$'}timestamp
            SET draft.updatedAt = ${'$'}timestamp
            MERGE (c)-[:HAS_VERSION]->(draft)
            MERGE (draft)-[:REVISION_OF]->(published)
            """.trimIndent(),
            mapOf(
                "conceptualUri" to conceptualUri,
                "newDraftUri" to newDraftUri,
                "resolvedVersionNumber" to resolvedVersionNumber,
                "timestamp" to timestamp,
            ),
        )

        draftData.entries.forEach { (k, v) ->
            val key = sanitizeCypherKey(sanitizeKey(k))
            client.runWrite(
                "MATCH (draft:Version {id: ${'$'}id}) SET draft.`$key` = ${'$'}value",
                mapOf("id" to newDraftUri, "value" to toPropertyValue(v)),
            )
        }

        provenanceService.recordDraftCreation(
            newVersionUri = newDraftUri,
            basedOnVersionUri = currentVersion.id,
            conceptualUri = conceptualUri,
            agent = agent,
            group = group,
        )

        logger.info("Created DRAFT: {} (based on {})", newDraftUri, currentVersion.id)
        return newDraftUri
    }

    private fun nextVersionFrom(currentVersionUri: String): String {
        val segment = currentVersionUri.substringAfterLast("/").removeSuffix("-draft")
        val match = Regex("v?(\\d+)(?:\\.(\\d+))?.*").find(segment)
        if (match == null) return "2.0"
        val major = match.groupValues[1].toIntOrNull() ?: 1
        return "${major + 1}.0"
    }

    fun submitDraft(
        draftUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ) {
        requireVersionExists(draftUri, "draft")

        val now =
            java.time.Instant
                .now()
                .toString()
        client.runWrite(
            """
            MATCH (v:Version {id: ${'$'}draftUri})
            WHERE toLower(v.status) = 'draft'
            SET v.status = 'submitted', v.updatedAt = ${'$'}now
            """.trimIndent(),
            mapOf("draftUri" to draftUri, "now" to now),
        )

        provenanceService.recordStateTransition(draftUri, "DRAFT", "SUBMITTED", agent, group)
        logger.info("Submitted: {} (DRAFT -> SUBMITTED)", draftUri)
    }

    fun publish(
        submittedUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ): String {
        requireVersionExists(submittedUri, "submitted")

        val conceptualUri =
            getConceptualUri(submittedUri)
                ?: throw IllegalStateException("Cannot find conceptual URI for $submittedUri")

        val currentPublished = getCurrentPublishedVersion(conceptualUri)
        val now =
            java.time.Instant
                .now()
                .toString()

        val finalUri = submittedUri.removeSuffix("-draft")
        // Publish submitted version first, then reconcile previous published versions and pointers.
        markSubmittedAsPublished(submittedUri, finalUri, now)

        val archivedRows = findArchivedPublishedRows(conceptualUri, finalUri)
        archivePreviousPublishedVersions(conceptualUri, finalUri, now, archivedRows, agent, group)

        updateCurrentVersionPointer(conceptualUri, finalUri)
        linkSupersededVersion(currentPublished?.id, finalUri)

        provenanceService.recordStateTransition(finalUri, "SUBMITTED", "PUBLISHED", agent, group)
        logger.info("Published: {} (SUBMITTED -> PUBLISHED)", finalUri)
        logger.info("Updated conceptual pointer: {} -> {}", conceptualUri, finalUri)
        return finalUri
    }

    private fun markSubmittedAsPublished(
        submittedUri: String,
        finalUri: String,
        now: String,
    ) {
        if (finalUri != submittedUri) {
            client.runWrite(
                """
                MATCH (v:Version {id: ${'$'}submittedUri})
                SET v.id = ${'$'}finalUri, v.status = 'published', v.updatedAt = ${'$'}now
                """.trimIndent(),
                mapOf("submittedUri" to submittedUri, "finalUri" to finalUri, "now" to now),
            )
            return
        }

        client.runWrite(
            """
            MATCH (v:Version {id: ${'$'}submittedUri})
            SET v.status = 'published', v.updatedAt = ${'$'}now
            """.trimIndent(),
            mapOf("submittedUri" to submittedUri, "now" to now),
        )
    }

    private fun findArchivedPublishedRows(
        conceptualUri: String,
        finalUri: String,
    ): List<Map<String, Any?>> =
        client.runWriteReturning(
            """
            MATCH (:Conceptual {id: ${'$'}conceptualUri})-[:HAS_VERSION]->(v:Version)
            WHERE toLower(v.status) = 'published' AND v.id <> ${'$'}finalUri
            RETURN coalesce(v.identifier, v.id) AS id
            """.trimIndent(),
            mapOf("conceptualUri" to conceptualUri, "finalUri" to finalUri),
        )

    private fun archivePreviousPublishedVersions(
        conceptualUri: String,
        finalUri: String,
        now: String,
        archivedRows: List<Map<String, Any?>>,
        agent: String,
        group: String,
    ) {
        if (archivedRows.isEmpty()) return

        client.runWrite(
            """
            MATCH (:Conceptual {id: ${'$'}conceptualUri})-[:HAS_VERSION]->(v:Version)
            WHERE toLower(v.status) = 'published' AND v.id <> ${'$'}finalUri
            SET v.status = 'archived', v.updatedAt = ${'$'}now
            """.trimIndent(),
            mapOf("conceptualUri" to conceptualUri, "finalUri" to finalUri, "now" to now),
        )

        archivedRows.forEach { row ->
            val archivedId = row["id"]?.toString()?.trim().orEmpty()
            if (archivedId.isBlank()) return@forEach
            provenanceService.recordStateTransition(archivedId, "PUBLISHED", "ARCHIVED", agent, group)
            logger.info("Archived old version: {}", archivedId)
        }
    }

    private fun updateCurrentVersionPointer(
        conceptualUri: String,
        finalUri: String,
    ) {
        client.runWrite(
            """
            MATCH (c:Conceptual {id: ${'$'}conceptualUri})
            OPTIONAL MATCH (c)-[old:CURRENT_VERSION]->(:Version)
            DELETE old
            WITH c
            MATCH (v:Version {id: ${'$'}finalUri})
            MERGE (c)-[:CURRENT_VERSION]->(v)
            """.trimIndent(),
            mapOf("conceptualUri" to conceptualUri, "finalUri" to finalUri),
        )
    }

    private fun linkSupersededVersion(
        previousPublishedUri: String?,
        finalUri: String,
    ) {
        if (previousPublishedUri.isNullOrBlank()) return
        client.runWrite(
            """
            MATCH (oldV:Version {id: ${'$'}previousPublishedUri}), (newV:Version {id: ${'$'}finalUri})
            MERGE (oldV)-[:SUPERSEDED_BY]->(newV)
            """.trimIndent(),
            mapOf("previousPublishedUri" to previousPublishedUri, "finalUri" to finalUri),
        )
    }

    fun rejectSubmission(
        submittedUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ) {
        val now =
            java.time.Instant
                .now()
                .toString()
        client.runWrite(
            """
            MATCH (v:Version {id: ${'$'}submittedUri})
            WHERE toLower(v.status) = 'submitted'
            SET v.status = 'draft', v.updatedAt = ${'$'}now
            """.trimIndent(),
            mapOf("submittedUri" to submittedUri, "now" to now),
        )

        provenanceService.recordStateTransition(submittedUri, "SUBMITTED", "DRAFT", agent, group)
        logger.info("Rejected: {} (SUBMITTED -> DRAFT)", submittedUri)
    }

    fun discard(
        versionUri: String,
        fromStatus: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ) {
        val normalizedFrom = fromStatus.lowercase()
        require(normalizedFrom == "draft" || normalizedFrom == "submitted") {
            "Only draft or submitted versions can be discarded"
        }

        val now =
            java.time.Instant
                .now()
                .toString()
        client.runWrite(
            """
            MATCH (v:Version {id: ${'$'}versionUri})
            WHERE toLower(v.status) = ${'$'}normalizedFrom
            SET v.status = 'discarded', v.updatedAt = ${'$'}now
            """.trimIndent(),
            mapOf("versionUri" to versionUri, "normalizedFrom" to normalizedFrom, "now" to now),
        )

        provenanceService.recordStateTransition(versionUri, normalizedFrom.uppercase(), "DISCARDED", agent, group)
        logger.info("Discarded: {} ({} -> DISCARDED)", versionUri, normalizedFrom.uppercase())
    }

    fun deleteDraft(draftUri: String) {
        client.runWrite(
            """
            MATCH (v:Version {id: ${'$'}draftUri})
            DETACH DELETE v
            """.trimIndent(),
            mapOf("draftUri" to draftUri),
        )
        logger.info("Deleted DRAFT: {}", draftUri)
    }

    private fun getConceptualUri(versionUri: String): String? {
        val result =
            client
                .runQuery(
                    "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {id: ${'$'}versionUri}) RETURN c.id AS value LIMIT 1",
                    mapOf("versionUri" to versionUri),
                ).firstOrNull() ?: return null
        return result["value"]?.toString()
    }

    private fun requireVersionExists(
        uri: String,
        expectedStatus: String,
    ) {
        val count =
            client
                .runQuery(
                    "MATCH (v:Version {id: ${'$'}uri}) WHERE toLower(v.status) = ${'$'}status RETURN count(v) AS value",
                    mapOf("uri" to uri, "status" to expectedStatus.lowercase()),
                ).firstOrNull()
                ?.get("value")
                ?.toString()
                ?.toLongOrNull() ?: 0L
        val exists = count > 0L

        if (!exists) {
            throw IllegalArgumentException("Version $uri not found in $expectedStatus status")
        }
    }

    private fun toVersionRecord(map: Map<*, *>): VersionRecord {
        val id = propAsString(map["identifier"]) ?: propAsString(map["id"]) ?: ""
        val status = propAsString(map["status"]) ?: ""
        val versionNumber = propAsString(map["versionNumber"]) ?: ""
        val createdAt = propAsString(map["createdAt"]) ?: ""
        val updatedAt = propAsString(map["updatedAt"]) ?: ""
        val data =
            map.entries
                .filter { it.key is String }
                .associate { (k, v) -> k as String to (prop(v) ?: "") }
                .filterKeys { it !in setOf("id", "label", "status", "versionNumber", "createdAt", "updatedAt") }
        return VersionRecord(id, status, versionNumber, data, createdAt, updatedAt)
    }

    private fun prop(v: Any?): Any? =
        when (v) {
            null -> null
            is List<*> -> if (v.size == 1) v.firstOrNull() else v
            else -> v
        }

    private fun propAsString(v: Any?): String? = prop(v)?.toString()

    private fun sanitizeKey(key: String): String {
        val reserved = setOf("id", "label", "key", "value", "property")
        return if (key in reserved) "pg_$key" else key
    }

    private fun sanitizeCypherKey(key: String): String = key.replace("`", "")

    private fun toPropertyValue(value: Any?): Any =
        when (value) {
            null -> ""
            is String, is Number, is Boolean -> value
            is List<*>, is Map<*, *> -> mapper.writeValueAsString(value)
            else -> value.toString()
        }
}
