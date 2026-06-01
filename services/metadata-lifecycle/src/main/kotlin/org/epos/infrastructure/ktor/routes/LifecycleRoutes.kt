package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ApiResponse
import org.epos.api.models.ErrorResponse
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.ProvenanceService
import java.net.URLDecoder

fun Route.lifecycleRoutes(
    resourceController: ResourceController,
    client: PropertyGraphClient,
    provenanceService: ProvenanceService
) {
    route("/api/v2/{resourceType}") {
        post("/{id}/submit") {
            val entityUri = decodePathId(call.parameters["id"])
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.submitEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respondLifecycleResponse(response)
        }

        post("/{id}/publish") {
            val entityUri = decodePathId(call.parameters["id"])
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.publishEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respondLifecycleResponse(response)
        }

        post("/{id}/draft") {
            val conceptualUri = decodePathId(call.parameters["id"])
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.createNewDraft(conceptualUri, actorHeaders.agent, actorHeaders.group)
            if (response.success) {
                call.respond(HttpStatusCode.Created, response)
            } else {
                call.respondLifecycleResponse(response)
            }
        }

        post("/{id}/discard") {
            val entityUri = decodePathId(call.parameters["id"])
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.discardEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respondLifecycleResponse(response)
        }

        get("/{id}/history") {
            val conceptualUri = call.parameters["id"] ?: ""
            val response = resourceController.getVersionHistory(conceptualUri)
            call.respond(response)
        }

        post("/{id}/repair") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            try {
                val exists =
                    client.exists(
                        "MATCH (v:Version {id: ${'$'}id}) WHERE toLower(v.status) = 'published' RETURN v.id AS value LIMIT 1",
                        mapOf("id" to entityUri),
                    )

                if (!exists) {
                    call.respond(
                        HttpStatusCode.NotFound,
                        ErrorResponse("NotFound", "Entity not found in PUBLISHED")
                    )
                    return@post
                }

                val conceptualUri = entityUri.substringBeforeLast("/")
                val versionUri = entityUri

                val conceptualExists =
                    client.exists(
                        "MATCH (c:Conceptual {id: ${'$'}id}) RETURN c.id AS value LIMIT 1",
                        mapOf("id" to conceptualUri),
                    )

                if (!conceptualExists) {
                    client.runWrite(
                        "MERGE (c:Conceptual {id: ${'$'}id}) ON CREATE SET c.type = 'Resource', c.createdAt = ${'$'}createdAt",
                        mapOf("id" to conceptualUri, "createdAt" to java.time.Instant.now().toString()),
                    )
                }

                client.runWrite(
                    """
                    MATCH (c:Conceptual {id: ${'$'}conceptualUri})
                    OPTIONAL MATCH (c)-[old:CURRENT_VERSION]->(:Version)
                    DELETE old
                    WITH c
                    MATCH (v:Version {id: ${'$'}versionUri})
                    MERGE (c)-[:CURRENT_VERSION]->(v)
                    """.trimIndent(),
                    mapOf("conceptualUri" to conceptualUri, "versionUri" to versionUri),
                )

                val hasProvenance =
                    client.exists(
                        "MATCH (a:Activity)-[:AFFECTED]->(v:Version {id: ${'$'}id}) RETURN a.id AS value LIMIT 1",
                        mapOf("id" to versionUri),
                    )

                if (!hasProvenance) {
                    provenanceService.recordInitialCreation(
                        versionUri = versionUri,
                        agent = actorHeaders.agent,
                        group = actorHeaders.group
                    )
                }

                call.respond(
                    ApiResponse(
                        success = true,
                        message = "Entity repaired successfully",
                        data = mapOf(
                            "uri" to entityUri,
                            "conceptualUri" to conceptualUri,
                            "conceptualCreated" to !conceptualExists,
                            "provenanceCreated" to !hasProvenance
                        )
                    )
                )
            } catch (e: Exception) {
                call.respond(
                    HttpStatusCode.InternalServerError,
                    ErrorResponse("Error", "Failed to repair entity: ${e.message}")
                )
            }
        }
    }
}

private fun decodePathId(raw: String?): String {
    val value = raw?.trim().orEmpty()
    if (value.isBlank()) return ""
    return runCatching { URLDecoder.decode(value, "UTF-8") }.getOrDefault(value)
}

private suspend fun io.ktor.server.application.ApplicationCall.respondLifecycleResponse(response: ApiResponse) {
    if (response.success) {
        respond(response)
        return
    }

    val message = response.message.lowercase()
    val status = when {
        message.contains("not found") -> HttpStatusCode.NotFound
        message.contains("create/use a draft") || message.contains("only drafts") -> HttpStatusCode.Conflict
        else -> HttpStatusCode.BadRequest
    }
    respond(status, response)
}
