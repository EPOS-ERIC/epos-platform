package org.epos.propertygraph.services.population

import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.client.statement.bodyAsText
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.eclipse.rdf4j.model.IRI
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
import org.epos.propertygraph.services.EntityLifecycleService
import org.epos.propertygraph.utils.UriUtils
import org.epos.propertygraph.utils.VersionRecord
import org.epos.services.ShaclService
import java.io.StringReader
import java.nio.file.Files
import java.nio.file.Path

data class RevisionImportOptions(
    val versionMode: VersionMode,
    val explicitVersion: String? = null,
    val replaceDraftIfExists: Boolean = true,
    val dryRun: Boolean = false,
)

enum class VersionMode {
    AUTO,
    EXPLICIT,
}

data class RevisionImportEntityResult(
    val importedUri: String,
    val conceptualUri: String,
    val draftUri: String,
    val status: String,
    val message: String? = null,
)

data class RevisionImportResult(
    val success: Boolean,
    val source: String,
    val dryRun: Boolean,
    val subjectsProcessed: Int,
    val triplesPrepared: Int,
    val triplesInserted: Int,
    val draftsCreated: Int,
    val draftsReused: Int,
    val linksRewritten: Int,
    val entities: List<RevisionImportEntityResult>,
    val errors: List<String>,
    val message: String,
)

private data class DraftPlan(
    val draftUri: String,
    val status: DraftPlanStatus,
    val message: String,
)

private enum class DraftPlanStatus {
    CREATED,
    REUSED,
    SKIPPED,
}

class RevisionImportService(
    private val client: PropertyGraphClient,
    private val httpClient: HttpClient,
    private val lifecycleService: EntityLifecycleService,
    private val shaclService: ShaclService,
) {
    suspend fun importFromContent(
        ttlContent: String,
        sourceName: String,
        options: RevisionImportOptions,
        agent: String,
        group: String,
    ): RevisionImportResult =
        withContext(Dispatchers.IO) {
            val model = parseTurtleWithRecovery(ttlContent, sourceName)
            importModel(model, sourceName, options, agent, group)
        }

    suspend fun importFromSingleFile(
        ttlUrl: String,
        options: RevisionImportOptions,
        agent: String,
        group: String,
    ): RevisionImportResult =
        withContext(Dispatchers.IO) {
            val ttlContent = httpClient.get(ttlUrl).bodyAsText()
            val model = parseTurtleWithRecovery(ttlContent, ttlUrl)
            importModel(model, ttlUrl, options, agent, group)
        }

    suspend fun importFromLocalFile(
        filePath: String,
        options: RevisionImportOptions,
        agent: String,
        group: String,
    ): RevisionImportResult =
        withContext(Dispatchers.IO) {
            val path = Path.of(filePath)
            require(Files.exists(path)) { "Local file not found: $filePath" }
            require(Files.isRegularFile(path)) { "Path is not a regular file: $filePath" }
            require(Files.isReadable(path)) { "Local file is not readable: $filePath" }

            val ttlContent = Files.readString(path)
            val sourceLabel = "file://${path.toAbsolutePath()}"
            val model = parseTurtleWithRecovery(ttlContent, sourceLabel)
            importModel(model, sourceLabel, options, agent, group)
        }

    private fun importModel(
        model: Model,
        source: String,
        options: RevisionImportOptions,
        agent: String,
        group: String,
    ): RevisionImportResult {
        val importedSubjects = extractSubjects(model)
        if (importedSubjects.isEmpty()) {
            return RevisionImportResult(
                success = false,
                source = source,
                dryRun = options.dryRun,
                subjectsProcessed = 0,
                triplesPrepared = 0,
                triplesInserted = 0,
                draftsCreated = 0,
                draftsReused = 0,
                linksRewritten = 0,
                entities = emptyList(),
                errors = listOf("No URI subjects found in imported TTL content"),
                message = "Revision import failed",
            )
        }

        val conceptualPlans = linkedMapOf<String, DraftPlan>()
        val entityResults = mutableListOf<RevisionImportEntityResult>()
        val errors = mutableListOf<String>()

        importedSubjects.forEach { importedUri ->
            val conceptualUri =
                if (UriUtils.hasVersion(importedUri)) {
                    UriUtils.extractConceptualUri(importedUri)
                } else {
                    importedUri
                }

            val plan =
                conceptualPlans.getOrPut(conceptualUri) {
                    try {
                        planDraft(conceptualUri, options, agent, group)
                    } catch (e: Exception) {
                        errors.add("$conceptualUri: ${e.message}")
                        DraftPlan(
                            draftUri = "",
                            status = DraftPlanStatus.SKIPPED,
                            message = e.message ?: "Planning failed",
                        )
                    }
                }

            entityResults.add(
                RevisionImportEntityResult(
                    importedUri = importedUri,
                    conceptualUri = conceptualUri,
                    draftUri = plan.draftUri,
                    status = plan.status.name.lowercase(),
                    message = plan.message,
                ),
            )
        }

        if (errors.isNotEmpty()) {
            return RevisionImportResult(
                success = false,
                source = source,
                dryRun = options.dryRun,
                subjectsProcessed = importedSubjects.size,
                triplesPrepared = 0,
                triplesInserted = 0,
                draftsCreated = conceptualPlans.values.count { it.status == DraftPlanStatus.CREATED },
                draftsReused = conceptualPlans.values.count { it.status == DraftPlanStatus.REUSED },
                linksRewritten = 0,
                entities = entityResults,
                errors = errors,
                message = "Revision import failed",
            )
        }

        val importedToDraft =
            importedSubjects.associateWith { importedUri ->
                val conceptualUri =
                    if (UriUtils.hasVersion(importedUri)) {
                        UriUtils.extractConceptualUri(importedUri)
                    } else {
                        importedUri
                    }
                conceptualPlans.getValue(conceptualUri).draftUri
            }

        val rewrittenModel = rewriteModel(model, importedToDraft)
        val rewrittenLinks = countRewrittenLinks(model, importedToDraft)

        val entityDataMap = mutableMapOf<String, MutableMap<String, Any>>()
        val entityEdgeMap = mutableMapOf<String, MutableList<EdgeDraft>>()
        val resourceTypesByName = shaclService.getResourceTypes().associateBy { normalizeShapeName(it.name) }
        val shaclPrefixes = shaclService.getParser().getCachedShapesGraph()?.prefixes.orEmpty()
        val subjectTypeByVersionUri = extractSubjectTypeByVersionUri(rewrittenModel)

        rewrittenModel.forEach { stmt ->
            val subject = stmt.subject.stringValue()
            if (subject.startsWith("genid-") || subject.startsWith("_:")) return@forEach
            if (!importedToDraft.values.contains(subject)) return@forEach

            val predicate = stmt.predicate.stringValue()
            val obj = stmt.`object`

            val propName = propertyKeyFromPredicate(predicate, shaclPrefixes)
            if (predicate.endsWith("type") || predicate == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type" ||
                predicate.endsWith("version") || predicate.endsWith("isVersionOf")) {
                return@forEach
            }

            val data = entityDataMap.getOrPut(subject) { mutableMapOf() }

            when (obj) {
                is IRI -> {
                    if (shouldCreateRelationEdge(subject, predicate, propName, subjectTypeByVersionUri, resourceTypesByName)) {
                        val edgeLabel = sanitizeEdgeLabel(propName)
                        val edges = entityEdgeMap.getOrPut(subject) { mutableListOf() }
                        edges.add(
                            EdgeDraft(
                                label = edgeLabel,
                                targetVersionUri = obj.stringValue(),
                                predicateIri = predicate,
                                predicateKey = propName,
                            ),
                        )
                    }

                    val existing = data[propName]
                    data[propName] = when (existing) {
                        null -> obj.stringValue()
                        is List<*> -> existing + obj.stringValue()
                        else -> listOf(existing, obj.stringValue())
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
                    rewrittenModel.filter { it.subject == obj }.forEach { bStmt ->
                        val bProp = bStmt.predicate.stringValue().substringAfterLast("/").substringAfterLast("#")
                        val bObj = bStmt.`object`
                        bnodeProps[bProp] = when (bObj) {
                            is IRI -> bObj.stringValue()
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
        if (!options.dryRun) {
            entityDataMap.forEach { (draftUri, data) ->
                val setClauses =
                    data.entries.flatMap { (key, value) ->
                        val safeKey = gStr(sanitizeKey(key))
                        when (value) {
                            is List<*> -> listOf("v.property($safeKey, ${gVal(value)})")
                            else -> listOf("v.property($safeKey, ${gVal(value)})")
                        }
                    }.joinToString("\n")
                val timestamp = java.time.Instant.now().toString()

                client.runWrite("""
                    def v = g.V().hasLabel('Version').has('id', ${gStr(draftUri)}).has('status','draft').next()
                    $setClauses
                    v.property('updatedAt', ${gStr(timestamp)})
                """.trimIndent())

                triplesInserted += data.size
            }

            entityEdgeMap.forEach { (sourceVersionId, edges) ->
                if (edges.isNotEmpty()) {
                    client.runWrite(buildAttachEdgesScript(sourceVersionId, edges))
                }
            }
        }

        return RevisionImportResult(
            success = true,
            source = source,
            dryRun = options.dryRun,
            subjectsProcessed = importedSubjects.size,
            triplesPrepared = entityDataMap.values.sumOf { it.size },
            triplesInserted = triplesInserted,
            draftsCreated = conceptualPlans.values.count { it.status == DraftPlanStatus.CREATED },
            draftsReused = conceptualPlans.values.count { it.status == DraftPlanStatus.REUSED },
            linksRewritten = rewrittenLinks,
            entities = entityResults,
            errors = emptyList(),
            message = if (options.dryRun) "Revision import dry-run completed" else "Revision import completed",
        )
    }

    private fun planDraft(
        conceptualUri: String,
        options: RevisionImportOptions,
        agent: String,
        group: String,
    ): DraftPlan {
        val currentPublished =
            lifecycleService.getCurrentPublishedVersion(conceptualUri)
                ?: throw IllegalStateException("No published baseline found")

        val targetVersion =
            when (options.versionMode) {
                VersionMode.EXPLICIT -> {
                    options.explicitVersion
                        ?: throw IllegalArgumentException("version is required when versionMode=explicit")
                }

                VersionMode.AUTO -> {
                    nextVersionFrom(currentPublished.id)
                }
            }

        val draftUri = "$conceptualUri/$targetVersion-draft"

        val draftExists = client.exists("g.V().hasLabel('Version').has('id', ${gStr(draftUri)}).has('status','draft').values('id').limit(1).toList()")

        if (draftExists && !options.replaceDraftIfExists) {
            throw IllegalStateException("Draft already exists for target version ($targetVersion)")
        }

        if (!draftExists && !options.dryRun) {
            lifecycleService.createDraft(conceptualUri, targetVersion, agent, group)
            return DraftPlan(draftUri, DraftPlanStatus.CREATED, "Draft created")
        }

        val message =
            when {
                options.dryRun && !draftExists -> "Draft would be created"
                draftExists -> "Existing draft reused"
                else -> "Draft prepared"
            }
        return DraftPlan(draftUri, DraftPlanStatus.REUSED, message)
    }

    private fun rewriteModel(
        model: Model,
        importedToDraft: Map<String, String>,
    ): Model {
        val remap = importedToDraft.mapKeys { org.eclipse.rdf4j.model.util.Values.iri(it.key) }
            .mapValues { org.eclipse.rdf4j.model.util.Values.iri(it.value) }
        val rewritten = LinkedHashModel()

        model.forEach { statement ->
            val newSubject =
                when (val subject = statement.subject) {
                    is IRI -> remap[subject] ?: subject
                    else -> subject
                }

            val newObject: org.eclipse.rdf4j.model.Value =
                when (val obj = statement.`object`) {
                    is IRI -> remap[obj] ?: obj
                    else -> obj
                }

            rewritten.add(newSubject, statement.predicate, newObject)
        }

        return rewritten
    }

    private fun countRewrittenLinks(
        model: Model,
        importedToDraft: Map<String, String>,
    ): Int {
        val remapKeys = importedToDraft.keys.map { org.eclipse.rdf4j.model.util.Values.iri(it) }.toSet()
        return model.count { statement -> statement.`object` is IRI && remapKeys.contains(statement.`object`) }
    }

    private fun extractSubjects(model: Model): List<String> {
        return model.asSequence()
            .filter { it.subject is org.eclipse.rdf4j.model.IRI }
            .map { it.subject.stringValue() }
            .filterNot { it.startsWith("genid-") || it.startsWith("_:") }
            .distinct()
            .toList()
    }

    private fun nextVersionFrom(currentVersionUri: String): String {
        val segment = currentVersionUri.substringAfterLast("/").removeSuffix("-draft")
        val match = Regex("v?(\\d+)(?:\\.(\\d+))?.*").find(segment)
        if (match == null) {
            return "2.0"
        }

        val major = match.groupValues[1].toIntOrNull() ?: 1
        return "${major + 1}.0"
    }

    private fun parseTurtleWithRecovery(
        ttlContent: String,
        source: String,
    ): Model =
        try {
            Rio.parse(StringReader(ttlContent), Config.BASE_URI, RDFFormat.TURTLE)
        } catch (e: Exception) {
            println("  WARNING: strict Turtle parsing failed for '$source': ${e.message}")
            println("  Retrying with relaxed parser and IRI recovery...")

            try {
                val recovered = LinkedHashModel()
                val parser = Rio.createParser(RDFFormat.TURTLE)
                parser.set(BasicParserSettings.VERIFY_URI_SYNTAX, false)
                parser.setParseErrorListener(ParseErrorLogger())
                parser.setRDFHandler(StatementCollector(recovered))
                parser.parse(StringReader(ttlContent), Config.BASE_URI)
                recovered
            } catch (relaxedError: Exception) {
                println("  WARNING: relaxed parsing also failed: ${relaxedError.message}")
                println("  Rewriting non-absolute IRI tokens and retrying...")

                val sanitized = sanitizeNonAbsoluteIriTokens(ttlContent)
                Rio.parse(StringReader(sanitized), Config.BASE_URI, RDFFormat.TURTLE)
            }
        }

    private fun sanitizeNonAbsoluteIriTokens(ttlContent: String): String {
        val iriToken = Regex("<([^<>]+)>")
        return iriToken.replace(ttlContent) { match ->
            val raw = match.groupValues[1].trim()
            if (UriUtils.isAbsolute(raw)) {
                match.value
            } else {
                "<${UriUtils.ensureAbsolute(raw, "revision-import")}>"
            }
        }
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

    private fun buildAttachEdgesScript(
        sourceVersionId: String,
        edges: List<EdgeDraft>,
    ): String {
        val edgeAssignments =
            edges.distinct().joinToString("\n") { edge ->
                """
                def t_${edge.safeVarName()} = g.V().hasLabel('Version').has('id', ${gStr(edge.targetVersionUri)}).limit(1).toList()
                if (t_${edge.safeVarName()}.isEmpty()) {
                    t_${edge.safeVarName()} = g.V().hasLabel('Conceptual').has('id', ${gStr(edge.targetVersionUri)})
                        .out('CURRENT_VERSION').hasLabel('Version').limit(1).toList()
                }
                if (!t_${edge.safeVarName()}.isEmpty()) {
                    def tId_${edge.safeVarName()} = t_${edge.safeVarName()}[0].value('id')
                    if (!g.V(v.id()).out(${gStr(edge.label)}).has('id', tId_${edge.safeVarName()}).hasNext()) {
                        g.V(v.id()).addE(${gStr(edge.label)}).to(t_${edge.safeVarName()}[0])
                            .property('predicateIri', ${gStr(edge.predicateIri)})
                            .property('predicateKey', ${gStr(edge.predicateKey)})
                            .iterate()
                    }
                }
                """.trimIndent()
            }

        return """
            def vs = g.V().hasLabel('Version').has('id', ${gStr(sourceVersionId)}).limit(1).toList()
            if (!vs.isEmpty()) {
                def v = vs[0]
                ${edgeAssignments}
            }
        """.trimIndent()
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

        val property = resourceType.properties.firstOrNull { prop ->
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
}

private data class EdgeDraft(
    val label: String,
    val targetVersionUri: String,
    val predicateIri: String,
    val predicateKey: String,
) {
    fun safeVarName(): String = (label + targetVersionUri).hashCode().toString().replace("-", "n")
}

private fun gStr(value: String): String =
    "'${value.replace("\\", "\\\\").replace("'", "\\'").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t")}'"
private fun gVal(value: Any): String = when (value) {
    is String -> gStr(value)
    is Number, is Boolean -> value.toString()
    is Map<*, *>, is List<*> -> gStr(com.fasterxml.jackson.databind.ObjectMapper().writeValueAsString(value))
    else -> gStr(value.toString())
}
private fun sanitizeKey(key: String): String {
    val reserved = setOf("id", "label", "key", "value", "property")
    return if (key in reserved) "pg_$key" else key
}

private fun propertyKeyFromPredicate(predicate: String, shaclPrefixes: Map<String, String>): String {
    val local = predicate.substringAfterLast('/').substringAfterLast('#').trim()
    if (local.isBlank()) return sanitizeKey("prop")

    val namespace = predicate.substringBeforeLast('#', predicate.substringBeforeLast('/') + "/")
    val namespaceToken = preferredShaclPrefix(namespace, shaclPrefixes)
        ?: predicate
            .substringBeforeLast('#', predicate)
            .substringBeforeLast('/', predicate)
            .substringAfterLast('/')
            .substringAfterLast('#')
            .lowercase()
            .replace(Regex("[^a-z0-9]"), "")

    val localKey = sanitizeKey(local)
    if (namespaceToken.isBlank() || !namespaceToken.first().isLetter()) return localKey
    return "$namespaceToken:$localKey"
}

private fun preferredShaclPrefix(namespace: String, shaclPrefixes: Map<String, String>): String? {
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

private fun predicateKeyLookupCandidates(predicateKey: String): List<String> {
    val candidates = linkedSetOf(predicateKey)
    predicateKey.substringAfter(':', predicateKey).takeIf { it != predicateKey }?.let { candidates.add(it) }
    predicateKey.substringAfter('_', predicateKey).takeIf { it != predicateKey }?.let { candidates.add(it) }
    return candidates.toList()
}

private fun sanitizeEdgeLabel(label: String): String {
    val cleaned = label.trim().ifBlank { "relatedTo" }
    val normalized = cleaned.replace(Regex("[^A-Za-z0-9_\\-]"), "_")
    return "rel_$normalized"
}
