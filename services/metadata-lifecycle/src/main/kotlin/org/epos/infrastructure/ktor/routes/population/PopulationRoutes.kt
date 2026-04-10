package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.application.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import org.epos.api.models.ApiResponse
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.services.InitialPopulationService

/**
 * Routes for database population operations.
 */
fun Route.populationRoutes(populationService: InitialPopulationService) {
    route("/api/v2/admin") {
        
        /**
         * POST /api/v2/admin/populate
         * Triggers initial population from the configured INDEX_URL
         */
        post("/populate") {
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM

            try {
                println("Population triggered by agent: $agent")
                println("Using INDEX_URL: ${Config.INDEX_URL}")

                withContext(Dispatchers.IO) {
                    populationService.populateInitial()
                }

                call.respondSuccess(
                    message = "Population completed successfully",
                    data = mapOf(
                        "indexUrl" to Config.INDEX_URL,
                        "agent" to agent
                    )
                )
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to populate database: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }
        
        /**
         * POST /api/v2/admin/populate/from-url
         * Triggers population from a custom index URL
         */
        post("/populate/from-url") {
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM
            val request = call.receive<PopulateFromUrlRequest>()

            val indexUrl = request.indexUrl
            if (indexUrl.isNullOrBlank()) {
                call.respondBadRequest("indexUrl is required in request body")
                return@post
            }

            try {
                println("Population triggered by agent: $agent")
                println("Using custom INDEX_URL: $indexUrl")

                withContext(Dispatchers.IO) {
                    populationService.populateInitial(indexUrl = indexUrl)
                }

                call.respondSuccess(
                    message = "Population completed successfully",
                    data = mapOf(
                        "indexUrl" to indexUrl,
                        "agent" to agent
                    )
                )
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to populate database: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }
        
        /**
         * GET /api/v2/admin/populate/status
         * Returns the current population configuration
         */
        get("/populate/status") {
            call.respond(
                mapOf(
                    "indexUrl" to Config.INDEX_URL,
                    "graphdbEndpoint" to Config.GRAPHDB_ENDPOINT,
                    "repositoryId" to Config.REPOSITORY_ID,
                    "graphs" to mapOf(
                        "published" to Config.GRAPH_PUBLISHED,
                        "conceptual" to Config.GRAPH_CONCEPTUAL,
                        "provenance" to Config.GRAPH_PROVENANCE
                    )
                )
            )
        }
        
        /**
         * POST /api/v2/admin/populate/file
         * Populates the database from a single TTL file URL or local file path.
         * 
         * Request body examples:
         * - { "url": "https://example.com/data.ttl" }
         * - { "path": "/data/metadata.ttl" }
         * 
         * Headers:
         * - X-Source-Authorization: Optional auth header to forward when fetching the URL
         *   (e.g., "token ghp_xxx" for private GitHub repos)
         * 
         * This is useful for:
         * - Loading a single TTL file directly
         * - Loading from GitHub raw URLs (including private repos with auth)
         * - Loading from any HTTP(S) accessible TTL file
         */
        post("/populate/file") {
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM
            val sourceAuthorization = call.request.header("X-Source-Authorization")
            val request = call.receive<PopulateFromFileRequest>()

            val ttlUrl = request.url?.trim()
            val localPath = request.path?.trim()

            if (ttlUrl.isNullOrBlank() && localPath.isNullOrBlank()) {
                call.respondBadRequest("Either url or path is required in request body")
                return@post
            }

            if (!ttlUrl.isNullOrBlank() && !localPath.isNullOrBlank()) {
                call.respondBadRequest("Provide only one of url or path")
                return@post
            }

            if (!ttlUrl.isNullOrBlank() && !ttlUrl.startsWith("http://") && !ttlUrl.startsWith("https://")) {
                call.respondBadRequest("url must be a valid HTTP(S) URL")
                return@post
            }

            try {
                println("Single file population triggered by agent: $agent")
                if (!ttlUrl.isNullOrBlank()) {
                    println("TTL URL: $ttlUrl")
                }
                if (!localPath.isNullOrBlank()) {
                    println("Local file path: $localPath")
                }
                if (!ttlUrl.isNullOrBlank() && sourceAuthorization != null) {
                    println("Using provided authorization header")
                }
                
                val result = withContext(Dispatchers.IO) {
                    when {
                        !ttlUrl.isNullOrBlank() -> populationService.populateFromSingleFile(ttlUrl, sourceAuthorization)
                        else -> populationService.populateFromLocalFile(localPath!!)
                    }
                }

                if (result.success) {
                    call.respondSuccess(
                        message = result.message,
                        data = mapOf(
                            "sourceUrl" to result.sourceUrl,
                            "subjectsProcessed" to result.subjectsProcessed,
                            "triplesInserted" to result.triplesInserted,
                            "agent" to agent
                        )
                    )
                } else {
                    call.respondPopulationError(
                        message = result.message,
                        details = result.error
                    )
                }
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to populate from file: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }
        
        /**
         * POST /api/v2/admin/populate/content
         * Populates the database from TTL content uploaded directly.
         * 
         * Request body: { "content": "@prefix ...", "sourceName": "my-upload" }
         * 
         * This is useful for:
         * - Uploading TTL content directly without hosting it
         * - Testing with small TTL snippets
         * - Programmatic population from other services
         */
        post("/populate/content") {
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM
            val request = call.receive<PopulateFromContentRequest>()

            val ttlContent = request.content
            if (ttlContent.isNullOrBlank()) {
                call.respondBadRequest("content is required in request body")
                return@post
            }

            val sourceName = request.sourceName ?: "direct-upload-${System.currentTimeMillis()}"

            try {
                println("Direct content population triggered by agent: $agent")
                println("Source name: $sourceName")
                println("Content length: ${ttlContent.length} characters")

                val result = withContext(Dispatchers.IO) {
                    populationService.populateFromContent(ttlContent, sourceName)
                }

                if (result.success) {
                    call.respondSuccess(
                        message = result.message,
                        data = mapOf(
                            "sourceName" to sourceName,
                            "subjectsProcessed" to result.subjectsProcessed,
                            "triplesInserted" to result.triplesInserted,
                            "agent" to agent
                        )
                    )
                } else {
                    call.respondPopulationError(
                        message = result.message,
                        details = result.error
                    )
                }
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to populate from content: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }
    }
}

private suspend fun ApplicationCall.respondSuccess(message: String, data: Any? = null) {
    respond(ApiResponse(success = true, message = message, data = data))
}

private suspend fun ApplicationCall.respondBadRequest(message: String) {
    respond(HttpStatusCode.BadRequest, ErrorResponse(error = "BadRequest", message = message))
}

private suspend fun ApplicationCall.respondPopulationError(message: String, details: String? = null) {
    respond(
        HttpStatusCode.InternalServerError,
        ErrorResponse(
            error = "PopulationFailed",
            message = message,
            details = details
        )
    )
}
