package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.application.ApplicationCall
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.EdmMappingService
import java.net.URLDecoder
import java.net.URLEncoder

fun Route.legacyRoutes(
    graphDBClient: GraphDBClient,
    edmMappingService: EdmMappingService
) {
    get("/resources/search") {
        handleLegacySearch(call, graphDBClient)
    }

    get("/resources/details/{instance_id}") {
        handleLegacyDetails(call, graphDBClient, edmMappingService)
    }

    get("/software/search") {
        handleLegacySearch(
            call,
            graphDBClient,
            forcedTypeTerms = listOf("softwareapplication", "softwaresourcecode", "software")
        )
    }

    get("/software/details/{instance_id}") {
        handleLegacyDetails(call, graphDBClient, edmMappingService)
    }

    get("/facilities/search") {
        handleLegacySearch(call, graphDBClient, forcedTypeTerms = listOf("facility"))
    }

    get("/facilities/details/{instance_id}") {
        handleLegacyDetails(call, graphDBClient, edmMappingService)
    }

    get("/equipments/{instance_id}") {
        handleLegacyDetails(call, graphDBClient, edmMappingService)
    }

    get("/resources/organizations") {
        val params = call.request.queryParameters
        val status = (params["versioningStatus"] ?: params["status"] ?: "published").lowercase()
        val graphUri = getGraphUri(status)
        val q = params["q"]?.trim()
        val country = params["country"]?.trim()
        val providerType = params["type"]?.trim()?.lowercase()
        val page = params["page"]?.toIntOrNull() ?: 1
        val pageSize = (params["pageSize"]?.toIntOrNull() ?: 20).coerceIn(1, 100)

        if (page < 1) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Page must be >= 1"))
            return@get
        }
        val offset = (page - 1) * pageSize

        val filters = mutableListOf<String>()
        filters.add("?s rdf:type ?type . FILTER(CONTAINS(LCASE(STR(?type)), \"organization\"))")

        if (!q.isNullOrBlank()) {
            filters.add(
                """
                OPTIONAL { ?s schema:name ?orgName }
                OPTIONAL { ?s foaf:name ?foafName }
                FILTER(
                    CONTAINS(LCASE(STR(?orgName)), "${q.lowercase()}") ||
                    CONTAINS(LCASE(STR(?foafName)), "${q.lowercase()}") ||
                    CONTAINS(LCASE(STR(?s)), "${q.lowercase()}")
                )
                """.trimIndent()
            )
        }

        if (!country.isNullOrBlank()) {
            filters.add(
                """
                OPTIONAL { ?s schema:address/schema:addressCountry ?countryName }
                FILTER(CONTAINS(LCASE(STR(?countryName)), "${country.lowercase()}"))
                """.trimIndent()
            )
        }

        when (providerType) {
            "dataproviders" -> filters.add("?r dcterms:publisher|schema:provider ?s .")
            "serviceproviders" -> filters.add("?r dcat:distribution/dcat:accessService/schema:provider ?s .")
            "facilitiesproviders" -> {
                filters.add("?r rdf:type ?rType . FILTER(CONTAINS(LCASE(STR(?rType)), \"facility\"))")
                filters.add("?r dcterms:publisher|schema:provider ?s .")
            }
        }

        val prefixes = legacyPrefixes()
        val filterClause = filters.joinToString("\n")
        val countQuery = """
            $prefixes
            SELECT (COUNT(DISTINCT ?s) AS ?count)
            FROM <$graphUri>
            WHERE {
                $filterClause
            }
        """.trimIndent()

        val query = """
            $prefixes
            SELECT DISTINCT ?s
                   (SAMPLE(?n) AS ?name)
                   (SAMPLE(?logo) AS ?logo)
                   (SAMPLE(?url) AS ?url)
                   (SAMPLE(?countryName) AS ?country)
            FROM <$graphUri>
            WHERE {
                $filterClause
                OPTIONAL { ?s schema:name|schema:legalName|foaf:name ?n }
                OPTIONAL { ?s foaf:logo|schema:logo ?logo }
                OPTIONAL { ?s foaf:homepage|schema:url ?url }
                OPTIONAL { ?s schema:address/schema:addressCountry ?countryName }
            }
            GROUP BY ?s
            ORDER BY ?s
            LIMIT $pageSize
            OFFSET $offset
        """.trimIndent()

        try {
            val totalCount = graphDBClient.query(countQuery).firstOrNull()?.get("count")?.toIntOrNull() ?: 0
            val rows = graphDBClient.query(query)
            val organizationsResponse = rows.map { row ->
                LegacyOrganizationBean(
                    id = row["s"],
                    logo = row["logo"],
                    url = row["url"],
                    name = row["name"],
                    country = row["country"]
                )
            }
            call.response.headers.append("X-Total-Count", totalCount.toString())
            call.response.headers.append("X-Page", page.toString())
            call.response.headers.append("X-Page-Size", pageSize.toString())
            call.respond(organizationsResponse)
        } catch (e: Exception) {
            call.respond(HttpStatusCode.InternalServerError, LegacySearchResponse(errorMessage = e.message ?: "Organization search failed"))
        }
    }

    get("/resources/monitoring") {
        val query = """
            PREFIX dcat: <http://www.w3.org/ns/dcat#>
            PREFIX dcterms: <http://purl.org/dc/terms/>
            PREFIX epos: <https://www.epos-eu.org/epos-dcat-ap#>

            SELECT DISTINCT ?dist ?uid ?metaId ?title ?downloadURL ?accessURL
            FROM <${Config.GRAPH_PUBLISHED}>
            WHERE {
                ?dist a dcat:Distribution .
                OPTIONAL { ?dist epos:uid ?uid }
                OPTIONAL { ?dist epos:metaId ?metaId }
                OPTIONAL { ?dist dcterms:title ?title }
                OPTIONAL { ?dist dcat:downloadURL ?downloadURL }
                OPTIONAL { ?dist dcat:accessURL ?accessURL }
            }
            LIMIT 500
        """.trimIndent()

        val rows = graphDBClient.query(query)
        val monitoring = rows.map { row ->
            val url = row["downloadURL"] ?: row["accessURL"]
            LegacyMonitoringBean(
                identifier = row["metaId"],
                name = row["title"],
                originalURL = url,
                validationRules = listOf(LegacyValidationRule(type = "none")),
                contactPoints = emptyList(),
                TCSGroup = "Undefined",
                id = row["metaId"],
                uid = row["uid"]
            )
        }.filter { !it.originalURL.isNullOrBlank() }

        call.respond(monitoring)
    }

    get("/api/v2/monitoring") {
        val summary = mapOf(
            "timestamp" to System.currentTimeMillis(),
            "graphdbEndpoint" to Config.GRAPHDB_ENDPOINT,
            "repositoryId" to Config.REPOSITORY_ID,
            "publishedTriples" to graphDBClient.query(
                "SELECT (COUNT(?s) AS ?count) FROM <${Config.GRAPH_PUBLISHED}> WHERE { ?s ?p ?o }"
            ).firstOrNull()?.get("count")?.toLongOrNull().orZero(),
            "draftTriples" to graphDBClient.query(
                "SELECT (COUNT(?s) AS ?count) FROM <${Config.GRAPH_DRAFT}> WHERE { ?s ?p ?o }"
            ).firstOrNull()?.get("count")?.toLongOrNull().orZero(),
            "submittedTriples" to graphDBClient.query(
                "SELECT (COUNT(?s) AS ?count) FROM <${Config.GRAPH_SUBMITTED}> WHERE { ?s ?p ?o }"
            ).firstOrNull()?.get("count")?.toLongOrNull().orZero(),
            "archivedTriples" to graphDBClient.query(
                "SELECT (COUNT(?s) AS ?count) FROM <${Config.GRAPH_ARCHIVED}> WHERE { ?s ?p ?o }"
            ).firstOrNull()?.get("count")?.toLongOrNull().orZero()
        )
        call.respond(summary)
    }
}

private suspend fun handleLegacySearch(
    call: ApplicationCall,
    graphDBClient: GraphDBClient,
    forcedTypeTerms: List<String> = emptyList()
) {
    val params = call.request.queryParameters
    val q = params["q"]
    val status = (params["versioningStatus"] ?: params["status"] ?: "published").lowercase()
    val bbox = params["bbox"]
    val startDate = params["startDate"]
    val endDate = params["endDate"]
    val keywords = params["keywords"]
    val organizations = params["organisations"] ?: params["organizations"] ?: params["organization"]
    val scienceDomains = params["sciencedomains"]
    val serviceTypes = params["servicetypes"]
    val includeFacets = params["facets"]?.toBoolean() ?: false
    val requestedFacetType = params["facetstype"]?.lowercase() ?: "none"
    val facetsType = when (requestedFacetType) {
        "facilityprovider", "facilityproviders" -> "dataproviders"
        else -> requestedFacetType
    }
    val page = params["page"]?.toIntOrNull() ?: 1
    val pageSize = params["pageSize"]?.toIntOrNull()

    if (page < 1) {
        call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Page must be >= 1"))
        return
    }
    if (pageSize != null && pageSize < 1) {
        call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "pageSize must be >= 1"))
        return
    }

    val graphUri = getGraphUri(status)
    val publicBaseUrl = call.resolvePublicBaseUrl()
    val offset = if (pageSize != null) (page - 1) * pageSize else 0
    val pagingClause = if (pageSize != null) "LIMIT $pageSize OFFSET $offset" else ""
    val filters = mutableListOf<String>()
    filters.add("?s rdf:type ?type .")

    if (forcedTypeTerms.isNotEmpty()) {
        filters.add(
            "FILTER(" + forcedTypeTerms.joinToString(" || ") {
                "CONTAINS(LCASE(STR(?type)), \"${it.lowercase()}\")"
            } + ")"
        )
    }

    if (!q.isNullOrBlank()) {
        val terms = q.split("\\s+".toRegex()).filter { it.isNotBlank() }
        if (terms.isNotEmpty()) {
            val textFilters = terms.joinToString(" && ") { term ->
                """(
                    CONTAINS(LCASE(STR(?title)), "${term.lowercase()}") ||
                    CONTAINS(LCASE(STR(?description)), "${term.lowercase()}") ||
                    CONTAINS(LCASE(STR(?label)), "${term.lowercase()}")
                )"""
            }
            filters.add(
                """
                OPTIONAL { ?s dcterms:title|schema:name|rdfs:label ?title }
                OPTIONAL { ?s dcterms:description|schema:description ?description }
                OPTIONAL { ?s rdfs:label ?label }
                FILTER($textFilters)
                """.trimIndent()
            )
        }
    }

    if (!keywords.isNullOrBlank()) {
        val keywordList = keywords.split(",").map { it.trim().lowercase() }.filter { it.isNotBlank() }
        if (keywordList.isNotEmpty()) {
            val keywordFilters = keywordList.joinToString(" || ") { kw ->
                "CONTAINS(LCASE(STR(?keyword)), \"$kw\")"
            }
            filters.add(
                """
                ?s dcat:keyword|schema:keywords ?keyword .
                FILTER($keywordFilters)
                """.trimIndent()
            )
        }
    }

    if (!organizations.isNullOrBlank()) {
        val term = organizations.lowercase()
        filters.add(
            """
            {
                ?s dcterms:publisher|schema:provider ?org .
                OPTIONAL { ?org schema:name|foaf:name ?orgName }
                FILTER(CONTAINS(LCASE(STR(?orgName)), "$term") || CONTAINS(LCASE(STR(?org)), "$term"))
            }
            """.trimIndent()
        )
    }

    if (!scienceDomains.isNullOrBlank()) {
        val domains = scienceDomains.split(",").map { it.trim().lowercase() }.filter { it.isNotBlank() }
        if (domains.isNotEmpty()) {
            filters.add(
                """
                ?s dcat:theme ?themeConcept .
                OPTIONAL { ?themeConcept skos:prefLabel|rdfs:label ?themeLabel }
                FILTER(${domains.joinToString(" || ") { d ->
                    "CONTAINS(LCASE(STR(?themeConcept)), \"$d\") || CONTAINS(LCASE(STR(?themeLabel)), \"$d\")"
                }})
                """.trimIndent()
            )
        }
    }

    if (!serviceTypes.isNullOrBlank()) {
        val serviceTypeTerms = serviceTypes.split(",").map { it.trim().lowercase() }.filter { it.isNotBlank() }
        if (serviceTypeTerms.isNotEmpty()) {
            filters.add(
                """
                OPTIONAL { ?s schema:serviceType|dcterms:type ?serviceType }
                FILTER(${serviceTypeTerms.joinToString(" || ") { s ->
                    "CONTAINS(LCASE(STR(?serviceType)), \"$s\") || CONTAINS(LCASE(STR(?type)), \"$s\")"
                }})
                """.trimIndent()
            )
        }
    }

    if (!startDate.isNullOrBlank() || !endDate.isNullOrBlank()) {
        filters.add("OPTIONAL { ?s dcterms:temporal ?temporal . ?temporal dcat:startDate|schema:startDate ?tempStart . ?temporal dcat:endDate|schema:endDate ?tempEnd }")
        if (!startDate.isNullOrBlank()) {
            filters.add("FILTER(!BOUND(?tempEnd) || ?tempEnd >= \"$startDate\"^^xsd:dateTime)")
        }
        if (!endDate.isNullOrBlank()) {
            filters.add("FILTER(!BOUND(?tempStart) || ?tempStart <= \"$endDate\"^^xsd:dateTime)")
        }
    }

    buildSpatialFilter(bbox = bbox, geo = null, geoRelation = "intersects", lat = null, lon = null, radius = null)
        ?.let { filters.add(it) }

    val prefixes = legacyPrefixes()
    val filterClause = filters.joinToString("\n")

    val countQuery = """
        $prefixes
        SELECT (COUNT(DISTINCT ?s) AS ?count)
        FROM <$graphUri>
        WHERE {
            $filterClause
        }
    """.trimIndent()

    val mainQuery = """
        $prefixes
        SELECT DISTINCT ?s ?type
               (SAMPLE(?t) AS ?title)
               (SAMPLE(?d) AS ?description)
               (SAMPLE(?m) AS ?modified)
        FROM <$graphUri>
        WHERE {
            $filterClause
            OPTIONAL { ?s dcterms:title|schema:name|rdfs:label ?t }
            OPTIONAL { ?s dcterms:description|schema:description ?d }
            OPTIONAL { ?s dcterms:modified|schema:dateModified ?m }
        }
        GROUP BY ?s ?type
        ORDER BY DESC(?modified) ?s
        $pagingClause
    """.trimIndent()

    try {
            val totalCount = graphDBClient.query(countQuery).firstOrNull()?.get("count")?.toIntOrNull() ?: 0
            val results = graphDBClient.query(mainQuery)
        val formatsByEntity = buildLegacyAvailableFormatsForEntities(
            graphDBClient = graphDBClient,
            graphUri = graphUri,
            entityUris = results.mapNotNull { it["s"] },
            publicBaseUrl = publicBaseUrl
        )
        val items = results.map { row ->
            val id = row["s"] ?: ""
            SearchResultItem(
                id = id,
                type = extractTypeName(row["type"] ?: ""),
                title = row["title"],
                description = row["description"]?.take(500),
                modified = row["modified"],
                href = "$publicBaseUrl/resources/details/${URLEncoder.encode(id, "UTF-8")}",
                availableFormats = null
            )
        }

        val facets = if (includeFacets) buildFacets(graphDBClient, graphUri) else null
        val grouped = if (includeFacets && facetsType != "none") {
            buildGroupedSearchResponse(
                graphDBClient = graphDBClient,
                graphUri = graphUri,
                items = items,
                facetsType = facetsType,
                facets = facets,
                totalCount = totalCount,
                page = page,
                pageSize = pageSize ?: totalCount.coerceAtLeast(1)
            )
        } else {
            null
        }

        call.respond(toLegacySearchResponse(items, grouped, status, formatsByEntity))
    } catch (e: Exception) {
        call.respond(HttpStatusCode.InternalServerError, LegacySearchResponse(errorMessage = e.message ?: "Search failed"))
    }
}

private suspend fun handleLegacyDetails(
    call: ApplicationCall,
    graphDBClient: GraphDBClient,
    edmMappingService: EdmMappingService
) {
    val id = call.parameters["instance_id"]
    if (id.isNullOrBlank()) {
        call.respond(HttpStatusCode.BadRequest, LegacyDistribution(errorMessage = "Missing instance_id"))
        return
    }

    val decodedUri = try {
        URLDecoder.decode(id, "UTF-8")
    } catch (_: Exception) {
        id
    }

    val status = (call.request.queryParameters["versioningStatus"] ?: call.request.queryParameters["status"] ?: "published").lowercase()
    val publicBaseUrl = call.resolvePublicBaseUrl()
    val graphUri = getGraphUri(status)
    val extended = call.request.queryParameters["extended"]?.toBoolean() ?: false

    val existsQuery = """
        ASK {
            GRAPH <$graphUri> {
                <$decodedUri> ?p ?o .
            }
        }
    """.trimIndent()

    if (!graphDBClient.ask(existsQuery)) {
        call.respond(HttpStatusCode.NotFound, LegacyDistribution(errorMessage = "Entity not found: $decodedUri"))
        return
    }

    val aggregated = buildAggregatedDetailsResponse(
        graphDBClient = graphDBClient,
        edmMappingService = edmMappingService,
        entityUri = decodedUri,
        graphUri = graphUri,
        status = status,
        publicBaseUrl = publicBaseUrl
    )

    val legacyFormats = buildLegacyAvailableFormatsForEntities(
        graphDBClient = graphDBClient,
        graphUri = graphUri,
        entityUris = listOf(decodedUri),
        publicBaseUrl = publicBaseUrl
    )[decodedUri]

    if (extended) {
        call.respond(toLegacyDistributionExtended(aggregated, legacyFormats, publicBaseUrl))
    } else {
        call.respond(toLegacyDistribution(aggregated, legacyFormats, publicBaseUrl))
    }
}

private fun legacyPrefixes(): String = """
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

private fun Long?.orZero(): Long = this ?: 0L

private fun toLegacySearchResponse(
    items: List<SearchResultItem>,
    grouped: GroupedSearchResponse?,
    status: String,
    formatsByEntity: Map<String, List<LegacyAvailableFormat>>
): LegacySearchResponse {
    if (grouped != null) {
        return LegacySearchResponse(
            results = grouped.results.toLegacyNode(status, formatsByEntity),
            filters = grouped.filters.map { it.toLegacyFilter() }
        )
    }

    return LegacySearchResponse(
        results = LegacyNode(
            name = "results",
            distributions = items.map { it.toLegacyDiscoveryItem(status, formatsByEntity[it.id]) }
        ),
        filters = emptyList()
    )
}

private fun SearchResultItem.toLegacyDiscoveryItem(status: String, legacyFormats: List<LegacyAvailableFormat>?): LegacyDiscoveryItem = LegacyDiscoveryItem(
    href = href,
    hrefExtended = "$href?extended=true",
    id = id,
    uid = null,
    metaId = null,
    title = title,
    description = description,
    versioningStatus = status.uppercase(),
    availableFormats = legacyFormats
)

private fun FacetNode.toLegacyNode(status: String, formatsByEntity: Map<String, List<LegacyAvailableFormat>>): LegacyNode = LegacyNode(
    children = children?.map { it.toLegacyNode(status, formatsByEntity) },
    distributions = distributions?.map { it.toLegacyDiscoveryItem(status, formatsByEntity[it.id]) },
    ddss = ddss,
    id = id,
    code = code,
    linkUrl = linkUrl,
    imgUrl = imgUrl,
    color = color,
    name = name
)

private fun FacetNode.toLegacyNode(): LegacyNode = toLegacyNode("published", emptyMap())

private fun FilterNode.toLegacyFilter(): LegacyNodeFilter = LegacyNodeFilter(
    children = children?.map { it.toLegacyFilter() },
    id = id,
    name = name
)

private fun DataServiceProvider?.toLegacyProvider(): LegacyDataServiceProvider? {
    if (this == null) return null
    return LegacyDataServiceProvider(
        dataProviderLegalName = dataProviderLegalName,
        dataProviderUrl = dataProviderUrl,
        country = country,
        uid = uid,
        metaid = metaid,
        instanceid = instanceid,
        relatedDataProvider = relatedDataProvider?.mapNotNull { it.toLegacyProvider() },
        relatedDataServiceProvider = relatedDataServiceProvider?.mapNotNull { it.toLegacyProvider() }
    )
}

private fun toLegacyDistribution(
    source: AggregatedEntityResponse,
    legacyFormats: List<LegacyAvailableFormat>?,
    publicBaseUrl: String
): LegacyDistribution {
    val (doi, internalIds) = splitIdentifiers(source.identifiers)
    val accessRights = source.relatedDataProducts
        ?.firstOrNull()
        ?.accessRights

    return LegacyDistribution(
    operationid = source.relatedWebservices?.firstOrNull()?.operations?.firstOrNull()?.uid,
    href = source.href,
    hrefExtended = source.hrefExtended,
    categories = source.categories?.toLegacyNode(),
    id = source.id,
    uid = source.uid,
    metaId = source.metaId,
    type = normalizeLegacyType(source),
    title = source.title,
    description = source.description,
    license = source.license,
    downloadURL = source.downloadURL,
    keywords = source.keywords,
    dataProvider = source.dataProvider?.mapNotNull { it.toLegacyProvider() },
    frequencyUpdate = source.frequencyUpdate,
    internalID = internalIds,
    DOI = doi,
    spatial = source.spatial,
    temporalCoverage = source.temporalCoverage,
    scienceDomain = source.scienceDomain,
    qualityAssurance = source.qualityAssurance,
    accessRights = accessRights,
    availableFormats = legacyFormats,
    availableContactPoints = buildLegacyAvailableContactPoints(source, publicBaseUrl),
    versioningStatus = source.status.uppercase(),
    editorId = source.editorId,
    serviceName = source.relatedWebservices?.firstOrNull()?.name,
    serviceDescription = source.relatedWebservices?.firstOrNull()?.description,
    serviceProvider = source.relatedWebservices?.firstOrNull()?.provider.toLegacyProvider(),
    serviceSpatial = source.relatedWebservices?.firstOrNull()?.spatial,
    serviceTemporalCoverage = source.relatedWebservices?.firstOrNull()?.temporalCoverage?.firstOrNull() ?: TemporalCoverage(),
    serviceEndpoint = source.relatedWebservices?.firstOrNull()?.operations?.firstOrNull()?.endpoint?.substringBefore("{"),
    serviceDocumentation = extractDocumentationUri(source.relatedWebservices?.firstOrNull()?.documentation),
    serviceType = source.relatedWebservices?.firstOrNull()?.type,
    endpoint = source.relatedWebservices?.firstOrNull()?.operations?.firstOrNull()?.endpoint,
    serviceParameters = source.relatedWebservices
        ?.firstOrNull()
        ?.operations
        ?.firstOrNull()
        ?.parameters
        ?.map { it.toLegacyServiceParameter() }
)
}

private fun toLegacyDistributionExtended(
    source: AggregatedEntityResponse,
    legacyFormats: List<LegacyAvailableFormat>?,
    publicBaseUrl: String
): LegacyDistributionExtended =
    LegacyDistributionExtended(
        href = source.href,
        hrefExtended = source.hrefExtended,
        categories = source.categories?.toLegacyNode(),
        id = source.id,
        uid = source.uid,
        metaId = source.metaId,
        type = normalizeLegacyType(source),
        title = source.title,
        description = source.description,
        license = source.license,
        downloadURL = source.downloadURL,
        keywords = source.keywords,
        frequencyUpdate = source.frequencyUpdate,
        scienceDomain = source.scienceDomain,
        hasQualityAnnotation = source.qualityAssurance,
        availableFormats = legacyFormats,
        availableContactPoints = buildLegacyAvailableContactPoints(source, publicBaseUrl),
        relatedDataProducts = source.relatedDataProducts?.map { dp ->
            LegacyDataProduct(
                href = dp.href,
                id = dp.id,
                uid = dp.uid,
                type = dp.type,
                title = dp.title,
                description = dp.description,
                license = dp.license,
                downloadURL = dp.downloadURL,
                keywords = dp.keywords,
                dataProvider = dp.dataProvider?.mapNotNull { it.toLegacyProvider() },
                frequencyUpdate = dp.frequencyUpdate,
                identifiers = source.identifiers?.map { mapOf("type" to it.type, "value" to it.value) },
                scienceDomain = dp.scienceDomain,
                hasQualityAnnotation = dp.qualityAssurance,
                accessRights = dp.accessRights,
                provenance = dp.provenance,
                contactPoints = dp.contactPoints?.map { cp -> cp.toLegacyContactPointMap() },
                spatial = dp.spatial,
                temporalCoverage = dp.temporalCoverage,
                categories = dp.categories?.toLegacyNode()
            )
        },
        relatedWebservice = source.relatedWebservices?.map { ws ->
            LegacyWebservice(
                href = ws.href,
                id = ws.id,
                metaid = ws.metaid,
                uid = ws.uid,
                name = ws.name,
                description = ws.description,
                provider = ws.provider.toLegacyProvider(),
                spatial = ws.spatial,
                documentation = extractDocumentationUri(ws.documentation),
                temporalCoverage = ws.temporalCoverage,
                type = ws.type,
                contactPoints = ws.contactPoints?.map { cp -> cp.toLegacyContactPointMap() },
                operations = ws.operations,
                categories = ws.categories
            )
        },
        versioningStatus = source.status.uppercase(),
        editorId = source.editorId
    )

private fun splitIdentifiers(identifiers: List<IdentifierInfo>?): Pair<List<String>?, List<String>?> {
    if (identifiers.isNullOrEmpty()) return null to null

    val doi = identifiers
        .filter { it.type.equals("DOI", ignoreCase = true) || it.type.contains("doi", ignoreCase = true) }
        .map { it.value }
        .distinct()

    val internalIds = identifiers
        .filter { it.type.equals("DDSS-ID", ignoreCase = true) || it.type.contains("ddss", ignoreCase = true) }
        .map { it.value }
        .distinct()

    return doi.ifEmpty { null } to internalIds.ifEmpty { null }
}

private fun buildLegacyAvailableContactPoints(
    source: AggregatedEntityResponse,
    publicBaseUrl: String
): List<LegacyAvailableContactPoint>? {
    val hasServiceProviderContacts = source.relatedWebservices
        ?.any { !it.contactPoints.isNullOrEmpty() }
        ?: false

    val hasDataProviderContacts = (!source.contactPoints.isNullOrEmpty()) ||
        (source.relatedDataProducts?.any { !it.contactPoints.isNullOrEmpty() } ?: false)

    val contacts = mutableListOf<LegacyAvailableContactPoint>()
    val idParam = source.id

    if (hasServiceProviderContacts) {
        contacts.add(
            LegacyAvailableContactPoint(
                href = "$publicBaseUrl/sender/send-email?id=$idParam&contactType=SERVICEPROVIDERS",
                type = "SERVICEPROVIDERS"
            )
        )
    }

    if (hasDataProviderContacts) {
        contacts.add(
            LegacyAvailableContactPoint(
                href = "$publicBaseUrl/sender/send-email?id=$idParam&contactType=DATAPROVIDERS",
                type = "DATAPROVIDERS"
            )
        )
    }

    if (hasServiceProviderContacts && hasDataProviderContacts) {
        contacts.add(
            LegacyAvailableContactPoint(
                href = "$publicBaseUrl/sender/send-email?id=$idParam&contactType=ALL",
                type = "ALL"
            )
        )
    }

    return contacts.ifEmpty { null }
}

private fun buildLegacyAvailableFormatsForEntities(
    graphDBClient: GraphDBClient,
    graphUri: String,
    entityUris: List<String>,
    publicBaseUrl: String
): Map<String, List<LegacyAvailableFormat>> {
    val uniqueUris = entityUris.filter { it.isNotBlank() }.distinct()
    if (uniqueUris.isEmpty()) return emptyMap()

    val valuesClause = uniqueUris.joinToString(" ") { "<$it>" }
    val query = """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX hydra: <http://www.w3.org/ns/hydra/core#>
        PREFIX schema: <http://schema.org/>

        SELECT ?entity ?dist
               (GROUP_CONCAT(DISTINCT STR(?downloadURL);SEPARATOR=",") AS ?downloadURLs)
               (SAMPLE(STR(?distFormat)) AS ?originalFormat)
               (COUNT(DISTINCT ?accessService) AS ?accessServiceCount)
               (GROUP_CONCAT(DISTINCT STR(?returns);SEPARATOR=",") AS ?returnsValues)
        FROM <$graphUri>
        WHERE {
            VALUES ?entity { $valuesClause }
            OPTIONAL {
                {
                    ?entity dcat:distribution ?dist .
                } UNION {
                    ?entity a dcat:Distribution .
                    BIND(?entity AS ?dist)
                }
                OPTIONAL { ?dist dcat:downloadURL ?downloadURL }
                OPTIONAL { ?dist dcterms:format ?distFormat }
                OPTIONAL {
                    {
                        ?dist dcat:accessService ?accessService .
                    } UNION {
                        ?dist dcat:accessService ?wsFromDist .
                        BIND(?wsFromDist AS ?accessService)
                    }
                    OPTIONAL {
                        ?accessService hydra:supportedOperation ?op .
                        OPTIONAL { ?op hydra:returns ?returns }
                    }
                }
            }
        }
        GROUP BY ?entity ?dist
    """.trimIndent()

    val rows = graphDBClient.query(query)
    val byEntity = mutableMapOf<String, MutableList<LegacyAvailableFormat>>()

    rows.forEach { row ->
        val entity = row["entity"] ?: return@forEach
        val downloadURLs = row["downloadURLs"]
        val originalFormat = row["originalFormat"]
        val hasAccessService = (row["accessServiceCount"]?.toIntOrNull() ?: 0) > 0
        val returnsValues = row["returnsValues"]

        val formats = byEntity.getOrPut(entity) { mutableListOf() }

        if (!downloadURLs.isNullOrBlank() && !hasAccessService && !originalFormat.isNullOrBlank()) {
            val formatToken = extractLegacyFormatToken(originalFormat)
            formats.add(
                LegacyAvailableFormat(
                    method = null,
                    label = formatToken.uppercase(),
                    format = formatToken,
                    originalFormat = formatToken,
                    href = downloadURLs,
                    type = "ORIGINAL"
                )
            )
            return@forEach
        }

        val returns = returnsValues
            ?.split(",")
            ?.map { it.trim() }
            ?.filter { it.isNotBlank() }
            .orEmpty()
            .distinct()

        if (returns.isNotEmpty()) {
            returns.forEach { ret ->
                val normalized = normalizeLegacyReturnFormat(ret)
                formats.add(
                    LegacyAvailableFormat(
                        method = null,
                        label = legacyLabelForFormat(normalized),
                        format = normalized,
                        originalFormat = normalized,
                        href = buildLegacyExecuteHref(publicBaseUrl, entity, normalized),
                        type = "ORIGINAL"
                    )
                )
            }
        } else if (!originalFormat.isNullOrBlank()) {
            val formatToken = extractLegacyFormatToken(originalFormat)
            formats.add(
                LegacyAvailableFormat(
                    method = null,
                    label = legacyLabelForFormat(formatToken),
                    format = formatToken,
                    originalFormat = formatToken,
                    href = buildLegacyExecuteHref(publicBaseUrl, entity, formatToken),
                    type = "ORIGINAL"
                )
            )
        }
    }

    return byEntity.mapValues { (_, formats) ->
        formats.distinctBy { listOf(it.href, it.format, it.originalFormat, it.label, it.type, it.inputFormat, it.pluginId) }
    }
}

private fun buildLegacyExecuteHref(publicBaseUrl: String, entityUri: String, format: String): String {
    val encoded = URLEncoder.encode(entityUri, "UTF-8")
    return "$publicBaseUrl/execute/$encoded?format=${URLEncoder.encode(format, "UTF-8")}"
}

private fun normalizeLegacyReturnFormat(value: String): String {
    return when {
        value.contains("geo+json", ignoreCase = true) || value.contains("geojson", ignoreCase = true) -> "application/epos.geo+json"
        value.equals("json", ignoreCase = true) -> "application/json"
        else -> value
    }
}

private fun extractLegacyFormatToken(value: String): String {
    val cleaned = value.substringBefore(',').trim().ifBlank { value }
    val fromUri = cleaned.substringAfterLast('/').substringAfterLast('#')
    return if (fromUri.isNotBlank() && fromUri != cleaned) fromUri else cleaned
}

private fun legacyLabelForFormat(format: String): String {
    return when {
        format.contains("epos.geo+json", ignoreCase = true) || format.contains("geojson", ignoreCase = true) -> "GEOJSON"
        format.contains("covjson", ignoreCase = true) -> "COVJSON"
        else -> extractLegacyFormatToken(format).uppercase()
    }
}

private fun ContactPointInfo.toLegacyContactPointMap(): Map<String, Any> {
    val base = linkedMapOf<String, Any>()
    id?.let { base["id"] = it }
    metaid?.let { base["metaid"] = it }
    uid?.let { base["uid"] = it }
    person?.let {
        val personMap = linkedMapOf<String, String>()
        it.id?.let { id -> personMap["id"] = id }
        it.metaid?.let { metaid -> personMap["metaid"] = metaid }
        it.uid?.let { uid -> personMap["uid"] = uid }
        if (personMap.isNotEmpty()) {
            base["person"] = personMap
        }
    }
    return base
}

private fun extractDocumentationUri(documentation: String?): String? {
    if (documentation.isNullOrBlank()) return null
    val trimmed = documentation.trim()
    if (!trimmed.startsWith("{")) return documentation

    val marker = "\"Uri\""
    val idx = trimmed.indexOf(marker)
    if (idx < 0) return documentation
    val colon = trimmed.indexOf(':', idx + marker.length)
    if (colon < 0) return documentation
    val firstQuote = trimmed.indexOf('"', colon + 1)
    if (firstQuote < 0) return documentation
    val secondQuote = trimmed.indexOf('"', firstQuote + 1)
    if (secondQuote < 0) return documentation
    return trimmed.substring(firstQuote + 1, secondQuote)
}

private fun normalizeLegacyType(source: AggregatedEntityResponse): String {
    if (!source.relatedWebservices.isNullOrEmpty()) return "WEB_SERVICE"
    if (!source.relatedDataProducts.isNullOrEmpty()) return "DATA"
    return source.type.uppercase().replace(' ', '_')
}

private fun ServiceParameter.toLegacyServiceParameter(): LegacyServiceParameter = LegacyServiceParameter(
    enum = enumValues,
    defaultValue = defaultValue,
    label = label,
    maxValue = maxValue,
    minValue = minValue,
    multipleValue = multipleValues.toString(),
    name = name,
    property = property,
    readOnlyValue = readOnlyValue ?: "false",
    required = required,
    type = type,
    valuePattern = valuePattern
)
