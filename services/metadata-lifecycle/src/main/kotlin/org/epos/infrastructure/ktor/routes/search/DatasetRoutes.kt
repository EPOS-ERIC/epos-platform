package org.epos.infrastructure.ktor.routes

import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.header
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.eclipse.rdf4j.rio.RDFFormat
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import java.net.URLDecoder
import java.net.URLEncoder

fun Route.datasetRoutes(
    graphDBClient: GraphDBClient,
    resourceController: ResourceController,
) {
    get("/api/v2/dataset/{id}") {
        val datasetId = call.parameters["id"]
        if (datasetId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing dataset ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(datasetId, "UTF-8")
            } catch (_: Exception) {
                datasetId
            }

        // Support status parameter for accessing different lifecycle stages
        val status = call.request.queryParameters["status"]?.lowercase() ?: "published"
        val graphUri =
            when (status) {
                "draft" -> Config.GRAPH_DRAFT
                "submitted" -> Config.GRAPH_SUBMITTED
                "published" -> Config.GRAPH_PUBLISHED
                "archived" -> Config.GRAPH_ARCHIVED
                else -> Config.GRAPH_PUBLISHED
            }

        // Check if dataset exists
        if (!entityExists(graphDBClient, decodedId, graphUri)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Dataset not found: $decodedId"))
            return@get
        }

        // Check for RDF format requests (format param or Accept header)
        val formatParam = call.request.queryParameters["format"]?.lowercase()
        val acceptHeader = call.request.header("Accept")?.lowercase() ?: "application/json"
        val requestedFormat = formatParam ?: acceptHeader

        // Handle RDF format requests
        val rdfFormat = detectRdfFormat(requestedFormat)
        if (rdfFormat != null) {
            try {
                val rdf = resourceController.getEntityAsRdf(decodedId, status, rdfFormat)
                if (rdf.isNotEmpty()) {
                    val contentType =
                        when (rdfFormat) {
                            RDFFormat.TURTLE -> ContentType.parse("text/turtle")
                            RDFFormat.JSONLD -> ContentType.parse("application/ld+json")
                            RDFFormat.RDFXML -> ContentType.parse("application/rdf+xml")
                            RDFFormat.NTRIPLES -> ContentType.parse("application/n-triples")
                            else -> ContentType.parse("text/turtle")
                        }
                    call.respondText(rdf, contentType)
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Dataset not found"))
                }
            } catch (e: Exception) {
                call.respond(
                    HttpStatusCode.InternalServerError,
                    ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to serialize dataset"),
                )
            }
            return@get
        }

        // Default: return JSON detail response
        try {
            val dataset =
                buildDatasetDetail(
                    graphDBClient = graphDBClient,
                    entityUri = decodedId,
                    graphUri = graphUri,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(dataset)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch dataset"),
            )
        }
    }
}

private fun detectRdfFormat(format: String): RDFFormat? =
    when {
        format.contains("turtle") || format == "ttl" -> RDFFormat.TURTLE
        format.contains("ld+json") || format.contains("json-ld") || format == "jsonld" -> RDFFormat.JSONLD
        format.contains("rdf+xml") || format == "rdfxml" || format == "xml" -> RDFFormat.RDFXML
        format.contains("n-triples") || format.contains("ntriples") || format == "nt" -> RDFFormat.NTRIPLES
        else -> null
    }

private fun buildDatasetDetail(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): DatasetDetail {
    // Query basic properties
    val basicQuery =
        """
        $DETAIL_PREFIXES
        
        SELECT ?title ?description
        FROM <$graphUri>
        WHERE {
            <$entityUri> a ?type .
            OPTIONAL { <$entityUri> dcterms:title ?title }
            OPTIONAL { <$entityUri> dcterms:description ?description }
        }
        LIMIT 1
        """.trimIndent()

    val basicResult = graphDBClient.query(basicQuery).firstOrNull()
    val title = basicResult?.get("title") ?: extractTypeName(entityUri)
    val description = basicResult?.get("description")

    // Get keywords
    val keywords = extractKeywords(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

    // Get categories
    val categories = extractCategoryLabels(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

    // Get publishers
    val publishers = extractPublishers(graphDBClient, entityUri, graphUri).takeIf { it.isNotEmpty() }

    // Get spatial
    val spatial = extractSpatialWkt(graphDBClient, entityUri, graphUri)

    // Get temporal
    val temporal = extractTemporalExtent(graphDBClient, entityUri, graphUri)

    // Get distributions with hrefs
    val distributions =
        getDatasetDistributions(
            graphDBClient = graphDBClient,
            datasetUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    // Get inbound relations (always included)
    val related =
        buildRelatedResources(
            graphDBClient = graphDBClient,
            entityUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    // Get inbound related resources (always included when present)
    val relatedBy =
        buildRelatedByResources(
            graphDBClient = graphDBClient,
            entityUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    val relations =
        buildImmediateRelationGraph(
            graphDBClient = graphDBClient,
            entityUri = entityUri,
            graphUri = graphUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    return DatasetDetail(
        id = entityUri,
        title = title,
        description = description,
        keywords = keywords,
        categories = categories,
        publisher = publishers,
        spatial = spatial,
        temporal = temporal,
        status = getStatusFromGraph(graphUri),
        distributions = distributions,
        relations = relations,
        related = related,
        relatedBy = relatedBy,
    )
}

private fun getDatasetDistributions(
    graphDBClient: GraphDBClient,
    datasetUri: String,
    graphUri: String,
    publicBaseUrl: String,
): List<ResourceRef> {
    val query =
        """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        
        SELECT DISTINCT ?dist ?title
        FROM <$graphUri>
        WHERE {
            <$datasetUri> dcat:distribution ?dist .
            OPTIONAL { ?dist dcterms:title|rdfs:label ?title }
        }
        """.trimIndent()

    return graphDBClient.query(query).mapNotNull { row ->
        val distUri = row["dist"] ?: return@mapNotNull null
        ResourceRef(
            id = distUri,
            title = row["title"],
            href = "$publicBaseUrl/api/v2/distribution/${URLEncoder.encode(distUri, "UTF-8")}",
        )
    }
}
