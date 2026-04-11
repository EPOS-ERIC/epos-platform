package org.epos.infrastructure.ktor.routes

import org.epos.graphdb.client.GraphDBClient
import org.epos.services.EdmMappingService
import java.net.URLEncoder

private const val RDF_TYPE_PREDICATE = "http://www.w3.org/1999/02/22-rdf-syntax-ns#type"

private fun selectPrimaryTypeUri(typeUris: List<String>): String? {
    val candidates = typeUris.filter { it.isNotBlank() }
    if (candidates.isEmpty()) return null

    // Prefer domain RDF classes over EDM classes to keep API type stable for clients.
    return candidates.firstOrNull { !it.startsWith(EdmMappingService.EDM_NAMESPACE) } ?: candidates.first()
}

private fun toApiType(typeUri: String?): String {
    if (typeUri.isNullOrBlank()) return "unknown"
    return extractTypeName(typeUri).lowercase()
}

internal fun buildEdmDetailsResponse(
    graphDBClient: GraphDBClient,
    edmMappingService: EdmMappingService,
    entityUri: String,
    graphUri: String,
    status: String,
    publicBaseUrl: String,
    includeInbound: Boolean = false,
): EdmEntityResponse {
    val detailsQuery =
        """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?p ?o ?oType ?oLabel
        FROM <$graphUri>
        WHERE {
            <$entityUri> ?p ?o .
            OPTIONAL {
                ?o rdf:type ?oType .
                FILTER(isIRI(?o))
            }
            OPTIONAL {
                ?o rdfs:label|dcterms:title|schema:name|foaf:name|skos:prefLabel ?oLabel .
                FILTER(isIRI(?o))
            }
        }
        """.trimIndent()

    val results = graphDBClient.query(detailsQuery)

    val types = mutableListOf<String>()
    val properties = mutableMapOf<String, MutableList<Any>>()
    val relations = mutableMapOf<String, MutableList<RelatedEntity>>()

    var title: String? = null
    var name: String? = null
    var description: String? = null
    val identifiers = mutableListOf<String>()
    val keywords = mutableListOf<String>()
    var created: String? = null
    var modified: String? = null
    var issued: String? = null
    var instanceId: String? = null
    var metaId: String? = null

    results.forEach { row ->
        val predicate = row["p"] ?: return@forEach
        val value = row["o"] ?: return@forEach
        val objectType = row["oType"]
        val objectLabel = row["oLabel"]
        val edmPropName = edmMappingService.getEdmPropertyName(predicate)

        if (predicate == RDF_TYPE_PREDICATE) {
            types.add(value)
            return@forEach
        }

        val isRelation = edmMappingService.isRelationProperty(predicate)
        val isUri = value.startsWith("http://") || value.startsWith("https://")

        if (isRelation && isUri) {
            val relatedType = objectType?.let { toApiType(it) }
            val relatedEntity =
                RelatedEntity(
                    id = value,
                    type = relatedType,
                    name = objectLabel,
                    href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(value, "UTF-8")}",
                )
            relations.getOrPut(edmPropName) { mutableListOf() }.add(relatedEntity)
        } else {
            when (edmPropName) {
                "title" -> if (title == null) title = value
                "name" -> if (name == null) name = value
                "description" -> if (description == null) description = value
                "identifier" -> identifiers.add(value)
                "keywords", "keyword" -> keywords.add(value)
                "created" -> if (created == null) created = value
                "modified" -> if (modified == null) modified = value
                "issued" -> if (issued == null) issued = value
                "instanceId" -> instanceId = value
                "metaId" -> metaId = value
                else -> properties.getOrPut(edmPropName) { mutableListOf() }.add(value)
            }
        }
    }

    val primaryTypeUri = selectPrimaryTypeUri(types) ?: ""
    val primaryType = toApiType(primaryTypeUri)

    val simplifiedProps =
        properties
            .mapValues { (_, values) ->
                if (values.size == 1) values.first() else values.toList()
            }.ifEmpty { null }

    val simplifiedRelations =
        relations
            .mapValues { (_, entities) ->
                if (entities.size == 1) entities.first() as Any else entities.toList() as Any
            }.ifEmpty { null }

    val inbound =
        if (includeInbound) {
            buildInboundRelations(graphDBClient, entityUri, graphUri, publicBaseUrl).ifEmpty { null }
        } else {
            null
        }

    return EdmEntityResponse(
        id = entityUri,
        instanceId = instanceId,
        metaId = metaId,
        type = primaryType,
        typeUri = primaryTypeUri,
        status = status,
        title = title,
        name = name ?: title,
        description = description,
        identifier =
            when {
                identifiers.isEmpty() -> null
                identifiers.size == 1 -> identifiers.first()
                else -> identifiers
            },
        keywords = keywords.ifEmpty { null },
        created = created,
        modified = modified,
        issued = issued,
        properties = simplifiedProps,
        relations = simplifiedRelations,
        inbound = inbound,
    )
}

internal fun buildAggregatedDetailsResponse(
    graphDBClient: GraphDBClient,
    edmMappingService: EdmMappingService,
    entityUri: String,
    graphUri: String,
    status: String,
    publicBaseUrl: String,
    includeInbound: Boolean = false,
): AggregatedEntityResponse {
    val prefixes =
        """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX hydra: <http://www.w3.org/ns/hydra/core#>
        PREFIX epos: <https://www.epos-eu.org/epos-dcat-ap#>
        PREFIX geo: <http://www.opengis.net/ont/geosparql#>
        PREFIX locn: <http://www.w3.org/ns/locn#>
        PREFIX adms: <http://www.w3.org/ns/adms#>
        PREFIX vcard: <http://www.w3.org/2006/vcard/ns#>
        PREFIX sh: <http://www.w3.org/ns/shacl#>
        PREFIX http: <http://www.w3.org/2006/http#>
        """.trimIndent()

    val basicQuery =
        """
        $prefixes
        SELECT ?p ?o
        FROM <$graphUri>
        WHERE {
            <$entityUri> ?p ?o .
        }
        """.trimIndent()

    val basicResults = graphDBClient.query(basicQuery)

    val entityTypes = mutableListOf<String>()
    var title: String? = null
    var description: String? = null
    var license: String? = null
    var downloadURL: String? = null
    var uid: String? = null
    var metaId: String? = null
    var editorId: String? = null
    var frequencyUpdate: String? = null
    var qualityAssurance: String? = null
    val keywords = mutableListOf<String>()
    val scienceDomains = mutableListOf<String>()
    val identifiers = mutableListOf<IdentifierInfo>()
    val relatedWebserviceUris = mutableListOf<String>()
    val distributionUris = mutableListOf<String>()
    val distributionOperationUris = mutableListOf<String>()
    val publisherUris = mutableListOf<String>()
    val contactPointUris = mutableListOf<String>()
    val categoryUris = mutableListOf<String>()
    var spatialUri: String? = null
    var temporalUri: String? = null

    basicResults.forEach { row ->
        val predicate = row["p"] ?: return@forEach
        val value = row["o"] ?: return@forEach

        when {
            predicate == RDF_TYPE_PREDICATE -> entityTypes.add(value)
            predicate.contains("title") -> title = value
            predicate.contains("description") -> description = value
            predicate.contains("license") -> license = value
            predicate.contains("downloadURL") -> downloadURL = value
            predicate.contains("keyword") -> keywords.add(value)
            predicate.contains("uid") -> uid = value
            predicate.contains("metaId") -> metaId = value
            predicate.contains("editor") -> editorId = value
            predicate.contains("accrualPeriodicity") -> frequencyUpdate = value
            predicate.contains("hasQualityAnnotation") -> qualityAssurance = value
            predicate.contains("publisher") || predicate.contains("provider") -> publisherUris.add(value)
            predicate.contains("contactPoint") -> contactPointUris.add(value)
            predicate.contains("theme") || predicate.contains("category") -> categoryUris.add(value)
            predicate.contains("spatial") -> spatialUri = value
            predicate.contains("temporal") -> temporalUri = value
            predicate.contains("distribution") -> distributionUris.add(value)
            predicate.contains("accessService") || predicate.contains("conformsTo") -> relatedWebserviceUris.add(value)
            predicate.contains("operation") -> distributionOperationUris.add(value)
        }
    }

    val dataProviders = buildDataProviders(graphDBClient, graphUri, prefixes, publisherUris)
    val distributionIdentifiers = buildDistributionIdentifiers(graphDBClient, graphUri, prefixes, entityUri)
    val distributionScienceDomains = buildDistributionScienceDomains(graphDBClient, graphUri, prefixes, entityUri)
    val spatial = spatialUri?.let { buildSpatialInfo(graphDBClient, graphUri, prefixes, it) }
    val temporal = temporalUri?.let { buildTemporalCoverage(graphDBClient, graphUri, prefixes, it) }
    val categories = if (categoryUris.isNotEmpty()) buildCategoriesFacet(graphDBClient, graphUri, prefixes, categoryUris) else null
    val relatedDataProducts = buildRelatedDataProducts(graphDBClient, graphUri, prefixes, entityUri, publicBaseUrl)
    val relatedWebservices =
        buildRelatedWebservices(
            graphDBClient,
            graphUri,
            prefixes,
            distributionUris + relatedWebserviceUris,
            publicBaseUrl,
            distributionOperationUris.distinct(),
        )
    val availableFormats = buildAvailableFormats(graphDBClient, graphUri, prefixes, distributionUris)
    val contactPoints = buildContactPoints(graphDBClient, graphUri, prefixes, contactPointUris)
    val primaryTypeUri = selectPrimaryTypeUri(entityTypes)
    val apiType = toApiType(primaryTypeUri)
    val inbound =
        if (includeInbound) {
            buildInboundRelations(graphDBClient, entityUri, graphUri, publicBaseUrl).ifEmpty { null }
        } else {
            null
        }

    return AggregatedEntityResponse(
        id = entityUri,
        uid = uid,
        metaId = metaId,
        type = apiType,
        status = status,
        title = title,
        description = description,
        license = license,
        downloadURL = downloadURL,
        keywords = keywords.ifEmpty { null },
        frequencyUpdate = frequencyUpdate,
        scienceDomain = (scienceDomains + distributionScienceDomains).distinct().ifEmpty { null },
        qualityAssurance = qualityAssurance,
        editorId = editorId,
        href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(entityUri, "UTF-8")}",
        hrefExtended = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(entityUri, "UTF-8")}?extended=true",
        spatial = spatial,
        temporalCoverage = temporal,
        dataProvider = dataProviders.ifEmpty { null },
        identifiers = (identifiers + distributionIdentifiers).distinctBy { "${it.type}:${it.value}" }.ifEmpty { null },
        contactPoints = contactPoints.ifEmpty { null },
        availableFormats = availableFormats.ifEmpty { null },
        categories = categories,
        relatedDataProducts = relatedDataProducts.ifEmpty { null },
        relatedWebservices = relatedWebservices.ifEmpty { null },
        inbound = inbound,
    )
}

internal fun buildInboundRelations(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): List<InboundRelation> {
    val query =
        """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT DISTINCT ?source ?predicate ?sourceType ?sourceLabel
        FROM <$graphUri>
        WHERE {
            ?source ?predicate <$entityUri> .
            FILTER(isIRI(?source))
            FILTER(?source != <$entityUri>)

            OPTIONAL { ?source rdf:type ?sourceType . }
            OPTIONAL { ?source rdfs:label|dcterms:title|schema:name|foaf:name|skos:prefLabel ?sourceLabel . }
        }
        ORDER BY ?predicate ?source
        """.trimIndent()

    val rows = graphDBClient.query(query)
    if (rows.isEmpty()) {
        return emptyList()
    }

    val grouped = linkedMapOf<String, LinkedHashMap<String, InboundEntity>>()

    rows.forEach { row ->
        val predicate = row["predicate"] ?: return@forEach
        val sourceId = row["source"] ?: return@forEach
        val sourceTypeUri = row["sourceType"]
        val sourceType = if (sourceTypeUri.isNullOrBlank()) null else toApiType(sourceTypeUri)
        val sourceTitle = row["sourceLabel"]

        val bySource = grouped.getOrPut(predicate) { linkedMapOf() }
        val current = bySource[sourceId]
        if (current == null) {
            bySource[sourceId] =
                InboundEntity(
                    id = sourceId,
                    type = sourceType,
                    title = sourceTitle,
                    href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(sourceId, "UTF-8")}",
                )
        } else if ((current.type == null && sourceType != null) || (current.title == null && sourceTitle != null)) {
            bySource[sourceId] =
                current.copy(
                    type = current.type ?: sourceType,
                    title = current.title ?: sourceTitle,
                )
        }
    }

    return grouped.map { (predicate, entities) ->
        InboundRelation(predicate = predicate, items = entities.values.toList())
    }
}

internal fun buildDataProviders(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    publisherUris: List<String>,
): List<DataServiceProvider> {
    if (publisherUris.isEmpty()) return emptyList()

    return publisherUris.mapNotNull { uri ->
        val query =
            """
            $prefixes
            SELECT ?name ?legalName ?url ?country
            FROM <$graphUri>
            WHERE {
                <$uri> a ?type .
                OPTIONAL { <$uri> schema:name ?name }
                OPTIONAL { <$uri> foaf:name ?foafName }
                OPTIONAL { <$uri> schema:legalName ?legalName }
                OPTIONAL { <$uri> schema:url ?url }
                OPTIONAL { <$uri> schema:address/schema:addressCountry ?country }
            }
            LIMIT 1
            """.trimIndent()

        val results = graphDBClient.query(query)
        results.firstOrNull()?.let { row ->
            DataServiceProvider(
                dataProviderLegalName = row["legalName"] ?: row["name"] ?: row["foafName"],
                dataProviderUrl = row["url"],
                country = row["country"],
                instanceid = uri,
            )
        }
    }
}

internal fun buildSpatialInfo(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    spatialUri: String,
): SpatialInfo? {
    val query =
        """
        $prefixes
        SELECT ?wkt ?location
        FROM <$graphUri>
        WHERE {
            {
                <$spatialUri> geo:asWKT ?wkt .
            } UNION {
                <$spatialUri> locn:geometry ?geom .
                ?geom geo:asWKT ?wkt .
            } UNION {
                <$spatialUri> dcat:bbox ?wkt .
            }
            OPTIONAL { <$spatialUri> locn:geometry ?location }
        }
        LIMIT 1
        """.trimIndent()

    val results = graphDBClient.query(query)
    return results.firstOrNull()?.let { row ->
        val location = row["wkt"] ?: row["location"]
        val point = parsePointWkt(location)
        SpatialInfo(
            wkid = 4326,
            location = location,
            x = point?.first,
            y = point?.second,
        )
    }
}

private fun parsePointWkt(wkt: String?): Pair<Double, Double>? {
    if (wkt.isNullOrBlank()) return null
    val trimmed = wkt.trim()
    if (!trimmed.startsWith("POINT", ignoreCase = true)) return null
    val start = trimmed.indexOf('(')
    val end = trimmed.lastIndexOf(')')
    if (start < 0 || end <= start) return null
    val coords = trimmed.substring(start + 1, end).trim().split(Regex("\\s+"))
    if (coords.size < 2) return null
    val x = coords[0].toDoubleOrNull() ?: return null
    val y = coords[1].toDoubleOrNull() ?: return null
    return x to y
}

internal fun buildTemporalCoverage(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    temporalUri: String,
): TemporalCoverage? {
    val query =
        """
        $prefixes
        SELECT ?startDate ?endDate
        FROM <$graphUri>
        WHERE {
            OPTIONAL { <$temporalUri> dcat:startDate ?startDate }
            OPTIONAL { <$temporalUri> dcat:endDate ?endDate }
            OPTIONAL { <$temporalUri> schema:startDate ?startDate }
            OPTIONAL { <$temporalUri> schema:endDate ?endDate }
        }
        LIMIT 1
        """.trimIndent()

    val results = graphDBClient.query(query)
    return results.firstOrNull()?.let { row ->
        if (row["startDate"] != null || row["endDate"] != null) {
            TemporalCoverage(startDate = row["startDate"], endDate = row["endDate"])
        } else {
            null
        }
    }
}

internal fun buildCategoriesFacet(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    categoryUris: List<String>,
): FacetNode? {
    if (categoryUris.isEmpty()) return null

    val categories =
        categoryUris.mapNotNull { uri ->
            val query =
                """
                $prefixes
                SELECT ?label ?scheme ?schemeLabel
                FROM <$graphUri>
                WHERE {
                    <$uri> a skos:Concept .
                    OPTIONAL { <$uri> skos:prefLabel ?label }
                    OPTIONAL { <$uri> rdfs:label ?rdfsLabel }
                    OPTIONAL {
                        <$uri> skos:inScheme ?scheme .
                        OPTIONAL { ?scheme skos:prefLabel ?schemeLabel }
                    }
                }
                LIMIT 1
                """.trimIndent()

            val results = graphDBClient.query(query)
            results.firstOrNull()?.let { row ->
                FacetNode(name = row["label"] ?: row["rdfsLabel"] ?: extractTypeName(uri), ddss = uri)
            }
        }

    return if (categories.isNotEmpty()) FacetNode(name = "categories", children = categories) else null
}

internal fun buildRelatedDataProducts(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    entityUri: String,
    publicBaseUrl: String,
): List<DataProductInfo> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?dp ?title ?description ?license ?uid ?accessRights ?qualityAssurance ?frequencyUpdate
        FROM <$graphUri>
        WHERE {
            ?dp a dcat:Dataset .
            ?dp dcat:distribution <$entityUri> .
            OPTIONAL { ?dp dcterms:title ?title }
            OPTIONAL { ?dp dcterms:description ?description }
            OPTIONAL { ?dp dcterms:license ?license }
            OPTIONAL { ?dp epos:uid ?uid }
            OPTIONAL { ?dp dcterms:accessRights ?accessRights }
            OPTIONAL { ?dp epos:hasQualityAnnotation ?qualityAssurance }
            OPTIONAL { ?dp dcterms:accrualPeriodicity ?frequencyUpdate }
        }
        """.trimIndent()

    val results = graphDBClient.query(query)

    return results.mapNotNull { row ->
        val dpUri = row["dp"] ?: return@mapNotNull null
        val identifiers = buildDataProductIdentifiers(graphDBClient, graphUri, prefixes, dpUri)
        val dataProviders = buildDataProductProviders(graphDBClient, graphUri, prefixes, dpUri)
        val scienceDomains = buildDataProductScienceDomains(graphDBClient, graphUri, prefixes, dpUri)
        val contactPoints = buildDataProductContactPoints(graphDBClient, graphUri, prefixes, dpUri)
        val categories = buildDataProductCategories(graphDBClient, graphUri, prefixes, dpUri)
        val provenance = buildDataProductProvenance(graphDBClient, graphUri, prefixes, dpUri)
        val spatial = buildDataProductSpatial(graphDBClient, graphUri, prefixes, dpUri)
        val temporalCoverage = buildDataProductTemporal(graphDBClient, graphUri, prefixes, dpUri)

        DataProductInfo(
            id = dpUri,
            uid = row["uid"],
            type = "dataset",
            title = row["title"],
            description = row["description"],
            license = row["license"],
            frequencyUpdate = row["frequencyUpdate"],
            identifiers = identifiers.ifEmpty { null },
            dataProvider = dataProviders.ifEmpty { null },
            scienceDomain = scienceDomains.ifEmpty { null },
            contactPoints = contactPoints.ifEmpty { null },
            categories = categories,
            provenance = provenance.ifEmpty { null },
            spatial = spatial,
            temporalCoverage = temporalCoverage.ifEmpty { null },
            qualityAssurance = row["qualityAssurance"],
            accessRights = row["accessRights"],
            href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(dpUri, "UTF-8")}",
        )
    }
}

private fun buildDataProductIdentifiers(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<IdentifierInfo> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?idType ?idValue
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> adms:identifier|dcterms:identifier ?identifier .
            OPTIONAL { ?identifier dcterms:type|schema:propertyID|adms:schemeAgency ?idType }
            OPTIONAL { ?identifier rdf:value|skos:notation|schema:value ?idNodeValue }
            BIND(COALESCE(?idNodeValue, STR(?identifier)) AS ?idValue)
        }
        """.trimIndent()
    return graphDBClient.query(query).mapNotNull { row ->
        val value = row["idValue"] ?: return@mapNotNull null
        IdentifierInfo(type = row["idType"] ?: inferIdentifierType(value), value = value)
    }
}

private fun buildDataProductProviders(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<DataServiceProvider> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?org
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> dcterms:publisher|schema:provider ?org .
        }
        """.trimIndent()
    val uris = graphDBClient.query(query).mapNotNull { it["org"] }
    return buildDataProviders(graphDBClient, graphUri, prefixes, uris)
}

private fun buildDataProductScienceDomains(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<String> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?label ?theme
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> dcat:theme ?theme .
            OPTIONAL { ?theme skos:prefLabel|rdfs:label ?label }
        }
        """.trimIndent()
    return graphDBClient.query(query).mapNotNull { it["label"] ?: it["theme"] }.distinct()
}

private fun buildDataProductContactPoints(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<ContactPointInfo> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?cp
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> dcat:contactPoint ?cp .
        }
        """.trimIndent()
    val cpUris = graphDBClient.query(query).mapNotNull { it["cp"] }
    return buildContactPoints(graphDBClient, graphUri, prefixes, cpUris)
}

private fun buildDataProductCategories(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): FacetNode? {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?theme
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> dcat:theme ?theme .
        }
        """.trimIndent()
    val categories = graphDBClient.query(query).mapNotNull { it["theme"] }
    return if (categories.isEmpty()) null else buildCategoriesFacet(graphDBClient, graphUri, prefixes, categories)
}

private fun buildDataProductProvenance(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<Map<String, String>> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?prov
        FROM <$graphUri>
        WHERE {
            <$dataProductUri> epos:provenance ?prov .
        }
        """.trimIndent()
    return graphDBClient.query(query).mapNotNull { row ->
        row["prov"]?.let { mapOf("provenance" to it) }
    }
}

private fun buildDataProductSpatial(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): SpatialInfo? {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?sp
        FROM <$graphUri>
        WHERE { <$dataProductUri> dcterms:spatial ?sp . }
        LIMIT 1
        """.trimIndent()
    val spUri = graphDBClient.query(query).firstOrNull()?.get("sp") ?: return null
    return buildSpatialInfo(graphDBClient, graphUri, prefixes, spUri)
}

private fun buildDataProductTemporal(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    dataProductUri: String,
): List<TemporalCoverage> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?tp
        FROM <$graphUri>
        WHERE { <$dataProductUri> dcterms:temporal ?tp . }
        """.trimIndent()
    val temporalUris = graphDBClient.query(query).mapNotNull { it["tp"] }
    return temporalUris.mapNotNull { uri -> buildTemporalCoverage(graphDBClient, graphUri, prefixes, uri) }
}

internal fun buildDistributionIdentifiers(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    distributionUri: String,
): List<IdentifierInfo> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?idType ?idValue
        FROM <$graphUri>
        WHERE {
            {
                ?dp a dcat:Dataset .
                ?dp dcat:distribution <$distributionUri> .
                ?dp adms:identifier|dcterms:identifier ?identifier .
                OPTIONAL { ?identifier dcterms:type|schema:propertyID|adms:schemeAgency ?idType }
                OPTIONAL { ?identifier rdf:value|skos:notation|schema:value ?idNodeValue }
                BIND(COALESCE(?idNodeValue, STR(?identifier)) AS ?idValue)
            }
        }
        """.trimIndent()

    return graphDBClient
        .query(query)
        .mapNotNull { row ->
            val value = row["idValue"] ?: return@mapNotNull null
            IdentifierInfo(
                type = row["idType"] ?: inferIdentifierType(value),
                value = value,
            )
        }
}

internal fun buildDistributionScienceDomains(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    distributionUri: String,
): List<String> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?label ?theme
        FROM <$graphUri>
        WHERE {
            ?dp a dcat:Dataset .
            ?dp dcat:distribution <$distributionUri> .
            ?dp dcat:theme ?theme .
            OPTIONAL { ?theme skos:prefLabel|rdfs:label ?label }
        }
        """.trimIndent()

    return graphDBClient
        .query(query)
        .mapNotNull { row -> row["label"] ?: row["theme"] }
        .distinct()
}

private fun inferIdentifierType(value: String): String =
    when {
        value.contains("doi", ignoreCase = true) || value.startsWith("10.") -> "DOI"
        value.contains("ddss", ignoreCase = true) -> "DDSS-ID"
        else -> "IDENTIFIER"
    }

internal fun buildRelatedWebservices(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    serviceUris: List<String>,
    publicBaseUrl: String,
    distributionOperationUris: List<String> = emptyList(),
): List<WebserviceInfo> {
    if (serviceUris.isEmpty()) return emptyList()

    return serviceUris.distinct().mapNotNull { uri ->
        val accessServiceQuery =
            """
            $prefixes
            SELECT ?ws
            FROM <$graphUri>
            WHERE {
                {
                    <$uri> dcat:accessService ?ws .
                } UNION {
                    <$uri> dcterms:conformsTo ?ws .
                } UNION {
                    BIND(<$uri> AS ?ws)
                    ?ws a epos:WebService .
                } UNION {
                    BIND(<$uri> AS ?ws)
                    ?ws a dcat:DataService .
                }
            }
            LIMIT 1
            """.trimIndent()

        val accessResults = graphDBClient.query(accessServiceQuery)
        val wsUri = accessResults.firstOrNull()?.get("ws") ?: return@mapNotNull null

        val wsQuery =
            """
            $prefixes
            SELECT ?name ?description ?providerUri ?providerName ?documentation ?uid ?metaid
            FROM <$graphUri>
            WHERE {
                OPTIONAL { <$wsUri> schema:name ?name }
                OPTIONAL { <$wsUri> dcterms:title ?title }
                OPTIONAL { <$wsUri> dcterms:description ?description }
                OPTIONAL {
                    <$wsUri> schema:provider ?providerUri .
                    OPTIONAL { ?providerUri schema:legalName ?providerName }
                }
                OPTIONAL {
                    <$wsUri> dcterms:publisher ?providerUriAlt .
                    OPTIONAL { ?providerUriAlt schema:legalName|schema:name|foaf:name ?providerNameAlt }
                }
                OPTIONAL { <$wsUri> schema:documentation ?documentation }
                OPTIONAL { <$wsUri> dcterms:conformsTo ?documentationAlt }
                OPTIONAL { <$wsUri> epos:uid ?uid }
                OPTIONAL { <$wsUri> epos:metaId ?metaid }
            }
            LIMIT 1
            """.trimIndent()

        val wsResults = graphDBClient.query(wsQuery)
        val wsRow = wsResults.firstOrNull() ?: return@mapNotNull null
        val operations =
            buildOperations(
                graphDBClient,
                graphUri,
                prefixes,
                wsUri,
                publicBaseUrl,
                distributionOperationUris,
            )
        val spatial = buildWebserviceSpatial(graphDBClient, graphUri, prefixes, wsUri)
        val temporalCoverage = buildWebserviceTemporal(graphDBClient, graphUri, prefixes, wsUri)
        val serviceTypes = buildWebserviceTypes(graphDBClient, graphUri, prefixes, wsUri)
        val contactPoints = buildWebserviceContactPoints(graphDBClient, graphUri, prefixes, wsUri)
        val categories = buildWebserviceCategories(graphDBClient, graphUri, prefixes, wsUri)

        WebserviceInfo(
            id = wsUri,
            uid = wsRow["uid"],
            metaid = wsRow["metaid"],
            name = wsRow["name"] ?: wsRow["title"],
            description = wsRow["description"],
            provider =
                (wsRow["providerUri"] ?: wsRow["providerUriAlt"])?.let {
                    DataServiceProvider(
                        dataProviderLegalName = wsRow["providerName"] ?: wsRow["providerNameAlt"],
                        instanceid = it,
                    )
                },
            spatial = spatial,
            temporalCoverage = temporalCoverage.ifEmpty { null },
            documentation = wsRow["documentation"] ?: wsRow["documentationAlt"],
            type = serviceTypes.ifEmpty { null },
            contactPoints = contactPoints.ifEmpty { null },
            operations = operations.ifEmpty { null },
            categories = categories,
            href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(wsUri, "UTF-8")}",
        )
    }
}

private fun buildWebserviceSpatial(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
): SpatialInfo? {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?sp
        FROM <$graphUri>
        WHERE { <$webserviceUri> dcterms:spatial|schema:spatialCoverage ?sp . }
        LIMIT 1
        """.trimIndent()
    val spUri = graphDBClient.query(query).firstOrNull()?.get("sp") ?: return null
    return buildSpatialInfo(graphDBClient, graphUri, prefixes, spUri)
}

private fun buildWebserviceTemporal(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
): List<TemporalCoverage> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?tp
        FROM <$graphUri>
        WHERE { <$webserviceUri> dcterms:temporal ?tp . }
        """.trimIndent()
    val temporalUris = graphDBClient.query(query).mapNotNull { it["tp"] }
    return temporalUris.mapNotNull { uri -> buildTemporalCoverage(graphDBClient, graphUri, prefixes, uri) }
}

private fun buildWebserviceTypes(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
): List<String> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?label ?cat
        FROM <$graphUri>
        WHERE {
            <$webserviceUri> dcterms:type|dcat:theme ?cat .
            OPTIONAL { ?cat skos:prefLabel|rdfs:label ?label }
        }
        """.trimIndent()
    return graphDBClient.query(query).mapNotNull { it["label"] ?: it["cat"] }
}

private fun buildWebserviceContactPoints(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
): List<ContactPointInfo> {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?cp
        FROM <$graphUri>
        WHERE { <$webserviceUri> dcat:contactPoint ?cp . }
        """.trimIndent()
    val cpUris = graphDBClient.query(query).mapNotNull { it["cp"] }
    return buildContactPoints(graphDBClient, graphUri, prefixes, cpUris)
}

private fun buildWebserviceCategories(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
): Map<String, List<String>>? {
    val query =
        """
        $prefixes
        SELECT DISTINCT ?schemeLabel ?label
        FROM <$graphUri>
        WHERE {
            <$webserviceUri> dcat:theme ?cat .
            OPTIONAL { ?cat skos:prefLabel|rdfs:label ?label }
            OPTIONAL {
                ?cat skos:inScheme ?scheme .
                ?scheme skos:prefLabel|rdfs:label ?schemeLabel
            }
        }
        """.trimIndent()
    val rows = graphDBClient.query(query)
    if (rows.isEmpty()) return null
    val grouped = linkedMapOf<String, MutableList<String>>()
    rows.forEach { row ->
        val key = row["schemeLabel"] ?: "categories"
        val value = row["label"] ?: return@forEach
        grouped.getOrPut(key) { mutableListOf() }.add(value)
    }
    return grouped.mapValues { (_, values) -> values.distinct() }
}

internal fun buildOperations(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    webserviceUri: String,
    publicBaseUrl: String,
    distributionOperationUris: List<String> = emptyList(),
): List<OperationInfo> {
    val query =
        """
        $prefixes
        SELECT ?op ?method ?resolvedTemplate ?uid
        FROM <$graphUri>
        WHERE {
            {
                <$webserviceUri> hydra:supportedOperation ?op .
            } UNION {
                <$webserviceUri> dcat:endpointDescription ?op .
            } UNION {
                <$webserviceUri> dcterms:relation ?op .
            } UNION {
                <$webserviceUri> ?opPredicate ?op .
                FILTER(CONTAINS(LCASE(STR(?opPredicate)), "operation"))
            }
            OPTIONAL { ?op hydra:method ?method }
            OPTIONAL { ?op hydra:template ?directTemplate }
            OPTIONAL {
                ?op hydra:returns ?returnsNode .
                ?returnsNode hydra:template ?templateFromReturns .
            }
            OPTIONAL {
                ?op hydra:property ?iriTemplate .
                ?iriTemplate hydra:template ?templateFromProperty .
            }
            OPTIONAL { ?op epos:uid ?uid }
            BIND(COALESCE(?templateFromReturns, ?templateFromProperty, ?directTemplate) AS ?resolvedTemplate)
        }
        """.trimIndent()

    val results = graphDBClient.query(query)

    val mapped =
        results.mapNotNull { row ->
            val opUri = row["op"] ?: return@mapNotNull null
            if (distributionOperationUris.isNotEmpty() && opUri !in distributionOperationUris) {
                return@mapNotNull null
            }
            val parameters = buildServiceParameters(graphDBClient, graphUri, prefixes, opUri)
            OperationInfo(
                id = opUri,
                uid = row["uid"],
                method = row["method"] ?: "GET",
                endpoint = row["resolvedTemplate"],
                parameters = parameters.ifEmpty { null },
                href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(opUri, "UTF-8")}",
            )
        }

    if (mapped.isNotEmpty() || distributionOperationUris.isEmpty()) {
        return mapped
    }

    // Fallback: distribution-operation identifiers may be in a different form (uid vs iri)
    return results.mapNotNull { row ->
        val opUri = row["op"] ?: return@mapNotNull null
        val parameters = buildServiceParameters(graphDBClient, graphUri, prefixes, opUri)
        OperationInfo(
            id = opUri,
            uid = row["uid"],
            method = row["method"] ?: "GET",
            endpoint = row["resolvedTemplate"],
            parameters = parameters.ifEmpty { null },
            href = "$publicBaseUrl/api/v2/details/${URLEncoder.encode(opUri, "UTF-8")}",
        )
    }
}

internal fun buildServiceParameters(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    operationUri: String,
): List<ServiceParameter> {
    val query =
        """
        $prefixes
        SELECT ?mapping ?variable ?property ?required ?minValue ?maxValue ?defaultValue ?label
               ?range ?valuePattern ?multiple ?readOnly
               (GROUP_CONCAT(DISTINCT STR(?paramValue);SEPARATOR=",") AS ?enumValues)
        FROM <$graphUri>
        WHERE {
            {
                <$operationUri> hydra:returns ?template .
                ?template hydra:mapping ?mapping .
            } UNION {
                <$operationUri> hydra:expects ?template .
                ?template hydra:mapping ?mapping .
            } UNION {
                <$operationUri> hydra:property ?template .
                ?template hydra:mapping ?mapping .
            } UNION {
                <$operationUri> hydra:mapping ?mapping .
            } UNION {
                <$operationUri> ?mappingPredicate ?mapping .
                FILTER(CONTAINS(LCASE(STR(?mappingPredicate)), "mapping"))
            }
            OPTIONAL { ?mapping hydra:variable|schema:identifier ?variable }
            OPTIONAL { ?mapping hydra:property|schema:propertyID|epos:property ?property }
            OPTIONAL { ?mapping hydra:required ?required }
            OPTIONAL { ?mapping hydra:range|schema:rangeIncludes|epos:range ?range }
            OPTIONAL { ?mapping schema:minValue|epos:minValue ?minValue }
            OPTIONAL { ?mapping schema:maxValue|epos:maxValue ?maxValue }
            OPTIONAL { ?mapping schema:defaultValue|epos:defaultValue ?defaultValue }
            OPTIONAL { ?mapping schema:pattern|epos:valuePattern ?valuePattern }
            OPTIONAL { ?mapping epos:multipleValues|schema:multipleValues ?multiple }
            OPTIONAL { ?mapping epos:readOnlyValue|schema:readOnly|schema:readonlyValue ?readOnly }
            OPTIONAL { ?mapping rdfs:label|dcterms:description ?label }
            OPTIONAL { ?mapping schema:value|epos:paramValue|http:paramValue ?paramValue }
            OPTIONAL {
                ?mapping sh:in ?enumList .
                ?enumList rdf:rest*/rdf:first ?enumNodeValue .
                BIND(STR(?enumNodeValue) AS ?paramValue)
            }
        }
        GROUP BY ?mapping ?variable ?property ?required ?minValue ?maxValue ?defaultValue ?label ?range ?valuePattern ?multiple ?readOnly
        """.trimIndent()

    val results = graphDBClient.query(query)

    val strictParameters =
        results.mapNotNull { row ->
            val variable = row["variable"] ?: return@mapNotNull null
            ServiceParameter(
                name = variable,
                label = row["label"],
                type = row["range"]?.substringAfterLast(":")?.substringAfterLast("#"),
                property = row["property"],
                required = row["required"]?.toBoolean() ?: false,
                minValue = row["minValue"],
                maxValue = row["maxValue"],
                defaultValue = row["defaultValue"],
                enumValues =
                    row["enumValues"]
                        ?.split(",")
                        ?.map { it.trim() }
                        ?.filter { it.isNotBlank() }
                        ?.distinct(),
                valuePattern = row["valuePattern"],
                multipleValues = row["multiple"]?.toBoolean() ?: false,
                readOnlyValue = row["readOnly"],
            )
        }

    if (strictParameters.isNotEmpty()) {
        return strictParameters
    }

    return buildServiceParametersFallback(graphDBClient, graphUri, operationUri)
}

private fun buildServiceParametersFallback(
    graphDBClient: GraphDBClient,
    graphUri: String,
    operationUri: String,
): List<ServiceParameter> {
    val mappingQuery =
        """
        PREFIX hydra: <http://www.w3.org/ns/hydra/core#>

        SELECT DISTINCT ?mapping
        FROM <$graphUri>
        WHERE {
            {
                <$operationUri> hydra:mapping ?mapping .
            } UNION {
                <$operationUri> ?predicate ?mapping .
                FILTER(CONTAINS(LCASE(STR(?predicate)), "mapping"))
            }
        }
        """.trimIndent()

    val mappings = graphDBClient.query(mappingQuery).mapNotNull { it["mapping"] }.distinct()
    if (mappings.isEmpty()) return emptyList()

    return mappings.mapNotNull { mappingUri ->
        val detailQuery =
            """
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

            SELECT ?predicate ?value
            FROM <$graphUri>
            WHERE {
                <$mappingUri> ?predicate ?value .
            }
            """.trimIndent()

        val rows = graphDBClient.query(detailQuery)
        if (rows.isEmpty()) return@mapNotNull null

        fun pick(vararg keywords: String): String? {
            return rows
                .firstOrNull { row ->
                    val p = row["predicate"]?.lowercase() ?: return@firstOrNull false
                    keywords.any { k -> p.contains(k) }
                }?.get("value")
        }

        val enumValues =
            rows
                .filter { row ->
                    val p = row["predicate"]?.lowercase() ?: return@filter false
                    p.contains("paramvalue") || p.contains("enum") || p.contains("allowed")
                }.mapNotNull { it["value"] }
                .distinct()
                .ifEmpty { null }

        val name = pick("variable", "identifier", "name") ?: return@mapNotNull null
        ServiceParameter(
            name = name,
            label = pick("label", "description"),
            type = pick("range", "type")?.substringAfterLast(":")?.substringAfterLast("#"),
            property = pick("property"),
            required = pick("required")?.toBoolean() ?: false,
            minValue = pick("minvalue"),
            maxValue = pick("maxvalue"),
            defaultValue = pick("defaultvalue"),
            enumValues = enumValues,
            valuePattern = pick("valuepattern", "pattern"),
            multipleValues = pick("multiple")?.toBoolean() ?: false,
            readOnlyValue = pick("readonly"),
        )
    }
}

internal fun buildAvailableFormats(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    distributionUris: List<String>,
): List<AvailableFormat> {
    if (distributionUris.isEmpty()) return emptyList()

    return distributionUris.flatMap { uri ->
        val query =
            """
            $prefixes
            SELECT ?format ?downloadURL ?accessURL ?title
            FROM <$graphUri>
            WHERE {
                <$uri> a dcat:Distribution .
                OPTIONAL { <$uri> dcterms:format ?format }
                OPTIONAL { <$uri> dcat:downloadURL ?downloadURL }
                OPTIONAL { <$uri> dcat:accessURL ?accessURL }
                OPTIONAL { <$uri> dcterms:title ?title }
            }
            LIMIT 1
            """.trimIndent()

        val results = graphDBClient.query(query)
        results.mapNotNull { row ->
            val url = row["downloadURL"] ?: row["accessURL"]
            val format = row["format"]
            if (url != null) {
                AvailableFormat(
                    label = row["title"] ?: legacyFormatLabel(format),
                    format = format,
                    originalFormat = format,
                    href = url,
                    type = "original",
                )
            } else {
                null
            }
        }
    }
}

private fun legacyFormatLabel(format: String?): String? {
    if (format.isNullOrBlank()) return null
    val normalized = format.substringAfterLast('/').substringAfterLast('#').ifBlank { format }
    return normalized.uppercase()
}

internal fun buildContactPoints(
    graphDBClient: GraphDBClient,
    graphUri: String,
    prefixes: String,
    contactPointUris: List<String>,
): List<ContactPointInfo> {
    if (contactPointUris.isEmpty()) return emptyList()

    return contactPointUris.mapNotNull { uri ->
        val query =
            """
            $prefixes
            SELECT ?cpMetaId ?cpUid ?email ?telephone ?role ?personUri ?personMetaId ?personUid ?personName
            FROM <$graphUri>
            WHERE {
                <$uri> a ?type .
                OPTIONAL { <$uri> epos:metaId ?cpMetaId }
                OPTIONAL { <$uri> epos:uid ?cpUid }
                OPTIONAL { <$uri> schema:email ?email }
                OPTIONAL { <$uri> vcard:hasEmail ?vcardEmail }
                OPTIONAL { <$uri> schema:telephone ?telephone }
                OPTIONAL { <$uri> epos:role ?role }
                OPTIONAL {
                    <$uri> epos:hasPerson ?personUri .
                    OPTIONAL { ?personUri epos:metaId ?personMetaId }
                    OPTIONAL { ?personUri epos:uid ?personUid }
                    OPTIONAL { ?personUri schema:name ?personName }
                }
            }
            LIMIT 1
            """.trimIndent()

        val results = graphDBClient.query(query)
        results.firstOrNull()?.let { row ->
            ContactPointInfo(
                id = uri,
                metaid = row["cpMetaId"],
                uid = row["cpUid"],
                email = row["email"] ?: row["vcardEmail"],
                telephone = row["telephone"],
                role = row["role"],
                person =
                    row["personUri"]?.let { personUri ->
                        PersonInfo(
                            id = personUri,
                            metaid = row["personMetaId"],
                            uid = row["personUid"],
                            name = row["personName"],
                        )
                    },
            )
        }
    }
}
