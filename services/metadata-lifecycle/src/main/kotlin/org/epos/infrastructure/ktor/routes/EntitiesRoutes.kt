package org.epos.infrastructure.ktor.routes

import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.header
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.controllers.ResourceController
import org.epos.api.models.EntityResponse
import org.epos.api.models.ErrorResponse
import org.epos.api.models.ResolvedEntityResponse
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.services.ShaclService
import java.net.URLDecoder

fun Route.entitiesRoutes(
    resourceController: ResourceController,
    shaclService: ShaclService,
    client: PropertyGraphClient,
) {
    val knownTypes = shaclService.getResourceTypes()
    resourceController.setResourceType(knownTypes)

    get("/api/v2/entities") {
        val entityTypes = knownTypes.map { it.name.lowercase() }.distinct().sorted()
        call.respond(mapOf("entityTypes" to entityTypes))
    }

    knownTypes.forEach { resourceTypeInfo ->
        val resourceType = resourceTypeInfo.name
        val routeType = resourceType.lowercase()

        get("/api/v2/entities/$routeType") {
            val status = call.request.queryParameters["status"] ?: "published"
            val page = call.request.queryParameters["page"]?.toIntOrNull() ?: 1
            val pageSize =
                (
                    call.request.queryParameters["resultsPerPage"]?.toIntOrNull()
                        ?: call.request.queryParameters["pageSize"]?.toIntOrNull()
                        ?: 20
                ).coerceIn(1, 100)

            val response = resourceController.listEntities(resourceType, status, page, pageSize)
            call.respond(response)
        }

        get("/api/v2/entities/$routeType/{id}") {
            val idParam = call.parameters["id"]?.trim().orEmpty()
            if (idParam.isBlank()) {
                call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing id"))
                return@get
            }
            if (idParam.all { it.isDigit() }) {
                call.respond(
                    HttpStatusCode.BadRequest,
                    ErrorResponse(
                        "BAD_REQUEST",
                        "Numeric internal graph IDs are not accepted; provide a conceptual/version resource identifier",
                    ),
                )
                return@get
            }

            val decodedId =
                try {
                    URLDecoder.decode(idParam, "UTF-8")
                } catch (_: Exception) {
                    idParam
                }

            val status = call.request.queryParameters["status"] ?: "published"
            val formatParam = call.request.queryParameters["format"]?.lowercase()
            val acceptHeader = call.request.header("Accept")?.lowercase() ?: "application/json"
            val requestedFormat = formatParam ?: acceptHeader
            val resolvedEntityId =
                org.epos.infrastructure.ktor.routes.search.ResourceLinkResolver
                    .resolveAccessEntityId(client, decodedId)
            if (resolvedEntityId == null) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resource not found: $decodedId"))
                return@get
            }

            val entity =
                resourceController.getEntity(resolvedEntityId, status)
                    ?: resourceController.getEntityByConceptualId(decodedId, status)
            if (entity == null) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resolved resource not found: $resolvedEntityId"))
                return@get
            }

            val rdfEntity = enrichEntityWithOutboundRelations(client, entity)

            if (requestedFormat.contains("turtle") || requestedFormat == "ttl") {
                call.respondText(convertEntityToTurtle(rdfEntity), ContentType.parse("text/turtle"))
                return@get
            }

            if (requestedFormat.contains("ld+json") || requestedFormat.contains("json-ld") || requestedFormat == "jsonld") {
                call.respondText(convertEntityToJsonLd(rdfEntity), ContentType.parse("application/ld+json"))
                return@get
            }

            if (requestedFormat.contains("rdf+xml") || requestedFormat == "rdfxml" || requestedFormat == "xml") {
                call.respondText(convertEntityToRdfXml(rdfEntity), ContentType.parse("application/rdf+xml"))
                return@get
            }

            if (requestedFormat.contains("n-triples") || requestedFormat.contains("ntriples") || requestedFormat == "nt") {
                call.respondText(convertEntityToNTriples(rdfEntity), ContentType.parse("application/n-triples"))
                return@get
            }

            call.respond(ResolvedEntityResponse(resolvedId = resolvedEntityId, entity = entity))
        }
    }
}

private fun enrichEntityWithOutboundRelations(
    client: PropertyGraphClient,
    entity: EntityResponse,
): EntityResponse {
    val rows =
        client.runQuery(
            """
            MATCH (s:Version {id: ${'$'}id})-[e]->(t:Version)
            RETURN {
                label: type(e),
                predicateKey: coalesce(e.predicateKey, '')
            } AS e,
            {
                identifier: coalesce(t.identifier, ''),
                id: coalesce(t.id, '')
            } AS t
            """.trimIndent(),
            mapOf("id" to entity.uri),
        )

    if (rows.isEmpty()) return entity

    val props = entity.properties.toMutableMap()
    rows.forEach { row ->
        val edge = row["e"] as? Map<*, *> ?: return@forEach
        val edgeLabel = edge["label"]?.toString() ?: return@forEach
        if (!edgeLabel.startsWith("rel_")) return@forEach
        val target = row["t"] as? Map<*, *> ?: return@forEach
        val targetId =
            (
                target["identifier"]?.toString()?.takeIf { it.isNotBlank() }
                    ?: target["id"]?.toString()?.takeIf { it.isNotBlank() }
            )
                ?: return@forEach
        val key = (edge["predicateKey"]?.toString()?.takeIf { it.isNotBlank() } ?: edgeLabel.removePrefix("rel_")).lowercase()
        val existing = props[key]
        props[key] =
            when (existing) {
                null -> listOf(targetId)
                is List<*> -> existing + targetId
                else -> listOf(existing, targetId)
            }
    }

    return entity.copy(properties = props)
}
