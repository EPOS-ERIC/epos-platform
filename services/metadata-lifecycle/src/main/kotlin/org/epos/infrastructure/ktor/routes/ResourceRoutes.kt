package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.controllers.ResourceController
import org.epos.api.models.*
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService

/**
 * Resource type listing routes.
 */
fun Route.resourceTypeRoutes(shaclService: ShaclService, resourceController: ResourceController) {
    get("/api/v2/resource-types") {
        val resourceTypes = shaclService.getResourceTypes()
        resourceController.setResourceType(resourceTypes)

        val includeShapeDefinitions = call.request.queryParameters["includeShapeDefinitions"]?.toBoolean() ?: false
        val includeTransitiveShapeDefinitions = call.request.queryParameters["includeTransitiveShapeDefinitions"]?.toBoolean() ?: false
        val includeResolvedRefs = call.request.queryParameters["includeResolvedRefs"]?.toBoolean() ?: false

        if (!includeShapeDefinitions && !includeTransitiveShapeDefinitions) {
            call.respond(resourceTypes)
            return@get
        }

        val shapeDefinitions = shaclService.getParser().getShapeDefinitions(
            onlyReachableFromResourceTypes = includeTransitiveShapeDefinitions,
            includeResolvedRefs = includeResolvedRefs
        )
        call.respond(ResourceTypesResponse(resourceTypes = resourceTypes, shapeDefinitions = shapeDefinitions))
    }
}

/**
 * CRUD routes for resources.
 */
fun Route.resourceCrudRoutes(
    resourceController: ResourceController,
    shaclService: ShaclService,
    graphDBClient: GraphDBClient
) {
    route("/api/v2/{resourceType}") {
        // List entities
        get {
            val resourceType = call.parameters["resourceType"] ?: ""
            val status = call.request.queryParameters["status"] ?: "published"
            val page = call.request.queryParameters["page"]?.toIntOrNull() ?: 1
            val pageSize = call.request.queryParameters["pageSize"]?.toIntOrNull() ?: 20

            val resourceTypes = shaclService.getResourceTypes()
            resourceController.setResourceType(resourceTypes)

            val response = resourceController.listEntities(resourceType, status, page, pageSize)
            call.respond(response)
        }

        // Create entity
        post {
            val resourceType = call.parameters["resourceType"] ?: ""
            val request = call.receive<CreateEntityRequest>()
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM

            val response = resourceController.createEntity(
                request.copy(type = resourceType),
                agent
            )
            call.respond(HttpStatusCode.Created, response)
        }

        // Get single entity
        get("/{id}") {
            val entityUri = call.parameters["id"] ?: ""
            val status = call.request.queryParameters["status"] ?: "published"

            val formatParam = call.request.queryParameters["format"]?.lowercase()
            val acceptHeader = call.request.header("Accept")?.lowercase() ?: "application/json"
            val requestedFormat = formatParam ?: acceptHeader

            val graphUri = when (status.lowercase()) {
                "draft" -> Config.GRAPH_DRAFT
                "submitted" -> Config.GRAPH_SUBMITTED
                "published" -> Config.GRAPH_PUBLISHED
                "archived" -> Config.GRAPH_ARCHIVED
                else -> Config.GRAPH_PUBLISHED
            }

            val exists = graphDBClient.ask(
                """
                ASK {
                    GRAPH <$graphUri> {
                        <$entityUri> ?p ?o .
                    }
                }
                """.trimIndent()
            )

            if (!exists) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                return@get
            }

            // Handle different RDF formats
            if (requestedFormat.contains("turtle") || requestedFormat == "ttl") {
                val ttl = resourceController.getEntityAsRdf(
                    entityUri, status, org.eclipse.rdf4j.rio.RDFFormat.TURTLE
                )
                if (ttl.isNotEmpty()) {
                    call.respondText(ttl, ContentType.parse("text/turtle"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }
            
            if (requestedFormat.contains("ld+json") || requestedFormat.contains("json-ld") || requestedFormat == "jsonld") {
                val jsonld = resourceController.getEntityAsRdf(
                    entityUri, status, org.eclipse.rdf4j.rio.RDFFormat.JSONLD
                )
                if (jsonld.isNotEmpty()) {
                    call.respondText(jsonld, ContentType.parse("application/ld+json"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }
            
            if (requestedFormat.contains("rdf+xml") || requestedFormat == "rdfxml" || requestedFormat == "xml") {
                val rdfxml = resourceController.getEntityAsRdf(
                    entityUri, status, org.eclipse.rdf4j.rio.RDFFormat.RDFXML
                )
                if (rdfxml.isNotEmpty()) {
                    call.respondText(rdfxml, ContentType.parse("application/rdf+xml"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }
            
            if (requestedFormat.contains("n-triples") || requestedFormat.contains("ntriples") || requestedFormat == "nt") {
                val ntriples = resourceController.getEntityAsRdf(
                    entityUri, status, org.eclipse.rdf4j.rio.RDFFormat.NTRIPLES
                )
                if (ntriples.isNotEmpty()) {
                    call.respondText(ntriples, ContentType.parse("application/n-triples"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }
            
            if (requestedFormat.contains("json") || requestedFormat == "*/*") {
                val response = resourceController.getEntity(entityUri, status, enrichReferences = true)
                if (response != null) {
                    call.respond(response)
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }
            
            call.respond(
                HttpStatusCode.NotAcceptable,
                ErrorResponse(
                    "NotAcceptable",
                    "Unsupported format. Use ?format= with: json, turtle (or ttl), jsonld, rdfxml (or xml), ntriples (or nt)"
                )
            )
        }

        // Update entity
        put("/{id}") {
            val entityUri = call.parameters["id"] ?: ""
            val request = call.receive<UpdateEntityRequest>()
            val agent = call.request.header("X-Agent") ?: Config.AGENT_SYSTEM

            val response = resourceController.updateEntity(entityUri, request, agent)
            call.respond(response)
        }

        // Delete entity
        delete("/{id}") {
            val entityUri = call.parameters["id"] ?: ""
            val response = resourceController.deleteEntity(entityUri)
            call.respond(response)
        }
    }
}
