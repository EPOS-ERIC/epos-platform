package org.epos.propertygraph.client

import org.apache.tinkerpop.gremlin.driver.Client
import org.apache.tinkerpop.gremlin.driver.Cluster
import org.apache.tinkerpop.gremlin.driver.Result
import org.epos.propertygraph.Config
import org.slf4j.LoggerFactory
import java.util.concurrent.TimeUnit

class PropertyGraphClient {
    @Volatile
    private var cluster: Cluster? = null

    @Volatile
    private var client: Client? = null

    private val connectionLock = Any()

    private val logger = LoggerFactory.getLogger(PropertyGraphClient::class.java)

    private val maxConnectAttempts = 12
    private val connectRetryDelayMs = 5_000L

    init {
        connectWithRetry()
    }

    fun runQuery(cypher: String, parameters: Map<String, Any> = emptyMap()): List<Map<String, Any?>> {
        val script = resolveParameters(cypher, parameters)
        return submitWithReconnect(script).map { result -> mapResult(result) }
    }

    fun runSingle(cypher: String, parameters: Map<String, Any> = emptyMap()): Map<String, Any?>? {
        val results = runQuery(cypher, parameters)
        return results.firstOrNull()
    }

    fun runWrite(cypher: String, parameters: Map<String, Any> = emptyMap()) {
        val script = resolveParameters(cypher, parameters)
        submitWithReconnect(script)
    }

    fun runWriteReturning(cypher: String, parameters: Map<String, Any> = emptyMap()): List<Map<String, Any?>> {
        val script = resolveParameters(cypher, parameters)
        return submitWithReconnect(script).map { result -> mapResult(result) }
    }

    fun exists(cypher: String, parameters: Map<String, Any> = emptyMap()): Boolean {
        val script = resolveParameters(cypher, parameters)
        val wrapped = "(($script).size() > 0)"
        val result = submitWithReconnect(wrapped).firstOrNull() ?: return false
        return when (val value = result.`object`) {
            is Boolean -> value
            is Number -> value.toInt() > 0
            else -> value?.toString()?.toBooleanStrictOrNull() ?: false
        }
    }

    fun close() {
        synchronized(connectionLock) {
            closeResources()
        }
    }

    private fun submitWithReconnect(script: String): List<Result> {
        return try {
            submit(script)
        } catch (e: Exception) {
            logger.warn("Gremlin query failed, reconnecting and retrying once: {}", e.message)
            reconnect()
            submit(script)
        }
    }

    private fun submit(script: String): List<Result> {
        val activeClient = ensureClient()
        return activeClient.submit(script).all().get()
    }

    private fun ensureClient(): Client {
        val current = client
        if (current != null) {
            return current
        }

        synchronized(connectionLock) {
            val existing = client
            if (existing != null) {
                return existing
            }

            connectWithRetry()
            return requireNotNull(client) { "Gremlin client is not initialized" }
        }
    }

    private fun reconnect() {
        synchronized(connectionLock) {
            closeResources()
            connectWithRetry()
        }
    }

    private fun connectWithRetry() {
        var lastError: Exception? = null

        repeat(maxConnectAttempts) { attempt ->
            try {
                val nextCluster = buildCluster()
                val nextClient: Client = nextCluster.connect()

                cluster = nextCluster
                client = nextClient

                logger.info(
                    "JanusGraph connected: {}:{}/{}",
                    Config.JANUSGRAPH_HOST,
                    Config.JANUSGRAPH_PORT,
                    Config.JANUSGRAPH_TRAVERSAL_SOURCE,
                )
                return
            } catch (e: Exception) {
                lastError = e
                val nextAttempt = attempt + 1
                if (nextAttempt < maxConnectAttempts) {
                    logger.warn(
                        "JanusGraph unavailable (attempt {}/{}), retrying in {} ms: {}",
                        nextAttempt,
                        maxConnectAttempts,
                        connectRetryDelayMs,
                        e.message,
                    )
                    TimeUnit.MILLISECONDS.sleep(connectRetryDelayMs)
                }
            }
        }

        throw IllegalStateException(
            "Unable to connect to JanusGraph after $maxConnectAttempts attempts",
            lastError,
        )
    }

    private fun closeResources() {
        runCatching { client?.close() }
        runCatching { cluster?.close() }
        client = null
        cluster = null
    }

    private fun buildCluster(): Cluster {
        return if (Config.JANUSGRAPH_USERNAME.isNotBlank() && Config.JANUSGRAPH_PASSWORD.isNotBlank()) {
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
