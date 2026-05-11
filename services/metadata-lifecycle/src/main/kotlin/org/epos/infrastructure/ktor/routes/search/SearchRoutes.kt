package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.response.respondRedirect
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService

private const val DATASET_TYPE_URI = "http://www.w3.org/ns/dcat#Dataset"

fun Route.searchRoutes(
    graphDBClient: GraphDBClient,
    shaclService: ShaclService,
) {
    get("/api/v2/search") {
        val params = call.request.queryParameters

        // Parse pagination parameters
        // Preferred: page/resultsPerPage
        // Backward compatibility: limit/offset
        val q = params["q"]
        val resultsPerPage =
            (params["resultsPerPage"]?.toIntOrNull() ?: params["limit"]?.toIntOrNull() ?: 10)
                .coerceIn(1, 100)
        val pageParam = params["page"]?.toIntOrNull()
        val page = (pageParam ?: 1).coerceAtLeast(1)
        val offset =
            if (pageParam != null) {
                (page - 1) * resultsPerPage
            } else {
                (params["offset"]?.toIntOrNull() ?: 0).coerceAtLeast(0)
            }
        val limit = resultsPerPage
        val types = params.getAll("type") ?: emptyList()
        val spatialFilter = params["spatialFilter"]
        val dateFrom = params["dateFrom"]
        val dateTo = params["dateTo"]
        val serviceProviders = params.getAll("serviceProvider") ?: emptyList()
        val versioningStatuses =
            params.getAll("versioningStatus")?.map { it.uppercase() }
                ?: listOf("PUBLISHED")
        val keywords = params.getAll("keywords") ?: emptyList()
        val resourceTypeInfos = shaclService.getResourceTypes()
        val shapeByTargetClass = resourceTypeInfos.associate { it.targetClass to it.name }

        // Determine which graphs to query based on versioning status
        val graphUris =
            versioningStatuses
                .map { status ->
                    getGraphUri(status.lowercase())
                }.distinct()

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
            PREFIX xsd: <http://www.w3.org/2001/XMLSchema#>
            """.trimIndent()

        // Type filter
        if (types.isNotEmpty()) {
            val typeFilters =
                types.mapNotNull { typeName ->
                    // Normalize the input type name for matching
                    val normalizedInput = normalizeTypeName(typeName)

                    val matchingType =
                        resourceTypeInfos.find { typeInfo ->
                            val normalizedShacl = normalizeTypeName(typeInfo.name)
                            normalizedShacl == normalizedInput
                        }
                    if (matchingType != null) {
                        "?type = <${matchingType.targetClass}>"
                    } else {
                        null
                    }
                }
            if (typeFilters.isEmpty()) {
                call.respond(SearchResponse(total = 0, items = emptyList()))
                return@get
            }
            filters.add("?s rdf:type ?type . FILTER(${typeFilters.joinToString(" || ")})")
        } else {
            filters.add("?s rdf:type ?type .")
        }

        // Text search
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

        // Keywords filter (all must match)
        if (keywords.isNotEmpty()) {
            keywords.forEachIndexed { index, kw ->
                filters.add(
                    """
                    ?s dcat:keyword ?kw$index .
                    FILTER(CONTAINS(LCASE(STR(?kw$index)), "${kw.lowercase()}"))
                    """.trimIndent(),
                )
            }
        }

        // Service provider filter
        if (serviceProviders.isNotEmpty()) {
            val providerFilters =
                serviceProviders.joinToString(" || ") { provider ->
                    """(
                    CONTAINS(LCASE(COALESCE(STR(?providerName), STR(?provider))), "${provider.lowercase()}") ||
                    STR(?provider) = "$provider"
                )"""
                }
            filters.add(
                """
                ?s dcterms:publisher|schema:provider ?provider .
                OPTIONAL { ?provider schema:legalName|schema:name|foaf:name|rdfs:label ?providerName }
                FILTER($providerFilters)
                """.trimIndent(),
            )
        }

        // Temporal filter
        if (!dateFrom.isNullOrBlank() || !dateTo.isNullOrBlank()) {
            filters.add(
                "OPTIONAL { ?s dcterms:temporal ?temporal . ?temporal dcat:startDate ?tempStart . ?temporal dcat:endDate ?tempEnd }",
            )
            if (!dateFrom.isNullOrBlank()) {
                filters.add("FILTER(!BOUND(?tempEnd) || ?tempEnd >= \"$dateFrom\"^^xsd:dateTime)")
            }
            if (!dateTo.isNullOrBlank()) {
                filters.add("FILTER(!BOUND(?tempStart) || ?tempStart <= \"$dateTo\"^^xsd:dateTime)")
            }
        }

        // Spatial filter (WKT polygon)
        if (!spatialFilter.isNullOrBlank()) {
            filters.add(buildWktSpatialFilter(spatialFilter))
        }

        val filterClause = filters.joinToString("\n            ")
        val fromClauses = graphUris.joinToString("\n") { "FROM <$it>" }

        // Count query
        val countQuery =
            """
            $prefixes
            SELECT (COUNT(DISTINCT ?s) AS ?count)
            $fromClauses
            WHERE {
                $filterClause
            }
            """.trimIndent()

        // Main query - fetch all needed data in one query
        val mainQuery =
            """
            $prefixes
            SELECT DISTINCT ?s ?type
                   (SAMPLE(?t) AS ?title)
                   (GROUP_CONCAT(DISTINCT ?provDisplay; SEPARATOR="|||") AS ?serviceProvider)
                   (GROUP_CONCAT(DISTINCT COALESCE(STR(?catLabel), STR(?cat)); SEPARATOR="|||") AS ?categories)
            $fromClauses
            WHERE {
                $filterClause
                OPTIONAL { ?s dcterms:title ?t }
                OPTIONAL { ?s rdfs:label ?lbl }
                OPTIONAL {
                    ?s dcterms:publisher|schema:provider ?prov .
                    OPTIONAL { ?prov schema:legalName|schema:name|foaf:name|rdfs:label ?provName }
                    BIND(
                        IF(isLiteral(?prov), STR(?prov), IF(BOUND(?provName), STR(?provName), "")) AS ?provDisplay
                    )
                    FILTER(STRLEN(?provDisplay) > 0)
                }
                OPTIONAL {
                    ?s dcat:theme ?cat .
                    OPTIONAL { ?cat skos:prefLabel|rdfs:label ?catLabel }
                }
            }
            GROUP BY ?s ?type
            ORDER BY ?s
            LIMIT $limit
            OFFSET $offset
            """.trimIndent()

        try {
            val countResult = graphDBClient.query(countQuery)
            val totalCount = countResult.firstOrNull()?.get("count")?.toIntOrNull() ?: 0

            val results = graphDBClient.query(mainQuery)
            val baseUrl = call.resolvePublicBaseUrl()
            val items: List<SearchItem> =
                results
                    .map { row ->
                        val id = row["s"] ?: ""
                        val typeUri = row["type"] ?: ""
                        val apiType =
                            mapSearchTypeUriToApiType(typeUri)
                                ?: shapeByTargetClass[typeUri]
                                ?: extractTypeName(typeUri).lowercase()
                        val name = row["title"] ?: extractTypeName(id)
                        val accessHref = "$baseUrl/api/v2/access/$apiType/${java.net.URLEncoder.encode(id, "UTF-8")}"

                        if (apiType == "dataset") {
                            val serviceProvidersList =
                                row["serviceProvider"]
                                    ?.split("|||")
                                    ?.map { it.trim() }
                                    ?.filter { it.isNotBlank() }
                                    ?.distinct()
                                    ?.takeIf { it.isNotEmpty() }

                            val categoriesList =
                                row["categories"]
                                    ?.split("|||")
                                    ?.filter { it.isNotBlank() }
                                    ?.distinct()
                                    ?.takeIf { it.isNotEmpty() }

                            DatasetSearchItem(
                                id = id,
                                name = name,
                                href = accessHref,
                                serviceProviders = serviceProvidersList,
                                categories = categoriesList,
                            )
                        } else {
                            GenericSearchItem(
                                type = apiType,
                                id = id,
                                name = name,
                                href = accessHref,
                            )
                        }
                    }.filterNotNull()

            call.respond(
                SearchResponse(
                    total = totalCount,
                    items = items,
                ),
            )
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Search failed"),
            )
        }
    }

    get("/api/v2/access/search") {
        val query = call.request.queryParameters.flattenToQueryString()
        val target = if (query.isBlank()) "/api/v2/search" else "/api/v2/search?$query"
        call.respondRedirect(target, permanent = false)
    }

    get("/api/v2/access/search/{shape}") {
        val shape = call.parameters["shape"]
        if (shape.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing shape"))
            return@get
        }

        val pairs =
            call.request.queryParameters
                .entries()
                .flatMap { (key, values) -> values.map { value -> key to value } }
                .toMutableList()
        pairs += "type" to shape
        val query = pairs.toQueryString()
        call.respondRedirect("/api/v2/search?$query", permanent = false)
    }
}

private fun io.ktor.http.Parameters.flattenToQueryString(): String =
    entries().flatMap { (key, values) -> values.map { value -> key to value } }.toQueryString()

private fun List<Pair<String, String>>.toQueryString(): String =
    joinToString("&") { (key, value) ->
        "${java.net.URLEncoder.encode(key, "UTF-8")}=${java.net.URLEncoder.encode(value, "UTF-8")}"
    }

/**
 * Type aliases for user-friendly API type names.
 * Maps API names to SHACL type names.
 */
private val TYPE_ALIASES =
    mapOf(
        "category" to "concept",
        "categories" to "concept",
    )

/**
 * Normalizes a type name for comparison by:
 * - Applying type aliases
 * - Converting to lowercase
 * - Removing hyphens and underscores
 */
private fun normalizeTypeName(typeName: String): String {
    val lower = typeName.lowercase()
    val aliased = TYPE_ALIASES[lower] ?: lower
    return aliased.replace("-", "").replace("_", "")
}

/**
 * Maps RDF type URI to API type strictly for search results.
 * Returns null for unsupported types to avoid misclassifying them as datasets.
 */
private fun mapSearchTypeUriToApiType(typeUri: String): String? =
    when {
        typeUri.equals(DATASET_TYPE_URI, ignoreCase = true) ||
            typeUri.contains("/Dataset", ignoreCase = true) ||
            typeUri.endsWith("#Dataset", ignoreCase = true) -> "dataset"

        typeUri.contains("Distribution", ignoreCase = true) -> "distribution"

        typeUri.contains("SoftwareApplication", ignoreCase = true) -> "software-application"

        typeUri.contains("SoftwareSourceCode", ignoreCase = true) -> "software-source-code"

        typeUri.contains("Concept", ignoreCase = true) -> "category"

        else -> null
    }

/**
 * Builds a spatial filter for WKT polygon intersection.
 */
private fun buildWktSpatialFilter(wkt: String): String {
    val geometryExtraction =
        """
        {
            ?s dcterms:spatial ?spatialEntity .
            ?spatialEntity geo:asWKT ?entityGeom .
        } UNION {
            ?s dcterms:spatial ?spatialEntity .
            ?spatialEntity <http://www.w3.org/ns/locn#geometry> ?geomNode .
            ?geomNode geo:asWKT ?entityGeom .
        } UNION {
            ?s geo:hasGeometry ?geomNode .
            ?geomNode geo:asWKT ?entityGeom .
        } UNION {
            ?s geo:asWKT ?entityGeom .
        } UNION {
            ?s dcat:bbox ?entityGeom .
            FILTER(DATATYPE(?entityGeom) = geo:wktLiteral)
        }
        """.trimIndent()

    return """
        $geometryExtraction
        FILTER(geof:sfIntersects(?entityGeom, "$wkt"^^geo:wktLiteral))
        """.trimIndent()
}
