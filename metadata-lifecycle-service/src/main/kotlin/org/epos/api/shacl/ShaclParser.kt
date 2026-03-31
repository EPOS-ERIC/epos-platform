package org.epos.api.shacl

import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import com.fasterxml.jackson.annotation.JsonInclude
import org.eclipse.rdf4j.model.*
import org.eclipse.rdf4j.model.impl.SimpleValueFactory
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.model.vocabulary.RDFS
import org.eclipse.rdf4j.model.vocabulary.SHACL
import org.eclipse.rdf4j.model.vocabulary.XSD
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.models.ShapeDefinitionInfo
import java.io.StringReader

// =============================================================================
// SHACL Data Model - Comprehensive representation of SHACL constructs
// =============================================================================

/**
 * Represents a complete SHACL shapes graph
 */
data class ShaclShapesGraph(
    val shapes: List<Shape>,
    val prefixes: Map<String, String> = emptyMap()
) {
    /** Get all node shapes */
    val nodeShapes: List<NodeShape> get() = shapes.filterIsInstance<NodeShape>()

    /** Get all property shapes */
    val propertyShapes: List<PropertyShape> get() = shapes.filterIsInstance<PropertyShape>()

    /** Find shape by IRI */
    fun findShape(iri: String): Shape? = shapes.find { it.id == iri }

    /** Find shapes targeting a specific class */
    fun shapesForClass(classIri: String): List<NodeShape> =
        nodeShapes.filter { shape ->
            shape.targets.any { it is TargetClass && it.targetClass == classIri }
        }
}

/**
 * Base class for all SHACL shapes
 */
sealed class Shape {
    abstract val id: String?
    abstract val targets: List<Target>
    abstract val severity: Severity
    abstract val deactivated: Boolean
    abstract val message: Map<String, String>  // Language-tagged messages
    abstract val name: Map<String, String>
    abstract val description: Map<String, String>
    abstract val group: String?
    abstract val order: Double?
}

/**
 * Node Shape - validates focus nodes
 */
data class NodeShape(
    override val id: String?,
    override val targets: List<Target> = emptyList(),
    override val severity: Severity = Severity.VIOLATION,
    override val deactivated: Boolean = false,
    override val message: Map<String, String> = emptyMap(),
    override val name: Map<String, String> = emptyMap(),
    override val description: Map<String, String> = emptyMap(),
    override val group: String? = null,
    override val order: Double? = null,
    val properties: List<PropertyShape> = emptyList(),
    val nodeConstraints: NodeConstraints = NodeConstraints(),
    val logicalConstraints: LogicalConstraints = LogicalConstraints(),
    val closed: Boolean = false,
    val ignoredProperties: List<String> = emptyList(),
    val sparqlConstraints: List<SparqlConstraint> = emptyList()
) : Shape()

/**
 * Property Shape - validates property values
 */
data class PropertyShape(
    override val id: String?,
    override val targets: List<Target> = emptyList(),
    override val severity: Severity = Severity.VIOLATION,
    override val deactivated: Boolean = false,
    override val message: Map<String, String> = emptyMap(),
    override val name: Map<String, String> = emptyMap(),
    override val description: Map<String, String> = emptyMap(),
    override val group: String? = null,
    override val order: Double? = null,
    val path: PropertyPath,
    val nodeConstraints: NodeConstraints = NodeConstraints(),
    val stringConstraints: StringConstraints = StringConstraints(),
    val cardinalityConstraints: CardinalityConstraints = CardinalityConstraints(),
    val valueConstraints: ValueConstraints = ValueConstraints(),
    val comparisonConstraints: ComparisonConstraints = ComparisonConstraints(),
    val logicalConstraints: LogicalConstraints = LogicalConstraints(),
    val qualifiedConstraints: QualifiedConstraints? = null,
    val nestedShapes: List<PropertyShape> = emptyList()  // For nested property shapes
) : Shape()

// =============================================================================
// SHACL Targets
// =============================================================================

sealed class Target

data class TargetClass(val targetClass: String) : Target()
data class TargetNode(val targetNode: String) : Target()
data class TargetSubjectsOf(val predicate: String) : Target()
data class TargetObjectsOf(val predicate: String) : Target()
data class TargetSparql(val select: String) : Target()  // SHACL-AF

/** Implicit target when shape is also an rdfs:Class */
data class ImplicitClassTarget(val classIri: String) : Target()

// =============================================================================
// SHACL Property Paths
// =============================================================================

sealed class PropertyPath {
    /** Convert to SPARQL path syntax */
    abstract fun toSparql(): String

    /** Get all predicates used in this path */
    abstract fun getPredicates(): Set<String>
}

/** Simple predicate path: ex:property */
data class PredicatePath(val predicate: String) : PropertyPath() {
    override fun toSparql() = "<$predicate>"
    override fun getPredicates() = setOf(predicate)
}

/** Inverse path: ^ex:property */
data class InversePath(val inner: PropertyPath) : PropertyPath() {
    override fun toSparql() = "^(${inner.toSparql()})"
    override fun getPredicates() = inner.getPredicates()
}

/** Sequence path: ex:a / ex:b */
data class SequencePath(val paths: List<PropertyPath>) : PropertyPath() {
    override fun toSparql() = paths.joinToString("/") { it.toSparql() }
    override fun getPredicates() = paths.flatMap { it.getPredicates() }.toSet()
}

/** Alternative path: ex:a | ex:b */
data class AlternativePath(val paths: List<PropertyPath>) : PropertyPath() {
    override fun toSparql() = paths.joinToString("|") { "(${it.toSparql()})" }
    override fun getPredicates() = paths.flatMap { it.getPredicates() }.toSet()
}

/** Zero-or-more path: ex:a* */
data class ZeroOrMorePath(val inner: PropertyPath) : PropertyPath() {
    override fun toSparql() = "(${inner.toSparql()})*"
    override fun getPredicates() = inner.getPredicates()
}

/** One-or-more path: ex:a+ */
data class OneOrMorePath(val inner: PropertyPath) : PropertyPath() {
    override fun toSparql() = "(${inner.toSparql()})+"
    override fun getPredicates() = inner.getPredicates()
}

/** Zero-or-one path: ex:a? */
data class ZeroOrOnePath(val inner: PropertyPath) : PropertyPath() {
    override fun toSparql() = "(${inner.toSparql()})?"
    override fun getPredicates() = inner.getPredicates()
}

// =============================================================================
// SHACL Constraints
// =============================================================================

/**
 * Node-level constraints
 */
data class NodeConstraints(
    val nodeKind: NodeKind? = null,
    val classConstraint: List<String> = emptyList(),  // sh:class
    val datatype: String? = null,                      // sh:datatype
    val node: List<String> = emptyList(),             // sh:node (reference to other shapes)
    val hasValue: List<Value> = emptyList(),          // sh:hasValue
    val inValues: List<Value>? = null                 // sh:in
)

enum class NodeKind {
    BLANK_NODE,          // sh:BlankNode
    IRI,                 // sh:IRI
    LITERAL,             // sh:Literal
    BLANK_NODE_OR_IRI,   // sh:BlankNodeOrIRI
    BLANK_NODE_OR_LITERAL, // sh:BlankNodeOrLiteral
    IRI_OR_LITERAL       // sh:IRIOrLiteral
}

/**
 * String-based constraints
 */
data class StringConstraints(
    val minLength: Int? = null,
    val maxLength: Int? = null,
    val pattern: String? = null,
    val flags: String? = null,
    val languageIn: List<String>? = null,
    val uniqueLang: Boolean = false
)

/**
 * Cardinality constraints
 */
data class CardinalityConstraints(
    val minCount: Int? = null,
    val maxCount: Int? = null
) {
    val isRequired: Boolean get() = minCount != null && minCount > 0
    val isSingleton: Boolean get() = maxCount == 1
}

/**
 * Value range constraints
 */
data class ValueConstraints(
    val minInclusive: Number? = null,
    val maxInclusive: Number? = null,
    val minExclusive: Number? = null,
    val maxExclusive: Number? = null
)

/**
 * Property comparison constraints
 */
data class ComparisonConstraints(
    val equals: List<String> = emptyList(),
    val disjoint: List<String> = emptyList(),
    val lessThan: List<String> = emptyList(),
    val lessThanOrEquals: List<String> = emptyList()
)

/**
 * Logical constraints
 */
data class LogicalConstraints(
    val and: List<String> = emptyList(),  // Shape references
    val or: List<String> = emptyList(),
    val xone: List<String> = emptyList(),
    val not: List<String> = emptyList()
)

/**
 * Qualified value shape constraints
 */
data class QualifiedConstraints(
    val qualifiedValueShape: String,
    val qualifiedMinCount: Int? = null,
    val qualifiedMaxCount: Int? = null,
    val qualifiedValueShapesDisjoint: Boolean = false
)

/**
 * SPARQL-based constraint
 */
data class SparqlConstraint(
    val select: String? = null,
    val ask: String? = null,
    val message: Map<String, String> = emptyMap(),
    val prefixes: Map<String, String> = emptyMap()
)

/**
 * Constraint severity levels
 */
enum class Severity {
    INFO,
    WARNING,
    VIOLATION
}

/**
 * RDF Value wrapper for sh:hasValue and sh:in
 */
sealed class Value {
    data class IriValue(val iri: String) : Value()
    data class LiteralValue(
        val value: String,
        val datatype: String? = null,
        val language: String? = null
    ) : Value()
    data class BlankNodeValue(val id: String) : Value()
}

// =============================================================================
// Legacy model classes for backwards compatibility
// =============================================================================
// These model classes are defined in the org.epos.api.shacl package.
// Use fully qualified names if you have conflicts with other packages.

/**
 * Simplified property info for API responses.
 * This is a local version - use fully qualified name if you have conflicts:
 * org.epos.api.shacl.PropertyInfo
 */
@JsonInclude(JsonInclude.Include.NON_NULL)
data class PropertyInfo(
    val path: String,
    val name: String,
    val datatype: String? = null,
    val datatypes: List<String>? = null,
    val nodeKind: String? = null,
    val nodeKinds: List<String>? = null,
    val classConstraint: String? = null,
    val classConstraints: List<String>? = null,
    val nodeShapeRef: String? = null,
    val nodeShapeRefs: List<String>? = null,
    val minCount: Int? = null,
    val maxCount: Int? = null,
    val required: Boolean = false,
    val recommended: Boolean = false,
    val pattern: String? = null,
    val minLength: Int? = null,
    val maxLength: Int? = null,
    val hasValueIn: List<String>? = null,
    val severity: String? = null,
    val severities: List<String>? = null,
    val helpText: String? = null,
    val description: String? = null
)

// =============================================================================
// Comprehensive SHACL Parser
// =============================================================================

/**
 * A comprehensive SHACL parser that handles all SHACL constructs.
 *
 * Supports:
 * - All target types (class, node, subjectsOf, objectsOf, implicit)
 * - Complex property paths (sequence, alternative, inverse, zero-or-more, etc.)
 * - All constraint types (node, string, cardinality, value, comparison, logical)
 * - Qualified value shapes
 * - SPARQL-based constraints
 * - Nested shapes
 * - Shape inheritance via sh:node
 */
class ShaclParser(private val httpClient: HttpClient) {

    private val vf: ValueFactory = SimpleValueFactory.getInstance()

    // Relationship graph for finding paths between types
    // Uses the generalized RelationshipGraph with EPOS-optimized configuration
    val relationshipGraph = RelationshipGraph(RelationshipGraphConfig.EPOS)

    // Cache parsed shapes graph
    private var cachedShapesGraph: ShaclShapesGraph? = null
    private var cachedModel: Model? = null

    // ==========================================================================
    // Public API
    // ==========================================================================

    /**
     * Parse SHACL from URL and return full shapes graph
     */
    suspend fun parseShapesGraph(shaclUrl: String): ShaclShapesGraph {
        val shaclContent = httpClient.get(shaclUrl).bodyAsText()
        return parseShapesGraphFromString(shaclContent, RDFFormat.TURTLE)
    }

    /**
     * Parse SHACL from string content
     */
    fun parseShapesGraphFromString(content: String, format: RDFFormat = RDFFormat.TURTLE): ShaclShapesGraph {
        val model = Rio.parse(StringReader(content), "", format)
        cachedModel = model
        val shapesGraph = parseModel(model)
        cachedShapesGraph = shapesGraph
        return shapesGraph
    }

    /**
     * Parse SHACL and return simplified resource types (backwards compatible)
     */
    suspend fun parseShacl(
        shaclUrl: String,
        graphDBClient: org.epos.graphdb.client.GraphDBClient? = null
    ): List<ResourceTypeInfo> {
        val shapesGraph = parseShapesGraph(shaclUrl)

        println("🔗 Building relationship graph from SHACL schema...")
        relationshipGraph.buildFromShacl(shapesGraph)

        // Enrich relationship graph with actual data if client provided
        if (graphDBClient != null) {
            println("🔗 Enriching relationship graph from actual data...")
            relationshipGraph.buildFromData(
                graphDBClient,
                org.epos.graphdb.Config.GRAPH_PUBLISHED,
                clearExisting = false
            )
            relationshipGraph.printGraph()
        }

        return convertToResourceTypes(shapesGraph)
    }

    /**
     * Get the cached model for additional processing
     */
    fun getCachedModel(): Model? = cachedModel

    /**
     * Get the cached shapes graph
     */
    fun getCachedShapesGraph(): ShaclShapesGraph? = cachedShapesGraph

    /**
     * Get resolvable shape definitions map for UI clients.
     * Key is the shape IRI, value contains constraints and aggregated properties.
     */
    fun getShapeDefinitions(
        onlyReachableFromResourceTypes: Boolean = false,
        includeResolvedRefs: Boolean = false
    ): Map<String, ShapeDefinitionInfo> {
        val shapesGraph = cachedShapesGraph ?: return emptyMap()

        val allDefinitions = shapesGraph.nodeShapes
            .filter { !it.id.isNullOrBlank() }
            .associate { shape ->
                val targetClasses = shape.targets.filterIsInstance<TargetClass>().map { it.targetClass } +
                    shape.targets.filterIsInstance<ImplicitClassTarget>().map { it.classIri }

                val description = shape.description[""] ?: shape.description.values.firstOrNull()

                shape.id!! to ShapeDefinitionInfo(
                    id = shape.id,
                    targetClasses = targetClasses.distinct(),
                    properties = aggregateProperties(shape.properties),
                    closed = shape.closed,
                    description = description
                )
            }

        val selectedDefinitions = if (onlyReachableFromResourceTypes) {
            filterReachableShapeDefinitions(shapesGraph, allDefinitions)
        } else {
            allDefinitions
        }

        if (!includeResolvedRefs) {
            return selectedDefinitions
        }

        return selectedDefinitions.mapValues { (shapeId, shapeInfo) ->
            shapeInfo.copy(resolvedRefs = resolveTransitiveRefs(shapeId, allDefinitions).ifEmpty { null })
        }
    }

    private fun filterReachableShapeDefinitions(
        shapesGraph: ShaclShapesGraph,
        allDefinitions: Map<String, ShapeDefinitionInfo>
    ): Map<String, ShapeDefinitionInfo> {
        val resourceTypes = convertToResourceTypes(shapesGraph)
        val seeds = resourceTypes.flatMap { type ->
            type.properties.flatMap { property ->
                listOfNotNull(property.nodeShapeRef) + property.nodeShapeRefs.orEmpty()
            }
        }.toSet()

        if (seeds.isEmpty()) return emptyMap()

        val visited = mutableSetOf<String>()
        val queue = ArrayDeque<String>()
        seeds.forEach { seed ->
            if (seed in allDefinitions) queue.add(seed)
        }

        while (queue.isNotEmpty()) {
            val current = queue.removeFirst()
            if (!visited.add(current)) continue

            val definition = allDefinitions[current] ?: continue
            val refs = definition.properties.flatMap { property ->
                listOfNotNull(property.nodeShapeRef) + property.nodeShapeRefs.orEmpty()
            }

            refs.filter { it in allDefinitions && it !in visited }
                .forEach { queue.add(it) }
        }

        return visited.associateWith { allDefinitions.getValue(it) }
    }

    private fun resolveTransitiveRefs(
        shapeId: String,
        definitions: Map<String, ShapeDefinitionInfo>
    ): List<String> {
        val visited = mutableSetOf<String>()
        val queue = ArrayDeque<String>()

        val firstLevel = definitions[shapeId]
            ?.properties
            ?.flatMap { property -> listOfNotNull(property.nodeShapeRef) + property.nodeShapeRefs.orEmpty() }
            .orEmpty()
            .distinct()

        firstLevel.filter { it in definitions && it != shapeId }.forEach { queue.add(it) }

        while (queue.isNotEmpty()) {
            val current = queue.removeFirst()
            if (!visited.add(current)) continue

            val refs = definitions[current]
                ?.properties
                ?.flatMap { property -> listOfNotNull(property.nodeShapeRef) + property.nodeShapeRefs.orEmpty() }
                .orEmpty()

            refs.filter { it in definitions && it != shapeId && it !in visited }
                .forEach { queue.add(it) }
        }

        return visited.toList()
    }

    // ==========================================================================
    // Model Parsing
    // ==========================================================================

    private fun parseModel(model: Model): ShaclShapesGraph {
        val shapes = mutableListOf<Shape>()
        val prefixes = model.namespaces.associate { it.prefix to it.name }

        // Find explicit NodeShapes
        val explicitNodeShapes = model.filter {
            it.predicate == RDF.TYPE && it.`object` == SHACL.NODE_SHAPE
        }.map { it.subject }.toSet()

        // Find explicit PropertyShapes
        val explicitPropertyShapes = model.filter {
            it.predicate == RDF.TYPE && it.`object` == SHACL.PROPERTY_SHAPE
        }.map { it.subject }.toSet()

        // Find implicit shapes (have sh:targetClass but no explicit type)
        val implicitShapes = model.filter { it.predicate == SHACL.TARGET_CLASS }
            .map { it.subject }
            .filter { it !in explicitNodeShapes && it !in explicitPropertyShapes }
            .toSet()

        // Find shapes that are also rdfs:Class (implicit class targets)
        val classShapes = model.filter {
            it.predicate == RDF.TYPE && it.`object` == RDFS.CLASS
        }.map { it.subject }
            .filter { subj ->
                // Must have some SHACL properties to be considered a shape
                model.any { it.subject == subj && isShaclProperty(it.predicate) }
            }.toSet()

        // Parse all node shapes
        val allNodeShapeSubjects = explicitNodeShapes + implicitShapes + classShapes
        for (shapeSubject in allNodeShapeSubjects) {
            val nodeShape = parseNodeShape(model, shapeSubject, classShapes.contains(shapeSubject))
            shapes.add(nodeShape)
        }

        // Parse standalone property shapes (those with targets)
        for (propShapeSubject in explicitPropertyShapes) {
            if (hasTargets(model, propShapeSubject)) {
                val propShape = parsePropertyShape(model, propShapeSubject)
                if (propShape != null) {
                    shapes.add(propShape)
                }
            }
        }

        return ShaclShapesGraph(shapes, prefixes)
    }

    private fun isShaclProperty(predicate: IRI): Boolean {
        return predicate.namespace == SHACL.NAMESPACE
    }

    private fun hasTargets(model: Model, subject: Resource): Boolean {
        return model.any { stmt ->
            stmt.subject == subject && stmt.predicate in listOf(
                SHACL.TARGET_CLASS,
                SHACL.TARGET_NODE,
                SHACL.TARGET_SUBJECTS_OF,
                SHACL.TARGET_OBJECTS_OF
            )
        }
    }

    // ==========================================================================
    // Node Shape Parsing
    // ==========================================================================

    private fun parseNodeShape(model: Model, subject: Resource, isClassShape: Boolean): NodeShape {
        return NodeShape(
            id = (subject as? IRI)?.stringValue(),
            targets = parseTargets(model, subject, isClassShape),
            severity = parseSeverity(model, subject),
            deactivated = parseBoolean(model, subject, SHACL.DEACTIVATED) ?: false,
            message = parseLanguageMap(model, subject, SHACL.MESSAGE),
            name = parseLanguageMap(model, subject, SHACL.NAME),
            description = parseLanguageMap(model, subject, SHACL.DESCRIPTION),
            group = parseIri(model, subject, SHACL.GROUP),
            order = parseDouble(model, subject, SHACL.ORDER),
            properties = parsePropertyShapes(model, subject),
            nodeConstraints = parseNodeConstraints(model, subject),
            logicalConstraints = parseLogicalConstraints(model, subject),
            closed = parseBoolean(model, subject, SHACL.CLOSED) ?: false,
            ignoredProperties = parseIriList(model, subject, SHACL.IGNORED_PROPERTIES),
            sparqlConstraints = parseSparqlConstraints(model, subject)
        )
    }

    // ==========================================================================
    // Property Shape Parsing
    // ==========================================================================

    private fun parsePropertyShapes(model: Model, nodeShape: Resource): List<PropertyShape> {
        return model.filter { it.subject == nodeShape && it.predicate == SHACL.PROPERTY }
            .mapNotNull { stmt ->
                (stmt.`object` as? Resource)?.let { parsePropertyShape(model, it) }
            }
    }

    private fun parsePropertyShape(model: Model, subject: Resource): PropertyShape? {
        val path = parsePath(model, subject) ?: return null

        return PropertyShape(
            id = (subject as? IRI)?.stringValue(),
            targets = parseTargets(model, subject, false),
            severity = parseSeverity(model, subject),
            deactivated = parseBoolean(model, subject, SHACL.DEACTIVATED) ?: false,
            message = parseLanguageMap(model, subject, SHACL.MESSAGE),
            name = parseLanguageMap(model, subject, SHACL.NAME),
            description = parseLanguageMap(model, subject, SHACL.DESCRIPTION),
            group = parseIri(model, subject, SHACL.GROUP),
            order = parseDouble(model, subject, SHACL.ORDER),
            path = path,
            nodeConstraints = parseNodeConstraints(model, subject),
            stringConstraints = parseStringConstraints(model, subject),
            cardinalityConstraints = parseCardinalityConstraints(model, subject),
            valueConstraints = parseValueConstraints(model, subject),
            comparisonConstraints = parseComparisonConstraints(model, subject),
            logicalConstraints = parseLogicalConstraints(model, subject),
            qualifiedConstraints = parseQualifiedConstraints(model, subject),
            nestedShapes = parsePropertyShapes(model, subject)
        )
    }

    // ==========================================================================
    // Path Parsing
    // ==========================================================================

    private fun parsePath(model: Model, subject: Resource): PropertyPath? {
        val pathValue = model.filter { it.subject == subject && it.predicate == SHACL.PATH }
            .firstOrNull()?.`object` ?: return null

        return parsePathValue(model, pathValue)
    }

    private fun parsePathValue(model: Model, value: org.eclipse.rdf4j.model.Value): PropertyPath? {
        return when (value) {
            is IRI -> PredicatePath(value.stringValue())
            is BNode -> parseComplexPath(model, value)
            else -> null
        }
    }

    private fun parseComplexPath(model: Model, bnode: BNode): PropertyPath? {
        // Check for inverse path: [ sh:inversePath <predicate> ]
        val inversePath = model.filter { it.subject == bnode && it.predicate == SHACL.INVERSE_PATH }
            .firstOrNull()?.`object`
        if (inversePath != null) {
            val inner = parsePathValue(model, inversePath) ?: return null
            return InversePath(inner)
        }

        // Check for alternative path: [ sh:alternativePath ( ... ) ]
        val altPath = model.filter { it.subject == bnode && it.predicate == SHACL.ALTERNATIVE_PATH }
            .firstOrNull()?.`object`
        if (altPath != null) {
            val paths = parseRdfList(model, altPath).mapNotNull { parsePathValue(model, it) }
            return if (paths.size > 1) AlternativePath(paths) else paths.firstOrNull()
        }

        // Check for zero-or-more path: [ sh:zeroOrMorePath <predicate> ]
        val zeroOrMore = model.filter { it.subject == bnode && it.predicate == SHACL.ZERO_OR_MORE_PATH }
            .firstOrNull()?.`object`
        if (zeroOrMore != null) {
            val inner = parsePathValue(model, zeroOrMore) ?: return null
            return ZeroOrMorePath(inner)
        }

        // Check for one-or-more path: [ sh:oneOrMorePath <predicate> ]
        val oneOrMore = model.filter { it.subject == bnode && it.predicate == SHACL.ONE_OR_MORE_PATH }
            .firstOrNull()?.`object`
        if (oneOrMore != null) {
            val inner = parsePathValue(model, oneOrMore) ?: return null
            return OneOrMorePath(inner)
        }

        // Check for zero-or-one path: [ sh:zeroOrOnePath <predicate> ]
        val zeroOrOne = model.filter { it.subject == bnode && it.predicate == SHACL.ZERO_OR_ONE_PATH }
            .firstOrNull()?.`object`
        if (zeroOrOne != null) {
            val inner = parsePathValue(model, zeroOrOne) ?: return null
            return ZeroOrOnePath(inner)
        }

        // Check for sequence path (RDF list directly on the bnode)
        val first = model.filter { it.subject == bnode && it.predicate == RDF.FIRST }
            .firstOrNull()?.`object`
        if (first != null) {
            val paths = parseRdfList(model, bnode).mapNotNull { parsePathValue(model, it) }
            return if (paths.size > 1) SequencePath(paths) else paths.firstOrNull()
        }

        return null
    }

    // ==========================================================================
    // Target Parsing
    // ==========================================================================

    private fun parseTargets(model: Model, subject: Resource, isClassShape: Boolean): List<Target> {
        val targets = mutableListOf<Target>()

        // sh:targetClass
        model.filter { it.subject == subject && it.predicate == SHACL.TARGET_CLASS }
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetClass(it.stringValue())) }
            }

        // sh:targetNode
        model.filter { it.subject == subject && it.predicate == SHACL.TARGET_NODE }
            .forEach { stmt ->
                targets.add(TargetNode(stmt.`object`.stringValue()))
            }

        // sh:targetSubjectsOf
        model.filter { it.subject == subject && it.predicate == SHACL.TARGET_SUBJECTS_OF }
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetSubjectsOf(it.stringValue())) }
            }

        // sh:targetObjectsOf
        model.filter { it.subject == subject && it.predicate == SHACL.TARGET_OBJECTS_OF }
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetObjectsOf(it.stringValue())) }
            }

        // sh:target with SHACL-AF target declarations (e.g., SHACL-SPARQL targets)
        val shTarget = vf.createIRI("http://www.w3.org/ns/shacl#target")
        model.filter { it.subject == subject && (it.predicate == SHACL.TARGET || it.predicate == shTarget) }
            .forEach { stmt ->
                val target = stmt.`object` as? Resource ?: return@forEach
                val select = parseString(model, target, SHACL.SELECT)
                    ?: model.filter {
                        it.subject == target && it.predicate.localName.equals("select", ignoreCase = true)
                    }.firstOrNull()?.`object`?.stringValue()

                select?.let {
                    targets.add(TargetSparql(select))
                }
            }

        // Implicit class target (shape is also an rdfs:Class)
        if (isClassShape && subject is IRI) {
            targets.add(ImplicitClassTarget(subject.stringValue()))
        }

        return targets
    }

    // ==========================================================================
    // Constraint Parsing
    // ==========================================================================

    private fun parseNodeConstraints(model: Model, subject: Resource): NodeConstraints {
        val directNodeKind = parseNodeKind(model, subject)
        val directClassConstraints = model.filter { it.subject == subject && it.predicate == SHACL.CLASS }
            .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        val directDatatype = parseIri(model, subject, SHACL.DATATYPE)
        val directNodeRefs = model.filter { it.subject == subject && it.predicate == SHACL.NODE }
            .mapNotNull { (it.`object` as? Resource)?.let { res -> resourceRef(res) } }

        val orAlternatives = parseListResources(model, subject, SHACL.OR)
        val orClassConstraints = orAlternatives.flatMap { alternative ->
            model.filter { it.subject == alternative && it.predicate == SHACL.CLASS }
                .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        }
        val orDatatypes = orAlternatives.mapNotNull { alternative ->
            parseIri(model, alternative, SHACL.DATATYPE)
        }
        val orNodeKinds = orAlternatives.mapNotNull { alternative ->
            parseNodeKind(model, alternative)
        }
        val orNodeRefs = orAlternatives.flatMap { alternative ->
            model.filter { it.subject == alternative && it.predicate == SHACL.NODE }
                .mapNotNull { (it.`object` as? Resource)?.let { res -> resourceRef(res) } }
        }

        return NodeConstraints(
            nodeKind = directNodeKind ?: orNodeKinds.firstOrNull(),
            classConstraint = (directClassConstraints + orClassConstraints).distinct(),
            datatype = directDatatype ?: orDatatypes.firstOrNull(),
            node = (directNodeRefs + orNodeRefs).distinct(),
            hasValue = model.filter { it.subject == subject && it.predicate == SHACL.HAS_VALUE }
                .map { parseValue(it.`object`) },
            inValues = parseInConstraint(model, subject)
        )
    }

    private fun parseNodeKind(model: Model, subject: Resource): NodeKind? {
        val nodeKindValue = model.filter { it.subject == subject && it.predicate == SHACL.NODE_KIND }
            .firstOrNull()?.`object` as? IRI ?: return null

        return when (nodeKindValue) {
            SHACL.BLANK_NODE -> NodeKind.BLANK_NODE
            SHACL.IRI -> NodeKind.IRI
            SHACL.LITERAL -> NodeKind.LITERAL
            SHACL.BLANK_NODE_OR_IRI -> NodeKind.BLANK_NODE_OR_IRI
            SHACL.BLANK_NODE_OR_LITERAL -> NodeKind.BLANK_NODE_OR_LITERAL
            SHACL.IRI_OR_LITERAL -> NodeKind.IRI_OR_LITERAL
            else -> null
        }
    }

    private fun parseInConstraint(model: Model, subject: Resource): List<Value>? {
        val inValue = model.filter { it.subject == subject && it.predicate == SHACL.IN }
            .firstOrNull()?.`object` ?: return null

        return parseRdfList(model, inValue).map { parseValue(it) }
    }

    private fun parseValue(value: org.eclipse.rdf4j.model.Value): Value {
        return when (value) {
            is IRI -> Value.IriValue(value.stringValue())
            is Literal -> Value.LiteralValue(
                value = value.stringValue(),
                datatype = value.datatype?.stringValue(),
                language = value.language.orElse(null)
            )
            is BNode -> Value.BlankNodeValue(value.id)
            else -> Value.LiteralValue(value.stringValue())
        }
    }

    private fun parseStringConstraints(model: Model, subject: Resource): StringConstraints {
        return StringConstraints(
            minLength = parseInt(model, subject, SHACL.MIN_LENGTH),
            maxLength = parseInt(model, subject, SHACL.MAX_LENGTH),
            pattern = parseString(model, subject, SHACL.PATTERN),
            flags = parseString(model, subject, SHACL.FLAGS),
            languageIn = parseStringList(model, subject, SHACL.LANGUAGE_IN),
            uniqueLang = parseBoolean(model, subject, SHACL.UNIQUE_LANG) ?: false
        )
    }

    private fun parseCardinalityConstraints(model: Model, subject: Resource): CardinalityConstraints {
        return CardinalityConstraints(
            minCount = parseInt(model, subject, SHACL.MIN_COUNT),
            maxCount = parseInt(model, subject, SHACL.MAX_COUNT)
        )
    }

    private fun parseValueConstraints(model: Model, subject: Resource): ValueConstraints {
        return ValueConstraints(
            minInclusive = parseNumber(model, subject, SHACL.MIN_INCLUSIVE),
            maxInclusive = parseNumber(model, subject, SHACL.MAX_INCLUSIVE),
            minExclusive = parseNumber(model, subject, SHACL.MIN_EXCLUSIVE),
            maxExclusive = parseNumber(model, subject, SHACL.MAX_EXCLUSIVE)
        )
    }

    private fun parseComparisonConstraints(model: Model, subject: Resource): ComparisonConstraints {
        return ComparisonConstraints(
            equals = parseIriList(model, subject, SHACL.EQUALS),
            disjoint = parseIriList(model, subject, SHACL.DISJOINT),
            lessThan = parseIriList(model, subject, SHACL.LESS_THAN),
            lessThanOrEquals = parseIriList(model, subject, SHACL.LESS_THAN_OR_EQUALS)
        )
    }

    private fun parseLogicalConstraints(model: Model, subject: Resource): LogicalConstraints {
        return LogicalConstraints(
            and = parseShapeListConstraint(model, subject, SHACL.AND),
            or = parseShapeListConstraint(model, subject, SHACL.OR),
            xone = parseShapeListConstraint(model, subject, SHACL.XONE),
            not = model.filter { it.subject == subject && it.predicate == SHACL.NOT }
                .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        )
    }

    private fun parseShapeListConstraint(model: Model, subject: Resource, predicate: IRI): List<String> {
        val listHead = model.filter { it.subject == subject && it.predicate == predicate }
            .firstOrNull()?.`object` ?: return emptyList()

        return parseRdfList(model, listHead)
            .mapNotNull { value -> (value as? Resource)?.let { resourceRef(it) } }
    }

    private fun parseQualifiedConstraints(model: Model, subject: Resource): QualifiedConstraints? {
        val qualifiedShape = model.filter {
            it.subject == subject && it.predicate == SHACL.QUALIFIED_VALUE_SHAPE
        }.firstOrNull()?.`object`
            ?.let { value -> (value as? Resource)?.let { resourceRef(it) } }
            ?: return null

        return QualifiedConstraints(
            qualifiedValueShape = qualifiedShape,
            qualifiedMinCount = parseInt(model, subject, SHACL.QUALIFIED_MIN_COUNT),
            qualifiedMaxCount = parseInt(model, subject, SHACL.QUALIFIED_MAX_COUNT),
            qualifiedValueShapesDisjoint = parseBoolean(model, subject, SHACL.QUALIFIED_VALUE_SHAPES_DISJOINT) ?: false
        )
    }

    private fun parseSparqlConstraints(model: Model, subject: Resource): List<SparqlConstraint> {
        return model.filter { it.subject == subject && it.predicate == SHACL.SPARQL }
            .mapNotNull { stmt ->
                (stmt.`object` as? Resource)?.let { constraintNode ->
                    SparqlConstraint(
                        select = parseString(model, constraintNode, SHACL.SELECT),
                        ask = parseString(model, constraintNode, SHACL.ASK),
                        message = parseLanguageMap(model, constraintNode, SHACL.MESSAGE),
                        prefixes = parsePrefixes(model, constraintNode)
                    )
                }
            }
    }

    private fun parseSeverity(model: Model, subject: Resource): Severity {
        val severityValue = model.filter { it.subject == subject && it.predicate == SHACL.SEVERITY }
            .firstOrNull()?.`object` as? IRI

        return when (severityValue) {
            SHACL.INFO -> Severity.INFO
            SHACL.WARNING -> Severity.WARNING
            else -> Severity.VIOLATION
        }
    }

    private fun parsePrefixes(model: Model, subject: Resource): Map<String, String> {
        val prefixes = mutableMapOf<String, String>()
        model.filter { it.subject == subject && it.predicate == SHACL.PREFIXES }
            .forEach { stmt ->
                (stmt.`object` as? Resource)?.let { prefixDecl ->
                    val inlinePrefix = parseString(model, prefixDecl, SHACL.PREFIX_PROP)
                    val inlineNamespace = parseIri(model, prefixDecl, SHACL.NAMESPACE_PROP)
                    if (inlinePrefix != null && inlineNamespace != null) {
                        prefixes[inlinePrefix] = inlineNamespace
                    }

                    model.filter { it.subject == prefixDecl && it.predicate == SHACL.DECLARE }
                        .forEach { declareStmt ->
                            val declaration = declareStmt.`object` as? Resource ?: return@forEach
                            val prefix = parseString(model, declaration, SHACL.PREFIX_PROP)
                            val namespace = parseIri(model, declaration, SHACL.NAMESPACE_PROP)
                            if (prefix != null && namespace != null) {
                                prefixes[prefix] = namespace
                            }
                        }
                }
            }
        return prefixes
    }

    private fun parseListResources(model: Model, subject: Resource, predicate: IRI): List<Resource> {
        val listHead = model.filter { it.subject == subject && it.predicate == predicate }
            .firstOrNull()?.`object` ?: return emptyList()

        return parseRdfList(model, listHead).mapNotNull { it as? Resource }
    }

    private fun resourceRef(resource: Resource): String {
        return when (resource) {
            is IRI -> resource.stringValue()
            is BNode -> "_:${resource.id}"
            else -> resource.stringValue()
        }
    }

    // ==========================================================================
    // Utility Parsing Methods
    // ==========================================================================

    private fun parseString(model: Model, subject: Resource, predicate: IRI): String? {
        return model.filter { it.subject == subject && it.predicate == predicate }
            .firstOrNull()?.`object`?.stringValue()
    }

    private fun parseIri(model: Model, subject: Resource, predicate: IRI): String? {
        return model.filter { it.subject == subject && it.predicate == predicate }
            .firstOrNull()?.`object`?.let { (it as? IRI)?.stringValue() }
    }

    private fun parseInt(model: Model, subject: Resource, predicate: IRI): Int? {
        return parseString(model, subject, predicate)?.toIntOrNull()
    }

    private fun parseDouble(model: Model, subject: Resource, predicate: IRI): Double? {
        return parseString(model, subject, predicate)?.toDoubleOrNull()
    }

    private fun parseNumber(model: Model, subject: Resource, predicate: IRI): Number? {
        val str = parseString(model, subject, predicate) ?: return null
        return str.toIntOrNull() ?: str.toLongOrNull() ?: str.toDoubleOrNull()
    }

    private fun parseBoolean(model: Model, subject: Resource, predicate: IRI): Boolean? {
        return parseString(model, subject, predicate)?.toBooleanStrictOrNull()
    }

    private fun parseIriList(model: Model, subject: Resource, predicate: IRI): List<String> {
        return model.filter { it.subject == subject && it.predicate == predicate }
            .mapNotNull { (it.`object` as? IRI)?.stringValue() }
    }

    private fun parseStringList(model: Model, subject: Resource, predicate: IRI): List<String>? {
        val listHead = model.filter { it.subject == subject && it.predicate == predicate }
            .firstOrNull()?.`object` ?: return null

        return parseRdfList(model, listHead).map { it.stringValue() }
    }

    private fun parseLanguageMap(model: Model, subject: Resource, predicate: IRI): Map<String, String> {
        val result = mutableMapOf<String, String>()
        model.filter { it.subject == subject && it.predicate == predicate }
            .forEach { stmt ->
                val literal = stmt.`object` as? Literal
                if (literal != null) {
                    val lang = literal.language.orElse("")
                    result[lang] = literal.stringValue()
                }
            }
        return result
    }

    private fun parseRdfList(model: Model, listHead: org.eclipse.rdf4j.model.Value): List<org.eclipse.rdf4j.model.Value> {
        val result = mutableListOf<org.eclipse.rdf4j.model.Value>()
        var current: org.eclipse.rdf4j.model.Value? = listHead

        while (current != null && current != RDF.NIL) {
            if (current !is Resource) break

            val first = model.filter { it.subject == current && it.predicate == RDF.FIRST }
                .firstOrNull()?.`object`
            if (first != null) {
                result.add(first)
            }

            current = model.filter { it.subject == current && it.predicate == RDF.REST }
                .firstOrNull()?.`object`
        }

        return result
    }

    // ==========================================================================
    // Conversion to Legacy Format
    // ==========================================================================

    private fun convertToResourceTypes(shapesGraph: ShaclShapesGraph): List<ResourceTypeInfo> {
        return shapesGraph.nodeShapes.mapNotNull { nodeShape ->
            // Get target class
            val targetClass = nodeShape.targets
                .filterIsInstance<TargetClass>()
                .firstOrNull()?.targetClass
                ?: nodeShape.targets
                    .filterIsInstance<ImplicitClassTarget>()
                    .firstOrNull()?.classIri
                ?: return@mapNotNull null

            val resourceName = extractResourceName(targetClass)
            val description = nodeShape.description[""] ?: nodeShape.description.values.firstOrNull()

            ResourceTypeInfo(
                name = resourceName,
                targetClass = targetClass,
                properties = aggregateProperties(nodeShape.properties),
                closed = nodeShape.closed,
                description = description
            )
        }
    }

    private fun aggregateProperties(properties: List<PropertyShape>): List<PropertyInfo> {
        val pathOrder = mutableListOf<String>()
        val groupedByPath = linkedMapOf<String, MutableList<PropertyShape>>()

        properties.forEach { propertyShape ->
            val path = toPropertyPathString(propertyShape.path)
            if (path !in groupedByPath) {
                groupedByPath[path] = mutableListOf()
                pathOrder.add(path)
            }
            groupedByPath.getValue(path).add(propertyShape)
        }

        return pathOrder.mapNotNull { path ->
            val variants = groupedByPath[path].orEmpty()
            if (variants.isEmpty()) null else aggregatePropertyVariants(path, variants)
        }
    }

    private fun toPropertyPathString(path: PropertyPath): String {
        return when (path) {
            is PredicatePath -> path.predicate
            else -> path.toSparql()
        }
    }

    private fun aggregatePropertyVariants(path: String, variants: List<PropertyShape>): PropertyInfo {
        val names = variants.mapNotNull { shape ->
            shape.name[""] ?: shape.name.values.firstOrNull()
        }.map { it.trim() }.filter { it.isNotBlank() }

        val descriptions = variants.mapNotNull { shape ->
            shape.description[""] ?: shape.description.values.firstOrNull()
        }.map { it.trim() }.filter { it.isNotBlank() }

        val datatypes = variants.mapNotNull { it.nodeConstraints.datatype }.distinct()
        val nodeKinds = variants.mapNotNull { it.nodeConstraints.nodeKind?.name }.distinct()
        val classConstraints = variants.flatMap { it.nodeConstraints.classConstraint }.distinct()
        val nodeShapeRefs = variants.flatMap { it.nodeConstraints.node }.distinct()
        val severities = variants.map { it.severity.name }.distinct()
        val messages = variants.mapNotNull { shape ->
            shape.message[""] ?: shape.message.values.firstOrNull()
        }.map { it.trim() }.filter { it.isNotBlank() }

        val minCount = variants.mapNotNull { it.cardinalityConstraints.minCount }.maxOrNull()
        val maxCount = variants.mapNotNull { it.cardinalityConstraints.maxCount }.minOrNull()

        val pattern = variants.mapNotNull { it.stringConstraints.pattern }.firstOrNull()
        val minLength = variants.mapNotNull { it.stringConstraints.minLength }.maxOrNull()
        val maxLength = variants.mapNotNull { it.stringConstraints.maxLength }.minOrNull()

        val hasValueIn = variants.flatMap { shape ->
            shape.nodeConstraints.inValues.orEmpty().mapNotNull { value ->
                when (value) {
                    is Value.IriValue -> value.iri
                    is Value.LiteralValue -> value.value
                    else -> null
                }
            }
        }.distinct().ifEmpty { null }

        return PropertyInfo(
            path = path,
            name = names.firstOrNull() ?: extractPropertyName(path),
            datatype = if (datatypes.size == 1) datatypes.first() else null,
            datatypes = if (datatypes.size > 1) datatypes else null,
            nodeKind = if (nodeKinds.size == 1) nodeKinds.first() else null,
            nodeKinds = if (nodeKinds.size > 1) nodeKinds else null,
            classConstraint = if (classConstraints.size == 1) classConstraints.first() else null,
            classConstraints = if (classConstraints.size > 1) classConstraints else null,
            nodeShapeRef = if (nodeShapeRefs.size == 1) nodeShapeRefs.first() else null,
            nodeShapeRefs = if (nodeShapeRefs.size > 1) nodeShapeRefs else null,
            minCount = minCount,
            maxCount = maxCount,
            required = (minCount ?: 0) > 0,
            recommended = (minCount ?: 0) > 0 && severities.contains(Severity.WARNING.name),
            pattern = pattern,
            minLength = minLength,
            maxLength = maxLength,
            hasValueIn = hasValueIn,
            severity = if (severities.size == 1) severities.first() else null,
            severities = if (severities.size > 1) severities else null,
            helpText = messages.firstOrNull(),
            description = descriptions.firstOrNull()
        )
    }

    private fun extractResourceName(uri: String): String {
        val parts = uri.split("/", "#")
        return parts.lastOrNull()?.lowercase() ?: "resource"
    }

    private fun extractPropertyName(uri: String): String {
        // Handle SPARQL path syntax - clean up wrapper characters first
        val cleanUri = uri.trim()
            .removePrefix("<").removeSuffix(">")
            .removePrefix("^")

        // Split by / and # to get the local name
        val localName = cleanUri
            .substringAfterLast("/")
            .substringAfterLast("#")

        // Clean up any remaining SPARQL path syntax characters from the local name only
        val cleanName = localName.replace(Regex("[()*/+?|<>]"), "").trim()

        return cleanName.takeIf { it.isNotBlank() } ?: "property"
    }
}
