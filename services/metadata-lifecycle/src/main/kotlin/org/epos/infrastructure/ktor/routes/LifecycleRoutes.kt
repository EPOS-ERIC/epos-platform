package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ApiResponse
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.services.ProvenanceService

/**
 * Lifecycle management routes (submit, publish, draft, repair).
 */
fun Route.lifecycleRoutes(
    resourceController: ResourceController,
    graphDBClient: GraphDBClient,
    provenanceService: ProvenanceService
) {
    route("/api/v2/{resourceType}") {
        // Submit for review
        post("/{id}/submit") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.submitEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        // Publish
        post("/{id}/publish") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.publishEntity(entityUri, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        // Create new draft from published version
        post("/{id}/draft") {
            val conceptualUri = call.parameters["id"] ?: ""
            val newVersion = call.request.queryParameters["version"] ?: "2.0"
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.createNewDraft(conceptualUri, newVersion, actorHeaders.agent, actorHeaders.group)
            call.respond(HttpStatusCode.Created, response)
        }

        // Get version history
        get("/{id}/history") {
            val conceptualUri = call.parameters["id"] ?: ""
            val response = resourceController.getVersionHistory(conceptualUri)
            call.respond(response)
        }

        // Repair missing conceptual/provenance
        post("/{id}/repair") {
            val entityUri = call.parameters["id"] ?: ""
            val actorHeaders = call.requireActorHeaders() ?: return@post

            try {
                val exists = graphDBClient.ask(
                    """
                    ASK { GRAPH <${Config.GRAPH_PUBLISHED}> { <$entityUri> ?p ?o . } }
                    """.trimIndent()
                )

                if (!exists) {
                    call.respond(
                        HttpStatusCode.NotFound,
                        ErrorResponse("NotFound", "Entity not found in PUBLISHED")
                    )
                    return@post
                }

                val conceptualUri = entityUri
                val versionUri = entityUri

                val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
                val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
                val conceptualIRI = vf.createIRI(conceptualUri)
                val versionIRI = vf.createIRI(versionUri)

                model.add(
                    conceptualIRI,
                    org.eclipse.rdf4j.model.vocabulary.RDF.TYPE,
                    vf.createIRI("http://www.w3.org/ns/dcat#Resource")
                )
                model.add(
                    conceptualIRI,
                    vf.createIRI("http://purl.org/pav/hasCurrentVersion"),
                    versionIRI
                )

                graphDBClient.deleteFromGraph(conceptualUri, Config.GRAPH_CONCEPTUAL)
                graphDBClient.insertIntoGraph(model, Config.GRAPH_CONCEPTUAL)

                val hasProvenance = graphDBClient.ask(
                    """
                    PREFIX dcterms: <http://purl.org/dc/terms/>
                    ASK { 
                        GRAPH <${Config.GRAPH_PROVENANCE}> { 
                            <$versionUri> dcterms:isVersionOf ?c . 
                        } 
                    }
                    """.trimIndent()
                )

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
                            "conceptualCreated" to true,
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
