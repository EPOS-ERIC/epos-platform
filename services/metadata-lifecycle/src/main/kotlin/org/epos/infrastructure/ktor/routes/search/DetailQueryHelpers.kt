package org.epos.infrastructure.ktor.routes

import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import java.net.URLEncoder

/**
 * Common SPARQL prefixes used across detail queries.
 */
internal val DETAIL_PREFIXES = """
    PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
    PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
    PREFIX dcterms: <http://purl.org/dc/terms/>
    PREFIX dcat: <http://www.w3.org/ns/dcat#>
    PREFIX schema: <http://schema.org/>
    PREFIX foaf: <http://xmlns.com/foaf/0.1/>
    PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
    PREFIX geo: <http://www.opengis.net/ont/geosparql#>
    PREFIX locn: <http://www.w3.org/ns/locn#>
    PREFIX epos: <https://www.epos-eu.org/epos-dcat-ap#>
""".trimIndent()

/**
 * Checks if an entity exists in the specified graph.
 */
internal fun entityExists(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): Boolean {
    val query = """
        ASK {
            GRAPH <$graphUri> {
                <$entityUri> ?p ?o .
            }
        }
    """.trimIndent()
    return graphDBClient.ask(query)
}

/**
 * Builds a plain list of inbound related resources.
 * Returns entities that reference this entity via any predicate.
 */
internal fun buildRelatedByResources(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): List<ResourceRef> {
    val query = """
        $DETAIL_PREFIXES
        
        SELECT DISTINCT ?source ?sourceType ?sourceTitle
        FROM <$graphUri>
        WHERE {
            ?source ?predicate <$entityUri> .
            FILTER(isIRI(?source))
            FILTER(?source != <$entityUri>)
            
            OPTIONAL { ?source rdf:type ?sourceType . }
            OPTIONAL { ?source dcterms:title|rdfs:label|schema:name|foaf:name|skos:prefLabel ?sourceTitle . }
        }
        ORDER BY ?source
    """.trimIndent()

    val rows = graphDBClient.query(query)
    if (rows.isEmpty()) return emptyList()

    val bySource = linkedMapOf<String, ResourceRef>()

    rows.forEach { row ->
        val sourceId = row["source"] ?: return@forEach
        if (bySource.containsKey(sourceId)) return@forEach

        val sourceType = row["sourceType"]?.let { mapTypeUriToApiType(it) } ?: "dataset"
        bySource[sourceId] = ResourceRef(
            id = sourceId,
            title = row["sourceTitle"],
            href = "$publicBaseUrl/api/v2/$sourceType/${URLEncoder.encode(sourceId, "UTF-8")}",
        )
    }

    return bySource.values.toList()
}

/**
 * Builds a plain list of outbound related resources.
 */
internal fun buildRelatedResources(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
    publicBaseUrl: String,
): List<ResourceRef> {
    val query = """
        $DETAIL_PREFIXES

        SELECT DISTINCT ?target ?targetType ?targetTitle
        FROM <$graphUri>
        WHERE {
            <$entityUri> ?predicate ?target .
            FILTER(isIRI(?target))
            FILTER(?target != <$entityUri>)

            OPTIONAL { ?target rdf:type ?targetType . }
            OPTIONAL { ?target dcterms:title|rdfs:label|schema:name|foaf:name|skos:prefLabel ?targetTitle . }
        }
        ORDER BY ?target
    """.trimIndent()

    val rows = graphDBClient.query(query)
    if (rows.isEmpty()) return emptyList()

    val byTarget = linkedMapOf<String, ResourceRef>()
    rows.forEach { row ->
        val targetId = row["target"] ?: return@forEach
        if (byTarget.containsKey(targetId)) return@forEach

        val targetType = row["targetType"]?.let { mapTypeUriToApiType(it) } ?: "dataset"
        byTarget[targetId] = ResourceRef(
            id = targetId,
            title = row["targetTitle"],
            href = "$publicBaseUrl/api/v2/$targetType/${URLEncoder.encode(targetId, "UTF-8")}",
        )
    }

    return byTarget.values.toList()
}

/**
 * Maps RDF type URI to API type name for href construction.
 */
internal fun mapTypeUriToApiType(typeUri: String): String {
    return when {
        typeUri.contains("Dataset", ignoreCase = true) -> "dataset"
        typeUri.contains("Distribution", ignoreCase = true) -> "distribution"
        typeUri.contains("SoftwareApplication", ignoreCase = true) -> "software-application"
        typeUri.contains("SoftwareSourceCode", ignoreCase = true) -> "software-source-code"
        typeUri.contains("Concept", ignoreCase = true) -> "category"
        typeUri.contains("WebService", ignoreCase = true) -> "distribution"
        typeUri.contains("DataService", ignoreCase = true) -> "distribution"
        else -> "dataset" // Default fallback
    }
}

/**
 * Extracts keywords from SPARQL results.
 */
internal fun extractKeywords(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): List<String> {
    val query = """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        SELECT DISTINCT ?keyword
        FROM <$graphUri>
        WHERE {
            <$entityUri> dcat:keyword ?keyword .
        }
    """.trimIndent()

    return graphDBClient.query(query).mapNotNull { it["keyword"] }
}

/**
 * Extracts category labels from SPARQL results.
 */
internal fun extractCategoryLabels(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): List<String> {
    val query = """
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        
        SELECT DISTINCT ?label
        FROM <$graphUri>
        WHERE {
            <$entityUri> dcat:theme ?theme .
            ?theme skos:prefLabel|rdfs:label ?label .
        }
    """.trimIndent()

    return graphDBClient.query(query).mapNotNull { it["label"] }
}

/**
 * Extracts publisher names from SPARQL results.
 */
internal fun extractPublishers(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): List<String> {
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX schema: <http://schema.org/>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        
        SELECT DISTINCT ?name
        FROM <$graphUri>
        WHERE {
            {
                <$entityUri> dcterms:publisher ?org .
                ?org schema:legalName|schema:name|foaf:name ?name .
            }
            UNION
            {
                <$entityUri> schema:provider ?org .
                ?org schema:legalName|schema:name|foaf:name ?name .
            }
        }
    """.trimIndent()

    return graphDBClient.query(query).mapNotNull { it["name"] }
}

/**
 * Extracts spatial WKT from SPARQL results.
 */
internal fun extractSpatialWkt(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): String? {
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX geo: <http://www.opengis.net/ont/geosparql#>
        PREFIX locn: <http://www.w3.org/ns/locn#>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        
        SELECT ?wkt
        FROM <$graphUri>
        WHERE {
            {
                <$entityUri> dcterms:spatial ?sp .
                ?sp geo:asWKT ?wkt .
            }
            UNION
            {
                <$entityUri> dcterms:spatial ?sp .
                ?sp locn:geometry ?geom .
                ?geom geo:asWKT ?wkt .
            }
            UNION
            {
                <$entityUri> geo:hasGeometry ?geom .
                ?geom geo:asWKT ?wkt .
            }
            UNION
            {
                <$entityUri> dcat:bbox ?wkt .
            }
        }
        LIMIT 1
    """.trimIndent()

    return graphDBClient.query(query).firstOrNull()?.get("wkt")
}

/**
 * Extracts temporal extent from SPARQL results.
 */
internal fun extractTemporalExtent(
    graphDBClient: GraphDBClient,
    entityUri: String,
    graphUri: String,
): TemporalExtent? {
    val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX dcat: <http://www.w3.org/ns/dcat#>
        PREFIX schema: <http://schema.org/>
        
        SELECT ?startDate ?endDate
        FROM <$graphUri>
        WHERE {
            <$entityUri> dcterms:temporal ?temporal .
            OPTIONAL { ?temporal dcat:startDate|schema:startDate ?startDate }
            OPTIONAL { ?temporal dcat:endDate|schema:endDate ?endDate }
        }
        LIMIT 1
    """.trimIndent()

    val result = graphDBClient.query(query).firstOrNull() ?: return null
    val start = result["startDate"]
    val end = result["endDate"]

    return if (start != null || end != null) {
        TemporalExtent(start = start, end = end)
    } else {
        null
    }
}

/**
 * Gets the versioning status based on which graph the entity is in.
 */
internal fun getStatusFromGraph(graphUri: String): String {
    return when (graphUri) {
        Config.GRAPH_DRAFT -> "DRAFT"
        Config.GRAPH_SUBMITTED -> "SUBMITTED"
        Config.GRAPH_PUBLISHED -> "PUBLISHED"
        Config.GRAPH_ARCHIVED -> "ARCHIVED"
        else -> "PUBLISHED"
    }
}
