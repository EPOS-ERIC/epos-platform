package org.epos.api.shacl

import org.slf4j.LoggerFactory

/**
 * =============================================================================
 * Generalized Relationship Graph
 * =============================================================================
 *
 * A flexible, configurable graph for discovering and traversing relationships
 * between RDF resource types. Can be built from:
 * - Actual data in a SPARQL endpoint
 * - SHACL shapes
 * - Manual configuration
 *
 * Features:
 * - Configurable SPARQL queries
 * - Pluggable path scoring strategies
 * - Customizable filters and exclusions
 * - Multiple data source support
 */

// =============================================================================
// Data Classes
// =============================================================================

/**
 * Represents a property relationship between two resource types
 */
data class PropertyRelationship(
    val propertyUri: String,
    val propertyName: String,
    val sourceType: String,
    val sourceTypeName: String,
    val targetClass: String,
    val targetClassName: String,
    val isInverse: Boolean = false,
    val metadata: Map<String, Any> = emptyMap()  // Extensible metadata
)

/**
 * Represents a path between two resource types through multiple steps
 */
data class RelationshipPath(
    val fromType: String,
    val fromTypeName: String,
    val toType: String,
    val toTypeName: String,
    val steps: List<PathStep>,
    val sparqlPropertyPath: String,
    val score: Double = 0.0
) {
    val length: Int get() = steps.size

    val pathDescription: String
        get() = steps.joinToString(" → ") {
            if (it.isInverse) "←${it.propertyName}" else it.propertyName
        }

    val typePathDescription: String
        get() = fromTypeName + " → " + steps.joinToString(" → ") { it.toTypeName }
}

/**
 * Represents a single step in a relationship path
 */
data class PathStep(
    val propertyUri: String,
    val propertyName: String,
    val fromType: String,
    val fromTypeName: String,
    val toType: String,
    val toTypeName: String,
    val isInverse: Boolean = false
) {
    fun toSparql(): String = if (isInverse) "^<$propertyUri>" else "<$propertyUri>"
}

// =============================================================================
// Configuration
// =============================================================================

/**
 * Configuration for relationship graph building and querying
 */
data class RelationshipGraphConfig(
    // Query configuration
    val excludedProperties: Set<String> = DEFAULT_EXCLUDED_PROPERTIES,
    val excludedTypes: Set<String> = DEFAULT_EXCLUDED_TYPES,
    val datatypePatterns: Set<String> = DEFAULT_DATATYPE_PATTERNS,
    val maxQueryResults: Int = 10000,

    // Path finding configuration
    val maxPathDepth: Int = 5,
    val maxPathsToConsider: Int = 100,
    val includeBidirectional: Boolean = true,

    // Scoring configuration
    val scoringStrategy: PathScoringStrategy = DefaultPathScoringStrategy(),

    // Debug configuration
    val verbose: Boolean = false
) {
    companion object {
        val DEFAULT_EXCLUDED_PROPERTIES = setOf(
            "http://www.w3.org/1999/02/22-rdf-syntax-ns#type",
            "http://purl.org/dc/terms/isVersionOf",
            "http://purl.org/pav/version",
            "http://www.w3.org/ns/prov#wasGeneratedBy",
            "http://www.w3.org/ns/prov#wasRevisionOf",
            "http://www.w3.org/2002/07/owl#sameAs",
            "http://www.w3.org/2000/01/rdf-schema#seeAlso"
        )

        val DEFAULT_EXCLUDED_TYPES = setOf(
            "http://www.w3.org/2002/07/owl#Thing",
            "http://www.w3.org/2000/01/rdf-schema#Resource"
        )

        val DEFAULT_DATATYPE_PATTERNS = setOf(
            "XMLSchema#",
            "datatype",
            "Literal",
            "/xsd#",
            "/xsd/"
        )

        /** Default configuration */
        val DEFAULT = RelationshipGraphConfig()

        /** Configuration optimized for EPOS/DCAT data */
        val EPOS = RelationshipGraphConfig(
            excludedProperties = DEFAULT_EXCLUDED_PROPERTIES + setOf(
                "http://purl.org/dc/terms/identifier",
                "http://purl.org/dc/terms/created",
                "http://purl.org/dc/terms/modified"
            ),
            scoringStrategy = EposPathScoringStrategy()
        )
    }
}

// =============================================================================
// Scoring Strategies
// =============================================================================

/**
 * Strategy for scoring relationship paths
 */
interface PathScoringStrategy {
    fun score(path: List<PropertyRelationship>): Double

    /** Optional: provide hints about preferred properties */
    fun getPreferredProperties(): Set<String> = emptySet()

    /** Optional: provide hints about properties to avoid */
    fun getAvoidedProperties(): Set<String> = emptySet()
}

/**
 * Default scoring strategy - prefers shorter paths and forward relationships
 */
class DefaultPathScoringStrategy : PathScoringStrategy {
    override fun score(path: List<PropertyRelationship>): Double {
        var score = 100.0

        // Penalty for length
        score -= path.size * 10.0

        // Small penalty for inverse relationships
        path.forEach { rel ->
            if (rel.isInverse) score -= 2.0
        }

        return score
    }
}

/**
 * Configurable scoring strategy
 */
class ConfigurablePathScoringStrategy(
    val lengthPenalty: Double = 10.0,
    val inversePenalty: Double = 2.0,
    val preferredProperties: Map<String, Double> = emptyMap(),  // property pattern -> bonus
    val avoidedProperties: Map<String, Double> = emptyMap(),    // property pattern -> penalty
    val preferredIntermediateTypes: Map<String, Double> = emptyMap(),
    val avoidedIntermediateTypes: Map<String, Double> = emptyMap()
) : PathScoringStrategy {

    override fun score(path: List<PropertyRelationship>): Double {
        var score = 100.0

        // Length penalty
        score -= path.size * lengthPenalty

        path.forEachIndexed { index, rel ->
            // Inverse penalty
            if (rel.isInverse) score -= inversePenalty

            // Property bonuses/penalties
            preferredProperties.forEach { (pattern, bonus) ->
                if (rel.propertyName.contains(pattern, ignoreCase = true) ||
                    rel.propertyUri.contains(pattern, ignoreCase = true)) {
                    score += bonus
                }
            }

            avoidedProperties.forEach { (pattern, penalty) ->
                if (rel.propertyName.contains(pattern, ignoreCase = true) ||
                    rel.propertyUri.contains(pattern, ignoreCase = true)) {
                    score -= penalty
                }
            }

            // Intermediate type bonuses/penalties (not for final step)
            if (index < path.size - 1) {
                preferredIntermediateTypes.forEach { (pattern, bonus) ->
                    if (rel.targetClassName.contains(pattern, ignoreCase = true)) {
                        score += bonus
                    }
                }

                avoidedIntermediateTypes.forEach { (pattern, penalty) ->
                    if (rel.targetClassName.contains(pattern, ignoreCase = true)) {
                        score -= penalty
                    }
                }
            }
        }

        return score
    }

    override fun getPreferredProperties() = preferredProperties.keys
    override fun getAvoidedProperties() = avoidedProperties.keys
}

/**
 * EPOS-specific scoring strategy
 */
class EposPathScoringStrategy : PathScoringStrategy {

    private val semanticProperties = setOf(
        "theme", "category", "concept", "inScheme", "broader",
        "narrower", "distribution", "dataset", "accessService"
    )

    private val avoidedTypes = setOf(
        "contactpoint", "person", "postaladdress", "address"
    )

    private val preferredIntermediates = setOf(
        "dataset", "distribution", "webservice", "dataservice"
    )

    override fun score(path: List<PropertyRelationship>): Double {
        var score = 100.0

        // Length penalty
        score -= path.size * 10.0

        path.forEachIndexed { index, rel ->
            val propName = rel.propertyName.lowercase().removePrefix("^")
            val targetType = rel.targetClassName.lowercase()

            // Bonus for semantic relationships
            if (semanticProperties.any { propName.contains(it) }) {
                score += 20.0
            }

            // Heavy penalty for contactPoint paths
            if (propName.contains("contactpoint")) {
                score -= 50.0
            }

            // Penalty for going through unrelated types (not as final step)
            if (index < path.size - 1 && targetType in avoidedTypes) {
                score -= 30.0
            }

            // Bonus for common intermediate types
            if (targetType in preferredIntermediates) {
                score += 5.0
            }

            // Small penalty for inverse
            if (rel.isInverse) {
                score -= 2.0
            }
        }

        return score
    }

    override fun getPreferredProperties() = semanticProperties
    override fun getAvoidedProperties() = setOf("contactPoint")
}

// =============================================================================
// SPARQL Client Interface
// =============================================================================

/**
 * Interface for SPARQL query execution - implement this for your triplestore
 */
interface SparqlClient {
    /**
     * Execute a SELECT query and return results as a sequence of binding maps
     */
    fun query(sparql: String): Sequence<Map<String, String>>

    /**
     * Execute an ASK query
     */
    fun ask(sparql: String): Boolean = false
}

/**
 * Adapter for PropertyGraphClient to discover relationships from data
 */
class PropertyGraphClientAdapter(
    private val propertyGraphClient: org.epos.propertygraph.client.PropertyGraphClient
) : SparqlClient {
    override fun query(sparql: String): Sequence<Map<String, String>> {
        val results = propertyGraphClient.runQuery(
            """
            MATCH (source:Version)-[rel]->(target:Version)
            WHERE source.id <> target.id
            RETURN properties(source) AS source, type(rel) AS property, properties(target) AS target
            LIMIT 1000
            """.trimIndent(),
        )

        return results.map { row ->
            val source = row["source"] as? Map<*, *> ?: emptyMap<Any, Any>()
            val target = row["target"] as? Map<*, *> ?: emptyMap<Any, Any>()
            mapOf(
                "sourceType" to (((source["type"] as? List<*>)?.firstOrNull() ?: source["type"])?.toString() ?: ""),
                "property" to (row["property"] as? String ?: ""),
                "targetType" to (((target["type"] as? List<*>)?.firstOrNull() ?: target["type"])?.toString() ?: "")
            )
        }.asSequence()
    }
}

// =============================================================================
// Main Relationship Graph Class
// =============================================================================

/**
 * Generalized relationship graph with configurable behavior
 */
class RelationshipGraph(
    private val config: RelationshipGraphConfig = RelationshipGraphConfig.DEFAULT
) {
    private val logger = LoggerFactory.getLogger(javaClass)

    // Graph storage
    private val relationshipGraph = mutableMapOf<String, MutableList<PropertyRelationship>>()
    private val typeNameToUri = mutableMapOf<String, String>()
    private val typeUriToName = mutableMapOf<String, String>()

    // Statistics
    private var forwardCount = 0
    private var inverseCount = 0

    // =========================================================================
    // Building the Graph
    // =========================================================================

    /**
     * Build from a SPARQL endpoint using default query
     */
    fun buildFromSparql(
        client: SparqlClient,
        graphUri: String? = null,
        additionalFilters: String = "",
        clearExisting: Boolean = true
    ) {
        if (clearExisting) clear()

        val graphClause = if (graphUri != null) "FROM <$graphUri>" else ""
        val excludedPropsFilter = config.excludedProperties.joinToString(", ") { "<$it>" }

        val query = """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            
            SELECT DISTINCT ?sourceType ?property ?targetType
            $graphClause
            WHERE {
                ?source rdf:type ?sourceType ;
                        ?property ?target .
                ?target rdf:type ?targetType .
                
                FILTER(isIRI(?target))
                FILTER(isIRI(?sourceType))
                FILTER(isIRI(?targetType))
                FILTER(?property NOT IN ($excludedPropsFilter))
                $additionalFilters
            }
            LIMIT ${config.maxQueryResults}
        """.trimIndent()

        if (config.verbose) {
            logger.debug("Executing relationship discovery query...")
        }

        processQueryResults(client.query(query))
    }

    /**
     * Build from a SPARQL endpoint using custom query
     * Query must return ?sourceType, ?property, ?targetType bindings
     */
    fun buildFromCustomQuery(client: SparqlClient, customQuery: String, clearExisting: Boolean = true) {
        if (clearExisting) clear()
        processQueryResults(client.query(customQuery))
    }

    /**
     * Build from PropertyGraphClient (convenience method)
     */
    fun buildFromData(
        propertyGraphClient: org.epos.propertygraph.client.PropertyGraphClient,
        clearExisting: Boolean = true
    ) {
        buildFromSparql(
            PropertyGraphClientAdapter(propertyGraphClient),
            additionalFilters = "",
            clearExisting = clearExisting
        )
    }

    /**
     * Manually add a relationship
     */
    fun addRelationship(
        sourceTypeUri: String,
        propertyUri: String,
        targetTypeUri: String,
        bidirectional: Boolean = config.includeBidirectional
    ) {
        if (isDatatype(sourceTypeUri) || isDatatype(targetTypeUri)) return
        if (sourceTypeUri in config.excludedTypes || targetTypeUri in config.excludedTypes) return
        if (propertyUri in config.excludedProperties) return

        val sourceTypeName = extractTypeName(sourceTypeUri)
        val targetTypeName = extractTypeName(targetTypeUri)
        val propertyName = extractPropertyName(propertyUri)

        registerType(sourceTypeUri, sourceTypeName)
        registerType(targetTypeUri, targetTypeName)

        // Forward relationship
        addForwardRelationship(PropertyRelationship(
            propertyUri = propertyUri,
            propertyName = propertyName,
            sourceType = sourceTypeUri,
            sourceTypeName = sourceTypeName,
            targetClass = targetTypeUri,
            targetClassName = targetTypeName,
            isInverse = false
        ))

        // Inverse relationship
        if (bidirectional) {
            addInverseRelationship(PropertyRelationship(
                propertyUri = propertyUri,
                propertyName = "^$propertyName",
                sourceType = targetTypeUri,
                sourceTypeName = targetTypeName,
                targetClass = sourceTypeUri,
                targetClassName = sourceTypeName,
                isInverse = true
            ))
        }
    }

    /**
     * Build from SHACL shapes (schema-level relationships)
     */
    fun buildFromShacl(shapesGraph: ShaclShapesGraph, clearExisting: Boolean = true) {
        if (clearExisting) clear()

        val shapeById = shapesGraph.nodeShapes
            .filter { !it.id.isNullOrBlank() }
            .associateBy { it.id!! }

        for (nodeShape in shapesGraph.nodeShapes) {
            val sourceClass = nodeShape.targets
                .filterIsInstance<TargetClass>()
                .firstOrNull()?.targetClass
                ?: nodeShape.targets
                    .filterIsInstance<ImplicitClassTarget>()
                    .firstOrNull()?.classIri
                ?: continue

            for (propShape in nodeShape.properties) {
                val targetClasses = mutableSetOf<String>()
                targetClasses.addAll(propShape.nodeConstraints.classConstraint)

                propShape.nodeConstraints.node.forEach { shapeRef ->
                    val resolvedShape = shapeById[shapeRef]
                    if (resolvedShape != null) {
                        targetClasses.addAll(resolvedShape.targets.filterIsInstance<TargetClass>().map { it.targetClass })
                        targetClasses.addAll(resolvedShape.targets.filterIsInstance<ImplicitClassTarget>().map { it.classIri })
                    }
                }

                val pathVariants = extractPathVariants(propShape.path)
                if (pathVariants.isEmpty()) continue

                for ((propertyUri, isInversePath) in pathVariants) {
                    if (targetClasses.isEmpty()) {
                        shapesGraph.ontologySchema.properties[propertyUri]
                            ?.ranges
                            ?.filterNot { isDatatype(it) }
                            ?.forEach { targetClasses.add(it) }
                    }

                    if (targetClasses.isEmpty()) continue

                    for (targetClass in targetClasses) {
                        if (isInversePath) {
                            addRelationship(targetClass, propertyUri, sourceClass)
                        } else {
                            addRelationship(sourceClass, propertyUri, targetClass)
                        }
                    }
                }
            }
        }

        if (config.verbose) {
            logger.debug("Built relationship graph from SHACL: {} types, {} relationships", typeNameToUri.size, forwardCount + inverseCount)
        }
    }

    // =========================================================================
    // Path Finding
    // =========================================================================

    /**
     * Find the best path from one resource type to another
     */
    fun findPath(fromTypeName: String, toTypeName: String): RelationshipPath? {
        val fromTypeUri = typeNameToUri[fromTypeName.lowercase()]
        val toTypeUri = typeNameToUri[toTypeName.lowercase()]

        if (fromTypeUri == null || toTypeUri == null) {
            if (config.verbose) {
                logger.debug("Cannot find path: Unknown type(s) - source: {}, target: {}", fromTypeName, toTypeName)
            }
            return null
        }

        if (fromTypeUri == toTypeUri) return null

        val allPaths = findAllPathsBFS(fromTypeUri, toTypeUri)
        if (allPaths.isEmpty()) {
            if (config.verbose) {
                logger.debug("No path found from {} to {}", fromTypeName, toTypeName)
            }
            return null
        }

        // Score and sort paths
        val scoredPaths = allPaths
            .map { path -> path to config.scoringStrategy.score(path) }
            .sortedByDescending { it.second }

        if (config.verbose && allPaths.size > 1) {
            logger.debug("Found {} possible paths:", allPaths.size)
            scoredPaths.take(3).forEach { (path, score) ->
                val desc = path.joinToString(" -> ") {
                    "${it.sourceTypeName}.${it.propertyName}"
                } + " -> $toTypeName"
                logger.debug("  - [score: {:.1f}] {}", score, desc)
            }
        }

        val (bestPath, bestScore) = scoredPaths.first()
        return buildRelationshipPath(bestPath, fromTypeName, toTypeName, bestScore)
    }

    /**
     * Find all paths between two types (up to maxDepth)
     */
    fun findAllPaths(
        fromTypeName: String,
        toTypeName: String,
        maxDepth: Int = config.maxPathDepth
    ): List<RelationshipPath> {
        val fromTypeUri = typeNameToUri[fromTypeName.lowercase()] ?: return emptyList()
        val toTypeUri = typeNameToUri[toTypeName.lowercase()] ?: return emptyList()

        if (fromTypeUri == toTypeUri) return emptyList()

        return findAllPathsBFS(fromTypeUri, toTypeUri, maxDepth)
            .map { path ->
                val score = config.scoringStrategy.score(path)
                buildRelationshipPath(path, fromTypeName, toTypeName, score)
            }
            .sortedByDescending { it.score }
    }

    private fun findAllPathsBFS(
        fromType: String,
        toType: String,
        maxDepth: Int = config.maxPathDepth
    ): List<List<PropertyRelationship>> {
        val allPaths = mutableListOf<List<PropertyRelationship>>()
        val queue = ArrayDeque<Triple<String, List<PropertyRelationship>, Set<String>>>()

        queue.add(Triple(fromType, emptyList(), setOf(fromType)))

        while (queue.isNotEmpty() && allPaths.size < config.maxPathsToConsider) {
            val (currentType, path, visited) = queue.removeFirst()

            if (path.size >= maxDepth) continue

            val relationships = relationshipGraph[currentType] ?: emptyList()

            for (rel in relationships) {
                val targetType = rel.targetClass

                if (targetType == toType) {
                    allPaths.add(path + rel)
                } else if (targetType !in visited) {
                    queue.add(Triple(targetType, path + rel, visited + targetType))
                }
            }
        }

        return allPaths
    }

    // =========================================================================
    // Querying
    // =========================================================================

    /**
     * Get all direct relationships from a type
     */
    fun getOutgoingRelationships(typeName: String): List<PropertyRelationship> {
        val typeUri = typeNameToUri[typeName.lowercase()] ?: return emptyList()
        return relationshipGraph[typeUri] ?: emptyList()
    }

    /**
     * Get forward (non-inverse) relationships only
     */
    fun getForwardRelationships(typeName: String): List<PropertyRelationship> {
        return getOutgoingRelationships(typeName).filter { !it.isInverse }
    }

    /**
     * Get inverse relationships only
     */
    fun getInverseRelationships(typeName: String): List<PropertyRelationship> {
        return getOutgoingRelationships(typeName).filter { it.isInverse }
    }

    /**
     * Get type URI from type name
     */
    fun getTypeUri(typeName: String): String? = typeNameToUri[typeName.lowercase()]

    /**
     * Get type name from URI
     */
    fun getTypeName(typeUri: String): String? = typeUriToName[typeUri]

    /**
     * Get all known type names
     */
    fun getAllTypeNames(): Set<String> = typeNameToUri.keys.toSet()

    /**
     * Get all known type URIs
     */
    fun getAllTypeUris(): Set<String> = typeUriToName.keys.toSet()

    /**
     * Check if a type exists in the graph
     */
    fun hasType(typeName: String): Boolean = typeName.lowercase() in typeNameToUri

    /**
     * Get statistics about the graph
     */
    fun getStatistics(): GraphStatistics = GraphStatistics(
        typeCount = typeNameToUri.size,
        forwardRelationshipCount = forwardCount,
        inverseRelationshipCount = inverseCount,
        totalRelationshipCount = forwardCount + inverseCount
    )

    // =========================================================================
    // Internal Helpers
    // =========================================================================

    private fun processQueryResults(results: Sequence<Map<String, String>>) {
        results.forEach { row ->
            val sourceType = row["sourceType"] ?: return@forEach
            val property = row["property"] ?: return@forEach
            val targetType = row["targetType"] ?: return@forEach

            addRelationship(sourceType, property, targetType)
        }

        if (config.verbose) {
            logger.debug("Relationship graph built: {} types, {} forward, {} inverse",
                typeNameToUri.size, forwardCount, inverseCount)
        }
    }

    private fun registerType(uri: String, name: String) {
        typeNameToUri[name.lowercase()] = uri
        typeUriToName[uri] = name
    }

    private fun addForwardRelationship(rel: PropertyRelationship) {
        val existing = relationshipGraph.getOrPut(rel.sourceType) { mutableListOf() }
        if (existing.none { it.propertyUri == rel.propertyUri && it.targetClass == rel.targetClass && !it.isInverse }) {
            existing.add(rel)
            forwardCount++
        }
    }

    private fun addInverseRelationship(rel: PropertyRelationship) {
        val existing = relationshipGraph.getOrPut(rel.sourceType) { mutableListOf() }
        if (existing.none { it.propertyUri == rel.propertyUri && it.targetClass == rel.targetClass && it.isInverse }) {
            existing.add(rel)
            inverseCount++
        }
    }

    private fun buildRelationshipPath(
        relationships: List<PropertyRelationship>,
        fromTypeName: String,
        toTypeName: String,
        score: Double = 0.0
    ): RelationshipPath {
        val steps = relationships.map { rel ->
            PathStep(
                propertyUri = rel.propertyUri,
                propertyName = rel.propertyName.removePrefix("^"),
                fromType = rel.sourceType,
                fromTypeName = rel.sourceTypeName,
                toType = rel.targetClass,
                toTypeName = rel.targetClassName,
                isInverse = rel.isInverse
            )
        }

        val sparqlPath = steps.joinToString("/") { it.toSparql() }

        return RelationshipPath(
            fromType = relationships.first().sourceType,
            fromTypeName = fromTypeName,
            toType = relationships.last().targetClass,
            toTypeName = toTypeName,
            steps = steps,
            sparqlPropertyPath = sparqlPath,
            score = score
        )
    }

    private fun isDatatype(uri: String): Boolean {
        return config.datatypePatterns.any { uri.contains(it, ignoreCase = true) }
    }

    private fun extractPathVariants(path: PropertyPath): List<Pair<String, Boolean>> {
        return when (path) {
            is PredicatePath -> listOf(path.predicate to false)
            is InversePath -> when (val inner = path.inner) {
                is PredicatePath -> listOf(inner.predicate to true)
                else -> emptyList()
            }
            is AlternativePath -> path.paths.flatMap { extractPathVariants(it) }.distinct()
            is ZeroOrMorePath -> extractPathVariants(path.inner)
            is OneOrMorePath -> extractPathVariants(path.inner)
            is ZeroOrOnePath -> extractPathVariants(path.inner)
            else -> emptyList()
        }
    }

    private fun extractTypeName(uri: String): String {
        return uri.substringAfterLast("/").substringAfterLast("#").lowercase()
    }

    private fun extractPropertyName(uri: String): String {
        return uri.substringAfterLast("/").substringAfterLast("#")
    }

    /**
     * Clear all data
     */
    fun clear() {
        relationshipGraph.clear()
        typeNameToUri.clear()
        typeUriToName.clear()
        forwardCount = 0
        inverseCount = 0
    }

    // =========================================================================
    // Debug / Export
    // =========================================================================

    /**
     * Print the relationship graph
     */
    fun printGraph() {
        logger.info("Relationship Graph ({} types, {} relationships):",
            typeNameToUri.size, forwardCount + inverseCount)
        relationshipGraph.entries
            .sortedBy { typeUriToName[it.key] }
            .forEach { (sourceType, relationships) ->
                val typeName = typeUriToName[sourceType] ?: "unknown"
                val forward = relationships.filter { !it.isInverse }
                val inverse = relationships.filter { it.isInverse }
                if (forward.isNotEmpty() || inverse.isNotEmpty()) {
                    logger.debug("{}: {} forward, {} inverse", typeName, forward.size, inverse.size)
                }
            }
    }

    /**
     * Export as Mermaid diagram
     */
    fun toMermaid(): String = buildString {
        appendLine("graph LR")
        relationshipGraph.forEach { (sourceType, relationships) ->
            val sourceName = typeUriToName[sourceType] ?: return@forEach
            relationships.filter { !it.isInverse }.forEach { rel ->
                val targetName = rel.targetClassName
                val propName = rel.propertyName
                appendLine("    $sourceName -->|$propName| $targetName")
            }
        }
    }

    /**
     * Export as JSON-friendly map
     */
    fun toMap(): Map<String, Any> = mapOf(
        "types" to typeNameToUri,
        "relationships" to relationshipGraph.mapValues { (_, rels) ->
            rels.map { rel ->
                mapOf(
                    "property" to rel.propertyUri,
                    "propertyName" to rel.propertyName,
                    "target" to rel.targetClass,
                    "targetName" to rel.targetClassName,
                    "isInverse" to rel.isInverse
                )
            }
        },
        "statistics" to getStatistics()
    )
}

/**
 * Graph statistics
 */
data class GraphStatistics(
    val typeCount: Int,
    val forwardRelationshipCount: Int,
    val inverseRelationshipCount: Int,
    val totalRelationshipCount: Int
)

// =============================================================================
// Backwards Compatibility Alias
// =============================================================================

/**
 * Type alias for backwards compatibility with existing code
 */
typealias ShaclRelationshipGraph = RelationshipGraph
