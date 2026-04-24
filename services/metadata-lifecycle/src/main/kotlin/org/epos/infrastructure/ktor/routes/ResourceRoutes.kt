package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.eclipse.rdf4j.model.Model
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.epos.api.controllers.ResourceController
import org.epos.api.models.*
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService
import java.io.StringWriter

/**
 * Resource type listing routes.
 */
fun Route.resourceTypeRoutes(shaclService: ShaclService, resourceController: ResourceController) {
    get("/api/v2/resource-types") {
        val resourceTypes = shaclService.getResourceTypes()
        resourceController.setResourceType(resourceTypes)

        val view = call.request.queryParameters["view"]?.lowercase() ?: "parsed"
        if (view == "shacl") {
            val model = shaclService.getParser().getCachedModel()
            if (model == null) {
                call.respond(
                    HttpStatusCode.ServiceUnavailable,
                    ErrorResponse("NotReady", "SHACL model is not initialized yet")
                )
                return@get
            }

            val formatParam = call.request.queryParameters["format"]?.lowercase()
            val acceptHeader = call.request.header(HttpHeaders.Accept)?.lowercase()
            val rdfFormat = resolveRdfFormat(formatParam, acceptHeader)
            val contentType = resolveRdfContentType(rdfFormat)
            val body = serializeModel(model, rdfFormat)
            call.respondText(body, contentType)
            return@get
        }

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

private fun serializeModel(model: Model, format: RDFFormat): String {
    val writer = StringWriter()
    Rio.write(model, writer, format)
    return writer.toString()
}

private fun resolveRdfFormat(formatParam: String?, acceptHeader: String?): RDFFormat {
    return when {
        formatParam == "ttl" || formatParam == "turtle" -> RDFFormat.TURTLE
        formatParam == "jsonld" || formatParam == "json-ld" -> RDFFormat.JSONLD
        formatParam == "rdfxml" || formatParam == "rdf+xml" || formatParam == "xml" -> RDFFormat.RDFXML
        formatParam == "ntriples" || formatParam == "n-triples" || formatParam == "nt" -> RDFFormat.NTRIPLES
        acceptHeader?.contains("text/turtle") == true -> RDFFormat.TURTLE
        acceptHeader?.contains("application/ld+json") == true || acceptHeader?.contains("application/json") == true -> RDFFormat.JSONLD
        acceptHeader?.contains("application/rdf+xml") == true -> RDFFormat.RDFXML
        acceptHeader?.contains("application/n-triples") == true -> RDFFormat.NTRIPLES
        else -> RDFFormat.TURTLE
    }
}

private fun resolveRdfContentType(format: RDFFormat): ContentType {
    return when (format) {
        RDFFormat.TURTLE -> ContentType.parse("text/turtle")
        RDFFormat.JSONLD -> ContentType.parse("application/ld+json")
        RDFFormat.RDFXML -> ContentType.parse("application/rdf+xml")
        RDFFormat.NTRIPLES -> ContentType.parse("application/n-triples")
        else -> ContentType.Text.Plain
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
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response = resourceController.createEntity(
                request.copy(type = resourceType),
                actorHeaders.agent,
                actorHeaders.group
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
            val actorHeaders = call.requireActorHeaders() ?: return@put

            val response = resourceController.updateEntity(entityUri, request, actorHeaders.agent, actorHeaders.group)
            call.respond(response)
        }

        // Patch entity (merge semantics)
        patch("/{id}") {
            val entityUri = call.parameters["id"] ?: ""
            val request = call.receive<UpdateEntityRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@patch

            val response = resourceController.patchEntity(entityUri, request, actorHeaders.agent, actorHeaders.group)
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
