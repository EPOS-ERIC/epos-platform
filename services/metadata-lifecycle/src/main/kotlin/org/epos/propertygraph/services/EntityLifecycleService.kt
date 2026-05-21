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
        val result = client.runQuery(
            "g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).out('CURRENT_VERSION').limit(1).valueMap(true).toList()"
        ).firstOrNull() ?: return null
        val map = result["value"] as? Map<*, *> ?: return null
        return toVersionRecord(map)
    }

    fun createDraft(
        conceptualUri: String,
        newVersionNumber: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ): String {
        val currentVersion = getCurrentPublishedVersion(conceptualUri)
            ?: throw IllegalStateException("No published version found for $conceptualUri")

        val newDraftUri = "$conceptualUri/$newVersionNumber-draft"

        val draftData = currentVersion.data.toMutableMap()
        draftData["versionNumber"] = newVersionNumber

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
              .property('versionNumber', ${gStr(newVersionNumber)})
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

    fun publish(submittedUri: String, agent: String = Config.AGENT_SYSTEM, group: String = Config.AGENT_SYSTEM) {
        requireVersionExists(submittedUri, "submitted")

        val conceptualUri = getConceptualUri(submittedUri)
            ?: throw IllegalStateException("Cannot find conceptual URI for $submittedUri")

        val currentPublished = getCurrentPublishedVersion(conceptualUri)
        val now = java.time.Instant.now().toString()

        if (currentPublished != null) {
            client.runWrite("""
                g.V().hasLabel('Version').has('id', ${gStr(currentPublished.id)})
                  .property('status', 'archived')
                  .property('updatedAt', ${gStr(now)})
                  .iterate()
            """.trimIndent())

            provenanceService.recordStateTransition(currentPublished.id, "PUBLISHED", "ARCHIVED", agent, group)
            println("  Archived old version: ${currentPublished.id}")
        }

        val finalUri = submittedUri.removeSuffix("-draft")
        if (finalUri != submittedUri) {
            client.runWrite("""
                g.V().hasLabel('Version').has('id', ${gStr(submittedUri)})
                  .property('id', ${gStr(finalUri)})
                  .property('status', 'published')
                  .property('updatedAt', ${gStr(now)})
                  .iterate()
            """.trimIndent())
        } else {
            client.runWrite("""
                g.V().hasLabel('Version').has('id', ${gStr(submittedUri)})
                  .property('status', 'published')
                  .property('updatedAt', ${gStr(now)})
                  .iterate()
            """.trimIndent())
        }

        client.runWrite("""
            def c = g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).next()
            g.V(c.id()).outE('CURRENT_VERSION').drop().iterate()
            def v = g.V().hasLabel('Version').has('id', ${gStr(finalUri)}).next()
            g.V(c.id()).addE('CURRENT_VERSION').to(V(v.id())).iterate()
        """.trimIndent())

        if (currentPublished != null) {
            client.runWrite("""
                def oldV = g.V().hasLabel('Version').has('id', ${gStr(currentPublished.id)}).next()
                def newV = g.V().hasLabel('Version').has('id', ${gStr(finalUri)}).next()
                if (!g.V(oldV.id()).out('SUPERSEDED_BY').hasId(newV.id()).hasNext()) {
                  g.V(oldV.id()).addE('SUPERSEDED_BY').to(V(newV.id())).iterate()
                }
            """.trimIndent())
        }

        provenanceService.recordStateTransition(finalUri, "SUBMITTED", "PUBLISHED", agent, group)
        println("Published: $finalUri (SUBMITTED -> PUBLISHED)")
        println("  Updated conceptual pointer: $conceptualUri -> $finalUri")
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
        val exists = client.exists(
            "g.V().hasLabel('Version').has('id', ${gStr(uri)}).has('status', ${gStr(expectedStatus)}).limit(1).toList()"
        )

        if (!exists) {
            throw IllegalArgumentException("Version $uri not found in $expectedStatus status")
        }
    }

    private fun toVersionRecord(map: Map<*, *>): VersionRecord {
        val id = propAsString(map["id"]) ?: ""
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
