package org.epos.infrastructure.ktor.routes

import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService
import java.io.StringWriter
import java.net.URLDecoder
import java.net.URLEncoder

fun Route.searchRoutes(
    graphDBClient: GraphDBClient,
    shaclService: ShaclService,
) {
    get("/api/v2/search") {
        val publicBaseUrl = call.resolvePublicBaseUrl()
        val params = call.request.queryParameters
        val q = params["q"]
        val type = params["type"]
        val status = params["status"] ?: "published"
        val bbox = params["bbox"]
        val geo = params["geo"]
        val geoRelation = params["geoRelation"]?.lowercase() ?: "intersects"
        val lat = params["lat"]?.toDoubleOrNull()
        val lon = params["lon"]?.toDoubleOrNull()
        val radius = params["radius"]?.toDoubleOrNull()
        val startDate = params["startDate"]
        val endDate = params["endDate"]
        val keywords = params["keywords"]
        val organization = params["organization"]
        val concept = params["concept"]
        val conceptScheme = params["conceptScheme"]
        val dataProvider = params["dataProvider"]
        val page = params["page"]?.toIntOrNull() ?: 1
        val pageSize = (params["pageSize"]?.toIntOrNull() ?: 20).coerceIn(1, 100)
        val includeFacets = params["facets"]?.toBoolean() ?: false
        val facetsType = params["facetstype"]?.lowercase() ?: "none"

        if (page < 1) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Page must be >= 1"))
            return@get
        }

        val graphUri = getGraphUri(status)
        val offset = (page - 1) * pageSize

        val filters = mutableListOf<String>()
        val prefixes =
            """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX dcterms: <http://purl.org/dc/terms/>
            PREFIX dcat: <http://www.w3.org/ns/dcat#>
            PREFIX schema: <http://schema.org/>
            PREFIX foaf: <http://xmlns.com/foaf/0.1/>
            PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
            PREFIX geo: <http://www.opengis.net/ont/geosparql#>
            PREFIX geof: <http://www.opengis.net/def/function/geosparql/>
            PREFIX uom: <http://www.opengis.net/def/uom/OGC/1.0/>
            PREFIX sf: <http://www.opengis.net/ont/sf#>
            PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
            """.trimIndent()

        if (type != null) {
            val resourceTypes = shaclService.getResourceTypes()
            val matchingType = resourceTypes.find { it.name.equals(type, ignoreCase = true) }
            if (matchingType != null) {
                // Keep ?type bound even for typed filtering so response mapping stays consistent.
                filters.add("?s rdf:type ?type . FILTER(?type = <${matchingType.targetClass}>)")
            } else {
                filters.add("?s rdf:type ?type . FILTER(CONTAINS(LCASE(STR(?type)), \"${type.lowercase()}\"))")
            }
        } else {
            filters.add("?s rdf:type ?type .")
        }

        if (!q.isNullOrBlank()) {
            val searchTerms = q.split("\\s+".toRegex()).filter { it.isNotBlank() }
            if (searchTerms.isNotEmpty()) {
                val textFilters =
                    searchTerms.joinToString(" && ") { term ->
                        """(
                        CONTAINS(LCASE(STR(?title)), "${term.lowercase()}") ||
                        CONTAINS(LCASE(STR(?description)), "${term.lowercase()}") ||
                        CONTAINS(LCASE(STR(?label)), "${term.lowercase()}")
                    )"""
                    }
                filters.add(
                    """
                    OPTIONAL { ?s dcterms:title ?title }
                    OPTIONAL { ?s dcterms:description ?description }
                    OPTIONAL { ?s rdfs:label ?label }
                    FILTER($textFilters)
                    """.trimIndent(),
                )
            }
        }

        if (!keywords.isNullOrBlank()) {
            val keywordList = keywords.split(",").map { it.trim().lowercase() }
            val keywordFilters =
                keywordList.joinToString(" || ") { kw ->
                    "CONTAINS(LCASE(STR(?keyword)), \"$kw\")"
                }
            filters.add(
                """
                ?s dcat:keyword ?keyword .
                FILTER($keywordFilters)
                """.trimIndent(),
            )
        }

        if (!organization.isNullOrBlank()) {
            filters.add(
                """
                {
                    ?s dcterms:publisher ?org .
                    { ?org foaf:name ?orgName . FILTER(CONTAINS(LCASE(STR(?orgName)), "${organization.lowercase()}")) }
                    UNION
                    { FILTER(?org = <$organization>) }
                }
                UNION
                {
                    ?s schema:provider ?org .
                    { ?org schema:name ?orgName . FILTER(CONTAINS(LCASE(STR(?orgName)), "${organization.lowercase()}")) }
                    UNION
                    { FILTER(?org = <$organization>) }
                }
                """.trimIndent(),
            )
        }

        if (!startDate.isNullOrBlank() || !endDate.isNullOrBlank()) {
            filters.add(
                "OPTIONAL { ?s dcterms:temporal ?temporal . ?temporal dcat:startDate ?tempStart . ?temporal dcat:endDate ?tempEnd }",
            )
            if (!startDate.isNullOrBlank()) {
                filters.add("FILTER(!BOUND(?tempEnd) || ?tempEnd >= \"$startDate\"^^xsd:dateTime)")
            }
            if (!endDate.isNullOrBlank()) {
                filters.add("FILTER(!BOUND(?tempStart) || ?tempStart <= \"$endDate\"^^xsd:dateTime)")
            }
        }

        if (!concept.isNullOrBlank()) {
            filters.add("?s dcat:theme <$concept> .")
        }

        if (!conceptScheme.isNullOrBlank()) {
            filters.add(
                """
                ?s dcat:theme ?themeConcept .
                ?themeConcept skos:inScheme <$conceptScheme> .
                """.trimIndent(),
            )
        }

        if (!dataProvider.isNullOrBlank()) {
            filters.add(
                """
                {
                    ?s dcterms:publisher <$dataProvider> .
                }
                UNION
                {
                    ?s schema:provider <$dataProvider> .
                }
                """.trimIndent(),
            )
        }

        buildSpatialFilter(bbox, geo, geoRelation, lat, lon, radius)?.let { filters.add(it) }

        val filterClause = filters.joinToString("\n            ")

        val countQuery =
            """
            $prefixes
            SELECT (COUNT(DISTINCT ?s) AS ?count)
            FROM <$graphUri>
            WHERE {
                $filterClause
            }
            """.trimIndent()

        val mainQuery =
            """
            $prefixes
            SELECT DISTINCT ?s ?type
                   (SAMPLE(?t) AS ?title)
                   (SAMPLE(?d) AS ?description)
                   (SAMPLE(?m) AS ?modified)
            FROM <$graphUri>
            WHERE {
                $filterClause
                OPTIONAL { ?s dcterms:title ?t }
                OPTIONAL { ?s dcterms:description ?d }
                OPTIONAL { ?s dcterms:modified ?m }
            }
            GROUP BY ?s ?type
            ORDER BY DESC(?modified) ?s
            LIMIT $pageSize
            OFFSET $offset
            """.trimIndent()

        try {
            val countResult = graphDBClient.query(countQuery)
            val totalCount = countResult.firstOrNull()?.get("count")?.toIntOrNull() ?: 0

            val results = graphDBClient.query(mainQuery)
            val items =
                results.map { row ->
                    val id = row["s"] ?: ""
                    SearchResultItem(
                        id = id,
                        type = extractTypeName(row["type"] ?: ""),
                        title = row["title"],
                        description = row["description"]?.take(500),
                        modified = row["modified"],
                        href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(id, "UTF-8")}",
                    )
                }

            val facets =
                if (includeFacets) {
                    buildFacets(graphDBClient, graphUri)
                } else {
                    null
                }

            if (includeFacets && facetsType != "none") {
                call.respond(
                    buildGroupedSearchResponse(
                        graphDBClient,
                        graphUri,
                        items,
                        facetsType,
                        facets,
                        totalCount,
                        page,
                        pageSize,
                    ),
                )
            } else {
                call.respond(
                    SearchResponse(
                        results = items,
                        total = totalCount,
                        page = page,
                        pageSize = pageSize,
                        totalPages = (totalCount + pageSize - 1) / pageSize,
                        facets = facets,
                    ),
                )
            }
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("InternalServerError", e.message ?: "Search failed"),
            )
        }
    }

    get("/api/v2/details/{id}") {
        val publicBaseUrl = call.resolvePublicBaseUrl()
        val entityUri = call.parameters["id"]
        if (entityUri.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Missing entity ID"))
            return@get
        }

        val decodedUri =
            try {
                URLDecoder.decode(entityUri, "UTF-8")
            } catch (_: Exception) {
                entityUri
            }

        val status = call.request.queryParameters["status"] ?: "published"
        val format = call.request.queryParameters["format"]?.lowercase() ?: "json"
        val extended = call.request.queryParameters["extended"]?.toBoolean() ?: false
        val includeInbound = call.request.queryParameters["includeInbound"]?.toBoolean() ?: false
        val graphUri = getGraphUri(status)

        val existsQuery =
            """
            ASK {
                GRAPH <$graphUri> {
                    <$decodedUri> ?p ?o .
                }
            }
            """.trimIndent()

        if (!graphDBClient.ask(existsQuery)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found: $decodedUri"))
            return@get
        }

        when (format) {
            "turtle", "ttl" -> {
                val model = graphDBClient.getSubjectTriples(decodedUri, graphUri)
                val writer = StringWriter()
                Rio.write(model, writer, RDFFormat.TURTLE)
                call.respondText(writer.toString(), ContentType.parse("text/turtle"))
            }

            "jsonld", "json-ld" -> {
                val model = graphDBClient.getSubjectTriples(decodedUri, graphUri)
                val writer = StringWriter()
                Rio.write(model, writer, RDFFormat.JSONLD)
                call.respondText(writer.toString(), ContentType.parse("application/ld+json"))
            }

            "rdfxml", "xml" -> {
                val model = graphDBClient.getSubjectTriples(decodedUri, graphUri)
                val writer = StringWriter()
                Rio.write(model, writer, RDFFormat.RDFXML)
                call.respondText(writer.toString(), ContentType.parse("application/rdf+xml"))
            }

            else -> {
                if (extended) {
                    call.respond(
                        buildAggregatedDetailsResponse(
                            graphDBClient,
                            shaclService,
                            decodedUri,
                            graphUri,
                            status,
                            publicBaseUrl,
                            includeInbound,
                        ),
                    )
                } else {
                    call.respond(
                        buildEdmDetailsResponse(
                            graphDBClient,
                            shaclService,
                            decodedUri,
                            graphUri,
                            status,
                            publicBaseUrl,
                            includeInbound,
                        ),
                    )
                }
            }
        }
    }
}
