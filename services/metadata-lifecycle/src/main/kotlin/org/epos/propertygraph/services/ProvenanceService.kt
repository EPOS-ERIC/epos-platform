package org.epos.propertygraph.services

import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils

class ProvenanceService(private val client: PropertyGraphClient) {

    fun recordInitialCreation(
        versionUri: String,
        conceptualUri: String,
        source: String,
        agent: String = Config.AGENT_INITIAL_POPULATION,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = java.time.Instant.now().toString()

        client.runWrite(createActivityScript(activityUri, "initial_creation", versionUri, null, "published", agent, group, timestamp))
    }

    fun recordDraftCreation(
        newVersionUri: String,
        basedOnVersionUri: String,
        conceptualUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = java.time.Instant.now().toString()

        client.runWrite(createActivityScript(activityUri, "draft_creation", newVersionUri, null, "draft", agent, group, timestamp))
    }

    fun recordStateTransition(
        versionUri: String,
        fromStatus: String,
        toStatus: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp = java.time.Instant.now().toString()

        client.runWrite(createActivityScript(activityUri, "state_transition", versionUri, fromStatus, toStatus, agent, group, timestamp))
    }

    fun getVersionHistory(conceptualUri: String): List<Map<String, String?>> {
        val results = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${toGremlinString(conceptualUri)})
              .out('HAS_VERSION')
              .project('version','activities')
                .by(valueMap(true))
                .by(__.in('AFFECTED').valueMap(true).fold())
              .toList()
            """.trimIndent()
        )

        return results.mapNotNull { row ->
            val v = row["version"] as? Map<*, *> ?: return@mapNotNull null
            val activities: List<*> = (row["activities"] as? List<*>) ?: emptyList<Any>()

            val latestActivity: Map<*, *>? = activities
                .filterIsInstance<Map<*, *>>()
                .filter { it["type"] == "state_transition" }
                .maxByOrNull { it["timestamp"] as? String ?: "" }

            mapOf(
                "version" to readGremlinProperty(v["id"]),
                "title" to readGremlinProperty(v["title"]),
                "timestamp" to readGremlinProperty(v["createdAt"]),
                "agent" to readGremlinProperty(latestActivity?.get("agent")),
                "group" to readGremlinProperty(latestActivity?.get("group")),
                "status" to (readGremlinProperty(v["status"]) ?: "unknown")
            )
        }
    }

    private fun readGremlinProperty(value: Any?): String? {
        return when (value) {
            null -> null
            is List<*> -> value.firstOrNull()?.toString()
            else -> value.toString()
        }
    }

    private fun createActivityScript(
        activityUri: String,
        type: String,
        versionUri: String,
        fromStatus: String?,
        toStatus: String?,
        agent: String,
        group: String,
        timestamp: String
    ): String {
        val fromStatusLine = fromStatus?.let { "a.property('fromStatus', ${toGremlinString(it)})" } ?: ""
        val toStatusLine = toStatus?.let { "a.property('toStatus', ${toGremlinString(it)})" } ?: ""

        return """
            import static org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.*
            def v = g.V().hasLabel('Version').has('id', ${toGremlinString(versionUri)}).tryNext().orElse(null)
            if (v != null) {
              def a = g.addV('Activity')
                .property('id', ${toGremlinString(activityUri)})
                .property('type', ${toGremlinString(type)})
                .property('agent', ${toGremlinString(agent)})
                .property('group', ${toGremlinString(group)})
                .property('timestamp', ${toGremlinString(timestamp)})
                .next()
              $fromStatusLine
              $toStatusLine
              g.V(a.id()).addE('AFFECTED').to(V(v.id())).iterate()
            }
        """.trimIndent()
    }

    private fun toGremlinString(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
}
