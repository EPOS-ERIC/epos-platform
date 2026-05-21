package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ApiResponse
import org.epos.api.models.ErrorResponse
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.services.ProvenanceService

fun Route.lifecycleRoutes(
    resourceController: ResourceController,
    client: PropertyGraphClient,
    provenanceService: ProvenanceService
) {
    route("/api/v2/{resourceType}") {
        post("/{id}/submit") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.submitEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        post("/{id}/publish") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.publishEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        post("/{id}/draft") {
            val conceptualUri = call.parameters["id"] ?: ""
            val newVersion = call.request.queryParameters["version"] ?: "2.0"
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.createNewDraft(conceptualUri, newVersion, actorHeaders.agent, actorHeaders.group)
            call.respond(HttpStatusCode.Created, response)
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
                val exists = client.exists("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).has('status','published').limit(1).toList()")

                if (!exists) {
                    call.respond(
                        HttpStatusCode.NotFound,
                        ErrorResponse("NotFound", "Entity not found in PUBLISHED")
                    )
                    return@post
                }

                val conceptualUri = entityUri.substringBeforeLast("/")
                val versionUri = entityUri

                val conceptualExists = client.exists("g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).limit(1).toList()")

                if (!conceptualExists) {
                    client.runWrite("""
                        g.addV('Conceptual').property('id', ${gStr(conceptualUri)}).property('type','Resource').property('createdAt', '${java.time.Instant.now()}').iterate()
                    """.trimIndent())
                }

                client.runWrite("""
                    def c = g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualUri)}).next()
                    def v = g.V().hasLabel('Version').has('id', ${gStr(versionUri)}).next()
                    g.V(c.id()).outE('CURRENT_VERSION').drop().iterate()
                    g.V(c.id()).addE('CURRENT_VERSION').to(V(v.id())).iterate()
                """.trimIndent())

                val hasProvenance = client.exists("g.V().hasLabel('Version').has('id', ${gStr(versionUri)}).in('AFFECTED').hasLabel('Activity').limit(1).toList()")

                if (!hasProvenance) {
                    provenanceService.recordInitialCreation(
                        versionUri = versionUri,
                        conceptualUri = conceptualUri,
                        source = "manual-repair",
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

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
