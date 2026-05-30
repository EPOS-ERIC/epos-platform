package org.epos.propertygraph

import com.fasterxml.jackson.databind.DeserializationFeature
import com.fasterxml.jackson.databind.ObjectMapper
import com.fasterxml.jackson.dataformat.yaml.YAMLFactory
import com.fasterxml.jackson.module.kotlin.registerKotlinModule
import java.io.File
import java.io.InputStream

data class Neo4jConfig(
    val uri: String = "bolt://localhost:7687",
    val username: String = "",
    val password: String = "",
    val maxConnectionPoolSize: Int = 50,
    val connectionTimeoutMs: Long = 30_000,
)

data class ServerConfig(
    val host: String = "0.0.0.0",
    val port: Int = 8080,
    val contextPath: String = "",
    val publicBaseUrl: String? = null,
)

data class LoggingConfig(
    val level: String = "INFO",
    val format: String = "text",
    val includeTimestamp: Boolean = true,
)

data class FeaturesConfig(
    val enableSwagger: Boolean = true,
    val enableCors: Boolean = true,
    val enableMetrics: Boolean = false,
)

data class AgentsConfig(
    val system: String = "/agent/system",
    val initialPopulation: String = "/agent/InitialPopulationService",
)

data class AppConfig(
    val neo4j: Neo4jConfig? = null,
    val indexUrl: String? = null,
    val baseUri: String? = null,
    val agents: AgentsConfig? = null,
    val server: ServerConfig? = null,
    val logging: LoggingConfig? = null,
    val features: FeaturesConfig? = null,
    val custom: Map<String, Any>? = null,
) {
    fun neo4jOrDefault() = neo4j ?: Neo4jConfig()

    fun indexUrlOrDefault() = indexUrl ?: "http://192.168.1.26:4500/index.txt"

    fun baseUriOrDefault() = baseUri ?: "https://epos-eu.org"

    fun agentsOrDefault() = agents ?: AgentsConfig()

    fun serverOrDefault() = server ?: ServerConfig()

    fun loggingOrDefault() = logging ?: LoggingConfig()

    fun featuresOrDefault() = features ?: FeaturesConfig()

    fun customOrDefault() = custom ?: emptyMap()
}

class ResolvedAgents(
    private val agents: AgentsConfig,
    private val baseUri: String,
) {
    val system: String get() = resolve(agents.system)
    val initialPopulation: String get() = resolve(agents.initialPopulation)

    private fun resolve(path: String): String =
        if (path.startsWith("http://") || path.startsWith("https://")) {
            path
        } else {
            "$baseUri$path"
        }
}

object Config {
    private val mapper =
        ObjectMapper(YAMLFactory())
            .registerKotlinModule()
            .configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false)

    private val rawConfig: AppConfig by lazy { loadConfiguration() }

    val neo4j: Neo4jConfig get() = rawConfig.neo4jOrDefault()
    val server: ServerConfig get() = rawConfig.serverOrDefault()
    val logging: LoggingConfig get() = rawConfig.loggingOrDefault()
    val features: FeaturesConfig get() = rawConfig.featuresOrDefault()
    val indexUrl: String get() = envOverride("EPOS_INDEX_URL") ?: rawConfig.indexUrlOrDefault()
    val baseUri: String get() = envOverride("EPOS_BASE_URI") ?: rawConfig.baseUriOrDefault()

    val agents: ResolvedAgents by lazy { ResolvedAgents(rawConfig.agentsOrDefault(), baseUri) }

    val NEO4J_URI: String get() = envOverride("EPOS_NEO4J_URI") ?: neo4j.uri
    val NEO4J_USERNAME: String get() = envOverride("EPOS_NEO4J_USERNAME") ?: neo4j.username
    val NEO4J_PASSWORD: String get() = envOverride("EPOS_NEO4J_PASSWORD") ?: neo4j.password
    val NEO4J_DATABASE: String get() = envOverride("EPOS_NEO4J_DATABASE") ?: "neo4j"
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
            current =
                when (current) {
                    is Map<*, *> -> current[part]
                    else -> null
                }
            if (current == null) break
        }
        return current
    }

    fun <T> getOrDefault(
        key: String,
        default: T,
    ): T {
        @Suppress("UNCHECKED_CAST")
        return (this[key] as? T) ?: default
    }

    private fun loadConfiguration(): AppConfig {
        val profile =
            System.getenv("EPOS_PROFILE")
                ?: System.getProperty("epos.profile")
                ?: "default"

        println("Loading configuration with profile: $profile")

        val baseConfig =
            loadConfigFile("application.yaml")
                ?: loadConfigFromClasspath("application.yaml")
                ?: AppConfig()

        val profileConfig =
            if (profile != "default") {
                loadConfigFile("application-$profile.yaml")
                    ?: loadConfigFromClasspath("application-$profile.yaml")
            } else {
                null
            }

        val mergedConfig =
            if (profileConfig != null) {
                mergeConfigs(baseConfig, profileConfig)
            } else {
                baseConfig
            }

        return applyEnvironmentOverrides(mergedConfig)
    }

    private fun loadConfigFile(filename: String): AppConfig? {
        val paths =
            listOfNotNull(
                System.getenv("EPOS_CONFIG_DIR")?.let { "$it/$filename" },
                System.getProperty("epos.config.dir")?.let { "$it/$filename" },
                "config/$filename",
                filename,
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

    private fun mergeConfigs(
        base: AppConfig,
        profile: AppConfig,
    ): AppConfig {
        val baseNeo4j = base.neo4jOrDefault()
        val profileNeo4j = profile.neo4jOrDefault()
        val baseAgents = base.agentsOrDefault()
        val profileAgents = profile.agentsOrDefault()
        val baseServer = base.serverOrDefault()
        val profileServer = profile.serverOrDefault()
        val baseLogging = base.loggingOrDefault()
        val profileLogging = profile.loggingOrDefault()
        val baseFeatures = base.featuresOrDefault()
        val profileFeatures = profile.featuresOrDefault()
        val defaultNeo4j = Neo4jConfig()
        val defaultAgents = AgentsConfig()
        val defaultServer = ServerConfig()
        val defaultLogging = LoggingConfig()

        return AppConfig(
            neo4j =
                Neo4jConfig(
                    uri = if (profile.neo4j != null && profileNeo4j.uri != defaultNeo4j.uri) profileNeo4j.uri else baseNeo4j.uri,
                    username =
                        if (profile.neo4j != null &&
                            profileNeo4j.username != defaultNeo4j.username
                        ) {
                            profileNeo4j.username
                        } else {
                            baseNeo4j.username
                        },
                    password =
                        if (profile.neo4j != null &&
                            profileNeo4j.password != defaultNeo4j.password
                        ) {
                            profileNeo4j.password
                        } else {
                            baseNeo4j.password
                        },
                    maxConnectionPoolSize =
                        if (profile.neo4j != null &&
                            profileNeo4j.maxConnectionPoolSize != defaultNeo4j.maxConnectionPoolSize
                        ) {
                            profileNeo4j.maxConnectionPoolSize
                        } else {
                            baseNeo4j.maxConnectionPoolSize
                        },
                    connectionTimeoutMs =
                        if (profile.neo4j != null &&
                            profileNeo4j.connectionTimeoutMs != defaultNeo4j.connectionTimeoutMs
                        ) {
                            profileNeo4j.connectionTimeoutMs
                        } else {
                            baseNeo4j.connectionTimeoutMs
                        },
                ),
            indexUrl = profile.indexUrl ?: base.indexUrl,
            baseUri = profile.baseUri ?: base.baseUri,
            agents =
                AgentsConfig(
                    system =
                        if (profile.agents != null &&
                            profileAgents.system != defaultAgents.system
                        ) {
                            profileAgents.system
                        } else {
                            baseAgents.system
                        },
                    initialPopulation =
                        if (profile.agents != null &&
                            profileAgents.initialPopulation != defaultAgents.initialPopulation
                        ) {
                            profileAgents.initialPopulation
                        } else {
                            baseAgents.initialPopulation
                        },
                ),
            server =
                ServerConfig(
                    host = if (profile.server != null && profileServer.host != defaultServer.host) profileServer.host else baseServer.host,
                    port = if (profile.server != null && profileServer.port != defaultServer.port) profileServer.port else baseServer.port,
                    contextPath =
                        if (profile.server != null &&
                            profileServer.contextPath != defaultServer.contextPath
                        ) {
                            profileServer.contextPath
                        } else {
                            baseServer.contextPath
                        },
                    publicBaseUrl = if (profile.server?.publicBaseUrl != null) profileServer.publicBaseUrl else baseServer.publicBaseUrl,
                ),
            logging =
                LoggingConfig(
                    level =
                        if (profile.logging != null &&
                            profileLogging.level != defaultLogging.level
                        ) {
                            profileLogging.level
                        } else {
                            baseLogging.level
                        },
                    format =
                        if (profile.logging != null &&
                            profileLogging.format != defaultLogging.format
                        ) {
                            profileLogging.format
                        } else {
                            baseLogging.format
                        },
                    includeTimestamp = if (profile.logging != null) profileLogging.includeTimestamp else baseLogging.includeTimestamp,
                ),
            features =
                FeaturesConfig(
                    enableSwagger = if (profile.features != null) profileFeatures.enableSwagger else baseFeatures.enableSwagger,
                    enableCors = if (profile.features != null) profileFeatures.enableCors else baseFeatures.enableCors,
                    enableMetrics = if (profile.features != null) profileFeatures.enableMetrics else baseFeatures.enableMetrics,
                ),
            custom = base.customOrDefault() + profile.customOrDefault(),
        )
    }

    private fun applyEnvironmentOverrides(config: AppConfig): AppConfig {
        val neo4jUri = envOverride("EPOS_NEO4J_URI")
        val neo4jPassword = envOverride("EPOS_NEO4J_PASSWORD")
        val neo4jUser = envOverride("EPOS_NEO4J_USERNAME")
        val index = envOverride("EPOS_INDEX_URL")
        val base = envOverride("EPOS_BASE_URI")
        val serverPort =
            envOverride("EPOS_SERVER_PORT")?.toIntOrNull()
                ?: envOverride("PORT")?.toIntOrNull()
        val serverHost =
            envOverride("EPOS_SERVER_HOST")
                ?: envOverride("HOST")
        val publicBaseUrl = envOverride("EPOS_PUBLIC_BASE_URL")

        if (listOfNotNull(neo4jUri, neo4jPassword, neo4jUser, index, base, serverPort, serverHost, publicBaseUrl).isEmpty()) {
            return config
        }

        val currentNeo4j = config.neo4jOrDefault()
        val currentServer = config.serverOrDefault()

        return config.copy(
            neo4j =
                currentNeo4j.copy(
                    uri = neo4jUri ?: currentNeo4j.uri,
                    password = neo4jPassword ?: currentNeo4j.password,
                    username = neo4jUser ?: currentNeo4j.username,
                ),
            indexUrl = index ?: config.indexUrl,
            baseUri = base ?: config.baseUri,
            server =
                currentServer.copy(
                    port = serverPort ?: currentServer.port,
                    host = serverHost ?: currentServer.host,
                    publicBaseUrl = publicBaseUrl ?: currentServer.publicBaseUrl,
                ),
        )
    }

    private fun envOverride(key: String): String? = System.getenv(key)?.takeIf { it.isNotBlank() }

    fun printConfig() {
        println(
            """
            |Configuration Summary
            |=====================
            | Neo4j
            |   URI:      $NEO4J_URI
            |   Username: $NEO4J_USERNAME
            |   Database: $NEO4J_DATABASE
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
            """.trimMargin(),
        )
    }

    fun toMap(): Map<String, Any> =
        mapOf(
            "neo4j" to
                mapOf(
                    "uri" to NEO4J_URI,
                    "database" to NEO4J_DATABASE,
                ),
            "uris" to
                mapOf(
                    "baseUri" to BASE_URI,
                    "indexUrl" to INDEX_URL,
                    "idBase" to ID_BASE,
                ),
            "agents" to
                mapOf(
                    "system" to AGENT_SYSTEM,
                    "initialPopulation" to AGENT_INITIAL_POPULATION,
                ),
            "server" to
                mapOf(
                    "host" to server.host,
                    "port" to server.port,
                    "contextPath" to server.contextPath,
                ),
            "features" to
                mapOf(
                    "swagger" to features.enableSwagger,
                    "cors" to features.enableCors,
                    "metrics" to features.enableMetrics,
                ),
        )

}
