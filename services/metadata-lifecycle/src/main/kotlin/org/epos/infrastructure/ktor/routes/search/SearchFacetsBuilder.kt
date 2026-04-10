package org.epos.infrastructure.ktor.routes

import org.epos.graphdb.client.GraphDBClient

internal fun buildFacets(graphDBClient: GraphDBClient, graphUri: String): SearchFacets {
    val typeQuery = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        SELECT ?type (COUNT(DISTINCT ?s) AS ?count)
        FROM <$graphUri>
        WHERE {
            ?s rdf:type ?type .
        }
        GROUP BY ?type
        ORDER BY DESC(?count)
        LIMIT 20
    """.trimIndent()

    val typeResults = graphDBClient.query(typeQuery)
    val typeFacets = typeResults.map { row ->
        FacetValue(
            value = extractTypeName(row["type"] ?: ""),
            label = extractTypeName(row["type"] ?: ""),
            count = row["count"]?.toIntOrNull() ?: 0
        )
    }

    val keywordQuery = """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        SELECT ?keyword (COUNT(DISTINCT ?s) AS ?count)
        FROM <$graphUri>
        WHERE {
            ?s dcat:keyword ?keyword .
        }
        GROUP BY ?keyword
        ORDER BY DESC(?count)
        LIMIT 20
    """.trimIndent()

    val keywordResults = graphDBClient.query(keywordQuery)
    val keywordFacets = keywordResults.map { row ->
        FacetValue(
            value = row["keyword"] ?: "",
            label = row["keyword"] ?: "",
            count = row["count"]?.toIntOrNull() ?: 0
        )
    }

    val conceptSchemeFacets = buildConceptSchemeFacets(graphDBClient, graphUri)
    val dataProviderFacets = buildDataProviderFacets(graphDBClient, graphUri)

    return SearchFacets(
        types = typeFacets,
        keywords = keywordFacets,
        conceptSchemes = conceptSchemeFacets.ifEmpty { null },
        dataProviders = dataProviderFacets.ifEmpty { null }
    )
}

internal fun buildConceptSchemeFacets(graphDBClient: GraphDBClient, graphUri: String): List<HierarchicalFacet> {
    val query = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?scheme ?schemeLabel ?concept ?conceptLabel (COUNT(DISTINCT ?entity) AS ?count)
        FROM <$graphUri>
        WHERE {
            ?entity dcat:theme ?concept .
            ?concept skos:inScheme ?scheme .
            OPTIONAL { ?scheme skos:prefLabel ?schemeLabel }
            OPTIONAL { ?scheme rdfs:label ?schemeLabelAlt }
            OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
            OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
        }
        GROUP BY ?scheme ?schemeLabel ?concept ?conceptLabel
        ORDER BY ?scheme DESC(?count)
    """.trimIndent()

    val results = graphDBClient.query(query)
    val schemeMap = mutableMapOf<String, MutableList<Pair<Map<String, String>, Int>>>()

    results.forEach { row ->
        val schemeUri = row["scheme"] ?: return@forEach
        val conceptUri = row["concept"] ?: return@forEach
        val count = row["count"]?.toIntOrNull() ?: 0
        schemeMap.getOrPut(schemeUri) { mutableListOf() }.add(row to count)
    }

    return schemeMap.map { (schemeUri, conceptRows) ->
        val firstRow = conceptRows.firstOrNull()?.first ?: emptyMap()
        val schemeLabel = firstRow["schemeLabel"] ?: firstRow["schemeLabelAlt"] ?: extractTypeName(schemeUri)
        val totalCount = conceptRows.sumOf { it.second }

        val children = conceptRows.map { (row, count) ->
            val conceptUri = row["concept"] ?: ""
            val conceptLabel = row["conceptLabel"] ?: row["conceptLabelAlt"] ?: extractTypeName(conceptUri)
            FacetValue(value = conceptUri, label = conceptLabel, count = count)
        }.sortedByDescending { it.count }

        HierarchicalFacet(
            id = schemeUri,
            label = schemeLabel,
            count = totalCount,
            children = children.ifEmpty { null }
        )
    }.sortedByDescending { it.count }
}

internal fun buildDataProviderFacets(graphDBClient: GraphDBClient, graphUri: String): List<HierarchicalFacet> {
    val query = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX schema: <http://schema.org/>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?org ?orgName ?orgLegalName ?resourceType (COUNT(DISTINCT ?resource) AS ?count)
        FROM <$graphUri>
        WHERE {
            {
                ?resource dcterms:publisher ?org .
            } UNION {
                ?resource schema:provider ?org .
            }
            ?resource rdf:type ?resourceType .
            OPTIONAL { ?org schema:legalName ?orgLegalName }
            OPTIONAL { ?org schema:name ?orgName }
            OPTIONAL { ?org foaf:name ?orgNameAlt }
            OPTIONAL { ?org rdfs:label ?orgLabel }
            FILTER(?resourceType IN (
                dcat:Dataset,
                <https://www.epos-eu.org/epos-dcat-ap#WebService>,
                dcat:Distribution,
                schema:Service,
                <https://www.epos-eu.org/epos-dcat-ap#Facility>,
                <https://www.epos-eu.org/epos-dcat-ap#Equipment>
            ))
        }
        GROUP BY ?org ?orgName ?orgLegalName ?resourceType
        ORDER BY ?org DESC(?count)
    """.trimIndent()

    val results = graphDBClient.query(query)
    val orgMap = mutableMapOf<String, MutableList<Pair<Map<String, String>, Int>>>()

    results.forEach { row ->
        val orgUri = row["org"] ?: return@forEach
        val count = row["count"]?.toIntOrNull() ?: 0
        orgMap.getOrPut(orgUri) { mutableListOf() }.add(row to count)
    }

    return orgMap.map { (orgUri, resourceRows) ->
        val firstRow = resourceRows.firstOrNull()?.first ?: emptyMap()
        val orgName = firstRow["orgLegalName"]
            ?: firstRow["orgName"]
            ?: firstRow["orgNameAlt"]
            ?: firstRow["orgLabel"]
            ?: extractTypeName(orgUri)
        val totalCount = resourceRows.sumOf { it.second }

        val children = resourceRows.map { (row, count) ->
            val resourceType = row["resourceType"] ?: ""
            FacetValue(value = resourceType, label = extractTypeName(resourceType), count = count)
        }.sortedByDescending { it.count }

        HierarchicalFacet(
            id = orgUri,
            label = orgName,
            count = totalCount,
            children = children.ifEmpty { null }
        )
    }.sortedByDescending { it.count }
}

internal fun buildGroupedSearchResponse(
    graphDBClient: GraphDBClient,
    graphUri: String,
    items: List<SearchResultItem>,
    facetsType: String,
    facets: SearchFacets?,
    totalCount: Int,
    page: Int,
    pageSize: Int
): GroupedSearchResponse {
    val rootNode = when (facetsType) {
        "categories" -> buildCategoriesGroupedResults(graphDBClient, graphUri, items)
        "dataproviders" -> buildDataProvidersGroupedResults(graphDBClient, graphUri, items)
        "serviceproviders" -> buildServiceProvidersGroupedResults(graphDBClient, graphUri, items)
        else -> FacetNode(name = "results", distributions = items)
    }

    val filters = buildFiltersFromFacets(facets)

    return GroupedSearchResponse(
        results = rootNode,
        filters = filters,
        total = totalCount,
        page = page,
        pageSize = pageSize,
        totalPages = (totalCount + pageSize - 1) / pageSize
    )
}

internal fun buildCategoriesGroupedResults(
    graphDBClient: GraphDBClient,
    graphUri: String,
    items: List<SearchResultItem>
): FacetNode {
    val categoryQuery = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?entity ?concept ?conceptLabel ?scheme ?schemeLabel ?schemeCode ?schemeLogo ?schemeColor ?schemeHomepage
        FROM <$graphUri>
        WHERE {
            ?entity dcat:theme ?concept .
            ?concept skos:inScheme ?scheme .

            OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
            OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
            OPTIONAL { ?scheme skos:prefLabel ?schemeLabel }
            OPTIONAL { ?scheme rdfs:label ?schemeLabelAlt }
            OPTIONAL { ?scheme <https://www.epos-eu.org/epos-dcat-ap#code> ?schemeCode }
            OPTIONAL { ?scheme <http://xmlns.com/foaf/0.1/logo> ?schemeLogo }
            OPTIONAL { ?scheme <https://www.epos-eu.org/epos-dcat-ap#color> ?schemeColor }
            OPTIONAL { ?scheme <http://xmlns.com/foaf/0.1/homepage> ?schemeHomepage }
        }
    """.trimIndent()

    val categoryResults = graphDBClient.query(categoryQuery)
    val entityCategories = mutableMapOf<String, MutableSet<String>>()
    val conceptInfo = mutableMapOf<String, Map<String, String?>>()
    val schemeInfo = mutableMapOf<String, Map<String, String?>>()
    val conceptToScheme = mutableMapOf<String, String>()

    categoryResults.forEach { row ->
        val entityUri = row["entity"] ?: return@forEach
        val conceptUri = row["concept"] ?: return@forEach
        val schemeUri = row["scheme"] ?: return@forEach

        entityCategories.getOrPut(entityUri) { mutableSetOf() }.add(conceptUri)
        conceptToScheme[conceptUri] = schemeUri

        conceptInfo[conceptUri] = mapOf(
            "label" to (row["conceptLabel"] ?: row["conceptLabelAlt"] ?: extractTypeName(conceptUri))
        )

        schemeInfo[schemeUri] = mapOf(
            "label" to (row["schemeLabel"] ?: row["schemeLabelAlt"] ?: extractTypeName(schemeUri)),
            "code" to row["schemeCode"],
            "logo" to row["schemeLogo"],
            "color" to row["schemeColor"],
            "homepage" to row["schemeHomepage"]
        )
    }

    val schemeToConceptToItems = mutableMapOf<String, MutableMap<String, MutableList<SearchResultItem>>>()
    items.forEach { item ->
        val categories = entityCategories[item.id] ?: return@forEach
        categories.forEach { conceptUri ->
            val schemeUri = conceptToScheme[conceptUri] ?: return@forEach
            schemeToConceptToItems
                .getOrPut(schemeUri) { mutableMapOf() }
                .getOrPut(conceptUri) { mutableListOf() }
                .add(item)
        }
    }

    val schemeNodes = schemeToConceptToItems.map { (schemeUri, conceptMap) ->
        val info = schemeInfo[schemeUri] ?: emptyMap()
        val conceptNodes = conceptMap.map { (conceptUri, conceptItems) ->
            val cInfo = conceptInfo[conceptUri] ?: emptyMap()
            FacetNode(
                name = cInfo["label"] ?: extractTypeName(conceptUri),
                ddss = conceptUri,
                distributions = conceptItems.distinctBy { it.id }
            )
        }.filter { it.distributions?.isNotEmpty() == true }

        FacetNode(
            name = info["label"] ?: extractTypeName(schemeUri),
            id = schemeUri,
            code = info["code"],
            imgUrl = info["logo"],
            color = info["color"],
            linkUrl = info["homepage"],
            children = conceptNodes.ifEmpty { null }
        )
    }.filter { it.children?.isNotEmpty() == true }

    return FacetNode(name = "domains", children = schemeNodes.ifEmpty { null })
}

internal fun buildDataProvidersGroupedResults(
    graphDBClient: GraphDBClient,
    graphUri: String,
    items: List<SearchResultItem>
): FacetNode {
    val providerQuery = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?entity ?org ?orgName
        FROM <$graphUri>
        WHERE {
            {
                ?entity dcterms:publisher ?org .
            } UNION {
                ?entity schema:provider ?org .
            }
            OPTIONAL { ?org schema:name ?orgName }
            OPTIONAL { ?org foaf:name ?orgNameAlt }
            OPTIONAL { ?org schema:legalName ?orgLegalName }
            OPTIONAL { ?org rdfs:label ?orgLabel }
        }
    """.trimIndent()

    val providerResults = graphDBClient.query(providerQuery)
    val entityOrgs = mutableMapOf<String, MutableSet<String>>()
    val orgNames = mutableMapOf<String, String>()

    providerResults.forEach { row ->
        val entityUri = row["entity"] ?: return@forEach
        val orgUri = row["org"] ?: return@forEach
        val orgName = row["orgLegalName"] ?: row["orgName"] ?: row["orgNameAlt"] ?: row["orgLabel"] ?: extractTypeName(orgUri)

        entityOrgs.getOrPut(entityUri) { mutableSetOf() }.add(orgUri)
        orgNames[orgUri] = orgName
    }

    val orgToItems = mutableMapOf<String, MutableList<SearchResultItem>>()
    items.forEach { item ->
        val orgs = entityOrgs[item.id]
        if (orgs.isNullOrEmpty()) {
            orgToItems.getOrPut("Undefined") { mutableListOf() }.add(item)
        } else {
            orgs.forEach { orgUri -> orgToItems.getOrPut(orgUri) { mutableListOf() }.add(item) }
        }
    }

    val orgNodes = orgToItems.map { (orgUri, orgItems) ->
        FacetNode(
            name = if (orgUri == "Undefined") "Undefined" else (orgNames[orgUri] ?: extractTypeName(orgUri)),
            id = if (orgUri == "Undefined") null else orgUri,
            distributions = orgItems.distinctBy { it.id }
        )
    }.filter { it.distributions?.isNotEmpty() == true }
        .sortedByDescending { it.distributions?.size ?: 0 }

    return FacetNode(name = "dataproviders", children = orgNodes.ifEmpty { null })
}

internal fun buildServiceProvidersGroupedResults(
    graphDBClient: GraphDBClient,
    graphUri: String,
    items: List<SearchResultItem>
): FacetNode {
    val serviceProviderQuery = """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?entity ?org ?orgName
        FROM <$graphUri>
        WHERE {
            ?entity dcat:distribution ?dist .
            ?dist dcat:accessService ?ws .
            ?ws schema:provider ?org .

            OPTIONAL { ?org schema:name ?orgName }
            OPTIONAL { ?org foaf:name ?orgNameAlt }
            OPTIONAL { ?org schema:legalName ?orgLegalName }
            OPTIONAL { ?org rdfs:label ?orgLabel }
        }
    """.trimIndent()

    val providerResults = graphDBClient.query(serviceProviderQuery)
    val entityOrgs = mutableMapOf<String, MutableSet<String>>()
    val orgNames = mutableMapOf<String, String>()

    providerResults.forEach { row ->
        val entityUri = row["entity"] ?: return@forEach
        val orgUri = row["org"] ?: return@forEach
        val orgName = row["orgLegalName"] ?: row["orgName"] ?: row["orgNameAlt"] ?: row["orgLabel"] ?: extractTypeName(orgUri)

        entityOrgs.getOrPut(entityUri) { mutableSetOf() }.add(orgUri)
        orgNames[orgUri] = orgName
    }

    val orgToItems = mutableMapOf<String, MutableList<SearchResultItem>>()
    items.forEach { item ->
        val orgs = entityOrgs[item.id]
        if (orgs.isNullOrEmpty()) {
            orgToItems.getOrPut("Undefined") { mutableListOf() }.add(item)
        } else {
            orgs.forEach { orgUri -> orgToItems.getOrPut(orgUri) { mutableListOf() }.add(item) }
        }
    }

    val orgNodes = orgToItems.map { (orgUri, orgItems) ->
        FacetNode(
            name = if (orgUri == "Undefined") "Undefined" else (orgNames[orgUri] ?: extractTypeName(orgUri)),
            id = if (orgUri == "Undefined") null else orgUri,
            distributions = orgItems.distinctBy { it.id }
        )
    }.filter { it.distributions?.isNotEmpty() == true }
        .sortedByDescending { it.distributions?.size ?: 0 }

    return FacetNode(name = "serviceproviders", children = orgNodes.ifEmpty { null })
}

internal fun buildFiltersFromFacets(facets: SearchFacets?): List<FilterNode> {
    if (facets == null) return emptyList()

    val filters = mutableListOf<FilterNode>()

    if (facets.keywords.isNotEmpty()) {
        filters.add(
            FilterNode(
                name = "keywords",
                children = facets.keywords.map { kw ->
                    FilterNode(
                        name = kw.label,
                        id = java.util.Base64.getEncoder().encodeToString(kw.value.toByteArray())
                    )
                }
            )
        )
    }

    if (facets.types.isNotEmpty()) {
        filters.add(
            FilterNode(
                name = "types",
                children = facets.types.map { t -> FilterNode(name = t.label, id = t.value) }
            )
        )
    }

    facets.dataProviders?.let { providers ->
        if (providers.isNotEmpty()) {
            filters.add(
                FilterNode(
                    name = "organisations",
                    children = providers.map { org -> FilterNode(name = org.label, id = org.id) }
                )
            )
        }
    }

    facets.conceptSchemes?.let { schemes ->
        if (schemes.isNotEmpty()) {
            filters.add(
                FilterNode(
                    name = "sciencedomains",
                    children = schemes.flatMap { scheme ->
                        scheme.children?.map { concept -> FilterNode(name = concept.label, id = concept.value) }
                            ?: emptyList()
                    }
                )
            )
        }
    }

    return filters
}
