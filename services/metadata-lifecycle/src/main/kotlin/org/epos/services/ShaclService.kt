package org.epos.services

import kotlinx.coroutines.runBlocking
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.shacl.ShaclParser
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient

/**
 * Service that manages SHACL-based resource type parsing and caching.
 * This eliminates the need to re-parse SHACL on every request.
 */
class ShaclService(
    private val shaclParser: ShaclParser,
    private val graphDBClient: GraphDBClient
) {
    companion object {
        private const val DEFAULT_SHACL_URL = 
            "https://raw.githubusercontent.com/epos-eu/EPOS-DCAT-AP/refs/heads/EPOS-DCAT-AP-shapes/epos-dcat-ap_shapes.ttl"
    }

    @Volatile
    private var resourceTypes: List<ResourceTypeInfo> = emptyList()

    @Volatile
    private var initialized: Boolean = false

    /**
     * Get the cached resource types, initializing if necessary.
     */
    fun getResourceTypes(): List<ResourceTypeInfo> {
        if (!initialized) {
            runBlocking { initialize() }
        }
        return resourceTypes
    }

    /**
     * Initialize the service by parsing SHACL and building the relationship graph.
     * This should be called once at application startup.
     */
    suspend fun initialize(shaclUrl: String = DEFAULT_SHACL_URL) {
        if (initialized && resourceTypes.isNotEmpty()) {
            println("ShaclService already initialized with ${resourceTypes.size} resource types")
            return
        }

        println("Initializing ShaclService from $shaclUrl...")
        
        try {
            resourceTypes = shaclParser.parseShacl(shaclUrl, graphDBClient)
            initialized = true
            println("ShaclService initialized with ${resourceTypes.size} resource types")
            
            // Log available types
            resourceTypes.forEach { type ->
                println("  - ${type.name} (${type.targetClass})")
            }
        } catch (e: Exception) {
            println("Failed to initialize ShaclService: ${e.message}")
            throw e
        }
    }

    /**
     * Force re-initialization (e.g., after SHACL schema changes)
     */
    suspend fun refresh(shaclUrl: String = DEFAULT_SHACL_URL) {
        initialized = false
        resourceTypes = emptyList()
        initialize(shaclUrl)
    }

    /**
     * Check if a resource type name is valid
     */
    fun isValidResourceType(name: String): Boolean {
        return getResourceTypes().any { it.name.equals(name, ignoreCase = true) }
    }

    /**
     * Get a specific resource type by name
     */
    fun getResourceType(name: String): ResourceTypeInfo? {
        return getResourceTypes().find { it.name.equals(name, ignoreCase = true) }
    }

    /**
     * Get the target class URI for a resource type name
     */
    fun getTargetClass(resourceTypeName: String): String? {
        return getResourceType(resourceTypeName)?.targetClass
    }

    /**
     * Get the underlying SHACL parser for advanced operations
     */
    fun getParser(): ShaclParser = shaclParser

    /**
     * Check if the service is initialized
     */
    fun isInitialized(): Boolean = initialized
}
