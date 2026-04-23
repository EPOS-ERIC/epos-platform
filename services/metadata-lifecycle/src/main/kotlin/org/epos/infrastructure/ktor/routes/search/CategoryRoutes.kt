package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import java.net.URLDecoder

fun Route.categoryRoutes(graphDBClient: GraphDBClient) {
    // GET /api/v2/category - Full category tree
    get("/api/v2/category") {
        val graphUri = Config.GRAPH_PUBLISHED

        val query = """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
            PREFIX dcterms: <http://purl.org/dc/terms/>
            PREFIX foaf: <http://xmlns.com/foaf/0.1/>
            PREFIX epos: <https://www.epos-eu.org/epos-dcat-ap#>

            SELECT DISTINCT
                ?scheme ?schemeLabel ?schemeTitle ?schemeDesc ?schemeLogo ?schemeHomepage ?schemeColor ?schemeOrder ?schemeType
                ?concept ?conceptLabel ?conceptDesc ?broader
            FROM <$graphUri>
            WHERE {
                ?scheme a skos:ConceptScheme .
                OPTIONAL { ?scheme skos:prefLabel ?schemeLabel }
                OPTIONAL { ?scheme rdfs:label ?schemeLabelAlt }
                OPTIONAL { ?scheme dcterms:title ?schemeTitle }
                OPTIONAL { ?scheme dcterms:description ?schemeDesc }
                OPTIONAL { ?scheme foaf:logo ?schemeLogo }
                OPTIONAL { ?scheme foaf:homepage ?schemeHomepage }
                OPTIONAL { ?scheme epos:color ?schemeColor }
                OPTIONAL { ?scheme epos:orderNumber ?schemeOrder }
                OPTIONAL { ?scheme epos:type ?schemeType }

                OPTIONAL {
                    ?concept skos:inScheme ?scheme .
                    ?concept a skos:Concept .
                    OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
                    OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
                    OPTIONAL { ?concept skos:definition ?conceptDesc }
                    OPTIONAL { ?concept dcterms:description ?conceptDescAlt }
                    OPTIONAL { ?concept skos:broader ?broader }
                }
            }
            ORDER BY ?schemeOrder ?scheme ?broader ?conceptLabel
        """.trimIndent()

        try {
            val results = graphDBClient.query(query)
            val tree = buildCategoryTree(results)
            call.respond(tree)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch category tree"),
            )
        }
    }

    // GET /api/v2/category/{id} - Single category with subtree
    get("/api/v2/category/{id}") {
        val categoryId = call.parameters["id"]
        if (categoryId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing category ID"))
            return@get
        }

        val decodedId = try {
            URLDecoder.decode(categoryId, "UTF-8")
        } catch (_: Exception) {
            categoryId
        }

        val graphUri = Config.GRAPH_PUBLISHED

        // First check if it exists
        val existsQuery = """
            PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
            ASK {
                GRAPH <$graphUri> {
                    <$decodedId> a skos:Concept .
                }
            }
        """.trimIndent()

        if (!graphDBClient.ask(existsQuery)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Category not found: $decodedId"))
            return@get
        }

        // Query the category and its children recursively
        val query = """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
            PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
            PREFIX dcterms: <http://purl.org/dc/terms/>

            SELECT DISTINCT ?concept ?conceptLabel ?conceptDesc ?broader
            FROM <$graphUri>
            WHERE {
                {
                    BIND(<$decodedId> AS ?concept)
                    OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
                    OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
                    OPTIONAL { ?concept skos:definition ?conceptDesc }
                    OPTIONAL { ?concept dcterms:description ?conceptDescAlt }
                    OPTIONAL { ?concept skos:broader ?broader }
                }
                UNION
                {
                    ?concept skos:broader+ <$decodedId> .
                    OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
                    OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
                    OPTIONAL { ?concept skos:definition ?conceptDesc }
                    OPTIONAL { ?concept dcterms:description ?conceptDescAlt }
                    OPTIONAL { ?concept skos:broader ?broader }
                }
                UNION
                {
                    ?concept skos:broader <$decodedId> .
                    OPTIONAL { ?concept skos:prefLabel ?conceptLabel }
                    OPTIONAL { ?concept rdfs:label ?conceptLabelAlt }
                    OPTIONAL { ?concept skos:definition ?conceptDesc }
                    OPTIONAL { ?concept dcterms:description ?conceptDescAlt }
                    BIND(<$decodedId> AS ?broader)
                }
            }
            ORDER BY ?broader ?conceptLabel
        """.trimIndent()

        try {
            val results = graphDBClient.query(query)
            val category = buildSingleCategory(decodedId, results)
            if (category != null) {
                call.respond(category)
            } else {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Category not found: $decodedId"))
            }
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch category"),
            )
        }
    }
}

/**
 * Builds the full category tree from SPARQL results.
 */
private fun buildCategoryTree(results: List<Map<String, String>>): CategoryTreeResponse {
    // Group by scheme
    val schemeMap = linkedMapOf<String, MutableMap<String, Any?>>()
    val conceptsByScheme = linkedMapOf<String, MutableList<Map<String, String>>>()

    results.forEach { row ->
        val schemeUri = row["scheme"] ?: return@forEach

        // Build scheme info
        if (!schemeMap.containsKey(schemeUri)) {
            schemeMap[schemeUri] = mutableMapOf(
                "id" to schemeUri,
                "label" to (row["schemeLabel"] ?: row["schemeLabelAlt"] ?: extractTypeName(schemeUri)),
                "title" to row["schemeTitle"],
                "description" to row["schemeDesc"],
                "logo" to row["schemeLogo"],
                "homepage" to row["schemeHomepage"],
                "color" to row["schemeColor"],
                "orderNumber" to row["schemeOrder"]?.toIntOrNull(),
                "type" to (row["schemeType"] ?: "DATA"),
            )
        }

        // Collect concepts for this scheme
        val conceptUri = row["concept"]
        if (!conceptUri.isNullOrBlank()) {
            conceptsByScheme.getOrPut(schemeUri) { mutableListOf() }.add(row)
        }
    }

    // Build scheme objects with their category trees
    val schemes = schemeMap.map { (schemeUri, info) ->
        val concepts = conceptsByScheme[schemeUri] ?: emptyList()
        val children = buildCategoryHierarchy(concepts, null)

        CategoryScheme(
            id = info["id"] as String,
            label = info["label"] as String,
            title = info["title"] as? String,
            description = info["description"] as? String,
            logo = info["logo"] as? String,
            homepage = info["homepage"] as? String,
            color = info["color"] as? String,
            orderNumber = info["orderNumber"] as? Int,
            type = info["type"] as String,
            children = children.takeIf { it.isNotEmpty() },
        )
    }.sortedBy { it.orderNumber ?: Int.MAX_VALUE }

    return CategoryTreeResponse(schemes = schemes)
}

/**
 * Builds a hierarchical category list from flat SPARQL results.
 * @param concepts List of concept rows from SPARQL
 * @param parentUri The parent concept URI (null for top-level)
 */
private fun buildCategoryHierarchy(
    concepts: List<Map<String, String>>,
    parentUri: String?,
): List<Category> {
    // Find concepts at this level (those whose broader == parentUri, or no broader for top-level)
    val atThisLevel = concepts.filter { row ->
        val conceptUri = row["concept"] ?: return@filter false
        val broader = row["broader"]
        if (parentUri == null) {
            broader.isNullOrBlank()
        } else {
            broader == parentUri
        }
    }

    return atThisLevel.mapNotNull { row ->
        val conceptUri = row["concept"] ?: return@mapNotNull null
        val children = buildCategoryHierarchy(concepts, conceptUri)

        Category(
            id = conceptUri,
            name = row["conceptLabel"] ?: row["conceptLabelAlt"] ?: extractTypeName(conceptUri),
            description = row["conceptDesc"] ?: row["conceptDescAlt"],
            children = children.takeIf { it.isNotEmpty() },
        )
    }.distinctBy { it.id }
}

/**
 * Builds a single category with its subtree.
 */
private fun buildSingleCategory(
    rootId: String,
    results: List<Map<String, String>>,
): Category? {
    val rootRow = results.find { it["concept"] == rootId } ?: return null

    val children = buildCategoryHierarchy(results, rootId)

    return Category(
        id = rootId,
        name = rootRow["conceptLabel"] ?: rootRow["conceptLabelAlt"] ?: extractTypeName(rootId),
        description = rootRow["conceptDesc"] ?: rootRow["conceptDescAlt"],
        children = children.takeIf { it.isNotEmpty() },
    )
}
