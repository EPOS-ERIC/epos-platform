package org.epos.propertygraph.services.population

import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.statement.bodyAsText
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.withContext
import org.eclipse.rdf4j.model.Model
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.eclipse.rdf4j.rio.helpers.BasicParserSettings
import org.eclipse.rdf4j.rio.helpers.ParseErrorLogger
import org.eclipse.rdf4j.rio.helpers.StatementCollector
import org.epos.api.models.ResourceTypeInfo
import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.utils.UriUtils
import org.epos.services.ShaclService
import org.slf4j.LoggerFactory
import java.io.StringReader
import java.net.URI
import java.nio.file.Files
import java.nio.file.Path

data class PopulationResult(
    val success: Boolean,
    val sourceUrl: String,
    val subjectsProcessed: Int,
    val triplesInserted: Int,
    val message: String,
    val error: String? = null,
)

class InitialPopulationService(
    private val client: PropertyGraphClient,
    private val httpClient: HttpClient,
    private val provenanceService: ProvenanceService,
    private val shaclService: ShaclService,
) {
    private val logger = LoggerFactory.getLogger(javaClass)

    suspend fun populateInitial(indexUrl: String = Config.INDEX_URL) {
        logger.info("=== Initial Population Started ===")

        ensureNeo4jSupport()
        ensureIndexesReadyForPopulation()

        val urls = fetchIndex(indexUrl)
        logger.info("Found {} TTL files to process", urls.size)

        val results = downloadAll(urls)

        var successCount = 0
        var errorCount = 0
        val failures = mutableListOf<String>()

        results.forEachIndexed { index, (url, model) ->
            logger.info("[{}/{}] Processing: {}", index + 1, results.size, url)

            try {
                processModelWithStats(url, model)
                successCount++
            } catch (e: Exception) {
                errorCount++
                logger.error("ERROR processing {}: {}", url, e.message)
                failures += "$url :: ${e.message ?: e::class.simpleName ?: "Unknown error"}"
            }
        }

        logger.info("=== Initial Population Completed ===")
        logger.info("Total: {} | Success: {} | Errors: {}", results.size, successCount, errorCount)

        if (errorCount > 0) {
            val preview = failures.take(10).joinToString(" | ")
            val suffix = if (failures.size > 10) " | ... (${failures.size - 10} more)" else ""
            throw IllegalStateException(
                "Population completed with $errorCount errors out of ${results.size} sources. Failures: $preview$suffix",
            )
        }
    }

    private fun ensureNeo4jSupport() {
        try {
            client.runQuery("RETURN 1 AS ok")
        } catch (e: Exception) {
            throw IllegalStateException("Neo4j preflight failed: ${e.message}", e)
        }
    }

    private fun ensureIndexesReadyForPopulation() {
        try {
            logger.info("Ensuring Neo4j constraints and indexes for population...")
            val statements =
                listOf(
                    "CREATE CONSTRAINT conceptual_id_unique IF NOT EXISTS FOR (c:Conceptual) REQUIRE c.id IS UNIQUE",
                    "CREATE CONSTRAINT version_id_unique IF NOT EXISTS FOR (v:Version) REQUIRE v.id IS UNIQUE",
                    "CREATE INDEX version_status_idx IF NOT EXISTS FOR (v:Version) ON (v.status)",
                    "CREATE INDEX conceptual_type_idx IF NOT EXISTS FOR (c:Conceptual) ON (c.type)",
                    "CREATE INDEX conceptual_id_idx IF NOT EXISTS FOR (c:Conceptual) ON (c.id)",
                )
            statements.forEach { client.runWrite(it) }
            logger.info("Neo4j indexes are ready; starting population")
        } catch (e: Exception) {
            throw IllegalStateException("Could not ensure Neo4j indexes before population: ${e.message}", e)
        }
    }

    private suspend fun fetchIndex(indexUrl: String): List<String> =
        withContext(Dispatchers.IO) {
            httpClient
                .get(indexUrl)
                .bodyAsText()
                .lines()
                .map { it.trim() }
                .filter { it.isNotEmpty() && !it.startsWith("#") }
        }

    private suspend fun downloadAll(urls: List<String>): List<Pair<String, Model>> =
        withContext(Dispatchers.IO) {
            urls
                .map { url ->
                    async {
                        logger.debug("Downloading: {}", url)
                        val ttl = httpClient.get(url).bodyAsText()
                        val model = parseTurtleWithRecovery(ttl, url)
                        url to model
                    }
                }.awaitAll()
        }

    private fun parseTurtleWithRecovery(
        ttlContent: String,
        source: String,
    ): Model =
        try {
            Rio.parse(StringReader(ttlContent), Config.BASE_URI, RDFFormat.TURTLE)
        } catch (e: Exception) {
            logger.warn("Strict Turtle parsing failed for '{}': {}", source, e.message)
            logger.warn("Retrying with relaxed parser and IRI recovery...")

            try {
                val recovered = LinkedHashModel()
                val parser = Rio.createParser(RDFFormat.TURTLE)
                parser.set(BasicParserSettings.VERIFY_URI_SYNTAX, false)
                parser.setParseErrorListener(ParseErrorLogger())
                parser.setRDFHandler(StatementCollector(recovered))
                parser.parse(StringReader(ttlContent), Config.BASE_URI)
                normalizeNonAbsoluteIris(recovered, "imported-iri")
            } catch (relaxedError: Exception) {
                logger.warn("Relaxed parsing also failed: {}", relaxedError.message)
                logger.warn("Rewriting non-absolute IRI tokens and retrying...")

                val sanitized = sanitizeNonAbsoluteIriTokens(ttlContent)
                Rio.parse(StringReader(sanitized), Config.BASE_URI, RDFFormat.TURTLE)
            }
        }

    private fun normalizeNonAbsoluteIris(
        model: Model,
        context: String,
    ): Model {
        val iriMap = mutableMapOf<String, org.eclipse.rdf4j.model.IRI>()

        fun normalizeIri(iri: org.eclipse.rdf4j.model.IRI): org.eclipse.rdf4j.model.IRI {
            val value = iri.stringValue()
            return iriMap.getOrPut(value) {
                val absolute = UriUtils.ensureAbsolute(value, context)
                org.eclipse.rdf4j.model.util.Values
                    .iri(absolute)
            }
        }

        fun normalizeValue(value: org.eclipse.rdf4j.model.Value): org.eclipse.rdf4j.model.Value =
            when (value) {
                is org.eclipse.rdf4j.model.IRI -> normalizeIri(value)
                else -> value
            }

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val subject =
                    when (val subj = stmt.subject) {
                        is org.eclipse.rdf4j.model.IRI -> normalizeIri(subj)
                        else -> subj
                    }
                val predicate = normalizeIri(stmt.predicate)
                val obj = normalizeValue(stmt.`object`)
                val contextValue = stmt.context

                if (contextValue is org.eclipse.rdf4j.model.IRI) {
                    add(subject, predicate, obj, normalizeIri(contextValue))
                } else if (contextValue != null) {
                    add(subject, predicate, obj, contextValue)
                } else {
                    add(subject, predicate, obj)
                }
            }
        }
    }

    private fun sanitizeNonAbsoluteIriTokens(ttlContent: String): String {
        val iriToken = Regex("<([^<>]+)>")
        return iriToken.replace(ttlContent) { match ->
            val raw = match.groupValues[1].trim()
            if (UriUtils.isAbsolute(raw)) {
                match.value
            } else {
                "<${UriUtils.ensureAbsolute(raw, "imported-iri")}>"
            }
        }
    }

    private fun extractSubjects(model: Model): List<String> =
        model
            .asSequence()
            .filter { it.subject is org.eclipse.rdf4j.model.IRI }
            .map { it.subject.stringValue() }
            .filterNot { it.startsWith("genid-") || it.startsWith("_:") }
            .distinct()
            .toList()

    private fun skolemizeBlankNodes(
        model: Model,
        parentUris: List<String>,
    ): Model {
        val blankNodes =
            model
                .asSequence()
                .flatMap { sequenceOf(it.subject, it.`object`) }
                .filterIsInstance<org.eclipse.rdf4j.model.BNode>()
                .distinct()
                .toList()

        if (blankNodes.isEmpty()) return model

        val mapping =
            blankNodes.associateWith { bnode ->
                val parent = resolveBlankNodeParentUri(bnode, model, parentUris)
                generateBlankNodeUri(bnode, model, parent)
            }

        val vf =
            org.eclipse.rdf4j.model.util.Values
                .getValueFactory()
        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val newSubject =
                    (stmt.subject as? org.eclipse.rdf4j.model.BNode)?.let {
                        vf.createIRI(mapping[it] ?: it.toString())
                    } ?: stmt.subject

                val newObject =
                    when (val obj = stmt.`object`) {
                        is org.eclipse.rdf4j.model.BNode -> vf.createIRI(mapping[obj] ?: obj.toString())
                        else -> obj
                    }

                add(newSubject, stmt.predicate, newObject)
            }
        }
    }

    private fun resolveBlankNodeParentUri(
        bnode: org.eclipse.rdf4j.model.BNode,
        model: Model,
        parentUris: List<String>,
    ): String =
        model
            .asSequence()
            .filter { it.`object` == bnode && it.subject is org.eclipse.rdf4j.model.IRI }
            .map { it.subject.stringValue() }
            .firstOrNull(::isSkolemParentUri)
            ?: parentUris.firstOrNull(::isSkolemParentUri)
            ?: Config.ID_BASE

    private fun isSkolemParentUri(uri: String): Boolean {
        return try {
            val parsed = URI(uri)
            if (!parsed.scheme.equals("http", ignoreCase = true) && !parsed.scheme.equals("https", ignoreCase = true)) {
                return false
            }
            uri.trimEnd('/').startsWith(Config.BASE_URI.trimEnd('/'))
        } catch (_: Exception) {
            false
        }
    }

    private fun generateBlankNodeUri(
        bnode: org.eclipse.rdf4j.model.BNode,
        model: Model,
        parentUri: String,
    ): String {
        val triples = model.filter { it.subject == bnode }.sortedBy { "${it.predicate}${it.`object`}" }
        val typeName =
            triples
                .find { it.predicate == org.eclipse.rdf4j.model.vocabulary.RDF.TYPE }
                ?.`object`
                ?.let { (it as? org.eclipse.rdf4j.model.IRI)?.localName?.lowercase() } ?: "resource"
        val hash = UriUtils.generateHash(triples.joinToString("|") { "${it.predicate}=${it.`object`}" })
        return "${blankNodeBaseUri(parentUri)}/bnode-$typeName-$hash"
    }

    private fun blankNodeBaseUri(parentUri: String): String {
        val normalizedParent = parentUri.trimEnd('/')
        val normalizedIdBase = Config.ID_BASE.trimEnd('/')
        if (normalizedParent == normalizedIdBase) return normalizedIdBase
        return normalizedParent.substringBeforeLast("/", normalizedParent)
    }

    private fun replaceUri(
        model: Model,
        oldUri: String,
        newUri: String,
    ): Model {
        if (!isAbsoluteIri(oldUri) || !isAbsoluteIri(newUri)) {
            return model
        }

        val oldIRI =
            try {
                org.eclipse.rdf4j.model.util.Values
                    .iri(oldUri)
            } catch (_: IllegalArgumentException) {
                return model
            }
        val newIRI =
            try {
                org.eclipse.rdf4j.model.util.Values
                    .iri(newUri)
            } catch (_: IllegalArgumentException) {
                return model
            }

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val subject = if (stmt.subject == oldIRI) newIRI else stmt.subject
                val predicate = if (stmt.predicate == oldIRI) newIRI else stmt.predicate
                val obj = if (stmt.`object` == oldIRI) newIRI else stmt.`object`
                add(subject, predicate, obj)
            }
        }
    }

    private fun isAbsoluteIri(value: String): Boolean =
        try {
            URI(value).isAbsolute
        } catch (_: Exception) {
            false
        }

    suspend fun populateFromSingleFile(
        ttlUrl: String,
        authorization: String? = null,
    ): PopulationResult =
        runPopulationJob("Single File", ttlUrl) {
            logger.info("Downloading TTL file...")
            val ttlContent =
                httpClient
                    .get(ttlUrl) {
                        authorization?.let { header("Authorization", it) }
                    }.bodyAsText()

            logger.debug("Parsing TTL content...")
            val model = parseTurtleWithRecovery(ttlContent, ttlUrl)
            ttlUrl to processModelWithStats(ttlUrl, model)
        }

    suspend fun populateFromLocalFile(filePath: String): PopulationResult =
        runPopulationJob("Local File", filePath) {
            val path = Path.of(filePath)

            require(Files.exists(path)) { "Local file not found: $filePath" }
            require(Files.isRegularFile(path)) { "Path is not a regular file: $filePath" }
            require(Files.isReadable(path)) { "Local file is not readable: $filePath" }

            logger.info("Reading local TTL file...")
            val ttlContent = Files.readString(path)
            val sourceLabel = "file://${path.toAbsolutePath()}"

            logger.debug("Parsing TTL content...")
            val model = parseTurtleWithRecovery(ttlContent, sourceLabel)
            sourceLabel to processModelWithStats(sourceLabel, model)
        }

    suspend fun populateFromContent(
        ttlContent: String,
        sourceName: String = "direct-upload",
    ): PopulationResult =
        runPopulationJob("Direct Content", sourceName) {
            logger.info("Parsing TTL content ({} chars)...", ttlContent.length)
            val model = parseTurtleWithRecovery(ttlContent, sourceName)
            sourceName to processModelWithStats(sourceName, model)
        }

    private suspend fun runPopulationJob(
        sourceType: String,
        sourceIdentifier: String,
        operation: suspend () -> Pair<String, ProcessingStats>,
    ): PopulationResult =
        withContext(Dispatchers.IO) {
            logger.info("=== {} Population Started ===", sourceType)
            logger.info("Source: {}", sourceIdentifier)

            try {
                val (sourceUrl, stats) = operation()

                logger.info("=== {} Population Completed ===", sourceType)
                logger.info("Subjects processed: {}", stats.subjectsProcessed)
                logger.info("Triples inserted: {}", stats.triplesInserted)

                PopulationResult(
                    success = true,
                    sourceUrl = sourceUrl,
                    subjectsProcessed = stats.subjectsProcessed,
                    triplesInserted = stats.triplesInserted,
                    message = "Population completed successfully",
                )
            } catch (e: Exception) {
                logger.error("ERROR: {}", e.message)
                PopulationResult(
                    success = false,
                    sourceUrl = sourceIdentifier,
                    subjectsProcessed = 0,
                    triplesInserted = 0,
                    message = "Population failed: ${e.message}",
                    error = e.message,
                )
            }
        }

    private fun processModelWithStats(
        sourceUrl: String,
        originalModel: Model,
    ): ProcessingStats {
        val subjects = extractSubjects(originalModel)
        if (subjects.isEmpty()) {
            logger.warn("No subjects found, skipping")
            return ProcessingStats(0, 0)
        }
        logger.info("Found {} subject(s)", subjects.size)

        val skolemized = skolemizeBlankNodes(originalModel, subjects)
        val allSubjects = extractSubjects(skolemized)
        logger.info("After skolemization: {} subjects", allSubjects.size)

        val uriMappings =
            allSubjects.associate { uri ->
                val hasVersion = UriUtils.hasVersion(uri)
                val conceptual = if (hasVersion) UriUtils.extractConceptualUri(uri) else uri
                val versioned = if (hasVersion) uri else UriUtils.ensureVersioned(uri)

                if (!hasVersion) logger.debug("{} -> {}", uri, versioned)

                uri to Pair(conceptual, versioned)
            }

        var processedModel = skolemized
        uriMappings.forEach { (oldUri, mapping) ->
            if (oldUri != mapping.second) {
                processedModel = replaceUri(processedModel, oldUri, mapping.second)
            }
        }

        // Phase 1: collect vertex properties and relation edge drafts.
        val entityDataMap = mutableMapOf<String, MutableMap<String, Any>>()
        val entityEdgeMap = mutableMapOf<String, MutableList<EdgeDraft>>()
        val importedVersionUris = uriMappings.values.map { it.second }.toSet()
        val resourceTypesByName = shaclService.getResourceTypes().associateBy { normalizeShapeName(it.name) }
        val shaclPrefixes =
            shaclService
                .getParser()
                .getCachedShapesGraph()
                ?.prefixes
                .orEmpty()
        val subjectTypeByVersionUri = extractSubjectTypeByVersionUri(processedModel)

        processedModel.forEach { stmt ->
            val subject = stmt.subject.stringValue()
            if (subject.startsWith("genid-") || subject.startsWith("_:")) return@forEach

            val predicate = stmt.predicate.stringValue()
            val obj = stmt.`object`

            val data = entityDataMap.getOrPut(subject) { mutableMapOf() }

            val propName = propertyKeyFromPredicate(predicate, shaclPrefixes)

            when (obj) {
                is org.eclipse.rdf4j.model.IRI -> {
                    val targetUri = obj.stringValue()
                    if (shouldCreateRelationEdge(subject, predicate, propName, subjectTypeByVersionUri, resourceTypesByName)) {
                        val edgeLabel = sanitizeEdgeLabel(propName)
                        val edges = entityEdgeMap.getOrPut(subject) { mutableListOf() }
                        edges.add(
                            EdgeDraft(
                                label = edgeLabel,
                                targetVersionUri = targetUri,
                                predicateIri = predicate,
                                predicateKey = propName,
                            ),
                        )
                    }
                    if (targetUri !in importedVersionUris) {
                        appendPropertyValue(data, propName, targetUri)
                    }
                }

                is org.eclipse.rdf4j.model.Literal -> {
                    val value = obj.stringValue()
                    appendPropertyValue(data, propName, value)
                }

                is org.eclipse.rdf4j.model.BNode -> {
                    val bnodeProps = extractBNodeProperties(processedModel, obj)
                    if (bnodeProps.isNotEmpty()) {
                        appendPropertyValue(data, propName, bnodeProps)
                    }
                }
            }
        }

        var triplesInserted = 0

        // Phase 2: create/update Conceptual + Version vertices.
        uriMappings.values.forEach { (conceptual, versioned) ->
            val typeUri =
                processedModel
                    .filter { it.subject.stringValue() == versioned }
                    .find { it.predicate == org.eclipse.rdf4j.model.vocabulary.RDF.TYPE }
                    ?.`object`
                    ?.stringValue() ?: "Resource"

            val typeShortName = typeUri.substringAfterLast("/").substringAfterLast("#").lowercase()

            val data = entityDataMap[versioned] ?: emptyMap()
            val timestamp =
                java.time.Instant
                    .now()
                    .toString()
            upsertVersion(
                conceptual = conceptual,
                versioned = versioned,
                typeShortName = typeShortName,
                timestamp = timestamp,
                data = data,
            )

            triplesInserted += data.size
        }

        // Phase 3: attach relation edges now that all vertices exist.
        entityEdgeMap.forEach { (sourceVersionId, edges) ->
            if (edges.isNotEmpty()) {
                attachEdges(sourceVersionId, edges)
            }
        }
        logger.info("Inserted into Property Graph ({} property sets)", triplesInserted)

        uriMappings.values.forEach { (conceptual, versioned) ->
            provenanceService.recordInitialCreation(
                versionUri = versioned,
            )
        }
        logger.info("Recorded provenance ({} entities)", uriMappings.size)

        return ProcessingStats(uriMappings.size, triplesInserted)
    }

    private fun appendPropertyValue(
        data: MutableMap<String, Any>,
        propName: String,
        value: Any,
    ) {
        val existing = data[propName]
        data[propName] =
            when (existing) {
                null -> value
                is List<*> -> existing + value
                else -> listOf(existing, value)
            }
    }

    private fun extractBNodeProperties(
        model: Model,
        bNode: org.eclipse.rdf4j.model.BNode,
    ): Map<String, Any> {
        val bnodeProps = mutableMapOf<String, Any>()
        model.filter { it.subject == bNode }.forEach { bStmt ->
            val bProp =
                bStmt.predicate
                    .stringValue()
                    .substringAfterLast("/")
                    .substringAfterLast("#")
            bnodeProps[bProp] = bStmt.`object`.stringValue()
        }
        return bnodeProps
    }

    private fun upsertVersion(
        conceptual: String,
        versioned: String,
        typeShortName: String,
        timestamp: String,
        data: Map<String, Any>,
    ) {
        client.runWrite(
            """
            MERGE (c:Conceptual {id: ${'$'}conceptual})
            ON CREATE SET c.type = ${'$'}typeShortName, c.createdAt = ${'$'}timestamp
            SET c.type = coalesce(c.type, ${'$'}typeShortName)
            WITH c
            MERGE (v:Version {id: ${'$'}versioned})
            ON CREATE SET v.status = 'published', v.versionNumber = '1.0', v.createdAt = ${'$'}timestamp
            SET v.updatedAt = ${'$'}timestamp
            MERGE (c)-[:HAS_VERSION]->(v)
            MERGE (c)-[:CURRENT_VERSION]->(v)
            """.trimIndent(),
            mapOf(
                "conceptual" to conceptual,
                "versioned" to versioned,
                "typeShortName" to typeShortName,
                "timestamp" to timestamp,
            ),
        )

        data.forEach { (key, value) ->
            val propertyName = sanitizeCypherPropertyKey(key)
            val cypher =
                """
                MATCH (v:Version {id: ${'$'}versioned})
                SET v.`$propertyName` = ${'$'}value
                """.trimIndent()
            client.runWrite(
                cypher,
                mapOf(
                    "versioned" to versioned,
                    "value" to toNeo4jPropertyValue(value),
                ),
            )
        }
    }

    private fun attachEdges(
        sourceVersionId: String,
        edges: List<EdgeDraft>,
    ) {
        edges.distinct().forEach { edge ->
            val edgeType = sanitizeCypherRelationshipType(edge.label)
            val query =
                """
                MATCH (s:Version {id: ${'$'}sourceVersionId})
                OPTIONAL MATCH (tv:Version {id: ${'$'}targetVersionUri})
                OPTIONAL MATCH (tc:Conceptual {id: ${'$'}targetVersionUri})-[:CURRENT_VERSION]->(tcv:Version)
                WITH s, coalesce(tv, tcv) AS t
                WHERE t IS NOT NULL
                MERGE (s)-[r:$edgeType]->(t)
                SET r.predicateIri = ${'$'}predicateIri,
                    r.predicateKey = ${'$'}predicateKey
                """.trimIndent()
            client.runWrite(
                query,
                mapOf(
                    "sourceVersionId" to sourceVersionId,
                    "targetVersionUri" to edge.targetVersionUri,
                    "predicateIri" to edge.predicateIri,
                    "predicateKey" to edge.predicateKey,
                ),
            )
        }
    }

    private fun sanitizeCypherPropertyKey(key: String): String = key.replace("`", "")

    private fun sanitizeCypherRelationshipType(label: String): String {
        val cleaned = label.replace(Regex("[^A-Za-z0-9_]"), "_")
        val normalized = if (cleaned.firstOrNull()?.isDigit() == true) "R_$cleaned" else cleaned
        return normalized.ifBlank { "RELATED_TO" }
    }

    private fun toNeo4jPropertyValue(value: Any?): Any =
        when (value) {
            null -> {
                ""
            }

            is String, is Number, is Boolean -> {
                value
            }

            is List<*>, is Map<*, *> -> {
                com.fasterxml.jackson.databind
                    .ObjectMapper()
                    .writeValueAsString(value)
            }

            else -> {
                value.toString()
            }
        }

    private fun sanitizePropertyKey(key: String): String {
        val cleaned = key.trim().ifBlank { "prop" }
        val reserved = setOf("id", "label", "key", "value", "property")
        return if (cleaned in reserved) "pg_$cleaned" else cleaned
    }

    private fun propertyKeyFromPredicate(
        predicate: String,
        shaclPrefixes: Map<String, String>,
    ): String {
        val local = predicate.substringAfterLast('/').substringAfterLast('#').trim()
        if (local.isBlank()) return sanitizePropertyKey("prop")

        val namespace = predicate.substringBeforeLast('#', predicate.substringBeforeLast('/') + "/")
        val namespaceToken =
            preferredShaclPrefix(namespace, shaclPrefixes)
                ?: predicate
                    .substringBeforeLast('#', predicate)
                    .substringBeforeLast('/', predicate)
                    .substringAfterLast('/')
                    .substringAfterLast('#')
                    .lowercase()
                    .replace(Regex("[^a-z0-9]"), "")

        val localKey = sanitizePropertyKey(local)
        if (namespaceToken.isBlank() || !namespaceToken.first().isLetter()) return localKey
        return "$namespaceToken:$localKey"
    }

    private fun preferredShaclPrefix(
        namespace: String,
        shaclPrefixes: Map<String, String>,
    ): String? {
        val normalizedNamespace = normalizeNamespace(namespace)
        val candidates =
            shaclPrefixes
                .filter { (_, ns) -> normalizeNamespace(ns) == normalizedNamespace }
                .keys
                .filter { it.isNotBlank() }
        if (candidates.isEmpty()) return null
        return candidates.minWithOrNull(compareBy<String>({ isGenericPrefix(it) }, { it.length }, { it }))
    }

    private fun isGenericPrefix(prefix: String): Boolean = prefix == "ns" || prefix.matches(Regex("ns\\d*"))

    private fun normalizeNamespace(namespace: String): String = namespace.trimEnd('#', '/')

    private fun sanitizeEdgeLabel(label: String): String {
        val cleaned = label.trim().ifBlank { "relatedTo" }
        val normalized = cleaned.replace(Regex("[^A-Za-z0-9_\\-]"), "_")
        return "rel_$normalized"
    }

    private fun normalizeShapeName(typeName: String): String =
        typeName
            .lowercase()
            .replace("_", "")
            .replace("-", "")
            .replace(" ", "")

    private fun extractSubjectTypeByVersionUri(model: Model): Map<String, String> =
        model
            .filter { it.predicate == org.eclipse.rdf4j.model.vocabulary.RDF.TYPE && it.`object` is org.eclipse.rdf4j.model.IRI }
            .associate { stmt ->
                val subject = stmt.subject.stringValue()
                val typeShort =
                    stmt.`object`
                        .stringValue()
                        .substringAfterLast('/')
                        .substringAfterLast('#')
                subject to typeShort
            }

    private fun shouldCreateRelationEdge(
        subjectVersionUri: String,
        predicateIri: String,
        predicateKey: String,
        subjectTypeByVersionUri: Map<String, String>,
        resourceTypesByName: Map<String, ResourceTypeInfo>,
    ): Boolean {
        val subjectType = subjectTypeByVersionUri[subjectVersionUri] ?: return true
        val resourceType = resourceTypesByName[normalizeShapeName(subjectType)] ?: return true
        val normalizedKeyCandidates = predicateKeyLookupCandidates(predicateKey).map { normalizeShapeName(it) }

        val property =
            resourceType.properties.firstOrNull { prop ->
                prop.path.equals(predicateIri, ignoreCase = true)
            } ?: resourceType.properties.firstOrNull { prop ->
                val propName = normalizeShapeName(prop.name)
                val propPathLocal = normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#'))
                normalizedKeyCandidates.any { it == propName || it == propPathLocal }
            } ?: return true

        return !property.classConstraint.isNullOrBlank() ||
            !property.classConstraints.isNullOrEmpty() ||
            !property.nodeShapeRef.isNullOrBlank() ||
            !property.nodeShapeRefs.isNullOrEmpty() ||
            propertyAllowsResourceObject(property)
    }

    private fun propertyAllowsResourceObject(property: org.epos.api.shacl.PropertyInfo): Boolean {
        val nodeKinds = (property.nodeKinds.orEmpty() + listOfNotNull(property.nodeKind)).map { it.uppercase() }
        val hasResourceNodeKind = nodeKinds.any { it.contains("IRI") || it.contains("BLANKNODE") }
        val hasLiteralNodeKind = nodeKinds.any { it.contains("LITERAL") }
        val hasDatatype = !property.datatype.isNullOrBlank() || !property.datatypes.isNullOrEmpty()

        if (hasResourceNodeKind) return true
        if (hasLiteralNodeKind && hasDatatype) return false
        if (hasLiteralNodeKind) return false

        return !hasDatatype
    }

    private fun predicateKeyLookupCandidates(predicateKey: String): List<String> {
        val candidates = linkedSetOf(predicateKey)
        predicateKey.substringAfter(':', predicateKey).takeIf { it != predicateKey }?.let { candidates.add(it) }
        predicateKey.substringAfter('_', predicateKey).takeIf { it != predicateKey }?.let { candidates.add(it) }
        return candidates.toList()
    }

    private data class EdgeDraft(
        val label: String,
        val targetVersionUri: String,
        val predicateIri: String,
        val predicateKey: String,
    )

    private data class ProcessingStats(
        val subjectsProcessed: Int,
        val triplesInserted: Int,
    )
}
