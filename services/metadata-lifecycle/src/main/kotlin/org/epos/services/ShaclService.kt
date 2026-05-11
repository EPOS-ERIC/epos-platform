package org.epos.services

import kotlinx.coroutines.runBlocking
import org.eclipse.rdf4j.model.Model
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.shacl.PropertyInfo
import org.epos.api.shacl.ShaclParser
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.slf4j.LoggerFactory
import java.nio.file.Files
import java.nio.file.Path

/**
 * Service that manages SHACL-based resource type parsing and caching.
 * This eliminates the need to re-parse SHACL on every request.
 */
class ShaclService(
    private val shaclParser: ShaclParser,
    private val graphDBClient: GraphDBClient,
) {
    companion object {
        private val logger = LoggerFactory.getLogger(ShaclService::class.java)
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
    suspend fun initialize(shaclUrl: String? = null) {
        if (initialized && resourceTypes.isNotEmpty()) {
            logger.info("ShaclService already initialized with {} resource types", resourceTypes.size)
            return
        }

        logger.info("Initializing ShaclService from GraphDB SHACL graph")
        val initStartedAt = System.currentTimeMillis()

        try {
            val localShaclFile = resolveLocalShaclFile()
            if (!localShaclFile.isNullOrBlank()) {
                val localStartedAt = System.currentTimeMillis()
                logger.info("SHACL dev mode: loading local SHACL file {}", localShaclFile)
                val ttl = Files.readString(Path.of(localShaclFile))
                val shapesGraph = shaclParser.parseShapesGraphFromString(ttl)
                resourceTypes = shaclParser.toResourceTypes(shapesGraph)
                initialized = true
                logger.info(
                    "ShaclService initialized from local SHACL file with {} resource types in {} ms",
                    resourceTypes.size,
                    System.currentTimeMillis() - localStartedAt,
                )
                return
            }

            val configuredGraph = resolveConfiguredShaclGraph(shaclUrl)
            val fastInit = (Config["shacl.fastInit"] as? Boolean) ?: true
            if (fastInit && !configuredGraph.isNullOrBlank()) {
                val fastStartedAt = System.currentTimeMillis()
                logger.info("SHACL fast init enabled, reading NodeShapes from configured graph <{}>", configuredGraph)
                resourceTypes = loadResourceTypesFast(configuredGraph)
                initialized = true
                logger.info(
                    "ShaclService fast-initialized with {} resource types in {} ms",
                    resourceTypes.size,
                    System.currentTimeMillis() - fastStartedAt,
                )
                return
            }

            val loadStartedAt = System.currentTimeMillis()
            val model = loadShaclModelFromGraphDb(shaclUrl)
            logger.info(
                "SHACL model loaded ({} triples) in {} ms",
                model.size,
                System.currentTimeMillis() - loadStartedAt,
            )

            val parseStartedAt = System.currentTimeMillis()
            val shapesGraph = shaclParser.parseShapesGraphFromModel(model)
            logger.info(
                "SHACL parsed {} node shapes in {} ms",
                shapesGraph.nodeShapes.size,
                System.currentTimeMillis() - parseStartedAt,
            )

            val relFromShaclStartedAt = System.currentTimeMillis()
            shaclParser.relationshipGraph.buildFromShacl(shapesGraph)
            logger.info(
                "SHACL relationship graph built from shapes in {} ms",
                System.currentTimeMillis() - relFromShaclStartedAt,
            )

            val enrichFromData = (Config["shacl.enrichRelationshipGraphFromData"] as? Boolean) ?: false
            if (enrichFromData) {
                val relFromDataStartedAt = System.currentTimeMillis()
                logger.info("SHACL relationship graph enrichment from published data graph started")
                shaclParser.relationshipGraph.buildFromData(graphDBClient, Config.GRAPH_PUBLISHED, clearExisting = false)
                logger.info(
                    "SHACL relationship graph enrichment completed in {} ms",
                    System.currentTimeMillis() - relFromDataStartedAt,
                )
            }

            val convertStartedAt = System.currentTimeMillis()
            resourceTypes = shaclParser.toResourceTypes(shapesGraph)
            logger.info(
                "SHACL converted to {} resource types in {} ms",
                resourceTypes.size,
                System.currentTimeMillis() - convertStartedAt,
            )

            initialized = true
            logger.info(
                "ShaclService initialized with {} resource types in {} ms",
                resourceTypes.size,
                System.currentTimeMillis() - initStartedAt,
            )

            // Log available types
            resourceTypes.forEach { type ->
                logger.debug("SHACL type: {} ({})", type.name, type.targetClass)
            }
        } catch (e: Exception) {
            logger.error("Failed to initialize ShaclService", e)
            throw e
        }
    }

    /**
     * Force re-initialization (e.g., after SHACL schema changes)
     */
    suspend fun refresh(shaclUrl: String? = null) {
        initialized = false
        resourceTypes = emptyList()
        initialize(shaclUrl)
    }

    private fun loadShaclModelFromGraphDb(configuredGraph: String?): Model {
        val graphUri = resolveConfiguredShaclGraph(configuredGraph)

        if (!graphUri.isNullOrBlank()) {
            logger.info("SHACL loading model from configured graph <{}>", graphUri)
            return graphDBClient.construct(
                """
                CONSTRUCT { ?s ?p ?o }
                FROM <$graphUri>
                WHERE { ?s ?p ?o }
                """.trimIndent(),
            )
        }

        val allowGlobalDiscovery = (Config["shacl.allowGlobalDiscovery"] as? Boolean) ?: false
        val candidateGraphs =
            listOf(
                "${Config.BASE_URI}/graphs/shacl",
                "${Config.BASE_URI}/graphs/schema/shacl",
                "${Config.BASE_URI}/graphs/shapes",
            )

        logger.info("SHACL graph not configured, probing known candidate graphs")
        val detectedCandidates = candidateGraphs.filter { hasNodeShapes(it) }
        if (detectedCandidates.isNotEmpty()) {
            val fromClauses = detectedCandidates.joinToString("\n") { "FROM <$it>" }
            logger.info("SHACL using detected candidate graph(s): {}", detectedCandidates.joinToString())
            return graphDBClient.construct(
                """
                CONSTRUCT { ?s ?p ?o }
                $fromClauses
                WHERE { ?s ?p ?o }
                """.trimIndent(),
            )
        }

        if (!allowGlobalDiscovery) {
            throw IllegalStateException(
                "SHACL graph is not configured and global discovery is disabled. " +
                    "Set EPOS_SHACL_GRAPH or custom.shacl.graph, " +
                    "or enable custom.shacl.allowGlobalDiscovery=true"
            )
        }

        logger.warn("SHACL running global graph discovery for sh:NodeShape (this can be slow)")
        val discoveryStartedAt = System.currentTimeMillis()
        val discoveredGraphs =
            graphDBClient
                .query(
                    """
                    PREFIX sh: <http://www.w3.org/ns/shacl#>
                    SELECT DISTINCT ?g
                    WHERE {
                        GRAPH ?g {
                            ?shape a sh:NodeShape .
                        }
                    }
                    """.trimIndent(),
                ).mapNotNull { it["g"] }

        if (discoveredGraphs.isEmpty()) {
            throw IllegalStateException("No SHACL graph found in GraphDB. Configure EPOS_SHACL_GRAPH or custom.shacl.graph")
        }

        logger.info(
            "SHACL discovered {} graph(s) with NodeShapes in {} ms",
            discoveredGraphs.size,
            System.currentTimeMillis() - discoveryStartedAt,
        )
        val fromClauses = discoveredGraphs.joinToString("\n") { "FROM <$it>" }
        return graphDBClient.construct(
            """
            CONSTRUCT { ?s ?p ?o }
            $fromClauses
            WHERE { ?s ?p ?o }
            """.trimIndent(),
        )
    }

    private fun resolveConfiguredShaclGraph(shaclUrl: String?): String? {
        return shaclUrl ?: System.getenv("EPOS_SHACL_GRAPH") ?: (Config["shacl.graph"] as? String)
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

        return null
    }

    private fun loadResourceTypesFast(graphUri: String): List<ResourceTypeInfo> {
        val shapesRows =
            graphDBClient.query(
                """
                PREFIX sh: <http://www.w3.org/ns/shacl#>
                SELECT ?shape ?targetClass ?shapeDescription
                FROM <$graphUri>
                WHERE {
                  ?shape a sh:NodeShape ;
                         sh:targetClass ?targetClass .
                  OPTIONAL { ?shape sh:description ?shapeDescription }
                }
                """.trimIndent(),
            )

        if (shapesRows.isEmpty()) {
            logger.warn("SHACL fast init found no sh:NodeShape with sh:targetClass in graph <{}>", graphUri)
            return emptyList()
        }

        val propertyRows =
            graphDBClient.query(
                """
                PREFIX sh: <http://www.w3.org/ns/shacl#>
                SELECT ?shape ?path ?name ?description ?datatype ?classConstraint ?nodeKind ?minCount ?maxCount
                FROM <$graphUri>
                WHERE {
                  ?shape a sh:NodeShape ;
                         sh:property ?propertyShape .
                  ?propertyShape sh:path ?path .
                  FILTER(isIRI(?path))
                  OPTIONAL { ?propertyShape sh:name ?name }
                  OPTIONAL { ?propertyShape sh:description ?description }
                  OPTIONAL { ?propertyShape sh:datatype ?datatype }
                  OPTIONAL { ?propertyShape sh:class ?classConstraint }
                  OPTIONAL { ?propertyShape sh:nodeKind ?nodeKind }
                  OPTIONAL { ?propertyShape sh:minCount ?minCount }
                  OPTIONAL { ?propertyShape sh:maxCount ?maxCount }
                }
                """.trimIndent(),
            )

        val propertiesByShape =
            propertyRows.groupBy { it["shape"].orEmpty() }
                .mapValues { (_, rows) ->
                    rows.groupBy { it["path"].orEmpty() }
                        .mapNotNull { (path, variants) ->
                            if (path.isBlank()) return@mapNotNull null
                            val name = variants.firstNotNullOfOrNull { it["name"]?.takeIf { v -> v.isNotBlank() } }
                                ?: localName(path)
                            val description = variants.firstNotNullOfOrNull { it["description"]?.takeIf { v -> v.isNotBlank() } }
                            val datatypes = variants.mapNotNull { it["datatype"] }.distinct()
                            val classConstraints = variants.mapNotNull { it["classConstraint"] }.distinct()
                            val nodeKinds = variants.mapNotNull { it["nodeKind"] }.distinct()
                            val minCount = variants.mapNotNull { it["minCount"]?.toIntOrNull() }.maxOrNull()
                            val maxCount = variants.mapNotNull { it["maxCount"]?.toIntOrNull() }.minOrNull()

                            PropertyInfo(
                                path = path,
                                name = name,
                                datatype = datatypes.singleOrNull(),
                                datatypes = datatypes.takeIf { it.size > 1 },
                                nodeKind = nodeKinds.singleOrNull(),
                                nodeKinds = nodeKinds.takeIf { it.size > 1 },
                                classConstraint = classConstraints.singleOrNull(),
                                classConstraints = classConstraints.takeIf { it.size > 1 },
                                minCount = minCount,
                                maxCount = maxCount,
                                required = (minCount ?: 0) > 0,
                                description = description,
                            )
                        }
                }

        return shapesRows
            .groupBy { it["shape"].orEmpty() }
            .mapNotNull { (shapeIri, rows) ->
                val targetClass = rows.firstNotNullOfOrNull { it["targetClass"]?.takeIf { v -> v.isNotBlank() } } ?: return@mapNotNull null
                val description = rows.firstNotNullOfOrNull { it["shapeDescription"]?.takeIf { v -> v.isNotBlank() } }
                ResourceTypeInfo(
                    name = localName(targetClass).lowercase(),
                    targetClass = targetClass,
                    properties = propertiesByShape[shapeIri].orEmpty(),
                    closed = false,
                    description = description,
                )
            }
            .sortedBy { it.name }
    }

    private fun localName(iri: String): String {
        return iri.substringAfterLast('/').substringAfterLast('#').ifBlank { iri }
    }

    private fun hasNodeShapes(graphUri: String): Boolean {
        return try {
            graphDBClient.ask(
                """
                PREFIX sh: <http://www.w3.org/ns/shacl#>
                ASK {
                  GRAPH <$graphUri> {
                    ?shape a sh:NodeShape .
                  }
                }
                """.trimIndent(),
            )
        } catch (_: Exception) {
            false
        }
    }

    /**
     * Check if a resource type name is valid
     */
    fun isValidResourceType(name: String): Boolean = getResourceTypes().any { it.name.equals(name, ignoreCase = true) }

    /**
     * Get a specific resource type by name
     */
    fun getResourceType(name: String): ResourceTypeInfo? = getResourceTypes().find { it.name.equals(name, ignoreCase = true) }

    /**
     * Get the target class URI for a resource type name
     */
    fun getTargetClass(resourceTypeName: String): String? = getResourceType(resourceTypeName)?.targetClass

    /**
     * Get the underlying SHACL parser for advanced operations
     */
    fun getParser(): ShaclParser = shaclParser

    /**
     * Check if the service is initialized
     */
    fun isInitialized(): Boolean = initialized
}
