package org.epos.graphdb.services

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
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.utils.*
import java.nio.file.Path
import java.io.StringReader
import java.nio.file.Files

/**
 * Service for initial population of the database from TTL files.
 * Optimized version using shared RDF utilities.
 */
class InitialPopulationService(
    private val graphDBClient: GraphDBClient,
    private val httpClient: HttpClient,
    private val provenanceService: ProvenanceService
) {

    /**
     * Main entry point: downloads all TTL files and populates the database.
     */
    suspend fun populateInitial(indexUrl: String = Config.INDEX_URL) {
        println("=== Initial Population Started ===")

        val urls = fetchIndex(indexUrl)
        println("Found ${urls.size} TTL files to process")

        val results = downloadAll(urls)

        var successCount = 0
        var errorCount = 0

        results.forEachIndexed { index, (url, model) ->
            println("\n[${index + 1}/${results.size}] Processing: $url")

            try {
                processModelWithStats(url, model)
                successCount++
            } catch (e: Exception) {
                errorCount++
                println("  ❌ ERROR: ${e.message}")
                e.printStackTrace()
            }
        }

        println("\n=== Initial Population Completed ===")
        println("Total: ${results.size} | ✅ Success: $successCount | ❌ Errors: $errorCount")
    }

    // ============== Private Methods ==============

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
            println("  ⚠️  Strict Turtle parsing failed for '$source': ${e.message}")
            println("  ℹ️  Retrying with relaxed parser and IRI recovery...")

            try {
                val recovered = LinkedHashModel()
                val parser = Rio.createParser(RDFFormat.TURTLE)
                parser.set(BasicParserSettings.VERIFY_URI_SYNTAX, false)
                parser.setParseErrorListener(ParseErrorLogger())
                parser.setRDFHandler(StatementCollector(recovered))
                parser.parse(StringReader(ttlContent), Config.BASE_URI)
                ModelUtils.normalizeNonAbsoluteIris(recovered, "imported-iri")
            } catch (relaxedError: Exception) {
                println("  ⚠️  Relaxed parsing also failed: ${relaxedError.message}")
                println("  ℹ️  Rewriting non-absolute IRI tokens and retrying...")

                val sanitized = sanitizeNonAbsoluteIriTokens(ttlContent)
                Rio.parse(StringReader(sanitized), Config.BASE_URI, RDFFormat.TURTLE)
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

    private fun updateConceptualGraph(conceptualUri: String, versionUri: String) {
        // Check if exists and delete
        if (graphDBClient.ask(Sparql.askExists(conceptualUri, Config.GRAPH_CONCEPTUAL))) {
            graphDBClient.deleteFromGraph(conceptualUri, Config.GRAPH_CONCEPTUAL)
        }

        // Insert new conceptual pointer
        val conceptualModel = model {
            subject(conceptualUri) {
                type(NS.DCAT_RESOURCE)
                hasCurrentVersion(versionUri)
            }
        }
        graphDBClient.insertIntoGraph(conceptualModel, Config.GRAPH_CONCEPTUAL)
    }

    /** Helper data class for URI mappings */
    private data class UriMapping(val conceptual: String, val versioned: String)

    /**
     * Populate from a single TTL file URL.
     * Downloads the TTL file and processes it directly.
     * 
     * @param ttlUrl URL to a single TTL file
     * @param authorization Optional authorization header to use when fetching the URL
     *                      (e.g., "token ghp_xxx" for private GitHub repos)
     * @return PopulationResult with statistics
     */
    suspend fun populateFromSingleFile(ttlUrl: String, authorization: String? = null): PopulationResult {
        return runPopulationJob("Single File", ttlUrl) {
            println("Downloading TTL file...")
            val ttlContent = httpClient.get(ttlUrl) {
                authorization?.let { header("Authorization", it) }
            }.bodyAsText()

            println("Parsing TTL content...")
            val model = parseTurtleWithRecovery(ttlContent, ttlUrl)
            PopulationExecution(ttlUrl, processModelWithStats(ttlUrl, model))
        }
    }

    /**
     * Populate from a local TTL file path available on the server filesystem.
     *
     * @param filePath Absolute or relative filesystem path to a TTL file
     * @return PopulationResult with statistics
     */
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
            PopulationExecution(sourceLabel, processModelWithStats(sourceLabel, model))
        }
    }

    /**
     * Populate from TTL content directly (not from URL).
     * Useful for uploading TTL content directly.
     * 
     * @param ttlContent The TTL content as a string
     * @param sourceName A name/identifier for the source (for provenance)
     * @return PopulationResult with statistics
     */
    suspend fun populateFromContent(ttlContent: String, sourceName: String = "direct-upload"): PopulationResult {
        return runPopulationJob("Direct Content", sourceName) {
            println("Parsing TTL content (${ttlContent.length} chars)...")
            val model = parseTurtleWithRecovery(ttlContent, sourceName)
            PopulationExecution(sourceName, processModelWithStats(sourceName, model))
        }
    }

    private suspend fun runPopulationJob(
        sourceType: String,
        sourceIdentifier: String,
        operation: suspend () -> PopulationExecution
    ): PopulationResult = withContext(Dispatchers.IO) {
        println("=== $sourceType Population Started ===")
        println("Source: $sourceIdentifier")

        try {
            val execution = operation()
            val stats = execution.stats

            println("\n=== $sourceType Population Completed ===")
            println("Subjects processed: ${stats.subjectsProcessed}")
            println("Triples inserted: ${stats.triplesInserted}")

            PopulationResult(
                success = true,
                sourceUrl = execution.sourceUrl,
                subjectsProcessed = stats.subjectsProcessed,
                triplesInserted = stats.triplesInserted,
                message = "Population completed successfully"
            )
        } catch (e: Exception) {
            println("❌ ERROR: ${e.message}")
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

    /**
     * Process model and return statistics.
     */
    private fun processModelWithStats(sourceUrl: String, originalModel: Model): ProcessingStats {
        val subjects = ModelUtils.extractSubjects(originalModel)
        if (subjects.isEmpty()) {
            println("  ⚠️  No subjects found, skipping")
            return ProcessingStats(0, 0)
        }
        println("  📦 Found ${subjects.size} subject(s)")

        val skolemized = ModelUtils.skolemizeBlankNodes(originalModel, subjects)
        val allSubjects = ModelUtils.extractSubjects(skolemized)
        println("  🔗 After skolemization: ${allSubjects.size} subjects")

        val uriMappings = allSubjects.associate { uri ->
            val hasVersion = UriUtils.hasVersion(uri)
            val conceptual = if (hasVersion) UriUtils.extractConceptualUri(uri) else uri
            val versioned = if (hasVersion) uri else UriUtils.ensureVersioned(uri)

            if (!hasVersion) println("  ➕ $uri → $versioned")

            uri to UriMapping(conceptual, versioned)
        }

        var processedModel = skolemized
        uriMappings.forEach { (oldUri, mapping) ->
            if (oldUri != mapping.versioned) {
                processedModel = ModelUtils.replaceUri(processedModel, oldUri, mapping.versioned)
            }
        }

        val versioningModel = LinkedHashModel()
        uriMappings.values.forEach { mapping ->
            val m = org.epos.graphdb.utils.model {
                subject(mapping.versioned) {
                    isVersionOf(mapping.conceptual)
                    version("1.0")
                }
            }
            versioningModel.addAll(m)
        }
        processedModel.addAll(versioningModel)

        val triplesCount = processedModel.size

        graphDBClient.insertIntoGraph(processedModel, Config.GRAPH_PUBLISHED)
        println("  ✅ Inserted into PUBLISHED ($triplesCount triples)")

        uriMappings.values.forEach { mapping ->
            updateConceptualGraph(mapping.conceptual, mapping.versioned)
        }
        println("  ✅ Updated CONCEPTUAL graph (${uriMappings.size} entities)")

        uriMappings.values.forEach { mapping ->
            provenanceService.recordInitialCreation(
                versionUri = mapping.versioned,
                conceptualUri = mapping.conceptual,
                source = sourceUrl
            )
        }
        println("  ✅ Recorded provenance (${uriMappings.size} entities)")

        return ProcessingStats(uriMappings.size, triplesCount)
    }

    /** Statistics for processing */
    private data class ProcessingStats(val subjectsProcessed: Int, val triplesInserted: Int)

    private data class PopulationExecution(val sourceUrl: String, val stats: ProcessingStats)
}
