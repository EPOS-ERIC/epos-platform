package org.epos.graphdb.services

import io.ktor.client.HttpClient
import io.ktor.client.request.get
import io.ktor.client.statement.bodyAsText
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.eclipse.rdf4j.model.BNode
import org.eclipse.rdf4j.model.IRI
import org.eclipse.rdf4j.model.Model
import org.eclipse.rdf4j.model.Resource
import org.eclipse.rdf4j.model.Value
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.model.util.Values
import org.eclipse.rdf4j.model.vocabulary.DCTERMS
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.eclipse.rdf4j.rio.helpers.BasicParserSettings
import org.eclipse.rdf4j.rio.helpers.ParseErrorLogger
import org.eclipse.rdf4j.rio.helpers.StatementCollector
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.utils.ModelUtils
import org.epos.graphdb.utils.NS
import org.epos.graphdb.utils.UriUtils
import org.epos.graphdb.utils.toIRI
import java.io.StringReader
import java.nio.file.Files
import java.nio.file.Path

class RevisionImportService(
    private val graphDBClient: GraphDBClient,
    private val httpClient: HttpClient,
    private val lifecycleService: EntityLifecycleService,
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
        val importedSubjects = ModelUtils.extractSubjects(model)
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
        val allowedModel = keepReachableSubgraph(rewrittenModel, importedToDraft.values.map { it.toIRI() }.toSet())
        val preparedModel = preserveCoreLifecycleTriples(allowedModel, importedToDraft.values.toSet())

        var triplesInserted = 0
        if (!options.dryRun) {
            importedToDraft.values.toSet().forEach { draftUri ->
                graphDBClient.deleteFromGraph(draftUri, Config.GRAPH_DRAFT)
            }
            graphDBClient.insertIntoGraph(preparedModel, Config.GRAPH_DRAFT)
            triplesInserted = preparedModel.size
        }

        return RevisionImportResult(
            success = true,
            source = source,
            dryRun = options.dryRun,
            subjectsProcessed = importedSubjects.size,
            triplesPrepared = preparedModel.size,
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
                    nextVersionFrom(currentPublished)
                }
            }

        val draftUri = "$conceptualUri/$targetVersion-draft"
        val draftExists =
            graphDBClient.ask(
                """
                ASK {
                    GRAPH <${Config.GRAPH_DRAFT}> {
                        <$draftUri> ?p ?o .
                    }
                }
                """.trimIndent(),
            )

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
        val remap = importedToDraft.mapKeys { it.key.toIRI() }.mapValues { it.value.toIRI() }
        val rewritten = LinkedHashModel()

        model.forEach { statement ->
            val newSubject =
                when (val subject = statement.subject) {
                    is IRI -> remap[subject] ?: subject
                    else -> subject
                }

            val newObject: Value =
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
        val remapKeys = importedToDraft.keys.map { it.toIRI() }.toSet()
        return model.count { statement -> statement.`object` is IRI && remapKeys.contains(statement.`object`) }
    }

    private fun keepReachableSubgraph(
        model: Model,
        rootSubjects: Set<IRI>,
    ): Model {
        val allowedSubjects = mutableSetOf<Resource>().apply { addAll(rootSubjects) }
        var changed = true

        while (changed) {
            changed = false
            model.forEach { statement ->
                if (allowedSubjects.contains(statement.subject) && statement.`object` is BNode) {
                    if (allowedSubjects.add(statement.`object` as BNode)) {
                        changed = true
                    }
                }
            }
        }

        return LinkedHashModel().apply {
            model.forEach { statement ->
                if (allowedSubjects.contains(statement.subject)) {
                    add(statement)
                }
            }
        }
    }

    private fun preserveCoreLifecycleTriples(
        model: Model,
        draftUris: Set<String>,
    ): Model {
        val draftIris = draftUris.map { it.toIRI() }.toSet()
        val prepared = LinkedHashModel()
        val keepPredicates = setOf(RDF.TYPE, NS.PAV_VERSION, DCTERMS.IS_VERSION_OF)

        model.forEach { statement ->
            if (statement.subject in draftIris && statement.predicate in keepPredicates) {
                return@forEach
            }
            prepared.add(statement)
        }

        draftUris.forEach { draftUri ->
            val existing = graphDBClient.getSubjectTriples(draftUri, Config.GRAPH_DRAFT)
            val draftIri = Values.getValueFactory().createIRI(draftUri)
            existing.forEach { statement ->
                if (statement.subject == draftIri && statement.predicate in keepPredicates) {
                    prepared.add(statement)
                }
            }
        }

        return prepared
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
                ModelUtils.normalizeNonAbsoluteIris(recovered, "revision-import")
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
}

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
