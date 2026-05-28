package org.epos.propertygraph.services

import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils
import org.epos.propertygraph.utils.VersionRecord
import com.fasterxml.jackson.databind.ObjectMapper

class EntityLifecycleService(
    private val client: PropertyGraphClient,
    private val provenanceService: ProvenanceService
) {
    private val mapper = ObjectMapper()

    fun getCurrentPublishedVersion(conceptualUri: String): VersionRecord? {
        val current = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)})
              .out('CURRENT_VERSION').hasLabel('Version').has('status', within('published','PUBLISHED'))
              .limit(1).valueMap(true).toList()
            """.trimIndent(),
        ).firstOrNull()

        if (current != null) {
            val currentMap = (current["value"] as? Map<*, *>) ?: (current as? Map<*, *>) ?: return null
            return toVersionRecord(currentMap)
        }

        val fallback = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)})
              .out('HAS_VERSION').hasLabel('Version').has('status', within('published','PUBLISHED'))
              .order().by(coalesce(values('updatedAt'), values('createdAt'), constant('')), decr)
              .limit(1).valueMap(true).toList()
            """.trimIndent(),
        ).firstOrNull() ?: return null

        val fallbackMap = (fallback["value"] as? Map<*, *>) ?: (fallback as? Map<*, *>) ?: return null
        return toVersionRecord(fallbackMap)
    }

    fun createDraft(
        conceptualUri: String,
        newVersionNumber: String? = null,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ): String {
        val currentVersion = getCurrentPublishedVersion(conceptualUri)
            ?: throw IllegalStateException("No published version found for $conceptualUri")

        val resolvedVersionNumber = newVersionNumber?.trim().takeUnless { it.isNullOrBlank() }
            ?: nextVersionFrom(currentVersion.id)

        val newDraftUri = "$conceptualUri/$resolvedVersionNumber-draft"

        val draftData = currentVersion.data.toMutableMap()
        draftData["versionNumber"] = resolvedVersionNumber

        val timestamp = java.time.Instant.now().toString()
        val assignments = draftData.entries.joinToString("\n") { (k, v) ->
            "draft.property(${gStr(sanitizeKey(k))}, ${gVal(v)})"
        }

        client.runWrite(
            """
            def c = g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).next()
            def published = g.V(c.id()).out('CURRENT_VERSION').hasLabel('Version').next()
            def draft = g.addV('Version')
              .property('id', ${gStr(newDraftUri)})
              .property('status', 'draft')
              .property('versionNumber', ${gStr(resolvedVersionNumber)})
              .property('createdAt', ${gStr(timestamp)})
              .property('updatedAt', ${gStr(timestamp)})
              .next()
            $assignments
            g.V(c.id()).addE('HAS_VERSION').to(V(draft.id())).iterate()
            g.V(draft.id()).addE('REVISION_OF').to(V(published.id())).iterate()
            """.trimIndent()
        )

        provenanceService.recordDraftCreation(
            newVersionUri = newDraftUri,
            basedOnVersionUri = currentVersion.id,
            conceptualUri = conceptualUri,
            agent = agent,
            group = group
        )

        println("Created DRAFT: $newDraftUri (based on ${currentVersion.id})")
        return newDraftUri
    }

    private fun nextVersionFrom(currentVersionUri: String): String {
        val segment = currentVersionUri.substringAfterLast("/").removeSuffix("-draft")
        val match = Regex("v?(\\d+)(?:\\.(\\d+))?.*").find(segment)
        if (match == null) return "2.0"
        val major = match.groupValues[1].toIntOrNull() ?: 1
        return "${major + 1}.0"
    }

    fun submitDraft(draftUri: String, agent: String = Config.AGENT_SYSTEM, group: String = Config.AGENT_SYSTEM) {
        requireVersionExists(draftUri, "draft")

        val now = java.time.Instant.now().toString()
        client.runWrite("""
            g.V().hasLabel('Version').has('id', ${gStr(draftUri)}).has('status', 'draft')
              .property('status', 'submitted')
              .property('updatedAt', ${gStr(now)})
              .iterate()
        """.trimIndent())

        provenanceService.recordStateTransition(draftUri, "DRAFT", "SUBMITTED", agent, group)
        println("Submitted: $draftUri (DRAFT -> SUBMITTED)")
    }

    fun publish(submittedUri: String, agent: String = Config.AGENT_SYSTEM, group: String = Config.AGENT_SYSTEM): String {
        requireVersionExists(submittedUri, "submitted")

        val conceptualUri = getConceptualUri(submittedUri)
            ?: throw IllegalStateException("Cannot find conceptual URI for $submittedUri")

        val currentPublished = getCurrentPublishedVersion(conceptualUri)
        val now = java.time.Instant.now().toString()

        val finalUri = submittedUri.removeSuffix("-draft")
        // Publish submitted version first, then reconcile previous published versions and pointers.
        markSubmittedAsPublished(submittedUri, finalUri, now)

        val archivedRows = findArchivedPublishedRows(conceptualUri, finalUri)
        archivePreviousPublishedVersions(conceptualUri, finalUri, now, archivedRows, agent, group)

        updateCurrentVersionPointer(conceptualUri, finalUri)
        linkSupersededVersion(currentPublished?.id, finalUri)

        provenanceService.recordStateTransition(finalUri, "SUBMITTED", "PUBLISHED", agent, group)
        println("Published: $finalUri (SUBMITTED -> PUBLISHED)")
        println("  Updated conceptual pointer: $conceptualUri -> $finalUri")
        return finalUri
    }

    private fun markSubmittedAsPublished(submittedUri: String, finalUri: String, now: String) {
        if (finalUri != submittedUri) {
            client.runWrite(
                """
                g.V().hasLabel('Version').has('id', ${gStr(submittedUri)})
                  .property('id', ${gStr(finalUri)})
                  .property('status', 'published')
                  .property('updatedAt', ${gStr(now)})
                  .iterate()
                """.trimIndent(),
            )
            return
        }

        client.runWrite(
            """
            g.V().hasLabel('Version').has('id', ${gStr(submittedUri)})
              .property('status', 'published')
              .property('updatedAt', ${gStr(now)})
              .iterate()
            """.trimIndent(),
        )
    }

    private fun findArchivedPublishedRows(conceptualUri: String, finalUri: String): List<Map<String, Any?>> =
        client.runWriteReturning(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)})
              .out('HAS_VERSION').hasLabel('Version')
              .has('status', within('published', 'PUBLISHED'))
              .has('id', neq(${gStr(finalUri)}))
              .project('id')
              .by(coalesce(values('identifier'), values('id'), constant('')))
              .toList()
            """.trimIndent(),
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
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)})
              .out('HAS_VERSION').hasLabel('Version')
              .has('status', within('published', 'PUBLISHED'))
              .has('id', neq(${gStr(finalUri)}))
              .property('status', 'archived')
              .property('updatedAt', ${gStr(now)})
              .iterate()
            """.trimIndent(),
        )

        archivedRows.forEach { row ->
            val archivedId = row["id"]?.toString()?.trim().orEmpty()
            if (archivedId.isBlank()) return@forEach
            provenanceService.recordStateTransition(archivedId, "PUBLISHED", "ARCHIVED", agent, group)
            println("  Archived old version: $archivedId")
        }
    }

    private fun updateCurrentVersionPointer(conceptualUri: String, finalUri: String) {
        client.runWrite(
            """
            def c = g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).next()
            g.V(c.id()).outE('CURRENT_VERSION').drop().iterate()
            def v = g.V().hasLabel('Version').has('id', ${gStr(finalUri)}).next()
            g.V(c.id()).addE('CURRENT_VERSION').to(V(v.id())).iterate()
            """.trimIndent(),
        )
    }

    private fun linkSupersededVersion(previousPublishedUri: String?, finalUri: String) {
        if (previousPublishedUri.isNullOrBlank()) return
        client.runWrite(
            """
            def oldQ = g.V().hasLabel('Version').has('id', ${gStr(previousPublishedUri)}).limit(1).toList()
            def newQ = g.V().hasLabel('Version').has('id', ${gStr(finalUri)}).limit(1).toList()
            if (!oldQ.isEmpty() && !newQ.isEmpty()) {
              def oldV = oldQ.get(0)
              def newV = newQ.get(0)
              if (!g.V(oldV.id()).out('SUPERSEDED_BY').hasId(newV.id()).hasNext()) {
                g.V(oldV.id()).addE('SUPERSEDED_BY').to(V(newV.id())).iterate()
              }
            }
            """.trimIndent(),
        )
    }

    fun rejectSubmission(submittedUri: String, agent: String = Config.AGENT_SYSTEM, group: String = Config.AGENT_SYSTEM) {
        val now = java.time.Instant.now().toString()
        client.runWrite("""
            g.V().hasLabel('Version').has('id', ${gStr(submittedUri)}).has('status', 'submitted')
              .property('status', 'draft')
              .property('updatedAt', ${gStr(now)})
              .iterate()
        """.trimIndent())

        provenanceService.recordStateTransition(submittedUri, "SUBMITTED", "DRAFT", agent, group)
        println("Rejected: $submittedUri (SUBMITTED -> DRAFT)")
    }

    fun discard(versionUri: String, fromStatus: String, agent: String = Config.AGENT_SYSTEM, group: String = Config.AGENT_SYSTEM) {
        val normalizedFrom = fromStatus.lowercase()
        require(normalizedFrom == "draft" || normalizedFrom == "submitted") {
            "Only draft or submitted versions can be discarded"
        }

        val now = java.time.Instant.now().toString()
        client.runWrite(
            """
            g.V().hasLabel('Version').has('id', ${gStr(versionUri)}).has('status', ${gStr(normalizedFrom)})
              .property('status', 'discarded')
              .property('updatedAt', ${gStr(now)})
              .iterate()
            """.trimIndent(),
        )

        provenanceService.recordStateTransition(versionUri, normalizedFrom.uppercase(), "DISCARDED", agent, group)
        println("Discarded: $versionUri (${normalizedFrom.uppercase()} -> DISCARDED)")
    }

    fun deleteDraft(draftUri: String) {
        client.runWrite("""
            g.V().hasLabel('Version').has('id', ${gStr(draftUri)}).drop().iterate()
        """.trimIndent())
        println("Deleted DRAFT: $draftUri")
    }

    private fun getConceptualUri(versionUri: String): String? {
        val result = client.runQuery(
            "g.V().hasLabel('Version').has('id', ${gStr(versionUri)}).in('HAS_VERSION').values('id').limit(1).toList()"
        ).firstOrNull() ?: return null
        return result["value"]?.toString()
    }

    private fun requireVersionExists(uri: String, expectedStatus: String) {
        val count = client.runQuery(
            "g.V().hasLabel('Version').has('id', ${gStr(uri)}).has('status', ${gStr(expectedStatus)}).count().next()",
        ).firstOrNull()?.get("value")?.toString()?.toLongOrNull() ?: 0L
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
        val data = map.entries
            .filter { it.key is String }
            .associate { (k, v) -> k as String to (prop(v) ?: "") }
            .filterKeys { it !in setOf("id", "label", "status", "versionNumber", "createdAt", "updatedAt") }
        return VersionRecord(id, status, versionNumber, data, createdAt, updatedAt)
    }

    private fun prop(v: Any?): Any? = when (v) {
        null -> null
        is List<*> -> if (v.size == 1) v.firstOrNull() else v
        else -> v
    }

    private fun propAsString(v: Any?): String? = prop(v)?.toString()

    private fun sanitizeKey(key: String): String {
        val reserved = setOf("id", "label", "key", "value", "property")
        return if (key in reserved) "pg_$key" else key
    }

    private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

    private fun gVal(value: Any): String = when (value) {
        is String -> gStr(value)
        is Number, is Boolean -> value.toString()
        is List<*>, is Map<*, *> -> gStr(mapper.writeValueAsString(value))
        else -> gStr(value.toString())
    }
}
