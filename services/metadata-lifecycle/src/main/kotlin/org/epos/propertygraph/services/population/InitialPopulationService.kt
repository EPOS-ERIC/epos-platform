package org.epos.propertygraph.services.population

import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
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
import org.epos.propertygraph.Config
import org.epos.api.models.ResourceTypeInfo
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.ProvenanceService
import org.epos.propertygraph.utils.UriUtils
import org.epos.services.ShaclService
import java.io.StringReader
import java.nio.file.Files
import java.nio.file.Path

data class PopulationResult(
    val success: Boolean,
    val sourceUrl: String,
    val subjectsProcessed: Int,
    val triplesInserted: Int,
    val message: String,
    val error: String? = null
)

class InitialPopulationService(
    private val client: PropertyGraphClient,
    private val httpClient: HttpClient,
    private val provenanceService: ProvenanceService,
    private val shaclService: ShaclService,
) {

    suspend fun populateInitial(indexUrl: String = Config.INDEX_URL) {
        println("=== Initial Population Started ===")

        ensureGremlinSupport()
        ensureIndexesReadyForPopulation()

        val urls = fetchIndex(indexUrl)
        println("Found ${urls.size} TTL files to process")

        val results = downloadAll(urls)

        var successCount = 0
        var errorCount = 0
        val failures = mutableListOf<String>()

        results.forEachIndexed { index, (url, model) ->
            println("\n[${index + 1}/${results.size}] Processing: $url")

            try {
                processModelWithStats(url, model)
                successCount++
            } catch (e: Exception) {
                errorCount++
                println("  ERROR: ${e.message}")
                failures += "$url :: ${e.message ?: e::class.simpleName ?: "Unknown error"}"
                e.printStackTrace()
            }
        }

        println("\n=== Initial Population Completed ===")
        println("Total: ${results.size} | Success: $successCount | Errors: $errorCount")

        if (errorCount > 0) {
            val preview = failures.take(10).joinToString(" | ")
            val suffix = if (failures.size > 10) " | ... (${failures.size - 10} more)" else ""
            throw IllegalStateException(
                "Population completed with $errorCount errors out of ${results.size} sources. Failures: $preview$suffix",
            )
        }
    }

    private fun ensureGremlinSupport() {
        try {
            client.runQuery("g.V().limit(1).count()")
        } catch (e: Exception) {
            throw IllegalStateException("Gremlin preflight failed: ${e.message}", e)
        }
    }

    private fun ensureIndexesReadyForPopulation() {
        try {
            println("  Ensuring JanusGraph indexes for population...")
            client.runWrite(
                """
                mgmt = graph.openManagement()
                idKey = mgmt.getPropertyKey('id') ?: mgmt.makePropertyKey('id').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
                statusKey = mgmt.getPropertyKey('status') ?: mgmt.makePropertyKey('status').dataType(String.class).cardinality(org.janusgraph.core.Cardinality.SINGLE).make()
                versionLabel = mgmt.getVertexLabel('Version') ?: mgmt.makeVertexLabel('Version').make()
                conceptualLabel = mgmt.getVertexLabel('Conceptual') ?: mgmt.makeVertexLabel('Conceptual').make()

                if (versionLabel != null && mgmt.getGraphIndex('idxVersionId') == null) {
                    mgmt.buildIndex('idxVersionId', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                        .addKey(idKey)
                        .indexOnly(versionLabel)
                        .buildCompositeIndex()
                }
                if (versionLabel != null && mgmt.getGraphIndex('idxVersionStatus') == null) {
                    mgmt.buildIndex('idxVersionStatus', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                        .addKey(statusKey)
                        .indexOnly(versionLabel)
                        .buildCompositeIndex()
                }
                if (conceptualLabel != null && mgmt.getGraphIndex('idxConceptualId') == null) {
                    mgmt.buildIndex('idxConceptualId', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                        .addKey(idKey)
                        .indexOnly(conceptualLabel)
                        .buildCompositeIndex()
                }
                if (mgmt.getGraphIndex('idxVertexIdGlobal') == null) {
                    mgmt.buildIndex('idxVertexIdGlobal', org.apache.tinkerpop.gremlin.structure.Vertex.class)
                        .addKey(idKey)
                        .buildCompositeIndex()
                }
                mgmt.commit()
                """.trimIndent()
            )

            val maxAttempts = 12
            val waitMillis = 5000L
            var enabled = false

            repeat(maxAttempts) { attempt ->
                val statusesRows =
                    client.runQuery(
                        """
                        mgmt = graph.openManagement()
                        def statusMap = [
                            versionId: mgmt.getGraphIndex('idxVersionId')?.getIndexStatus(mgmt.getPropertyKey('id'))?.name(),
                            versionStatus: mgmt.getGraphIndex('idxVersionStatus')?.getIndexStatus(mgmt.getPropertyKey('status'))?.name(),
                            conceptualId: mgmt.getGraphIndex('idxConceptualId')?.getIndexStatus(mgmt.getPropertyKey('id'))?.name(),
                            globalId: mgmt.getGraphIndex('idxVertexIdGlobal')?.getIndexStatus(mgmt.getPropertyKey('id'))?.name()
                        ]
                        mgmt.rollback()
                        statusMap
                        """.trimIndent(),
                    )

                val statuses = mutableMapOf<String, String>()
                statusesRows.forEach { row ->
                    row.forEach { (key, value) ->
                        if (value != null) statuses[key] = value.toString()
                    }
                }

                val versionIdStatus = statuses["versionId"] ?: "UNKNOWN"
                val versionStatusStatus = statuses["versionStatus"] ?: "UNKNOWN"
                val conceptualIdStatus = statuses["conceptualId"] ?: "UNKNOWN"
                val globalIdStatus = statuses["globalId"] ?: "UNKNOWN"

                println(
                    "  Index status check ${attempt + 1}/$maxAttempts: " +
                        "idxVersionId=$versionIdStatus, idxVersionStatus=$versionStatusStatus, idxConceptualId=$conceptualIdStatus, idxVertexIdGlobal=$globalIdStatus",
                )

                val allEnabled =
                    versionIdStatus == "ENABLED" &&
                        conceptualIdStatus == "ENABLED" &&
                        (versionStatusStatus == "ENABLED" || versionStatusStatus == "REGISTERED" || versionStatusStatus == "UNKNOWN")

                if (allEnabled) {
                    enabled = true
                    return@repeat
                }

                Thread.sleep(waitMillis)
            }

            if (!enabled) {
                throw IllegalStateException(
                    "Indexes not ENABLED after ${maxAttempts * waitMillis / 1000}s; aborting populate to avoid full scans",
                )
            }

            println("  JanusGraph indexes are ENABLED; starting population")
        } catch (e: Exception) {
            throw IllegalStateException("Could not ensure JanusGraph indexes before population: ${e.message}", e)
        }
    }

    private suspend fun fetchIndex(indexUrl: String): List<String> = withContext(Dispatchers.IO) {
        httpClient.get(indexUrl).bodyAsText()
            .lines()
            .map { it.trim() }
            .filter { it.isNotEmpty() && !it.startsWith("#") }
    }

    private suspend fun downloadAll(urls: List<String>): List<Pair<String, Model>> =
        withContext(Dispatchers.IO) {
            urls.map { url ->
                async {
                    println("Downloading: $url")
                    val ttl = httpClient.get(url).bodyAsText()
                    val model = parseTurtleWithRecovery(ttl, url)
                    url to model
                }
            }.awaitAll()
        }

    private fun parseTurtleWithRecovery(ttlContent: String, source: String): Model {
        return try {
            Rio.parse(StringReader(ttlContent), Config.BASE_URI, RDFFormat.TURTLE)
        } catch (e: Exception) {
            println("  Strict Turtle parsing failed for '$source': ${e.message}")
            println("  Retrying with relaxed parser and IRI recovery...")

            try {
                val recovered = LinkedHashModel()
                val parser = Rio.createParser(RDFFormat.TURTLE)
                parser.set(BasicParserSettings.VERIFY_URI_SYNTAX, false)
                parser.setParseErrorListener(ParseErrorLogger())
                parser.setRDFHandler(StatementCollector(recovered))
                parser.parse(StringReader(ttlContent), Config.BASE_URI)
                normalizeNonAbsoluteIris(recovered, "imported-iri")
            } catch (relaxedError: Exception) {
                println("  Relaxed parsing also failed: ${relaxedError.message}")
                println("  Rewriting non-absolute IRI tokens and retrying...")

                val sanitized = sanitizeNonAbsoluteIriTokens(ttlContent)
                Rio.parse(StringReader(sanitized), Config.BASE_URI, RDFFormat.TURTLE)
            }
        }
    }

    private fun normalizeNonAbsoluteIris(model: Model, context: String): Model {
        val iriMap = mutableMapOf<String, org.eclipse.rdf4j.model.IRI>()

        fun normalizeIri(iri: org.eclipse.rdf4j.model.IRI): org.eclipse.rdf4j.model.IRI {
            val value = iri.stringValue()
            return iriMap.getOrPut(value) {
                val absolute = UriUtils.ensureAbsolute(value, context)
                org.eclipse.rdf4j.model.util.Values.iri(absolute)
            }
        }

        fun normalizeValue(value: org.eclipse.rdf4j.model.Value): org.eclipse.rdf4j.model.Value = when (value) {
            is org.eclipse.rdf4j.model.IRI -> normalizeIri(value)
            else -> value
        }

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val subject = when (val subj = stmt.subject) {
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
            if (UriUtils.isAbsolute(raw)) match.value
            else "<${UriUtils.ensureAbsolute(raw, "imported-iri")}>"
        }
    }

    private fun extractSubjects(model: Model): List<String> {
        return model.asSequence()
            .filter { it.subject is org.eclipse.rdf4j.model.IRI }
            .map { it.subject.stringValue() }
            .filterNot { it.startsWith("genid-") || it.startsWith("_:") }
            .distinct()
            .toList()
    }

    private fun skolemizeBlankNodes(model: Model, parentUris: List<String>): Model {
        val blankNodes = model.asSequence()
            .flatMap { sequenceOf(it.subject, it.`object`) }
            .filterIsInstance<org.eclipse.rdf4j.model.BNode>()
            .distinct()
            .toList()

        if (blankNodes.isEmpty()) return model

        val mapping = blankNodes.associateWith { bnode ->
            val parent = model.find { it.`object` == bnode && it.subject is org.eclipse.rdf4j.model.IRI }
                ?.subject?.stringValue()
                ?: parentUris.firstOrNull()
                ?: Config.ID_BASE
            generateBlankNodeUri(bnode, model, parent)
        }

        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val newSubject = (stmt.subject as? org.eclipse.rdf4j.model.BNode)?.let {
                    vf.createIRI(mapping[it] ?: it.toString())
                } ?: stmt.subject

                val newObject = when (val obj = stmt.`object`) {
                    is org.eclipse.rdf4j.model.BNode -> vf.createIRI(mapping[obj] ?: obj.toString())
                    else -> obj
                }

                add(newSubject, stmt.predicate, newObject)
            }
        }
    }

    private fun generateBlankNodeUri(bnode: org.eclipse.rdf4j.model.BNode, model: Model, parentUri: String): String {
        val triples = model.filter { it.subject == bnode }.sortedBy { "${it.predicate}${it.`object`}" }
        val typeName = triples.find { it.predicate == org.eclipse.rdf4j.model.vocabulary.RDF.TYPE }
            ?.`object`?.let { (it as? org.eclipse.rdf4j.model.IRI)?.localName?.lowercase() } ?: "resource"
        val hash = UriUtils.generateHash(triples.joinToString("|") { "${it.predicate}=${it.`object`}" })
        return "${parentUri.substringBeforeLast("/")}/bnode-$typeName-$hash"
    }

    private fun replaceUri(model: Model, oldUri: String, newUri: String): Model {
        val oldIRI = org.eclipse.rdf4j.model.util.Values.iri(oldUri)
        val newIRI = org.eclipse.rdf4j.model.util.Values.iri(newUri)
        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()

        return LinkedHashModel().apply {
            model.forEach { stmt ->
                val subject = if (stmt.subject == oldIRI) newIRI else stmt.subject
                val predicate = if (stmt.predicate == oldIRI) newIRI else stmt.predicate
                val obj = if (stmt.`object` == oldIRI) newIRI else stmt.`object`
                add(subject, predicate, obj)
            }
        }
    }

    suspend fun populateFromSingleFile(ttlUrl: String, authorization: String? = null): PopulationResult {
        return runPopulationJob("Single File", ttlUrl) {
            println("Downloading TTL file...")
            val ttlContent = httpClient.get(ttlUrl) {
                authorization?.let { header("Authorization", it) }
            }.bodyAsText()

            println("Parsing TTL content...")
            val model = parseTurtleWithRecovery(ttlContent, ttlUrl)
            ttlUrl to processModelWithStats(ttlUrl, model)
        }
    }

    suspend fun populateFromLocalFile(filePath: String): PopulationResult {
        return runPopulationJob("Local File", filePath) {
            val path = Path.of(filePath)

            require(Files.exists(path)) { "Local file not found: $filePath" }
            require(Files.isRegularFile(path)) { "Path is not a regular file: $filePath" }
            require(Files.isReadable(path)) { "Local file is not readable: $filePath" }

            println("Reading local TTL file...")
            val ttlContent = Files.readString(path)
            val sourceLabel = "file://${path.toAbsolutePath()}"

            println("Parsing TTL content...")
            val model = parseTurtleWithRecovery(ttlContent, sourceLabel)
            sourceLabel to processModelWithStats(sourceLabel, model)
        }
    }

    suspend fun populateFromContent(ttlContent: String, sourceName: String = "direct-upload"): PopulationResult {
        return runPopulationJob("Direct Content", sourceName) {
            println("Parsing TTL content (${ttlContent.length} chars)...")
            val model = parseTurtleWithRecovery(ttlContent, sourceName)
            sourceName to processModelWithStats(sourceName, model)
        }
    }

    private suspend fun runPopulationJob(
        sourceType: String,
        sourceIdentifier: String,
        operation: suspend () -> Pair<String, ProcessingStats>
    ): PopulationResult = withContext(Dispatchers.IO) {
        println("=== $sourceType Population Started ===")
        println("Source: $sourceIdentifier")

        try {
            val (sourceUrl, stats) = operation()

            println("\n=== $sourceType Population Completed ===")
            println("Subjects processed: ${stats.subjectsProcessed}")
            println("Triples inserted: ${stats.triplesInserted}")

            PopulationResult(
                success = true,
                sourceUrl = sourceUrl,
                subjectsProcessed = stats.subjectsProcessed,
                triplesInserted = stats.triplesInserted,
                message = "Population completed successfully"
            )
        } catch (e: Exception) {
            println("ERROR: ${e.message}")
            PopulationResult(
                success = false,
                sourceUrl = sourceIdentifier,
                subjectsProcessed = 0,
                triplesInserted = 0,
                message = "Population failed: ${e.message}",
                error = e.message
            )
        }
    }

    private fun processModelWithStats(sourceUrl: String, originalModel: Model): ProcessingStats {
        val subjects = extractSubjects(originalModel)
        if (subjects.isEmpty()) {
            println("  No subjects found, skipping")
            return ProcessingStats(0, 0)
        }
        println("  Found ${subjects.size} subject(s)")

        val skolemized = skolemizeBlankNodes(originalModel, subjects)
        val allSubjects = extractSubjects(skolemized)
        println("  After skolemization: ${allSubjects.size} subjects")

        val uriMappings = allSubjects.associate { uri ->
            val hasVersion = UriUtils.hasVersion(uri)
            val conceptual = if (hasVersion) UriUtils.extractConceptualUri(uri) else uri
            val versioned = if (hasVersion) uri else UriUtils.ensureVersioned(uri)

            if (!hasVersion) println("  $uri -> $versioned")

            uri to Pair(conceptual, versioned)
        }

        var processedModel = skolemized
        uriMappings.forEach { (oldUri, mapping) ->
            if (oldUri != mapping.second) {
                processedModel = replaceUri(processedModel, oldUri, mapping.second)
            }
        }

        val entityDataMap = mutableMapOf<String, MutableMap<String, Any>>()
        val entityEdgeMap = mutableMapOf<String, MutableList<EdgeDraft>>()
        val importedVersionUris = uriMappings.values.map { it.second }.toSet()
        val resourceTypesByName = shaclService.getResourceTypes().associateBy { normalizeShapeName(it.name) }
        val subjectTypeByVersionUri = extractSubjectTypeByVersionUri(processedModel)

        processedModel.forEach { stmt ->
            val subject = stmt.subject.stringValue()
            if (subject.startsWith("genid-") || subject.startsWith("_:")) return@forEach

            val predicate = stmt.predicate.stringValue()
            val obj = stmt.`object`

            val data = entityDataMap.getOrPut(subject) { mutableMapOf() }

            val propName = sanitizePropertyKey(predicate.substringAfterLast("/").substringAfterLast("#"))

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
                        val existing = data[propName]
                        data[propName] = when (existing) {
                            null -> targetUri
                            is List<*> -> existing + targetUri
                            else -> listOf(existing, targetUri)
                        }
                    }
                }
                is org.eclipse.rdf4j.model.Literal -> {
                    val value = obj.stringValue()
                    val existing = data[propName]
                    data[propName] = when (existing) {
                        null -> value
                        is List<*> -> existing + value
                        else -> listOf(existing, value)
                    }
                }
                is org.eclipse.rdf4j.model.BNode -> {
                    val bnodeProps = mutableMapOf<String, Any>()
                    processedModel.filter { it.subject == obj }.forEach { bStmt ->
                        val bProp = bStmt.predicate.stringValue().substringAfterLast("/").substringAfterLast("#")
                        val bObj = bStmt.`object`
                        bnodeProps[bProp] = when (bObj) {
                            is org.eclipse.rdf4j.model.IRI -> bObj.stringValue()
                            is org.eclipse.rdf4j.model.Literal -> bObj.stringValue()
                            else -> bObj.stringValue()
                        }
                    }
                    if (bnodeProps.isNotEmpty()) {
                        val existing = data[propName]
                        data[propName] = when (existing) {
                            null -> bnodeProps
                            is List<*> -> existing + bnodeProps
                            else -> listOf(existing, bnodeProps)
                        }
                    }
                }
            }
        }

        var triplesInserted = 0

        uriMappings.values.forEach { (conceptual, versioned) ->
            val typeUri = processedModel.filter { it.subject.stringValue() == versioned }
                .find { it.predicate.stringValue().endsWith("type") || it.predicate == org.eclipse.rdf4j.model.vocabulary.RDF.TYPE }
                ?.`object`?.stringValue() ?: "Resource"

            val typeShortName = typeUri.substringAfterLast("/").substringAfterLast("#").lowercase()

            val data = entityDataMap[versioned] ?: emptyMap()
            val timestamp = java.time.Instant.now().toString()
            val script = buildInsertVersionScript(
                conceptual = conceptual,
                versioned = versioned,
                typeShortName = typeShortName,
                timestamp = timestamp,
                data = data,
                edges = entityEdgeMap[versioned].orEmpty(),
            )
            client.runWrite(script)

            triplesInserted += data.size
        }

        entityEdgeMap.forEach { (sourceVersionId, edges) ->
            if (edges.isNotEmpty()) {
                client.runWrite(buildAttachEdgesScript(sourceVersionId, edges))
            }
        }
        println("  Inserted into Property Graph ($triplesInserted property sets)")

        uriMappings.values.forEach { (conceptual, versioned) ->
            provenanceService.recordInitialCreation(
                versionUri = versioned,
                conceptualUri = conceptual,
                source = sourceUrl
            )
        }
        println("  Recorded provenance (${uriMappings.size} entities)")

        return ProcessingStats(uriMappings.size, triplesInserted)
    }

    private fun buildInsertVersionScript(
        conceptual: String,
        versioned: String,
        typeShortName: String,
        timestamp: String,
        data: Map<String, Any>,
        edges: List<EdgeDraft>,
    ): String {
        val propertyAssignments =
            data.entries.flatMap { (key, value) ->
                val propKey = gremlinString(key)
                when (value) {
                    is List<*> -> listOf("v.property($propKey, ${gremlinValue(value)})")
                    else -> listOf("v.property($propKey, ${gremlinScalarValue(value)})")
                }
            }.joinToString("\n")

        return """
            import static org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.*
            def c = g.V().hasLabel('Conceptual').has('id', ${gremlinString(conceptual)}).fold().coalesce(
                unfold(),
                addV('Conceptual')
                    .property('id', ${gremlinString(conceptual)})
                    .property('type', ${gremlinString(typeShortName)})
                    .property('createdAt', ${gremlinString(timestamp)})
            ).next()

            def v = g.V().hasLabel('Version').has('id', ${gremlinString(versioned)}).fold().coalesce(
                unfold(),
                addV('Version')
                    .property('id', ${gremlinString(versioned)})
                    .property('status', 'published')
                    .property('versionNumber', '1.0')
                    .property('createdAt', ${gremlinString(timestamp)})
                    .property('updatedAt', ${gremlinString(timestamp)})
            ).next()

            ${propertyAssignments}

            if (!g.V(c.id()).out('HAS_VERSION').has('id', ${gremlinString(versioned)}).hasNext()) {
                g.V(c.id()).addE('HAS_VERSION').to(V(v.id())).iterate()
            }

            if (!g.V(c.id()).out('CURRENT_VERSION').has('id', ${gremlinString(versioned)}).hasNext()) {
                g.V(c.id()).addE('CURRENT_VERSION').to(V(v.id())).iterate()
            }
        """.trimIndent()
    }

    private fun buildAttachEdgesScript(
        sourceVersionId: String,
        edges: List<EdgeDraft>,
    ): String {
        val edgeAssignments =
            edges.distinct().joinToString("\n") { edge ->
                """
                def t_${edge.safeVarName()} = g.V().hasLabel('Version').has('id', ${gremlinString(edge.targetVersionUri)}).limit(1).toList()
                if (t_${edge.safeVarName()}.isEmpty()) {
                    t_${edge.safeVarName()} = g.V().hasLabel('Conceptual').has('id', ${gremlinString(edge.targetVersionUri)})
                        .out('CURRENT_VERSION').hasLabel('Version').limit(1).toList()
                }
                if (!t_${edge.safeVarName()}.isEmpty()) {
                    def tId_${edge.safeVarName()} = t_${edge.safeVarName()}[0].value('id')
                    if (!g.V(v.id()).out(${gremlinString(edge.label)}).has('id', tId_${edge.safeVarName()}).hasNext()) {
                        g.V(v.id()).addE(${gremlinString(edge.label)}).to(t_${edge.safeVarName()}[0])
                            .property('predicateIri', ${gremlinString(edge.predicateIri)})
                            .property('predicateKey', ${gremlinString(edge.predicateKey)})
                            .iterate()
                    }
                }
                """.trimIndent()
            }

        return """
            def vs = g.V().hasLabel('Version').has('id', ${gremlinString(sourceVersionId)}).limit(1).toList()
            if (!vs.isEmpty()) {
                def v = vs[0]
                ${edgeAssignments}
            }
        """.trimIndent()
    }

    private fun gremlinString(value: String): String =
        "'${value.replace("\\", "\\\\").replace("'", "\\'").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t")}'"

    private fun sanitizePropertyKey(key: String): String {
        val cleaned = key.trim().ifBlank { "prop" }
        val reserved = setOf("id", "label", "key", "value", "property")
        return if (cleaned in reserved) "pg_$cleaned" else cleaned
    }

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
                val typeShort = stmt.`object`.stringValue().substringAfterLast('/').substringAfterLast('#')
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
        val normalizedKey = normalizeShapeName(predicateKey)

        val property = resourceType.properties.firstOrNull { prop ->
            normalizeShapeName(prop.name) == normalizedKey ||
                normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#')) == normalizedKey ||
                prop.path.equals(predicateIri, ignoreCase = true)
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

    private fun gremlinValue(value: Any): String = when (value) {
        is String -> gremlinString(value)
        is Number, is Boolean -> value.toString()
        is List<*>, is Map<*, *> -> gremlinString(com.fasterxml.jackson.databind.ObjectMapper().writeValueAsString(value))
        else -> gremlinString(value.toString())
    }

    private fun gremlinScalarValue(value: Any?): String = when (value) {
        null -> gremlinString("")
        is String -> gremlinString(value)
        is Number, is Boolean -> value.toString()
        else -> gremlinString(value.toString())
    }

    private data class EdgeDraft(
        val label: String,
        val targetVersionUri: String,
        val predicateIri: String,
        val predicateKey: String,
    ) {
        fun safeVarName(): String = (label + targetVersionUri).hashCode().toString().replace("-", "n")
    }

    private data class ProcessingStats(val subjectsProcessed: Int, val triplesInserted: Int)
}
