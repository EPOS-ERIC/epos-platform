package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import java.net.URLDecoder

fun Route.softwareRoutes(graphDBClient: GraphDBClient) {
    // GET /api/v2/software-application/{id}
    get("/api/v2/software-application/{id}") {
        val appId = call.parameters["id"]
        if (appId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing software application ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(appId, "UTF-8")
            } catch (_: Exception) {
                appId
            }

        val graphUri = Config.GRAPH_PUBLISHED

        // Check if software application exists
        if (!entityExists(graphDBClient, decodedId, graphUri)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Software application not found: $decodedId"))
            return@get
        }

        try {
            val application =
                buildSoftwareApplicationDetail(
                    graphDBClient = graphDBClient,
                    entityUri = decodedId,
                    graphUri = graphUri,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(application)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch software application"),
            )
        }
    }

    // GET /api/v2/software-source-code/{id}
    get("/api/v2/software-source-code/{id}") {
        val codeId = call.parameters["id"]
        if (codeId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing software source code ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(codeId, "UTF-8")
            } catch (_: Exception) {
                codeId
            }

        val graphUri = Config.GRAPH_PUBLISHED

        // Check if software source code exists
        if (!entityExists(graphDBClient, decodedId, graphUri)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Software source code not found: $decodedId"))
            return@get
        }

        try {
            val sourceCode =
                buildSoftwareSourceCodeDetail(
                    graphDBClient = graphDBClient,
                    entityUri = decodedId,
                    graphUri = graphUri,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(sourceCode)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch software source code"),
            )
        }
    }
}

private fun buildSoftwareApplicationDetail(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): SoftwareApplicationDetail {
    val query =
        """
        $DETAIL_PREFIXES
        
        SELECT ?title ?description ?downloadUrl
        FROM <$graphUri>
        WHERE {
            <$entityUri> a ?type .
            OPTIONAL { <$entityUri> dcterms:title|schema:name ?title }
            OPTIONAL { <$entityUri> rdfs:label ?titleAlt }
            OPTIONAL { <$entityUri> dcterms:description|schema:description ?description }
            OPTIONAL { <$entityUri> schema:downloadUrl|schema:installUrl ?downloadUrl }
        }
        LIMIT 1
        """.trimIndent()

    val result = graphDBClient.query(query).firstOrNull()
    val title = result?.get("title") ?: result?.get("titleAlt") ?: extractTypeName(entityUri)
    val description = result?.get("description")
    val downloadUrl = result?.get("downloadUrl")

    // Get keywords
    val keywords = extractKeywords(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

    // Get categories
    val categories = extractCategoryLabels(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

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

    return SoftwareApplicationDetail(
        id = entityUri,
        title = title,
        description = description,
        keywords = keywords,
        categories = categories,
        downloadUrl = downloadUrl,
        related = related,
        relatedBy = relatedBy,
    )
}

private fun buildSoftwareSourceCodeDetail(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): SoftwareSourceCodeDetail {
    val query =
        """
        $DETAIL_PREFIXES
        
        SELECT ?title ?description ?programmingLanguage ?repository
        FROM <$graphUri>
        WHERE {
            <$entityUri> a ?type .
            OPTIONAL { <$entityUri> dcterms:title|schema:name ?title }
            OPTIONAL { <$entityUri> rdfs:label ?titleAlt }
            OPTIONAL { <$entityUri> dcterms:description|schema:description ?description }
            OPTIONAL { <$entityUri> schema:programmingLanguage ?programmingLanguage }
            OPTIONAL { <$entityUri> schema:codeRepository ?repository }
        }
        LIMIT 1
        """.trimIndent()

    val result = graphDBClient.query(query).firstOrNull()
    val title = result?.get("title") ?: result?.get("titleAlt") ?: extractTypeName(entityUri)
    val description = result?.get("description")
    val programmingLanguage = result?.get("programmingLanguage")
    val repository = result?.get("repository")

    // Get keywords
    val keywords = extractKeywords(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

    // Get categories
    val categories = extractCategoryLabels(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

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

    return SoftwareSourceCodeDetail(
        id = entityUri,
        title = title,
        description = description,
        programmingLanguage = programmingLanguage,
        keywords = keywords,
        categories = categories,
        repository = repository,
        related = related,
        relatedBy = relatedBy,
    )
}
