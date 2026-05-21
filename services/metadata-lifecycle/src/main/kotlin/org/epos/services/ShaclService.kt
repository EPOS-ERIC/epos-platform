package org.epos.services

import kotlinx.coroutines.runBlocking
import org.eclipse.rdf4j.model.Model
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.shacl.PropertyInfo
import org.epos.api.shacl.ShaclParser
import org.epos.propertygraph.Config
import org.slf4j.LoggerFactory
import java.io.StringReader
import java.net.URL
import java.nio.file.Files
import java.nio.file.Path

class ShaclService(
    private val shaclParser: ShaclParser,
) {
    companion object {
        private val logger = LoggerFactory.getLogger(ShaclService::class.java)
    }

    @Volatile
    private var resourceTypes: List<ResourceTypeInfo> = emptyList()

    @Volatile
    private var initialized: Boolean = false

    fun getResourceTypes(): List<ResourceTypeInfo> {
        if (!initialized) {
            runBlocking { initialize() }
        }
        return resourceTypes
    }

    suspend fun initialize(shaclUrl: String? = null) {
        if (initialized && resourceTypes.isNotEmpty()) {
            logger.info("ShaclService already initialized with {} resource types", resourceTypes.size)
            return
        }

        logger.info("Initializing ShaclService")
        val initStartedAt = System.currentTimeMillis()

        try {
            val localShaclFile = resolveLocalShaclFile()
            if (!localShaclFile.isNullOrBlank()) {
                val localStartedAt = System.currentTimeMillis()
                logger.info("SHACL dev mode: loading local SHACL file {}", localShaclFile)
                val ttl = Files.readString(Path.of(localShaclFile))
                val baseUri = Path.of(localShaclFile).toUri().toString()
                val shapesGraph = shaclParser.parseShapesGraphFromString(ttl, baseUri = baseUri)
                resourceTypes = shaclParser.toResourceTypes(shapesGraph)
                initialized = true
                logger.info(
                    "ShaclService initialized from local SHACL file with {} resource types in {} ms",
                    resourceTypes.size,
                    System.currentTimeMillis() - localStartedAt,
                )
                return
            }

            val shaclUrlResolved = shaclUrl ?: System.getenv("EPOS_SHACL_URL")
            if (!shaclUrlResolved.isNullOrBlank()) {
                val urlStartedAt = System.currentTimeMillis()
                logger.info("SHACL loading from URL: {}", shaclUrlResolved)
                val ttl = URL(shaclUrlResolved).readText()
                val shapesGraph = shaclParser.parseShapesGraphFromString(ttl)
                resourceTypes = shaclParser.toResourceTypes(shapesGraph)
                initialized = true
                logger.info(
                    "ShaclService initialized from URL with {} resource types in {} ms",
                    resourceTypes.size,
                    System.currentTimeMillis() - urlStartedAt,
                )
                return
            }

            logger.warn("No SHACL source configured. Set EPOS_SHACL_FILE, EPOS_SHACL_URL, or custom.shacl.file")
            resourceTypes = emptyList()
            initialized = true
        } catch (e: Exception) {
            logger.error("Failed to initialize ShaclService", e)
            throw e
        }
    }

    suspend fun refresh(shaclUrl: String? = null) {
        initialized = false
        resourceTypes = emptyList()
        initialize(shaclUrl)
    }

    private fun resolveLocalShaclFile(): String? {
        val envFile = System.getenv("EPOS_SHACL_FILE")
        if (!envFile.isNullOrBlank() && Files.exists(Path.of(envFile))) {
            return envFile
        }

        val configFile = Config["shacl.file"] as? String
        if (!configFile.isNullOrBlank() && Files.exists(Path.of(configFile))) {
            return configFile
        }

        val devFile = Config["custom.shacl.file"] as? String
        if (!devFile.isNullOrBlank() && Files.exists(Path.of(devFile))) {
            return devFile
        }

        val defaultCandidates =
            listOf(
                "/workspace/packages/shacl/shacl.ttl",
                "../../packages/shacl/shacl.ttl",
                "../packages/shacl/shacl.ttl",
                "packages/shacl/shacl.ttl",
            )

        defaultCandidates.firstOrNull { Files.exists(Path.of(it)) }?.let { return it }

        return null
    }

    fun isValidResourceType(name: String): Boolean = getResourceTypes().any { it.name.equals(name, ignoreCase = true) }

    fun getResourceType(name: String): ResourceTypeInfo? = getResourceTypes().find { it.name.equals(name, ignoreCase = true) }

    fun getTargetClass(resourceTypeName: String): String? = getResourceType(resourceTypeName)?.targetClass

    fun getParser(): ShaclParser = shaclParser

    fun isInitialized(): Boolean = initialized
}
