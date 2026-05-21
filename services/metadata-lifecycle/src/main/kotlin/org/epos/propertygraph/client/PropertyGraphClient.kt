package org.epos.propertygraph.client

import org.apache.tinkerpop.gremlin.driver.Client
import org.apache.tinkerpop.gremlin.driver.Cluster
import org.apache.tinkerpop.gremlin.driver.Result
import org.epos.propertygraph.Config

class PropertyGraphClient {
    private val cluster: Cluster
    private val client: Client

    init {
        cluster = if (Config.JANUSGRAPH_USERNAME.isNotBlank() && Config.JANUSGRAPH_PASSWORD.isNotBlank()) {
            Cluster.build()
                .addContactPoint(Config.JANUSGRAPH_HOST)
                .port(Config.JANUSGRAPH_PORT)
                .credentials(Config.JANUSGRAPH_USERNAME, Config.JANUSGRAPH_PASSWORD)
                .create()
        } else {
            Cluster.build()
                .addContactPoint(Config.JANUSGRAPH_HOST)
                .port(Config.JANUSGRAPH_PORT)
                .create()
        }
        client = cluster.connect()
        println("JanusGraph connected: ${Config.JANUSGRAPH_HOST}:${Config.JANUSGRAPH_PORT}/${Config.JANUSGRAPH_TRAVERSAL_SOURCE}")
    }

    fun runQuery(cypher: String, parameters: Map<String, Any> = emptyMap()): List<Map<String, Any?>> {
        val script = resolveParameters(cypher, parameters)
        val results = client.submit(script).all().get()
        return results.map { result -> mapResult(result) }
    }

    fun runSingle(cypher: String, parameters: Map<String, Any> = emptyMap()): Map<String, Any?>? {
        val results = runQuery(cypher, parameters)
        return results.firstOrNull()
    }

    fun runWrite(cypher: String, parameters: Map<String, Any> = emptyMap()) {
        val script = resolveParameters(cypher, parameters)
        client.submit(script).all().get()
    }

    fun runWriteReturning(cypher: String, parameters: Map<String, Any> = emptyMap()): List<Map<String, Any?>> {
        val script = resolveParameters(cypher, parameters)
        val results = client.submit(script).all().get()
        return results.map { result -> mapResult(result) }
    }

    fun exists(cypher: String, parameters: Map<String, Any> = emptyMap()): Boolean {
        return runQuery(cypher, parameters).isNotEmpty()
    }

    fun close() {
        client.close()
        cluster.close()
    }

    private fun resolveParameters(cypher: String, parameters: Map<String, Any>): String {
        return Regex("""\$([a-zA-Z_][a-zA-Z0-9_]*)""").replace(cypher) { match ->
            val key = match.groupValues[1]
            val value = parameters[key] ?: return@replace match.value
            when (value) {
                is String -> "'${escapeString(value)}'"
                is Number -> value.toString()
                is Boolean -> value.toString()
                else -> "'${escapeString(value.toString())}'"
            }
        }
    }

    private fun mapResult(result: Result): Map<String, Any?> {
        val value = result.`object`
        return when (value) {
            is Map<*, *> -> value.entries.associate { (k, v) -> k.toString() to toJavaValue(v) }
            else -> mapOf("value" to toJavaValue(value))
        }
    }

    private fun escapeString(value: String): String {
        return value.replace("\\", "\\\\")
            .replace("'", "\\'")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("\t", "\\t")
    }

    private val jsonMapper = com.fasterxml.jackson.databind.ObjectMapper()

    private fun toJavaValue(value: Any?): Any? {
        return when (value) {
            null -> null
            is Number -> value
            is Boolean -> value
            is String -> tryParseJson(value)
            is List<*> -> value.map { toJavaValue(it) }
            is Map<*, *> -> value.mapValues { toJavaValue(it.value) }
            else -> value.toString()
        }
    }

    private fun tryParseJson(value: String): Any? {
        if (!value.startsWith("{") && !value.startsWith("[")) return value
        return try {
            jsonMapper.readValue(value, Any::class.java)
        } catch (_: Exception) {
            value
        }
    }
}
