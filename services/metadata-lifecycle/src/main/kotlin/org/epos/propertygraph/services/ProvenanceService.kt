package org.epos.propertygraph.services

import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils

class ProvenanceService(
    private val client: PropertyGraphClient,
) {
    fun recordInitialCreation(
        versionUri: String,
        agent: String = Config.AGENT_INITIAL_POPULATION,
        group: String = Config.AGENT_SYSTEM,
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp =
            java.time.Instant
                .now()
                .toString()

        client.runWrite(
            createActivityCypher(),
            buildActivityParams(
                activityUri = activityUri,
                type = "initial_creation",
                versionUri = versionUri,
                fromStatus = null,
                toStatus = "published",
                agent = agent,
                group = group,
                timestamp = timestamp,
            ),
        )
    }

    fun recordDraftCreation(
        newVersionUri: String,
        basedOnVersionUri: String,
        conceptualUri: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp =
            java.time.Instant
                .now()
                .toString()

        client.runWrite(
            createActivityCypher(),
            buildActivityParams(
                activityUri = activityUri,
                type = "draft_creation",
                versionUri = newVersionUri,
                fromStatus = null,
                toStatus = "draft",
                agent = agent,
                group = group,
                timestamp = timestamp,
                extraProperties =
                    mapOf(
                        "startingVersionId" to basedOnVersionUri,
                        "startingDatasetId" to conceptualUri,
                    ),
            ),
        )
    }

    fun recordStateTransition(
        versionUri: String,
        fromStatus: String,
        toStatus: String,
        agent: String = Config.AGENT_SYSTEM,
        group: String = Config.AGENT_SYSTEM,
    ) {
        val activityUri = UriUtils.generateActivityUri()
        val timestamp =
            java.time.Instant
                .now()
                .toString()

        client.runWrite(
            createActivityCypher(),
            buildActivityParams(
                activityUri = activityUri,
                type = "state_transition",
                versionUri = versionUri,
                fromStatus = fromStatus,
                toStatus = toStatus,
                agent = agent,
                group = group,
                timestamp = timestamp,
            ),
        )
    }

    fun getVersionHistory(conceptualUri: String): List<Map<String, String?>> {
        val results =
            client.runQuery(
                """
                MATCH (:Conceptual {id: ${'$'}conceptualUri})-[:HAS_VERSION]->(v:Version)
                OPTIONAL MATCH (a:Activity)-[:AFFECTED]->(v)
                WITH properties(v) AS version, collect(properties(a)) AS activities
                RETURN version, activities
                """.trimIndent(),
                mapOf("conceptualUri" to conceptualUri),
            )

        return results.mapNotNull { row ->
            val v = row["version"] as? Map<*, *> ?: return@mapNotNull null
            val activities: List<*> = (row["activities"] as? List<*>) ?: emptyList<Any>()

            val activityMaps = activities.filterIsInstance<Map<*, *>>()

            val latestStateTransition: Map<*, *>? =
                activityMaps
                    .filter { readGremlinProperty(it["type"]) == "state_transition" }
                    .maxByOrNull { readGremlinProperty(it["timestamp"]) ?: "" }

            val latestActivity: Map<*, *>? =
                latestStateTransition
                    ?: activityMaps.maxByOrNull { readGremlinProperty(it["timestamp"]) ?: "" }

            mapOf(
                "version" to readGremlinProperty(v["id"]),
                "title" to readGremlinProperty(v["title"]),
                "timestamp" to (readGremlinProperty(latestActivity?.get("timestamp")) ?: readGremlinProperty(v["createdAt"])),
                "agent" to readGremlinProperty(latestActivity?.get("agent")),
                "group" to readGremlinProperty(latestActivity?.get("group")),
                "status" to (readGremlinProperty(v["status"]) ?: "unknown"),
            )
        }
    }

    private fun readGremlinProperty(value: Any?): String? =
        when (value) {
            null -> null
            is List<*> -> value.firstOrNull()?.toString()
            else -> value.toString()
        }

    private fun createActivityCypher(): String =
        """
        MATCH (v:Version {id: ${'$'}versionUri})
        MERGE (a:Activity {id: ${'$'}activityUri})
        SET a.type = ${'$'}type,
            a.agent = ${'$'}agent,
            a.group = ${'$'}group,
            a.timestamp = ${'$'}timestamp,
            a.fromStatus = ${'$'}fromStatus,
            a.toStatus = ${'$'}toStatus
        SET a += ${'$'}extraProperties
        MERGE (a)-[:AFFECTED]->(v)
        """.trimIndent()

    private fun buildActivityParams(
        activityUri: String,
        type: String,
        versionUri: String,
        fromStatus: String?,
        toStatus: String?,
        agent: String,
        group: String,
        timestamp: String,
        extraProperties: Map<String, String> = emptyMap(),
    ): Map<String, Any> {
        return mapOf(
            "activityUri" to activityUri,
            "type" to type,
            "versionUri" to versionUri,
            "fromStatus" to (fromStatus ?: ""),
            "toStatus" to (toStatus ?: ""),
            "agent" to agent,
            "group" to group,
            "timestamp" to timestamp,
            "extraProperties" to extraProperties,
        )
    }

    private fun toGremlinString(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
}
