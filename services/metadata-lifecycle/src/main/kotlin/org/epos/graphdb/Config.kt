package org.epos.graphdb

import com.fasterxml.jackson.databind.DeserializationFeature
import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import java.io.File
import java.io.InputStream

/**
 * =============================================================================
 * EPOS Modular Configuration System
 * =============================================================================
 *
 * Features:
 * - Profile-based configuration (dev, prod, test, etc.)
 * - Automatic config merging (base + profile)
 * - Environment variable overrides
 * - Type-safe section access
 * - Easy extensibility
 *
 * Configuration Resolution Order (highest priority first):
 * 1. Environment variables (EPOS_*)
 * 2. Profile-specific config (application-{profile}.yaml)
 * 3. Base config (application.yaml)
 * 4. Default values in data classes
 *
 * Usage:
 *   Config.graphdb.endpoint          // Access GraphDB settings
 *   Config.graphs.published          // Access named graphs (resolved URIs)
 *   Config.agents.system             // Access agent URIs
 *   Config["custom.key"]             // Dynamic property access
 */

// =============================================================================
// Configuration Data Classes
// =============================================================================

/**
 * GraphDB connection settings
 */
data class GraphDbConfig(
    val endpoint: String = "http://localhost:7200",
    val repositoryId: String = "epos",
    val timeout: TimeoutConfig = TimeoutConfig()
) {
    /** Computed SPARQL query endpoint */
    val sparqlEndpoint: String get() = "$endpoint/repositories/$repositoryId"

    /** Computed SPARQL update endpoint */
    val sparqlUpdateEndpoint: String get() = "$sparqlEndpoint/statements"
}

/**
 * Timeout settings for HTTP clients
 */
data class TimeoutConfig(
    val connectMs: Long = 10_000,
    val readMs: Long = 60_000,
    val writeMs: Long = 60_000
)

/**
 * Named graph configuration (stored as relative paths, resolved on access)
 */
data class GraphsConfig(
    val conceptual: String = "/graphs/conceptual",
    val draft: String = "/graphs/data/draft",
    val submitted: String = "/graphs/data/submitted",
    val published: String = "/graphs/data/published",
    val archived: String = "/graphs/data/archived",
    val provenance: String = "/graphs/provenance"
) {
    /** Get all graph paths as a map */
    fun toMap(): Map<String, String> = mapOf(
        "conceptual" to conceptual,
        "draft" to draft,
        "submitted" to submitted,
        "published" to published,
        "archived" to archived,
        "provenance" to provenance
    )

    /** Get graph by status name */
    fun forStatus(status: String): String? = when (status.lowercase()) {
        "draft" -> draft
        "submitted" -> submitted
        "published" -> published
        "archived" -> archived
        else -> null
    }
}

/**
 * Agent URI configuration (stored as relative paths, resolved on access)
 */
data class AgentsConfig(
    val system: String = "/agent/system",
    val initialPopulation: String = "/agent/InitialPopulationService"
)

/**
 * Server configuration
 */
data class ServerConfig(
    val host: String = "0.0.0.0",
    val port: Int = 8080,
    val contextPath: String = "",
    val publicBaseUrl: String? = null
)

/**
 * Logging configuration
 */
data class LoggingConfig(
    val level: String = "INFO",
    val format: String = "text",  // "text" or "json"
    val includeTimestamp: Boolean = true
)

/**
 * Feature flags for optional functionality
 */
data class FeaturesConfig(
    val enableSwagger: Boolean = true,
    val enableCors: Boolean = true,
    val enableMetrics: Boolean = false
)

/**
 * Root application configuration
 */
data class AppConfig(
    val graphdb: GraphDbConfig? = null,
    val indexUrl: String? = null,
    val baseUri: String? = null,
    val graphs: GraphsConfig? = null,
    val agents: AgentsConfig? = null,
    val server: ServerConfig? = null,
    val logging: LoggingConfig? = null,
    val features: FeaturesConfig? = null,
    val custom: Map<String, Any>? = null  // For arbitrary custom properties
) {
    // Provide non-null accessors with defaults
    fun graphdbOrDefault() = graphdb ?: GraphDbConfig()
    fun indexUrlOrDefault() = indexUrl ?: "http://192.168.1.26:4500/index.txt"
    fun baseUriOrDefault() = baseUri ?: "https://epos-eu.org"
    fun graphsOrDefault() = graphs ?: GraphsConfig()
    fun agentsOrDefault() = agents ?: AgentsConfig()
    fun serverOrDefault() = server ?: ServerConfig()
    fun loggingOrDefault() = logging ?: LoggingConfig()
    fun featuresOrDefault() = features ?: FeaturesConfig()
    fun customOrDefault() = custom ?: emptyMap()
}

// =============================================================================
// Resolved Configuration (with absolute URIs)
// =============================================================================

/**
 * Graphs with URIs resolved to absolute paths
 */
class ResolvedGraphs(private val graphs: GraphsConfig, private val baseUri: String) {
    val conceptual: String get() = resolve(graphs.conceptual)
    val draft: String get() = resolve(graphs.draft)
    val submitted: String get() = resolve(graphs.submitted)
    val published: String get() = resolve(graphs.published)
    val archived: String get() = resolve(graphs.archived)
    val provenance: String get() = resolve(graphs.provenance)

    fun forStatus(status: String): String? = graphs.forStatus(status)?.let { resolve(it) }

    fun toMap(): Map<String, String> = graphs.toMap().mapValues { resolve(it.value) }

    private fun resolve(path: String): String =
        if (path.startsWith("http://") || path.startsWith("https://")) path
        else "$baseUri$path"
}

/**
 * Agents with URIs resolved to absolute paths
 */
class ResolvedAgents(private val agents: AgentsConfig, private val baseUri: String) {
    val system: String get() = resolve(agents.system)
    val initialPopulation: String get() = resolve(agents.initialPopulation)

    private fun resolve(path: String): String =
        if (path.startsWith("http://") || path.startsWith("https://")) path
        else "$baseUri$path"
}

// =============================================================================
// Configuration Loader
// =============================================================================

/**
 * Main configuration object - singleton access point
 */
object Config {
    private val mapper = ObjectMapper(YAMLFactory())
        .registerKotlinModule()
        .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false)

    private val rawConfig: AppConfig by lazy { loadConfiguration() }

    // -------------------------------------------------------------------------
    // Direct Section Access
    // -------------------------------------------------------------------------

    /** GraphDB configuration */
    val graphdb: GraphDbConfig get() = rawConfig.graphdbOrDefault()

    /** Server configuration */
    val server: ServerConfig get() = rawConfig.serverOrDefault()

    /** Logging configuration */
    val logging: LoggingConfig get() = rawConfig.loggingOrDefault()

    /** Feature flags */
    val features: FeaturesConfig get() = rawConfig.featuresOrDefault()

    /** Index URL for TTL files */
    val indexUrl: String get() = envOverride("EPOS_INDEX_URL") ?: rawConfig.indexUrlOrDefault()

    /** Base URI for all generated URIs */
    val baseUri: String get() = envOverride("EPOS_BASE_URI") ?: rawConfig.baseUriOrDefault()

    // -------------------------------------------------------------------------
    // Resolved Sections (with absolute URIs)
    // -------------------------------------------------------------------------

    /** Named graphs with resolved absolute URIs */
    val graphs: ResolvedGraphs by lazy { ResolvedGraphs(rawConfig.graphsOrDefault(), baseUri) }

    /** Agents with resolved absolute URIs */
    val agents: ResolvedAgents by lazy { ResolvedAgents(rawConfig.agentsOrDefault(), baseUri) }

    // -------------------------------------------------------------------------
    // Convenience Accessors (backwards compatibility)
    // -------------------------------------------------------------------------

    val GRAPHDB_ENDPOINT: String get() = envOverride("EPOS_GRAPHDB_ENDPOINT") ?: graphdb.endpoint
    val REPOSITORY_ID: String get() = envOverride("EPOS_REPOSITORY_ID") ?: graphdb.repositoryId
    val SPARQL_ENDPOINT: String get() = "$GRAPHDB_ENDPOINT/repositories/$REPOSITORY_ID"
    val SPARQL_UPDATE_ENDPOINT: String get() = "$SPARQL_ENDPOINT/statements"
    val INDEX_URL: String get() = indexUrl
    val BASE_URI: String get() = baseUri
    val ID_BASE: String get() = "$BASE_URI/id"

    // Graph shortcuts
    val GRAPH_CONCEPTUAL: String get() = graphs.conceptual
    val GRAPH_DRAFT: String get() = graphs.draft
    val GRAPH_SUBMITTED: String get() = graphs.submitted
    val GRAPH_PUBLISHED: String get() = graphs.published
    val GRAPH_ARCHIVED: String get() = graphs.archived
    val GRAPH_PROVENANCE: String get() = graphs.provenance

    // Agent shortcuts
    val AGENT_SYSTEM: String get() = agents.system
    val AGENT_INITIAL_POPULATION: String get() = agents.initialPopulation

    // -------------------------------------------------------------------------
    // Dynamic Property Access
    // -------------------------------------------------------------------------

    /**
     * Access custom properties by dot-notation key
     * Example: Config["myapp.feature.enabled"]
     */
    operator fun get(key: String): Any? {
        val parts = key.split(".")
        var current: Any? = rawConfig.customOrDefault()
        for (part in parts) {
            current = when (current) {
                is Map<*, *> -> current[part]
                else -> null
            }
            if (current == null) break
        }
        return current
    }

    /**
     * Get custom property with default value
     */
    fun <T> getOrDefault(key: String, default: T): T {
        @Suppress("UNCHECKED_CAST")
        return (this[key] as? T) ?: default
    }

    // -------------------------------------------------------------------------
    // Configuration Loading
    // -------------------------------------------------------------------------

    private fun loadConfiguration(): AppConfig {
        val profile = System.getenv("EPOS_PROFILE")
            ?: System.getProperty("epos.profile")
            ?: "default"

        println("📋 Loading configuration with profile: $profile")

        // Load base configuration
        val baseConfig = loadConfigFile("application.yaml")
            ?: loadConfigFromClasspath("application.yaml")
            ?: AppConfig()

        // Load profile-specific configuration if not default
        val profileConfig = if (profile != "default") {
            loadConfigFile("application-$profile.yaml")
                ?: loadConfigFromClasspath("application-$profile.yaml")
        } else null

        // Merge configurations
        val mergedConfig = if (profileConfig != null) {
            mergeConfigs(baseConfig, profileConfig)
        } else {
            baseConfig
        }

        // Apply environment variable overrides
        return applyEnvironmentOverrides(mergedConfig)
    }

    private fun loadConfigFile(filename: String): AppConfig? {
        val paths = listOfNotNull(
            System.getenv("EPOS_CONFIG_DIR")?.let { "$it/$filename" },
            System.getProperty("epos.config.dir")?.let { "$it/$filename" },
            "config/$filename",
            filename
        )

        for (path in paths) {
            val file = File(path)
            if (file.exists() && file.canRead()) {
                println("📄 Loading: ${file.absolutePath}")
                return mapper.readValue(file, AppConfig::class.java)
            }
        }
        return null
    }

    private fun loadConfigFromClasspath(filename: String): AppConfig? {
        val stream: InputStream? = javaClass.classLoader.getResourceAsStream(filename)
        return stream?.let {
            println("📄 Loading from classpath: $filename")
            mapper.readValue(it, AppConfig::class.java)
        }
    }

    /**
     * Merge two configurations (profile overrides base)
     */
    private fun mergeConfigs(base: AppConfig, profile: AppConfig): AppConfig {
        val baseGraphdb = base.graphdbOrDefault()
        val profileGraphdb = profile.graphdbOrDefault()
        val baseGraphs = base.graphsOrDefault()
        val profileGraphs = profile.graphsOrDefault()
        val baseAgents = base.agentsOrDefault()
        val profileAgents = profile.agentsOrDefault()
        val baseServer = base.serverOrDefault()
        val profileServer = profile.serverOrDefault()
        val baseLogging = base.loggingOrDefault()
        val profileLogging = profile.loggingOrDefault()
        val baseFeatures = base.featuresOrDefault()
        val profileFeatures = profile.featuresOrDefault()
        val defaultGraphdb = GraphDbConfig()
        val defaultGraphs = GraphsConfig()
        val defaultAgents = AgentsConfig()
        val defaultServer = ServerConfig()
        val defaultLogging = LoggingConfig()
        val defaultTimeout = TimeoutConfig()

        return AppConfig(
            graphdb = GraphDbConfig(
                endpoint = if (profile.graphdb != null && profileGraphdb.endpoint != defaultGraphdb.endpoint) profileGraphdb.endpoint else baseGraphdb.endpoint,
                repositoryId = if (profile.graphdb != null && profileGraphdb.repositoryId != defaultGraphdb.repositoryId) profileGraphdb.repositoryId else baseGraphdb.repositoryId,
                timeout = TimeoutConfig(
                    connectMs = if (profile.graphdb?.timeout != null && profileGraphdb.timeout.connectMs != defaultTimeout.connectMs) profileGraphdb.timeout.connectMs else baseGraphdb.timeout.connectMs,
                    readMs = if (profile.graphdb?.timeout != null && profileGraphdb.timeout.readMs != defaultTimeout.readMs) profileGraphdb.timeout.readMs else baseGraphdb.timeout.readMs,
                    writeMs = if (profile.graphdb?.timeout != null && profileGraphdb.timeout.writeMs != defaultTimeout.writeMs) profileGraphdb.timeout.writeMs else baseGraphdb.timeout.writeMs
                )
            ),
            indexUrl = profile.indexUrl ?: base.indexUrl,
            baseUri = profile.baseUri ?: base.baseUri,
            graphs = GraphsConfig(
                conceptual = if (profile.graphs != null && profileGraphs.conceptual != defaultGraphs.conceptual) profileGraphs.conceptual else baseGraphs.conceptual,
                draft = if (profile.graphs != null && profileGraphs.draft != defaultGraphs.draft) profileGraphs.draft else baseGraphs.draft,
                submitted = if (profile.graphs != null && profileGraphs.submitted != defaultGraphs.submitted) profileGraphs.submitted else baseGraphs.submitted,
                published = if (profile.graphs != null && profileGraphs.published != defaultGraphs.published) profileGraphs.published else baseGraphs.published,
                archived = if (profile.graphs != null && profileGraphs.archived != defaultGraphs.archived) profileGraphs.archived else baseGraphs.archived,
                provenance = if (profile.graphs != null && profileGraphs.provenance != defaultGraphs.provenance) profileGraphs.provenance else baseGraphs.provenance
            ),
            agents = AgentsConfig(
                system = if (profile.agents != null && profileAgents.system != defaultAgents.system) profileAgents.system else baseAgents.system,
                initialPopulation = if (profile.agents != null && profileAgents.initialPopulation != defaultAgents.initialPopulation) profileAgents.initialPopulation else baseAgents.initialPopulation
            ),
            server = ServerConfig(
                host = if (profile.server != null && profileServer.host != defaultServer.host) profileServer.host else baseServer.host,
                port = if (profile.server != null && profileServer.port != defaultServer.port) profileServer.port else baseServer.port,
                contextPath = if (profile.server != null && profileServer.contextPath != defaultServer.contextPath) profileServer.contextPath else baseServer.contextPath,
                publicBaseUrl = if (profile.server?.publicBaseUrl != null) profileServer.publicBaseUrl else baseServer.publicBaseUrl
            ),
            logging = LoggingConfig(
                level = if (profile.logging != null && profileLogging.level != defaultLogging.level) profileLogging.level else baseLogging.level,
                format = if (profile.logging != null && profileLogging.format != defaultLogging.format) profileLogging.format else baseLogging.format,
                includeTimestamp = if (profile.logging != null) profileLogging.includeTimestamp else baseLogging.includeTimestamp
            ),
            features = FeaturesConfig(
                enableSwagger = if (profile.features != null) profileFeatures.enableSwagger else baseFeatures.enableSwagger,
                enableCors = if (profile.features != null) profileFeatures.enableCors else baseFeatures.enableCors,
                enableMetrics = if (profile.features != null) profileFeatures.enableMetrics else baseFeatures.enableMetrics
            ),
            custom = base.customOrDefault() + profile.customOrDefault()
        )
    }

    /**
     * Apply environment variable overrides
     */
    private fun applyEnvironmentOverrides(config: AppConfig): AppConfig {
        val endpoint = envOverride("EPOS_GRAPHDB_ENDPOINT")
        val repoId = envOverride("EPOS_REPOSITORY_ID")
        val index = envOverride("EPOS_INDEX_URL")
        val base = envOverride("EPOS_BASE_URI")
        val serverPort = envOverride("EPOS_SERVER_PORT")?.toIntOrNull()
            ?: envOverride("PORT")?.toIntOrNull()
        val serverHost = envOverride("EPOS_SERVER_HOST")
            ?: envOverride("HOST")
        val publicBaseUrl = envOverride("EPOS_PUBLIC_BASE_URL")

        // Only create new objects if there are overrides
        if (listOfNotNull(endpoint, repoId, index, base, serverPort, serverHost, publicBaseUrl).isEmpty()) {
            return config
        }

        val currentGraphdb = config.graphdbOrDefault()
        val currentServer = config.serverOrDefault()

        return config.copy(
            graphdb = currentGraphdb.copy(
                endpoint = endpoint ?: currentGraphdb.endpoint,
                repositoryId = repoId ?: currentGraphdb.repositoryId
            ),
            indexUrl = index ?: config.indexUrl,
            baseUri = base ?: config.baseUri,
            server = currentServer.copy(
                port = serverPort ?: currentServer.port,
                host = serverHost ?: currentServer.host,
                publicBaseUrl = publicBaseUrl ?: currentServer.publicBaseUrl
            )
        )
    }

    private fun envOverride(key: String): String? =
        System.getenv(key)?.takeIf { it.isNotBlank() }

    // -------------------------------------------------------------------------
    // Diagnostics
    // -------------------------------------------------------------------------

    /**
     * Print current configuration for debugging
     */
    fun printConfig() {
        println("""
            |╔══════════════════════════════════════════════════════════════════════╗
            |║                    EPOS Configuration Summary                        ║
            |╠══════════════════════════════════════════════════════════════════════╣
            |║ GraphDB                                                              ║
            |║   Endpoint:      $GRAPHDB_ENDPOINT
            |║   Repository:    $REPOSITORY_ID
            |║   SPARQL:        $SPARQL_ENDPOINT
            |╠──────────────────────────────────────────────────────────────────────╣
            |║ Server                                                               ║
            |║   Host:          ${server.host}
            |║   Port:          ${server.port}
            |╠──────────────────────────────────────────────────────────────────────╣
            |║ URIs                                                                 ║
            |║   Base URI:      $BASE_URI
            |║   Index URL:     $INDEX_URL
            |║   ID Base:       $ID_BASE
            |╠──────────────────────────────────────────────────────────────────────╣
            |║ Named Graphs                                                         ║
            |║   Conceptual:    $GRAPH_CONCEPTUAL
            |║   Draft:         $GRAPH_DRAFT
            |║   Submitted:     $GRAPH_SUBMITTED
            |║   Published:     $GRAPH_PUBLISHED
            |║   Archived:      $GRAPH_ARCHIVED
            |║   Provenance:    $GRAPH_PROVENANCE
            |╠──────────────────────────────────────────────────────────────────────╣
            |║ Agents                                                               ║
            |║   System:        $AGENT_SYSTEM
            |║   Initial Pop:   $AGENT_INITIAL_POPULATION
            |╠──────────────────────────────────────────────────────────────────────╣
            |║ Features                                                             ║
            |║   Swagger:       ${features.enableSwagger}
            |║   CORS:          ${features.enableCors}
            |║   Metrics:       ${features.enableMetrics}
            |╚══════════════════════════════════════════════════════════════════════╝
        """.trimMargin())
    }

    /**
     * Export configuration as a map (useful for /debug/config endpoint)
     */
    fun toMap(): Map<String, Any> = mapOf(
        "graphdb" to mapOf(
            "endpoint" to GRAPHDB_ENDPOINT,
            "repositoryId" to REPOSITORY_ID,
            "sparqlEndpoint" to SPARQL_ENDPOINT,
            "sparqlUpdateEndpoint" to SPARQL_UPDATE_ENDPOINT
        ),
        "uris" to mapOf(
            "baseUri" to BASE_URI,
            "indexUrl" to INDEX_URL,
            "idBase" to ID_BASE
        ),
        "graphs" to graphs.toMap(),
        "agents" to mapOf(
            "system" to AGENT_SYSTEM,
            "initialPopulation" to AGENT_INITIAL_POPULATION
        ),
        "server" to mapOf(
            "host" to server.host,
            "port" to server.port,
            "contextPath" to server.contextPath
        ),
        "features" to mapOf(
            "swagger" to features.enableSwagger,
            "cors" to features.enableCors,
            "metrics" to features.enableMetrics
        )
    )
}
