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
import org.eclipse.rdf4j.model.vocabulary.OWL
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
    val prefixes: Map<String, String> = emptyMap(),
    val ontologySchema: OntologySchema = OntologySchema()
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

data class OntologySchema(
    val classes: Map<String, OntologyClassInfo> = emptyMap(),
    val properties: Map<String, OntologyPropertyInfo> = emptyMap()
)

data class OntologyClassInfo(
    val iri: String,
    val labels: Map<String, String> = emptyMap(),
    val descriptions: Map<String, String> = emptyMap(),
    val equivalentClasses: List<String> = emptyList(),
    val subClassOf: List<String> = emptyList()
)

data class OntologyPropertyInfo(
    val iri: String,
    val labels: Map<String, String> = emptyMap(),
    val descriptions: Map<String, String> = emptyMap(),
    val propertyTypes: List<String> = emptyList(),
    val domains: List<String> = emptyList(),
    val ranges: List<String> = emptyList(),
    val equivalentProperties: List<String> = emptyList(),
    val subPropertyOf: List<String> = emptyList()
)

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
    val closed: Boolean = false,
    val ignoredProperties: List<String> = emptyList()
) : Shape()

/**
 * Property Shape - validates property values
 */
data class PropertyShape(
    override val id: String? = null,
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
    val qualifiedConstraints: QualifiedConstraints? = null
) : Shape()

// =============================================================================
// SHACL Targets
// =============================================================================

sealed class Target

data class TargetClass(val targetClass: String) : Target()
data class TargetNode(val targetNode: String? = null) : Target()
data class TargetSubjectsOf(val predicate: String? = null) : Target()
data class TargetObjectsOf(val predicate: String? = null) : Target()
data class TargetSparql(val query: String? = null) : Target()  // SHACL-AF

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
    val nodeKinds: List<NodeKind> = emptyList(),
    val classConstraint: List<String> = emptyList(),  // sh:class
    val datatype: String? = null,                      // sh:datatype
    val datatypes: List<String> = emptyList(),
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
        val value: String
    ) : Value()
    data class BlankNodeValue(val id: String? = null) : Value()
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
    private val parseCache = ThreadLocal<ParseCache?>()

    private data class SubjectPredicateKey(val subject: Resource, val predicate: IRI)

    private data class ParseCache(
        val statementsBySubjectPredicate: MutableMap<SubjectPredicateKey, List<Statement>> = mutableMapOf(),
        val subjectsByPredicate: MutableMap<IRI, List<Resource>> = mutableMapOf(),
        val statementsBySubject: MutableMap<Resource, List<Statement>> = mutableMapOf(),
    )

    // ==========================================================================
    // Public API
    // ==========================================================================

    /**
     * Parse SHACL from URL and return full shapes graph
     */
    suspend fun parseShapesGraph(shaclUrl: String): ShaclShapesGraph {
        val shaclContent = httpClient.get(shaclUrl).bodyAsText()
        return parseShapesGraphFromString(shaclContent, RDFFormat.TURTLE, shaclUrl)
    }

    /**
     * Parse SHACL from string content
     */
    fun parseShapesGraphFromString(
        content: String,
        format: RDFFormat = RDFFormat.TURTLE,
        baseUri: String = "",
    ): ShaclShapesGraph {
        val model = Rio.parse(StringReader(content), baseUri, format)
        cachedModel = model
        val parsedGraph = parseModel(model)
        val declaredPrefixes = extractDeclaredPrefixes(content)
        val mergedPrefixes = linkedMapOf<String, String>().apply {
            putAll(parsedGraph.prefixes)
            putAll(declaredPrefixes)
        }
        val shapesGraph = parsedGraph.copy(prefixes = mergedPrefixes)
        cachedShapesGraph = shapesGraph
        return shapesGraph
    }

    private fun extractDeclaredPrefixes(content: String): Map<String, String> {
        val prefixes = linkedMapOf<String, String>()
        val regex = Regex("""@prefix\s+([A-Za-z][A-Za-z0-9_-]*)\s*:\s*<([^>]+)>\s*\.""")
        regex.findAll(content).forEach { match ->
            val prefix = match.groupValues[1].trim()
            val namespace = match.groupValues[2].trim()
            if (prefix.isNotBlank() && namespace.isNotBlank()) {
                prefixes[prefix] = namespace
            }
        }
        return prefixes
    }

    /**
     * Parse SHACL from an in-memory RDF model.
     */
    fun parseShapesGraphFromModel(model: Model): ShaclShapesGraph {
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
        propertyGraphClient: org.epos.propertygraph.client.PropertyGraphClient? = null
    ): List<ResourceTypeInfo> {
        val shapesGraph = parseShapesGraph(shaclUrl)

        println("Building relationship graph from SHACL schema...")
        relationshipGraph.buildFromShacl(shapesGraph)

        if (propertyGraphClient != null) {
            println("Enriching relationship graph from actual data...")
            relationshipGraph.buildFromData(
                propertyGraphClient,
                clearExisting = false
            )
            relationshipGraph.printGraph()
        }

        return convertToResourceTypes(shapesGraph)
    }

    fun toResourceTypes(shapesGraph: ShaclShapesGraph): List<ResourceTypeInfo> = convertToResourceTypes(shapesGraph)

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

                val nodeKinds = shape.nodeConstraints.nodeKinds.map { it.name }.distinct()
                val datatypes = shape.nodeConstraints.datatypes.distinct()
                val classConstraints = shape.nodeConstraints.classConstraint.distinct()
                val nodeShapeRefs = shape.nodeConstraints.node.distinct()

                val ontologyClass = shape.id?.let { shapesGraph.ontologySchema.classes[it] }

                val description = shape.description[""] ?: shape.description.values.firstOrNull()
                val name = shape.name[""] ?: shape.name.values.firstOrNull()

                shape.id!! to ShapeDefinitionInfo(
                    id = shape.id,
                    targetClasses = targetClasses.distinct().ifEmpty {
                        if (shape.id in shapesGraph.ontologySchema.classes) listOf(shape.id) else emptyList()
                    },
                    properties = aggregateProperties(shape.properties, shapesGraph.ontologySchema, targetClasses.firstOrNull()),
                    closed = shape.closed,
                    description = description,
                    name = name,
                    datatype = if (datatypes.size == 1) datatypes.first() else null,
                    datatypes = if (datatypes.size > 1) datatypes else null,
                    nodeKind = if (nodeKinds.size == 1) nodeKinds.first() else null,
                    nodeKinds = if (nodeKinds.size > 1) nodeKinds else null,
                    classConstraint = if (classConstraints.size == 1) classConstraints.first() else null,
                    classConstraints = if (classConstraints.size > 1) classConstraints else null,
                    nodeShapeRef = if (nodeShapeRefs.size == 1) nodeShapeRefs.first() else null,
                    nodeShapeRefs = if (nodeShapeRefs.size > 1) nodeShapeRefs else null,
                    ontologyEquivalentClasses = ontologyClass?.equivalentClasses?.ifEmpty { null },
                    ontologySubClassesOf = ontologyClass?.subClassOf?.ifEmpty { null }
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
        parseCache.set(ParseCache())
        try {
            val shapes = mutableListOf<Shape>()
            val prefixes = model.namespaces.associate { it.prefix to it.name }

            // Find explicit NodeShapes
            val explicitNodeShapes = subjectsForPredicate(model, RDF.TYPE)
                .filter { subject -> valuesForSubjectPredicate(model, subject, RDF.TYPE).any { it == SHACL.NODE_SHAPE } }
                .toSet()

            // Find explicit PropertyShapes
            val explicitPropertyShapes = subjectsForPredicate(model, RDF.TYPE)
                .filter { subject -> valuesForSubjectPredicate(model, subject, RDF.TYPE).any { it == SHACL.PROPERTY_SHAPE } }
                .toSet()

            // Find implicit shapes (have sh:targetClass but no explicit type)
            val implicitShapes = subjectsForPredicate(model, SHACL.TARGET_CLASS)
                .filter { it !in explicitNodeShapes && it !in explicitPropertyShapes }
                .toSet()

            // Find shapes that are also classes (implicit class targets)
            val classShapes = subjectsForPredicate(model, RDF.TYPE)
                .filter { subj ->
                    val rdfTypes = valuesForSubjectPredicate(model, subj, RDF.TYPE)
                    (rdfTypes.any { it == RDFS.CLASS } || rdfTypes.any { it == OWL.CLASS }) && hasShaclPredicates(model, subj)
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

            val ontologySchema = parseOntologySchema(model)

            return ShaclShapesGraph(shapes, prefixes, ontologySchema)
        } finally {
            parseCache.remove()
        }
    }

    private fun isShaclProperty(predicate: IRI): Boolean {
        return predicate.namespace == SHACL.NAMESPACE
    }

    private fun hasTargets(model: Model, subject: Resource): Boolean {
        return statementsForSubjectPredicate(model, subject, SHACL.TARGET_CLASS).isNotEmpty() ||
            statementsForSubjectPredicate(model, subject, SHACL.TARGET_NODE).isNotEmpty() ||
            statementsForSubjectPredicate(model, subject, SHACL.TARGET_SUBJECTS_OF).isNotEmpty() ||
            statementsForSubjectPredicate(model, subject, SHACL.TARGET_OBJECTS_OF).isNotEmpty()
    }

    private fun hasShaclPredicates(model: Model, subject: Resource): Boolean {
        return statementsForSubject(model, subject).any { isShaclProperty(it.predicate) }
    }

    private fun statementsForSubjectPredicate(model: Model, subject: Resource, predicate: IRI): List<Statement> {
        val cache = parseCache.get() ?: return model.filter { it.subject == subject && it.predicate == predicate }.toList()
        val key = SubjectPredicateKey(subject, predicate)
        return cache.statementsBySubjectPredicate.getOrPut(key) {
            model.filter { it.subject == subject && it.predicate == predicate }.toList()
        }
    }

    private fun valuesForSubjectPredicate(model: Model, subject: Resource, predicate: IRI): List<org.eclipse.rdf4j.model.Value> {
        return statementsForSubjectPredicate(model, subject, predicate).map { it.`object` }
    }

    private fun firstValueForSubjectPredicate(model: Model, subject: Resource, predicate: IRI): org.eclipse.rdf4j.model.Value? {
        return statementsForSubjectPredicate(model, subject, predicate).firstOrNull()?.`object`
    }

    private fun subjectsForPredicate(model: Model, predicate: IRI): List<Resource> {
        val cache = parseCache.get() ?: return model.filter { it.predicate == predicate }.map { it.subject }.toList()
        return cache.subjectsByPredicate.getOrPut(predicate) {
            model.filter { it.predicate == predicate }.map { it.subject }.toList()
        }
    }

    private fun statementsForSubject(model: Model, subject: Resource): List<Statement> {
        val cache = parseCache.get() ?: return model.filter { it.subject == subject }.toList()
        return cache.statementsBySubject.getOrPut(subject) {
            model.filter { it.subject == subject }.toList()
        }
    }

    // ==========================================================================
    // Node Shape Parsing
    // ==========================================================================

    private fun parseNodeShape(model: Model, subject: Resource, isClassShape: Boolean): NodeShape {
        return NodeShape(
            id = (subject as? IRI)?.stringValue(),
            targets = parseTargets(model, subject, isClassShape),
            name = parseLanguageMap(model, subject, SHACL.NAME),
            description = parseLanguageMap(model, subject, SHACL.DESCRIPTION),
            properties = parsePropertyShapes(model, subject),
            nodeConstraints = parseNodeConstraints(model, subject),
            closed = parseBoolean(model, subject, SHACL.CLOSED) ?: false,
            ignoredProperties = parseIriList(model, subject, SHACL.IGNORED_PROPERTIES)
        )
    }

    // ==========================================================================
    // Property Shape Parsing
    // ==========================================================================

    private fun parsePropertyShapes(model: Model, nodeShape: Resource): List<PropertyShape> {
        return statementsForSubjectPredicate(model, nodeShape, SHACL.PROPERTY)
            .mapNotNull { stmt ->
                (stmt.`object` as? Resource)?.let { parsePropertyShape(model, it) }
            }
    }

    private fun parsePropertyShape(model: Model, subject: Resource): PropertyShape? {
        val path = parsePath(model, subject) ?: return null

        return PropertyShape(
            severity = parseSeverity(model, subject),
            message = parseLanguageMap(model, subject, SHACL.MESSAGE),
            name = parseLanguageMap(model, subject, SHACL.NAME),
            description = parseLanguageMap(model, subject, SHACL.DESCRIPTION),
            path = path,
            nodeConstraints = parseNodeConstraints(model, subject),
            stringConstraints = parseStringConstraints(model, subject),
            cardinalityConstraints = parseCardinalityConstraints(model, subject),
            qualifiedConstraints = parseQualifiedConstraints(model, subject)
        )
    }

    // ==========================================================================
    // Path Parsing
    // ==========================================================================

    private fun parsePath(model: Model, subject: Resource): PropertyPath? {
        val pathValue = firstValueForSubjectPredicate(model, subject, SHACL.PATH) ?: return null

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
        val inversePath = firstValueForSubjectPredicate(model, bnode, SHACL.INVERSE_PATH)
        if (inversePath != null) {
            val inner = parsePathValue(model, inversePath) ?: return null
            return InversePath(inner)
        }

        // Check for alternative path: [ sh:alternativePath ( ... ) ]
        val altPath = firstValueForSubjectPredicate(model, bnode, SHACL.ALTERNATIVE_PATH)
        if (altPath != null) {
            val paths = parseRdfList(model, altPath).mapNotNull { parsePathValue(model, it) }
            return if (paths.size > 1) AlternativePath(paths) else paths.firstOrNull()
        }

        // Check for zero-or-more path: [ sh:zeroOrMorePath <predicate> ]
        val zeroOrMore = firstValueForSubjectPredicate(model, bnode, SHACL.ZERO_OR_MORE_PATH)
        if (zeroOrMore != null) {
            val inner = parsePathValue(model, zeroOrMore) ?: return null
            return ZeroOrMorePath(inner)
        }

        // Check for one-or-more path: [ sh:oneOrMorePath <predicate> ]
        val oneOrMore = firstValueForSubjectPredicate(model, bnode, SHACL.ONE_OR_MORE_PATH)
        if (oneOrMore != null) {
            val inner = parsePathValue(model, oneOrMore) ?: return null
            return OneOrMorePath(inner)
        }

        // Check for zero-or-one path: [ sh:zeroOrOnePath <predicate> ]
        val zeroOrOne = firstValueForSubjectPredicate(model, bnode, SHACL.ZERO_OR_ONE_PATH)
        if (zeroOrOne != null) {
            val inner = parsePathValue(model, zeroOrOne) ?: return null
            return ZeroOrOnePath(inner)
        }

        // Check for sequence path (RDF list directly on the bnode)
        val first = firstValueForSubjectPredicate(model, bnode, RDF.FIRST)
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
        statementsForSubjectPredicate(model, subject, SHACL.TARGET_CLASS)
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetClass(it.stringValue())) }
            }

        // sh:targetNode
        statementsForSubjectPredicate(model, subject, SHACL.TARGET_NODE)
            .forEach { stmt ->
                targets.add(TargetNode())
            }

        // sh:targetSubjectsOf
        statementsForSubjectPredicate(model, subject, SHACL.TARGET_SUBJECTS_OF)
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetSubjectsOf()) }
            }

        // sh:targetObjectsOf
        statementsForSubjectPredicate(model, subject, SHACL.TARGET_OBJECTS_OF)
            .forEach { stmt ->
                (stmt.`object` as? IRI)?.let { targets.add(TargetObjectsOf()) }
            }

        // sh:target with SHACL-AF target declarations (e.g., SHACL-SPARQL targets)
        val shTarget = vf.createIRI("http://www.w3.org/ns/shacl#target")
        (statementsForSubjectPredicate(model, subject, SHACL.TARGET) + statementsForSubjectPredicate(model, subject, shTarget))
            .forEach { stmt ->
                val target = stmt.`object` as? Resource ?: return@forEach
                val select = parseString(model, target, SHACL.SELECT)
                    ?: statementsForSubject(model, target).firstOrNull {
                        it.subject == target && it.predicate.localName.equals("select", ignoreCase = true)
                    }?.`object`?.stringValue()

                select?.let {
                    targets.add(TargetSparql())
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
        val directNodeKinds = listOfNotNull(directNodeKind)
        val directClassConstraints = statementsForSubjectPredicate(model, subject, SHACL.CLASS)
            .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        val directDatatype = parseIri(model, subject, SHACL.DATATYPE)
        val directDatatypes = listOfNotNull(directDatatype)
        val directNodeRefs = statementsForSubjectPredicate(model, subject, SHACL.NODE)
            .mapNotNull { (it.`object` as? Resource)?.let { res -> resourceRef(res) } }

        val orAlternatives = parseListResources(model, subject, SHACL.OR)
        val orClassConstraints = orAlternatives.flatMap { alternative ->
            statementsForSubjectPredicate(model, alternative, SHACL.CLASS)
                .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        }
        val orDatatypes = orAlternatives.mapNotNull { alternative ->
            parseIri(model, alternative, SHACL.DATATYPE)
        }
        val orNodeKinds = orAlternatives.mapNotNull { alternative ->
            parseNodeKind(model, alternative)
        }
        val orNodeRefs = orAlternatives.flatMap { alternative ->
            statementsForSubjectPredicate(model, alternative, SHACL.NODE)
                .mapNotNull { (it.`object` as? Resource)?.let { res -> resourceRef(res) } }
        }

        val allNodeKinds = (directNodeKinds + orNodeKinds).distinct()
        val allDatatypes = (directDatatypes + orDatatypes).distinct()

        return NodeConstraints(
            nodeKind = allNodeKinds.firstOrNull(),
            nodeKinds = allNodeKinds,
            classConstraint = (directClassConstraints + orClassConstraints).distinct(),
            datatype = allDatatypes.firstOrNull(),
            datatypes = allDatatypes,
            node = (directNodeRefs + orNodeRefs).distinct(),
            hasValue = statementsForSubjectPredicate(model, subject, SHACL.HAS_VALUE)
                .map { parseValue(it.`object`) },
            inValues = parseInConstraint(model, subject)
        )
    }

    private fun parseNodeKind(model: Model, subject: Resource): NodeKind? {
        val nodeKindValue = firstValueForSubjectPredicate(model, subject, SHACL.NODE_KIND) as? IRI ?: return null

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
        val inValue = firstValueForSubjectPredicate(model, subject, SHACL.IN) ?: return null

        return parseRdfList(model, inValue).map { parseValue(it) }
    }

    private fun parseValue(value: org.eclipse.rdf4j.model.Value): Value {
        return when (value) {
            is IRI -> Value.IriValue(value.stringValue())
            is Literal -> Value.LiteralValue(
                value = value.stringValue()
            )
            is BNode -> Value.BlankNodeValue()
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
            not = statementsForSubjectPredicate(model, subject, SHACL.NOT)
                .mapNotNull { (it.`object` as? IRI)?.stringValue() }
        )
    }

    private fun parseShapeListConstraint(model: Model, subject: Resource, predicate: IRI): List<String> {
        val listHead = firstValueForSubjectPredicate(model, subject, predicate) ?: return emptyList()

        return parseRdfList(model, listHead)
            .mapNotNull { value -> (value as? Resource)?.let { resourceRef(it) } }
    }

    private fun parseQualifiedConstraints(model: Model, subject: Resource): QualifiedConstraints? {
        val qualifiedShape = firstValueForSubjectPredicate(model, subject, SHACL.QUALIFIED_VALUE_SHAPE)
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
        return statementsForSubjectPredicate(model, subject, SHACL.SPARQL)
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
        val severityValue = firstValueForSubjectPredicate(model, subject, SHACL.SEVERITY) as? IRI

        return when (severityValue) {
            SHACL.INFO -> Severity.INFO
            SHACL.WARNING -> Severity.WARNING
            else -> Severity.VIOLATION
        }
    }

    private fun parsePrefixes(model: Model, subject: Resource): Map<String, String> {
        val prefixes = mutableMapOf<String, String>()
        statementsForSubjectPredicate(model, subject, SHACL.PREFIXES)
            .forEach { stmt ->
                (stmt.`object` as? Resource)?.let { prefixDecl ->
                    val inlinePrefix = parseString(model, prefixDecl, SHACL.PREFIX_PROP)
                    val inlineNamespace = parseIri(model, prefixDecl, SHACL.NAMESPACE_PROP)
                    if (inlinePrefix != null && inlineNamespace != null) {
                        prefixes[inlinePrefix] = inlineNamespace
                    }

                    statementsForSubjectPredicate(model, prefixDecl, SHACL.DECLARE)
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

    private fun parseOntologySchema(model: Model): OntologySchema {
        val classIris = model.filter {
            it.predicate == RDF.TYPE && (it.`object` == RDFS.CLASS || it.`object` == OWL.CLASS)
        }.mapNotNull { (it.subject as? IRI)?.stringValue() }.toSet()

        val classes = classIris.associateWith { classIri ->
            val classResource = vf.createIRI(classIri)
            OntologyClassInfo(
                iri = classIri,
                labels = parseLanguageMap(model, classResource, RDFS.LABEL),
                descriptions = parseLanguageMap(model, classResource, RDFS.COMMENT),
                equivalentClasses = parseIriList(model, classResource, OWL.EQUIVALENTCLASS).distinct(),
                subClassOf = model.filter { it.subject == classResource && it.predicate == RDFS.SUBCLASSOF }
                    .flatMap { parseClassExpression(it.`object`, model) }
                    .distinct()
            )
        }

        val propertyIris = model.filter {
            it.predicate == RDF.TYPE && it.`object` in setOf(
                OWL.OBJECTPROPERTY,
                OWL.DATATYPEPROPERTY,
                OWL.ANNOTATIONPROPERTY,
                RDF.PROPERTY
            )
        }.mapNotNull { (it.subject as? IRI)?.stringValue() }.toSet()

        val properties = propertyIris.associateWith { propertyIri ->
            val propertyResource = vf.createIRI(propertyIri)
            val propertyTypes = model.filter { it.subject == propertyResource && it.predicate == RDF.TYPE }
                .mapNotNull { (it.`object` as? IRI)?.stringValue() }
                .distinct()

            OntologyPropertyInfo(
                iri = propertyIri,
                labels = parseLanguageMap(model, propertyResource, RDFS.LABEL),
                descriptions = parseLanguageMap(model, propertyResource, RDFS.COMMENT),
                propertyTypes = propertyTypes,
                domains = model.filter { it.subject == propertyResource && it.predicate == RDFS.DOMAIN }
                    .flatMap { parseClassExpression(it.`object`, model) }
                    .distinct(),
                ranges = model.filter { it.subject == propertyResource && it.predicate == RDFS.RANGE }
                    .flatMap { parseClassExpression(it.`object`, model) }
                    .distinct(),
                equivalentProperties = parseIriList(model, propertyResource, OWL.EQUIVALENTPROPERTY).distinct(),
                subPropertyOf = parseIriList(model, propertyResource, RDFS.SUBPROPERTYOF).distinct()
            )
        }

        return OntologySchema(classes = classes, properties = properties)
    }

    private fun parseClassExpression(value: org.eclipse.rdf4j.model.Value, model: Model): List<String> {
        return when (value) {
            is IRI -> listOf(value.stringValue())
            is BNode -> {
                val unionOf = firstValueForSubjectPredicate(model, value, OWL.UNIONOF) ?: return emptyList()
                parseRdfList(model, unionOf).flatMap { parseClassExpression(it, model) }
            }
            else -> emptyList()
        }
    }

    private fun parseListResources(model: Model, subject: Resource, predicate: IRI): List<Resource> {
        val listHead = firstValueForSubjectPredicate(model, subject, predicate) ?: return emptyList()

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
        return firstValueForSubjectPredicate(model, subject, predicate)?.stringValue()
    }

    private fun parseIri(model: Model, subject: Resource, predicate: IRI): String? {
        return firstValueForSubjectPredicate(model, subject, predicate)?.let { (it as? IRI)?.stringValue() }
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
        return statementsForSubjectPredicate(model, subject, predicate)
            .mapNotNull { (it.`object` as? IRI)?.stringValue() }
    }

    private fun parseStringList(model: Model, subject: Resource, predicate: IRI): List<String>? {
        val listHead = firstValueForSubjectPredicate(model, subject, predicate) ?: return null

        return parseRdfList(model, listHead).map { it.stringValue() }
    }

    private fun parseLanguageMap(model: Model, subject: Resource, predicate: IRI): Map<String, String> {
        val result = mutableMapOf<String, String>()
        statementsForSubjectPredicate(model, subject, predicate)
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

            val first = firstValueForSubjectPredicate(model, current, RDF.FIRST)
            if (first != null) {
                result.add(first)
            }

            current = firstValueForSubjectPredicate(model, current, RDF.REST)
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
                properties = aggregateProperties(nodeShape.properties, shapesGraph.ontologySchema, targetClass),
                closed = nodeShape.closed,
                description = description
            )
        }
    }

    private fun aggregateProperties(
        properties: List<PropertyShape>,
        ontologySchema: OntologySchema = OntologySchema(),
        sourceClass: String? = null
    ): List<PropertyInfo> {
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
            if (variants.isEmpty()) null else aggregatePropertyVariants(path, variants, ontologySchema, sourceClass)
        }
    }

    private fun toPropertyPathString(path: PropertyPath): String {
        return when (path) {
            is PredicatePath -> path.predicate
            else -> path.toSparql()
        }
    }

    private fun aggregatePropertyVariants(
        path: String,
        variants: List<PropertyShape>,
        ontologySchema: OntologySchema,
        sourceClass: String?
    ): PropertyInfo {
        val names = variants.mapNotNull { shape ->
            shape.name[""] ?: shape.name.values.firstOrNull()
        }.map { it.trim() }.filter { it.isNotBlank() }

        val descriptions = variants.mapNotNull { shape ->
            shape.description[""] ?: shape.description.values.firstOrNull()
        }.map { it.trim() }.filter { it.isNotBlank() }

        val datatypes = variants.flatMap { shape ->
            if (shape.nodeConstraints.datatypes.isNotEmpty()) {
                shape.nodeConstraints.datatypes
            } else {
                listOfNotNull(shape.nodeConstraints.datatype)
            }
        }.distinct()
        val nodeKinds = variants.flatMap { shape ->
            if (shape.nodeConstraints.nodeKinds.isNotEmpty()) {
                shape.nodeConstraints.nodeKinds.map { it.name }
            } else {
                listOfNotNull(shape.nodeConstraints.nodeKind?.name)
            }
        }.distinct()
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

        val ontologyProperty = ontologySchema.properties[path]
        val ontologyRanges = if (sourceClass == null || ontologyProperty == null || ontologyProperty.domains.isEmpty() || sourceClass in ontologyProperty.domains) {
            ontologyProperty?.ranges.orEmpty()
        } else {
            emptyList()
        }

        val ontologyDatatypes = ontologyRanges.filter { isDatatypeIri(it) }
        val ontologyClassRanges = ontologyRanges.filterNot { isDatatypeIri(it) }

        val effectiveDatatypes = if (datatypes.isEmpty()) ontologyDatatypes.distinct() else datatypes
        val effectiveClassConstraints = if (classConstraints.isEmpty()) ontologyClassRanges.distinct() else classConstraints

        return PropertyInfo(
            path = path,
            name = names.firstOrNull() ?: extractPropertyName(path),
            datatype = if (effectiveDatatypes.size == 1) effectiveDatatypes.first() else null,
            datatypes = if (effectiveDatatypes.size > 1) effectiveDatatypes else null,
            nodeKind = if (nodeKinds.size == 1) nodeKinds.first() else null,
            nodeKinds = if (nodeKinds.size > 1) nodeKinds else null,
            classConstraint = if (effectiveClassConstraints.size == 1) effectiveClassConstraints.first() else null,
            classConstraints = if (effectiveClassConstraints.size > 1) effectiveClassConstraints else null,
            nodeShapeRef = if (nodeShapeRefs.size == 1) nodeShapeRefs.first() else null,
            nodeShapeRefs = if (nodeShapeRefs.size > 1) nodeShapeRefs else null,
            minCount = minCount,
            maxCount = maxCount,
            required = (minCount ?: 0) > 0,
            recommended = severities.contains(Severity.WARNING.name),
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

    private fun isDatatypeIri(uri: String): Boolean {
        val normalized = uri.lowercase()
        return normalized.contains("/xmlschema#") ||
            normalized.contains("/xsd#") ||
            normalized.contains("/xsd/") ||
            normalized.endsWith("#literal") ||
            normalized.endsWith("/literal") ||
            normalized.contains("datatype")
    }
}
