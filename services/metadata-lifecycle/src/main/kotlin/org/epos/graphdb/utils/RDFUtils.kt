package org.epos.graphdb.utils

import org.eclipse.rdf4j.model.*
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.model.util.Values
import org.eclipse.rdf4j.model.vocabulary.DCTERMS
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.model.vocabulary.RDFS
import org.epos.graphdb.Config
import java.security.MessageDigest
import java.time.Instant
import java.util.UUID

/**
 * Common RDF namespace URIs with cached IRIs
 */
object NS {
    const val PROV = "http://www.w3.org/ns/prov#"
    const val PAV = "http://purl.org/pav/"
    const val DCAT = "http://www.w3.org/ns/dcat#"
    const val XSD = "http://www.w3.org/2001/XMLSchema#"

    // Cached IRIs for frequently used predicates (lazy to avoid initialization issues)
    val PROV_GENERATED_BY: IRI by lazy { iri("${PROV}wasGeneratedBy") }
    val PROV_GENERATED_AT: IRI by lazy { iri("${PROV}generatedAtTime") }
    val PROV_ATTRIBUTED_TO: IRI by lazy { iri("${PROV}wasAttributedTo") }
    val PROV_REVISION_OF: IRI by lazy { iri("${PROV}wasRevisionOf") }
    val PROV_USED: IRI by lazy { iri("${PROV}used") }
    val PROV_STARTED_AT: IRI by lazy { iri("${PROV}startedAtTime") }
    val PROV_ASSOCIATED_WITH: IRI by lazy { iri("${PROV}wasAssociatedWith") }
    val PROV_INFLUENCED_BY: IRI by lazy { iri("${PROV}wasInfluencedBy") }
    val PROV_ACTIVITY: IRI by lazy { iri("${PROV}Activity") }
    val PAV_VERSION: IRI by lazy { iri("${PAV}version") }
    val PAV_HAS_CURRENT_VERSION: IRI by lazy { iri("${PAV}hasCurrentVersion") }
    val DCAT_RESOURCE: IRI by lazy { iri("${DCAT}Resource") }
    val XSD_DATETIME: IRI by lazy { iri("${XSD}dateTime") }
}

/** Shared ValueFactory - thread-safe singleton */
private val vf: ValueFactory = Values.getValueFactory()

// ============== Extension Functions ==============

fun iri(value: String): IRI = vf.createIRI(UriUtils.ensureAbsolute(value, "runtime-iri"))
fun String.toIRI(): IRI = vf.createIRI(UriUtils.ensureAbsolute(this, "runtime-iri"))
fun String.toLiteral(): Literal = vf.createLiteral(this)
fun String.toDateTimeLiteral(): Literal = vf.createLiteral(this, NS.XSD_DATETIME)
fun Instant.toLiteral(): Literal = this.toString().toDateTimeLiteral()
fun now(): String = Instant.now().toString()

// ============== Model Builder DSL ==============

/** DSL for building RDF models */
inline fun model(block: ModelBuilder.() -> Unit): Model = ModelBuilder().apply(block).build()

class ModelBuilder {
    val model: Model = LinkedHashModel()

    fun subject(uri: String, block: SubjectBuilder.() -> Unit) {
        SubjectBuilder(uri.toIRI(), model).apply(block)
    }

    fun build(): Model = model
}

class SubjectBuilder(private val subject: IRI, private val model: Model) {

    fun type(typeUri: String) = model.add(subject, RDF.TYPE, typeUri.toIRI())
    fun type(typeIri: IRI) = model.add(subject, RDF.TYPE, typeIri)
    fun label(value: String) = model.add(subject, RDFS.LABEL, value.toLiteral())
    fun prop(predicate: IRI, value: Value) = model.add(subject, predicate, value)
    fun prop(predicate: IRI, uri: String) = model.add(subject, predicate, uri.toIRI())

    // Versioning
    fun isVersionOf(conceptualUri: String) = model.add(subject, DCTERMS.IS_VERSION_OF, conceptualUri.toIRI())
    fun version(number: String) = model.add(subject, NS.PAV_VERSION, number.toLiteral())
    fun hasCurrentVersion(uri: String) = model.add(subject, NS.PAV_HAS_CURRENT_VERSION, uri.toIRI())

    // Provenance
    fun generatedBy(activityUri: String) = model.add(subject, NS.PROV_GENERATED_BY, activityUri.toIRI())
    fun generatedAt(timestamp: String) = model.add(subject, NS.PROV_GENERATED_AT, timestamp.toDateTimeLiteral())
    fun attributedTo(agentUri: String) = model.add(subject, NS.PROV_ATTRIBUTED_TO, agentUri.toIRI())
    fun revisionOf(previousUri: String) = model.add(subject, NS.PROV_REVISION_OF, previousUri.toIRI())
    fun used(resourceUri: String) = model.add(subject, NS.PROV_USED, resourceUri.toIRI())
    fun startedAt(timestamp: String) = model.add(subject, NS.PROV_STARTED_AT, timestamp.toDateTimeLiteral())
    fun associatedWith(agentUri: String) = model.add(subject, NS.PROV_ASSOCIATED_WITH, agentUri.toIRI())
    fun influencedBy(activityUri: String) = model.add(subject, NS.PROV_INFLUENCED_BY, activityUri.toIRI())
}

// ============== URI Utilities ==============

object UriUtils {
    private val VERSION_PATTERN = Regex(".*/(v\\d+\\.\\d+.*|\\d+\\.\\d+.*)$")
    private val ABSOLUTE_URI_PATTERN = Regex("^[a-zA-Z][a-zA-Z0-9+.-]*:.+")

    fun isAbsolute(uri: String): Boolean = ABSOLUTE_URI_PATTERN.matches(uri)
    fun hasVersion(uri: String): Boolean = VERSION_PATTERN.matches(uri)

    fun extractConceptualUri(versionedUri: String): String =
        if (hasVersion(versionedUri)) versionedUri.substringBeforeLast("/") else versionedUri

    fun ensureVersioned(uri: String, version: String = "v1.0"): String =
        if (hasVersion(uri)) uri else "$uri/$version"

    fun ensureAbsolute(value: String, context: String): String {
        if (isAbsolute(value)) return value
        println("  ⚠️  WARNING: '$value' is not an absolute URI ($context). Converting.")
        val sanitized = value.replace(Regex("[^a-zA-Z0-9_-]"), "_")
        return "${Config.BASE_URI}/$context/$sanitized".also {
            println("  ℹ️  Converted to: $it")
        }
    }

    fun generateActivityUri(): String = "${Config.BASE_URI}/activity/${UUID.randomUUID()}"

    fun generateHash(content: String, length: Int = 8): String =
        MessageDigest.getInstance("MD5")
            .digest(content.toByteArray())
            .joinToString("") { "%02x".format(it) }
            .take(length)
}

// ============== Model Utilities ==============

object ModelUtils {

    /**
     * Replace non-absolute IRIs with generated absolute IRIs.
     *
     * This is used as a recovery step for malformed input data where IRI values
     * are present but do not comply with absolute IRI syntax.
     */
    fun normalizeNonAbsoluteIris(model: Model, context: String = "imported-iri"): Model {
        val iriMap = mutableMapOf<String, IRI>()

        fun normalizeIri(iri: IRI): IRI {
            val value = iri.stringValue()
            return iriMap.getOrPut(value) {
                val absolute = UriUtils.ensureAbsolute(value, context)
                absolute.toIRI()
            }
        }

        fun normalizeValue(value: Value): Value = when (value) {
            is IRI -> normalizeIri(value)
            else -> value
        }

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val subject = when (val subj = stmt.subject) {
                    is IRI -> normalizeIri(subj)
                    else -> subj
                }
                val predicate = normalizeIri(stmt.predicate)
                val obj = normalizeValue(stmt.`object`)
                val contextValue = stmt.context

                if (contextValue is IRI) {
                    add(subject, predicate, obj, normalizeIri(contextValue))
                } else if (contextValue != null) {
                    add(subject, predicate, obj, contextValue)
                } else {
                    add(subject, predicate, obj)
                }
            }
        }
    }

    /** Extract all URI subjects (excluding blank nodes) */
    fun extractSubjects(model: Model): List<String> =
        model.asSequence()
            .filter { it.subject is IRI }
            .map { it.subject.stringValue() }
            .filterNot { it.startsWith("genid-") || it.startsWith("_:") }
            .distinct()
            .toList()

    /** Replace a URI everywhere in the model */
    fun replaceUri(model: Model, oldUri: String, newUri: String): Model {
        val oldIRI = oldUri.toIRI()
        val newIRI = newUri.toIRI()

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                add(
                    if (stmt.subject == oldIRI) newIRI else stmt.subject,
                    if (stmt.predicate == oldIRI) newIRI else stmt.predicate,
                    if (stmt.`object` == oldIRI) newIRI else stmt.`object`
                )
            }
        }
    }

    /** Replace subject URI only */
    fun replaceSubject(model: Model, oldUri: String, newUri: String): Model {
        val oldIRI = oldUri.toIRI()
        val newIRI = newUri.toIRI()

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                add(
                    if (stmt.subject == oldIRI) newIRI else stmt.subject,
                    stmt.predicate,
                    stmt.`object`
                )
            }
        }
    }

    /** Skolemize blank nodes to deterministic URIs */
    fun skolemizeBlankNodes(model: Model, parentUris: List<String>): Model {
        val blankNodes = model.asSequence()
            .flatMap { sequenceOf(it.subject, it.`object`) }
            .filterIsInstance<BNode>()
            .distinct()
            .toList()

        if (blankNodes.isEmpty()) return model

        val mapping = blankNodes.associateWith { bnode ->
            val parent = model.find { it.`object` == bnode && it.subject is IRI }
                ?.subject?.stringValue()
                ?: parentUris.firstOrNull()
                ?: Config.ID_BASE
            generateBlankNodeUri(bnode, model, parent).toIRI()
        }

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                add(
                    (stmt.subject as? BNode)?.let { mapping[it] } ?: stmt.subject,
                    stmt.predicate,
                    (stmt.`object` as? BNode)?.let { mapping[it] } ?: stmt.`object`
                )
            }
        }
    }

    private fun generateBlankNodeUri(bnode: BNode, model: Model, parentUri: String): String {
        val triples = model.filter { it.subject == bnode }.sortedBy { "${it.predicate}${it.`object`}" }
        val typeName = triples.find { it.predicate == RDF.TYPE }
            ?.`object`?.let { (it as? IRI)?.localName?.lowercase() } ?: "resource"
        val hash = UriUtils.generateHash(triples.joinToString("|") { "${it.predicate}=${it.`object`}" })
        return "${parentUri.substringBeforeLast("/")}/bnode-$typeName-$hash"
    }
}

// ============== SPARQL Queries ==============

object Sparql {
    private val PREFIXES = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dct: <http://purl.org/dc/terms/>
        PREFIX prov: <${NS.PROV}>
        PREFIX pav: <${NS.PAV}>
        PREFIX dcat: <${NS.DCAT}>
    """.trimIndent()

    fun withPrefixes(query: String): String = "$PREFIXES\n\n$query"

    fun askExists(subjectUri: String, graphUri: String): String =
        "ASK { GRAPH <$graphUri> { <$subjectUri> ?p ?o . } }"

    fun selectConceptualUri(versionUri: String): String = withPrefixes("""
        SELECT ?conceptual FROM <${Config.GRAPH_PROVENANCE}>
        WHERE { <$versionUri> dcterms:isVersionOf ?conceptual . } LIMIT 1
    """.trimIndent())

    fun selectCurrentVersion(conceptualUri: String): String = withPrefixes("""
        SELECT ?currentVersion FROM <${Config.GRAPH_CONCEPTUAL}>
        WHERE { <$conceptualUri> pav:hasCurrentVersion ?currentVersion . } LIMIT 1
    """.trimIndent())

    fun selectVersionHistory(conceptualUri: String): String = withPrefixes("""
        SELECT ?version ?title ?timestamp (SAMPLE(?agentRaw) AS ?agent) (SAMPLE(?groupRaw) AS ?group)
        FROM <${Config.GRAPH_PROVENANCE}> FROM <${Config.GRAPH_PUBLISHED}>
        FROM <${Config.GRAPH_ARCHIVED}> FROM <${Config.GRAPH_DRAFT}> FROM <${Config.GRAPH_SUBMITTED}>
        WHERE {
            ?version dcterms:isVersionOf <$conceptualUri> ; prov:generatedAtTime ?timestamp .
            OPTIONAL { ?version prov:wasAttributedTo ?agentRaw . }
            OPTIONAL {
                ?version prov:wasGeneratedBy ?activity .
                ?activity prov:wasAssociatedWith ?associatedRaw .
                FILTER(!BOUND(?agentRaw) || ?associatedRaw != ?agentRaw)
                BIND(?associatedRaw AS ?groupRaw)
            }
            OPTIONAL { ?version dct:title ?title . }
        } GROUP BY ?version ?title ?timestamp ORDER BY DESC(?timestamp)
    """.trimIndent())

    fun selectLatestTransitionStatus(versionUri: String): String = withPrefixes("""
        SELECT ?toStatus FROM <${Config.GRAPH_PROVENANCE}>
        WHERE {
            ?activity a prov:Activity ;
                      prov:used <$versionUri> ;
                      rdfs:label ?label ;
                      prov:startedAtTime ?startedAt .
            FILTER(STRSTARTS(STR(?label), "State transition: "))
            BIND(LCASE(STRAFTER(STRAFTER(STR(?label), "State transition: "), " -> ")) AS ?toStatus)
        }
        ORDER BY DESC(?startedAt)
        LIMIT 1
    """.trimIndent())
}
