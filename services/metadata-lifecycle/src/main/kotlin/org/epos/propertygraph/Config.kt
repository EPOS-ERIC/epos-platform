package org.epos.propertygraph

import com.fasterxml.jackson.databind.DeserializationFeature
import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import java.io.File
import java.io.InputStream

data class JanusGraphConfig(
    val host: String = "localhost",
    val port: Int = 8182,
    val traversalSource: String = "g",
    val username: String = "",
    val password: String = "",
    val maxConnectionPoolSize: Int = 50,
    val connectionTimeoutMs: Long = 30_000
)

data class ServerConfig(
    val host: String = "0.0.0.0",
    val port: Int = 8080,
    val contextPath: String = "",
    val publicBaseUrl: String? = null
)

data class LoggingConfig(
    val level: String = "INFO",
    val format: String = "text",
    val includeTimestamp: Boolean = true
)

data class FeaturesConfig(
    val enableSwagger: Boolean = true,
    val enableCors: Boolean = true,
    val enableMetrics: Boolean = false
)

data class AgentsConfig(
    val system: String = "/agent/system",
    val initialPopulation: String = "/agent/InitialPopulationService"
)

data class AppConfig(
    val janusgraph: JanusGraphConfig? = null,
    val indexUrl: String? = null,
    val baseUri: String? = null,
    val agents: AgentsConfig? = null,
    val server: ServerConfig? = null,
    val logging: LoggingConfig? = null,
    val features: FeaturesConfig? = null,
    val custom: Map<String, Any>? = null
) {
    fun janusgraphOrDefault() = janusgraph ?: JanusGraphConfig()
    fun indexUrlOrDefault() = indexUrl ?: "http://192.168.1.26:4500/index.txt"
    fun baseUriOrDefault() = baseUri ?: "https://epos-eu.org"
    fun agentsOrDefault() = agents ?: AgentsConfig()
    fun serverOrDefault() = server ?: ServerConfig()
    fun loggingOrDefault() = logging ?: LoggingConfig()
    fun featuresOrDefault() = features ?: FeaturesConfig()
    fun customOrDefault() = custom ?: emptyMap()
}

class ResolvedAgents(private val agents: AgentsConfig, private val baseUri: String) {
    val system: String get() = resolve(agents.system)
    val initialPopulation: String get() = resolve(agents.initialPopulation)

    private fun resolve(path: String): String =
        if (path.startsWith("http://") || path.startsWith("https://")) path
        else "$baseUri$path"
}

object Config {
    private val mapper = ObjectMapper(YAMLFactory())
        .registerKotlinModule()
        .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false)

    private val rawConfig: AppConfig by lazy { loadConfiguration() }

    val janusgraph: JanusGraphConfig get() = rawConfig.janusgraphOrDefault()
    val server: ServerConfig get() = rawConfig.serverOrDefault()
    val logging: LoggingConfig get() = rawConfig.loggingOrDefault()
    val features: FeaturesConfig get() = rawConfig.featuresOrDefault()
    val indexUrl: String get() = envOverride("EPOS_INDEX_URL") ?: rawConfig.indexUrlOrDefault()
    val baseUri: String get() = envOverride("EPOS_BASE_URI") ?: rawConfig.baseUriOrDefault()

    val agents: ResolvedAgents by lazy { ResolvedAgents(rawConfig.agentsOrDefault(), baseUri) }

    val JANUSGRAPH_HOST: String get() = envOverride("EPOS_JANUSGRAPH_HOST") ?: janusgraph.host
    val JANUSGRAPH_PORT: Int get() = envOverride("EPOS_JANUSGRAPH_PORT")?.toIntOrNull() ?: janusgraph.port
    val JANUSGRAPH_TRAVERSAL_SOURCE: String get() = envOverride("EPOS_JANUSGRAPH_TRAVERSAL_SOURCE") ?: janusgraph.traversalSource
    val JANUSGRAPH_USERNAME: String get() = envOverride("EPOS_JANUSGRAPH_USERNAME") ?: janusgraph.username
    val JANUSGRAPH_PASSWORD: String get() = envOverride("EPOS_JANUSGRAPH_PASSWORD") ?: janusgraph.password
    val FALKORDB_HOST: String get() = JANUSGRAPH_HOST
    val FALKORDB_PORT: Int get() = JANUSGRAPH_PORT
    val FALKORDB_GRAPH: String get() = JANUSGRAPH_TRAVERSAL_SOURCE
    val FALKORDB_PASSWORD: String get() = JANUSGRAPH_PASSWORD
    val INDEX_URL: String get() = indexUrl
    val BASE_URI: String get() = baseUri
    val ID_BASE: String get() = "$BASE_URI/id"
    val RELATION_STRICT_MODE: Boolean
        get() = envOverride("EPOS_RELATION_STRICT_MODE")?.toBooleanStrictOrNull() ?: true

    val AGENT_SYSTEM: String get() = agents.system
    val AGENT_INITIAL_POPULATION: String get() = agents.initialPopulation

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

    fun <T> getOrDefault(key: String, default: T): T {
        @Suppress("UNCHECKED_CAST")
        return (this[key] as? T) ?: default
    }

    private fun loadConfiguration(): AppConfig {
        val profile = System.getenv("EPOS_PROFILE")
            ?: System.getProperty("epos.profile")
            ?: "default"

        println("Loading configuration with profile: $profile")

        val baseConfig = loadConfigFile("application.yaml")
            ?: loadConfigFromClasspath("application.yaml")
            ?: AppConfig()

        val profileConfig = if (profile != "default") {
            loadConfigFile("application-$profile.yaml")
                ?: loadConfigFromClasspath("application-$profile.yaml")
        } else null

        val mergedConfig = if (profileConfig != null) {
            mergeConfigs(baseConfig, profileConfig)
        } else {
            baseConfig
        }

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
                println("Loading: ${file.absolutePath}")
                return mapper.readValue(file, AppConfig::class.java)
            }
        }
        return null
    }

    private fun loadConfigFromClasspath(filename: String): AppConfig? {
        val stream: InputStream? = javaClass.classLoader.getResourceAsStream(filename)
        return stream?.let {
            println("Loading from classpath: $filename")
            mapper.readValue(it, AppConfig::class.java)
        }
    }

    private fun mergeConfigs(base: AppConfig, profile: AppConfig): AppConfig {
        val baseJanusGraph = base.janusgraphOrDefault()
        val profileJanusGraph = profile.janusgraphOrDefault()
        val baseAgents = base.agentsOrDefault()
        val profileAgents = profile.agentsOrDefault()
        val baseServer = base.serverOrDefault()
        val profileServer = profile.serverOrDefault()
        val baseLogging = base.loggingOrDefault()
        val profileLogging = profile.loggingOrDefault()
        val baseFeatures = base.featuresOrDefault()
        val profileFeatures = profile.featuresOrDefault()
        val defaultJanusGraph = JanusGraphConfig()
        val defaultAgents = AgentsConfig()
        val defaultServer = ServerConfig()
        val defaultLogging = LoggingConfig()

        return AppConfig(
            janusgraph = JanusGraphConfig(
                host = if (profile.janusgraph != null && profileJanusGraph.host != defaultJanusGraph.host) profileJanusGraph.host else baseJanusGraph.host,
                port = if (profile.janusgraph != null && profileJanusGraph.port != defaultJanusGraph.port) profileJanusGraph.port else baseJanusGraph.port,
                traversalSource = if (profile.janusgraph != null && profileJanusGraph.traversalSource != defaultJanusGraph.traversalSource) profileJanusGraph.traversalSource else baseJanusGraph.traversalSource,
                username = if (profile.janusgraph != null && profileJanusGraph.username != defaultJanusGraph.username) profileJanusGraph.username else baseJanusGraph.username,
                password = if (profile.janusgraph != null && profileJanusGraph.password != defaultJanusGraph.password) profileJanusGraph.password else baseJanusGraph.password,
                maxConnectionPoolSize = if (profile.janusgraph != null && profileJanusGraph.maxConnectionPoolSize != defaultJanusGraph.maxConnectionPoolSize) profileJanusGraph.maxConnectionPoolSize else baseJanusGraph.maxConnectionPoolSize,
                connectionTimeoutMs = if (profile.janusgraph != null && profileJanusGraph.connectionTimeoutMs != defaultJanusGraph.connectionTimeoutMs) profileJanusGraph.connectionTimeoutMs else baseJanusGraph.connectionTimeoutMs
            ),
            indexUrl = profile.indexUrl ?: base.indexUrl,
            baseUri = profile.baseUri ?: base.baseUri,
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

    private fun applyEnvironmentOverrides(config: AppConfig): AppConfig {
        val janusgraphHost = envOverride("EPOS_JANUSGRAPH_HOST")
        val janusgraphPort = envOverride("EPOS_JANUSGRAPH_PORT")?.toIntOrNull()
        val janusgraphGraph = envOverride("EPOS_JANUSGRAPH_TRAVERSAL_SOURCE")
        val janusgraphPassword = envOverride("EPOS_JANUSGRAPH_PASSWORD")
        val janusUser = envOverride("EPOS_JANUSGRAPH_USERNAME")
        val index = envOverride("EPOS_INDEX_URL")
        val base = envOverride("EPOS_BASE_URI")
        val serverPort = envOverride("EPOS_SERVER_PORT")?.toIntOrNull()
            ?: envOverride("PORT")?.toIntOrNull()
        val serverHost = envOverride("EPOS_SERVER_HOST")
            ?: envOverride("HOST")
        val publicBaseUrl = envOverride("EPOS_PUBLIC_BASE_URL")

        if (listOfNotNull(janusgraphHost, janusgraphPort, janusgraphGraph, janusgraphPassword, index, base, serverPort, serverHost, publicBaseUrl).isEmpty()) {
            return config
        }

        val currentJanusGraph = config.janusgraphOrDefault()
        val currentServer = config.serverOrDefault()

        return config.copy(
            janusgraph = currentJanusGraph.copy(
                host = janusgraphHost ?: currentJanusGraph.host,
                port = janusgraphPort ?: currentJanusGraph.port,
                traversalSource = janusgraphGraph ?: currentJanusGraph.traversalSource,
                password = janusgraphPassword ?: currentJanusGraph.password,
                username = janusUser ?: currentJanusGraph.username
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

    fun printConfig() {
        println("""
            |Configuration Summary
            |=====================
            | JanusGraph
            |   Host:     $JANUSGRAPH_HOST
            |   Port:     $JANUSGRAPH_PORT
            |   Traversal Source: $JANUSGRAPH_TRAVERSAL_SOURCE
            | Server
            |   Host:     ${server.host}
            |   Port:     ${server.port}
            | URIs
            |   Base URI: $BASE_URI
            |   Index URL: $INDEX_URL
            |   ID Base:   $ID_BASE
            | Agents
            |   System:    $AGENT_SYSTEM
            |   Initial Pop: $AGENT_INITIAL_POPULATION
            | Features
            |   Swagger:   ${features.enableSwagger}
            |   CORS:      ${features.enableCors}
            |   Metrics:   ${features.enableMetrics}
        """.trimMargin())
    }

    fun toMap(): Map<String, Any> = mapOf(
        "janusgraph" to mapOf(
            "host" to JANUSGRAPH_HOST,
            "port" to JANUSGRAPH_PORT,
            "traversalSource" to JANUSGRAPH_TRAVERSAL_SOURCE
        ),
        "uris" to mapOf(
            "baseUri" to BASE_URI,
            "indexUrl" to INDEX_URL,
            "idBase" to ID_BASE
        ),
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
