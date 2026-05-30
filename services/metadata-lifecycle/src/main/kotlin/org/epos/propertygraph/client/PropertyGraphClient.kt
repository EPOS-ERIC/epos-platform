package org.epos.propertygraph.client

import org.epos.propertygraph.Config
import org.neo4j.driver.AuthTokens
import org.neo4j.driver.Driver
import org.neo4j.driver.GraphDatabase
import org.neo4j.driver.Record
import org.neo4j.driver.Values
import org.neo4j.driver.types.Node
import org.neo4j.driver.types.Relationship
import org.slf4j.LoggerFactory

class PropertyGraphClient {
    @Volatile
    private var driver: Driver? = null

    private val connectionLock = Any()

    private val logger = LoggerFactory.getLogger(PropertyGraphClient::class.java)

    init {
        connect()
    }

    fun runQuery(
        cypher: String,
        parameters: Map<String, Any> = emptyMap(),
    ): List<Map<String, Any?>> {
        val activeDriver = ensureDriver()
        val session = activeDriver.session()
        return session.use {
            it.run(cypher, Values.value(parameters)).list().map(::mapRecord)
        }
    }

    fun runSingle(
        cypher: String,
        parameters: Map<String, Any> = emptyMap(),
    ): Map<String, Any?>? = runQuery(cypher, parameters).firstOrNull()

    fun runWrite(
        cypher: String,
        parameters: Map<String, Any> = emptyMap(),
    ) {
        val activeDriver = ensureDriver()
        val session = activeDriver.session()
        session.use {
            it.executeWrite { tx ->
                tx.run(cypher, Values.value(parameters)).consume()
                null
            }
        }
    }

    fun runWriteReturning(
        cypher: String,
        parameters: Map<String, Any> = emptyMap(),
    ): List<Map<String, Any?>> {
        val activeDriver = ensureDriver()
        val session = activeDriver.session()
        return session.use {
            it.executeWrite { tx ->
                tx.run(cypher, Values.value(parameters)).list().map(::mapRecord)
            }
        }
    }

    fun exists(
        cypher: String,
        parameters: Map<String, Any> = emptyMap(),
    ): Boolean = runQuery(cypher, parameters).isNotEmpty()

    fun close() {
        synchronized(connectionLock) {
            runCatching { driver?.close() }
            driver = null
        }
    }

    private fun ensureDriver(): Driver {
        val current = driver
        if (current != null) {
            return current
        }

        synchronized(connectionLock) {
            val existing = driver
            if (existing != null) {
                return existing
            }

            connect()
            return requireNotNull(driver) { "Neo4j driver is not initialized" }
        }
    }

    private fun connect() {
        val auth =
            if (Config.NEO4J_USERNAME.isNotBlank()) {
                AuthTokens.basic(Config.NEO4J_USERNAME, Config.NEO4J_PASSWORD)
            } else {
                AuthTokens.none()
            }

        var lastError: Exception? = null
        repeat(Config.NEO4J_CONNECT_MAX_ATTEMPTS) { attemptIndex ->
            val attempt = attemptIndex + 1
            try {
                val candidate = GraphDatabase.driver(Config.NEO4J_URI, auth)
                candidate.verifyConnectivity()
                driver = candidate
                logger.info(
                    "Neo4j connected: {} (database={}, attempt={}/{})",
                    Config.NEO4J_URI,
                    Config.NEO4J_DATABASE,
                    attempt,
                    Config.NEO4J_CONNECT_MAX_ATTEMPTS,
                )
                return
            } catch (e: Exception) {
                lastError = e
                logger.warn(
                    "Neo4j connection attempt {}/{} failed: {}",
                    attempt,
                    Config.NEO4J_CONNECT_MAX_ATTEMPTS,
                    e.message,
                )
                if (attempt < Config.NEO4J_CONNECT_MAX_ATTEMPTS) {
                    Thread.sleep(Config.NEO4J_CONNECT_RETRY_DELAY_MS)
                }
            }
        }

        throw IllegalStateException(
            "Failed to connect to Neo4j after ${Config.NEO4J_CONNECT_MAX_ATTEMPTS} attempts",
            lastError,
        )
    }

    private fun mapRecord(record: Record): Map<String, Any?> = record.keys().associateWith { key -> toJavaValue(record[key].asObject()) }

    private val jsonMapper =
        com.fasterxml.jackson.databind
            .ObjectMapper()

    private fun toJavaValue(value: Any?): Any? =
        when (value) {
            null -> null
            is Number -> value
            is Boolean -> value
            is String -> tryParseJson(value)
            is Node -> {
                val props = value.asMap { raw -> toJavaValue(raw) }
                props + mapOf("_id" to value.id(), "_labels" to value.labels().toList())
            }
            is Relationship -> {
                val props = value.asMap { raw -> toJavaValue(raw) }
                props +
                    mapOf(
                        "_id" to value.id(),
                        "_type" to value.type(),
                        "_start" to value.startNodeId(),
                        "_end" to value.endNodeId(),
                    )
            }
            is List<*> -> value.map { toJavaValue(it) }
            is Map<*, *> -> value.mapValues { toJavaValue(it.value) }
            else -> value.toString()
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
