package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import java.net.URLDecoder

fun Route.distributionRoutes(graphDBClient: GraphDBClient) {
    get("/api/v2/distribution/{id}") {
        val distributionId = call.parameters["id"]
        if (distributionId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing distribution ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(distributionId, "UTF-8")
            } catch (_: Exception) {
                distributionId
            }

        val graphUri = Config.GRAPH_PUBLISHED

        // Check if distribution exists
        if (!entityExists(graphDBClient, decodedId, graphUri)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Distribution not found: $decodedId"))
            return@get
        }

        try {
            val distribution =
                buildDistributionDetail(
                    graphDBClient = graphDBClient,
                    entityUri = decodedId,
                    graphUri = graphUri,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(distribution)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch distribution"),
            )
        }
    }
}

private fun buildDistributionDetail(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): DistributionDetail {
    val query =
        """
        $DETAIL_PREFIXES
        
        SELECT ?title ?description ?accessUrl ?downloadUrl ?operation
        FROM <$graphUri>
        WHERE {
            <$entityUri> a ?type .
            OPTIONAL { <$entityUri> dcterms:title ?title }
            OPTIONAL { <$entityUri> rdfs:label ?titleAlt }
            OPTIONAL { <$entityUri> dcterms:description ?description }
            OPTIONAL { <$entityUri> dcat:accessURL ?accessUrl }
            OPTIONAL { <$entityUri> dcat:downloadURL ?downloadUrl }
            OPTIONAL { 
                <$entityUri> dcat:accessService ?service .
                ?service epos:uid ?operation .
            }
        }
        LIMIT 1
        """.trimIndent()

    val result = graphDBClient.query(query).firstOrNull()
    val title = result?.get("title") ?: result?.get("titleAlt") ?: extractTypeName(entityUri)
    val description = result?.get("description")
    val accessUrl = result?.get("accessUrl") ?: result?.get("downloadUrl")
    val operation = result?.get("operation")

    // Get outbound and inbound relations (always included when present)
    val related =
        buildRelatedResources(
            graphDBClient = graphDBClient,
            entityUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    val relatedBy =
        buildRelatedByResources(
            graphDBClient = graphDBClient,
            entityUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    return DistributionDetail(
        id = entityUri,
        title = title,
        description = description,
        accessUrl = accessUrl,
        operation = operation,
        related = related,
        relatedBy = relatedBy,
    )
}
