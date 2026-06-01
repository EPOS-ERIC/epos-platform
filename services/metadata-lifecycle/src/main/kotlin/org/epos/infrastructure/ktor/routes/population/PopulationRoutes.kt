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
import org.epos.propertygraph.Config
import org.epos.propertygraph.services.population.InitialPopulationService
import org.epos.propertygraph.services.population.RelationReconciliationService
import org.epos.propertygraph.services.population.RevisionImportOptions
import org.epos.propertygraph.services.population.RevisionImportService
import org.epos.propertygraph.services.population.VersionMode

/**
 * Routes for database population operations.
 */
fun Route.populationRoutes(
    populationService: InitialPopulationService,
    revisionImportService: RevisionImportService,
    relationReconciliationService: RelationReconciliationService,
) {
    route("/api/v2/admin") {
        
        /**
         * POST /api/v2/admin/populate
         * Populates from configured INDEX_URL (no body) or from {url|path} body.
         */
        post("/populate") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = runCatching { call.receive<PopulateFromFileRequest>() }.getOrNull()
            val ttlUrl = request?.url?.trim()
            val localPath = request?.path?.trim()
            val indexUrl = request?.indexUrl?.trim()

            val sourceCount = listOf(ttlUrl, localPath, indexUrl).count { !it.isNullOrBlank() }
            if (sourceCount > 1) {
                call.respondBadRequest("Provide only one of url, path, or indexUrl")
                return@post
            }

            if (!ttlUrl.isNullOrBlank() && !ttlUrl.startsWith("http://") && !ttlUrl.startsWith("https://")) {
                call.respondBadRequest("url must be a valid HTTP(S) URL")
                return@post
            }

            if (!indexUrl.isNullOrBlank() && !indexUrl.startsWith("http://") && !indexUrl.startsWith("https://")) {
                call.respondBadRequest("indexUrl must be a valid HTTP(S) URL")
                return@post
            }

            val sourceAuthorization = call.request.header("X-Source-Authorization")

            try {
                println("Population triggered by agent: ${actorHeaders.agent}")
                println("Population group: ${actorHeaders.group}")

                when {
                    ttlUrl.isNullOrBlank() && localPath.isNullOrBlank() && indexUrl.isNullOrBlank() -> {
                        println("Using INDEX_URL: ${Config.INDEX_URL}")
                        withContext(Dispatchers.IO) {
                            populationService.populateInitial()
                        }
                        call.respondSuccess(
                            message = "Population completed successfully",
                            data = mapOf(
                                "indexUrl" to Config.INDEX_URL,
                                "agent" to actorHeaders.agent,
                                "group" to actorHeaders.group,
                            ),
                        )
                    }
                    !indexUrl.isNullOrBlank() -> {
                        println("Using request indexUrl: $indexUrl")
                        withContext(Dispatchers.IO) {
                            populationService.populateInitial(indexUrl)
                        }
                        call.respondSuccess(
                            message = "Population completed successfully",
                            data = mapOf(
                                "indexUrl" to indexUrl,
                                "agent" to actorHeaders.agent,
                                "group" to actorHeaders.group,
                            ),
                        )
                    }
                    else -> {
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
                                    "agent" to actorHeaders.agent,
                                    "group" to actorHeaders.group,
                                ),
                            )
                        } else {
                            call.respondPopulationError(
                                message = result.message,
                                details = result.error,
                            )
                        }
                    }
                }
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to populate database: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }

        /**
         * POST /api/v2/admin/relations/reconcile
         * Reconciles missing semantic relation edges and backfills legacy edge metadata.
         */
        post("/relations/reconcile") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = runCatching { call.receive<ReconcileRelationsRequest>() }.getOrNull()
            val dryRun = request?.dryRun ?: true
            val resourceId = request?.resourceId?.trim()?.takeIf { it.isNotBlank() }
            val maxVersions = request?.maxVersions
            val batchSize = request?.batchSize

            try {
                val result = withContext(Dispatchers.IO) {
                    relationReconciliationService.reconcile(
                        dryRun = dryRun,
                        resourceId = resourceId,
                        maxVersions = maxVersions,
                        batchSize = batchSize,
                    )
                }

                call.respondSuccess(
                    message = if (dryRun) "Relation reconciliation dry-run completed" else "Relation reconciliation completed",
                    data = mapOf(
                        "dryRun" to result.dryRun,
                        "resourceId" to resourceId,
                        "versionsScanned" to result.versionsScanned,
                        "missingEdgesDiscovered" to result.missingEdgesDiscovered,
                        "edgesCreated" to result.edgesCreated,
                        "legacyEdgesScanned" to result.legacyEdgesScanned,
                        "legacyEdgesBackfilled" to result.legacyEdgesBackfilled,
                        "agent" to actorHeaders.agent,
                        "group" to actorHeaders.group,
                    ),
                )
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to reconcile relations: ${e.message}",
                    details = e.stackTraceToString(),
                )
            }
        }

        /**
         * POST /api/v2/admin/relations/audit
         * Audits one resource relation/property cardinality and mixed conceptual/version values.
         */
        post("/relations/audit") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = runCatching { call.receive<AuditRelationsRequest>() }.getOrNull()
            val resourceId = request?.resourceId?.trim()
            if (resourceId.isNullOrBlank()) {
                call.respondBadRequest("resourceId is required")
                return@post
            }

            try {
                val result = withContext(Dispatchers.IO) {
                    relationReconciliationService.auditResource(resourceId)
                }
                if (result == null) {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resource not found: $resourceId"))
                    return@post
                }

                call.respondSuccess(
                    message = "Relation audit completed",
                    data = mapOf(
                        "resourceId" to result.resourceId,
                        "resolvedVersionId" to result.resolvedVersionId,
                        "propertyCounts" to result.propertyCounts,
                        "relationCandidates" to result.relationCandidates,
                        "mixedConceptualAndVersionValues" to result.mixedConceptualAndVersionValues,
                        "agent" to actorHeaders.agent,
                        "group" to actorHeaders.group,
                    ),
                )
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to audit relations: ${e.message}",
                    details = e.stackTraceToString(),
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
                    "neo4jUri" to Config.NEO4J_URI,
                    "neo4jDatabase" to Config.NEO4J_DATABASE,
                )
            )
        }
        
        /**
         * POST /api/v2/admin/populate/content
         * Populates the database from TTL content uploaded directly.
         * 
         * Request body: { "content": "@prefix ...", "sourceName": "my-upload" }
         * 
         * This is useful for:
         * - Uploading ttl content directly without hosting it
         * - Testing with small TTL snippets
         * - Programmatic population from other services
         */
        post("/populate/content") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = call.receive<PopulateFromContentRequest>()

            val ttlContent = request.content
            if (ttlContent.isNullOrBlank()) {
                call.respondBadRequest("content is required in request body")
                return@post
            }

            val sourceName = request.sourceName ?: "direct-upload-${System.currentTimeMillis()}"

            try {
                println("Direct content population triggered by agent: ${actorHeaders.agent}")
                println("Population group: ${actorHeaders.group}")
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
                            "agent" to actorHeaders.agent,
                            "group" to actorHeaders.group
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

        /**
         * POST /api/v2/admin/import/revision/file
         * Imports a TTL source into DRAFT revisions for existing published entities.
         */
        post("/import/revision/file") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = call.receive<ImportRevisionFromFileRequest>()

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

            val options = parseRevisionImportOptions(
                versionModeRaw = request.versionMode,
                version = request.version,
                replaceDraftIfExists = request.replaceDraftIfExists,
                dryRun = request.dryRun
            )

            if (options == null) {
                call.respondBadRequest("Invalid versionMode. Use 'auto' or 'explicit' (requires version)")
                return@post
            }

            try {
                val result = withContext(Dispatchers.IO) {
                    when {
                        !ttlUrl.isNullOrBlank() -> revisionImportService.importFromSingleFile(ttlUrl, options, actorHeaders.agent, actorHeaders.group)
                        else -> revisionImportService.importFromLocalFile(localPath!!, options, actorHeaders.agent, actorHeaders.group)
                    }
                }

                if (result.success) {
                    call.respondSuccess(
                        message = result.message,
                        data = mapOf(
                            "source" to result.source,
                            "dryRun" to result.dryRun,
                            "subjectsProcessed" to result.subjectsProcessed,
                            "triplesPrepared" to result.triplesPrepared,
                            "triplesInserted" to result.triplesInserted,
                            "draftsCreated" to result.draftsCreated,
                            "draftsReused" to result.draftsReused,
                            "linksRewritten" to result.linksRewritten,
                            "entities" to result.entities,
                            "agent" to actorHeaders.agent,
                            "group" to actorHeaders.group
                        )
                    )
                } else {
                    call.respondPopulationError(
                        message = result.message,
                        details = result.errors.joinToString("; ")
                    )
                }
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to import revisions from file: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }

        /**
         * POST /api/v2/admin/import/revision/content
         * Imports TTL content into DRAFT revisions for existing published entities.
         */
        post("/import/revision/content") {
            val actorHeaders = call.requireActorHeaders() ?: return@post
            val request = call.receive<ImportRevisionFromContentRequest>()

            val ttlContent = request.content
            if (ttlContent.isNullOrBlank()) {
                call.respondBadRequest("content is required in request body")
                return@post
            }

            val options = parseRevisionImportOptions(
                versionModeRaw = request.versionMode,
                version = request.version,
                replaceDraftIfExists = request.replaceDraftIfExists,
                dryRun = request.dryRun
            )

            if (options == null) {
                call.respondBadRequest("Invalid versionMode. Use 'auto' or 'explicit' (requires version)")
                return@post
            }

            val sourceName = request.sourceName ?: "revision-import-${System.currentTimeMillis()}"

            try {
                val result = withContext(Dispatchers.IO) {
                    revisionImportService.importFromContent(ttlContent, sourceName, options, actorHeaders.agent, actorHeaders.group)
                }

                if (result.success) {
                    call.respondSuccess(
                        message = result.message,
                        data = mapOf(
                            "source" to result.source,
                            "dryRun" to result.dryRun,
                            "subjectsProcessed" to result.subjectsProcessed,
                            "triplesPrepared" to result.triplesPrepared,
                            "triplesInserted" to result.triplesInserted,
                            "draftsCreated" to result.draftsCreated,
                            "draftsReused" to result.draftsReused,
                            "linksRewritten" to result.linksRewritten,
                            "entities" to result.entities,
                            "agent" to actorHeaders.agent,
                            "group" to actorHeaders.group
                        )
                    )
                } else {
                    call.respondPopulationError(
                        message = result.message,
                        details = result.errors.joinToString("; ")
                    )
                }
            } catch (e: Exception) {
                call.respondPopulationError(
                    message = "Failed to import revisions from content: ${e.message}",
                    details = e.stackTraceToString()
                )
            }
        }
    }
}

private fun parseRevisionImportOptions(
    versionModeRaw: String?,
    version: String?,
    replaceDraftIfExists: Boolean?,
    dryRun: Boolean?
): RevisionImportOptions? {
    val versionMode = when (versionModeRaw?.lowercase() ?: "auto") {
        "auto" -> VersionMode.AUTO
        "explicit" -> VersionMode.EXPLICIT
        else -> return null
    }

    if (versionMode == VersionMode.EXPLICIT && version.isNullOrBlank()) {
        return null
    }

    return RevisionImportOptions(
        versionMode = versionMode,
        explicitVersion = version?.trim(),
        replaceDraftIfExists = replaceDraftIfExists ?: true,
        dryRun = dryRun ?: false
    )
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
