package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.infrastructure.ktor.routes.Category
import org.epos.infrastructure.ktor.routes.CategoryScheme
import org.epos.infrastructure.ktor.routes.CategoryTreeResponse
import org.epos.infrastructure.ktor.routes.extractTypeName
import org.epos.propertygraph.client.PropertyGraphClient
import java.net.URLDecoder

fun Route.categoryRoutes(client: PropertyGraphClient) {
    get("/api/v2/category") {
        try {
            val results = client.runQuery("g.V().hasLabel('Version').has('status','published').valueMap(true).toList()")

            val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
            val dataResults = results.map { row ->
                val v = row["value"] as? Map<*, *> ?: emptyMap<String, Any?>()
                v.filterKeys { it is String && it !in structuralKeys }
                    .mapKeys { (k, _) -> k as String }
                    .mapValues { (_, v2) -> prop(v2) }
            }
            val tree = buildCategoryTree(dataResults)
            call.respond(tree)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch category tree"),
            )
        }
    }

    get("/api/v2/category/{id}") {
        val categoryId = call.parameters["id"]
        if (categoryId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing category ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(categoryId, "UTF-8")
            } catch (_: Exception) {
                categoryId
            }

        val exists = client.exists("g.V().hasLabel('Version').has('id', ${gStr(decodedId)}).limit(1).toList()")

        if (!exists) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Category not found: $decodedId"))
            return@get
        }

        try {
            val results = client.runQuery("g.V().hasLabel('Version').has('status','published').valueMap(true).toList()")

            val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
            val dataResults = results.map { row ->
                val v = row["value"] as? Map<*, *> ?: emptyMap<String, Any?>()
                v.filterKeys { it is String && it !in structuralKeys }
                    .mapKeys { (k, _) -> k as String }
                    .mapValues { (_, v2) -> prop(v2) }
            }
            val category = buildSingleCategory(decodedId, dataResults)
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

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun prop(value: Any?): Any? = when (value) {
    is List<*> -> if (value.size == 1) value.firstOrNull() else value
    else -> value
}

private fun buildCategoryTree(results: List<Map<String, Any?>>): CategoryTreeResponse {
    val schemeMap = linkedMapOf<String, MutableMap<String, Any?>>()
    val conceptsByScheme = linkedMapOf<String, MutableList<Map<String, Any?>>>()

    results.forEach { row ->
        val data = row["data"] as? Map<*, *> ?: return@forEach
        val schemeUri = data["inScheme"] as? String ?: data["skos:inScheme"] as? String ?: return@forEach

        if (!schemeMap.containsKey(schemeUri)) {
            schemeMap[schemeUri] = mutableMapOf(
                "id" to schemeUri,
                "label" to (data["schemeLabel"] as? String ?: data["label"] as? String ?: extractTypeName(schemeUri)),
                "title" to data["schemeTitle"] as? String,
                "description" to data["schemeDescription"] as? String,
                "type" to (data["schemeType"] as? String ?: "DATA"),
            )
        }

        val conceptUri = data["id"] as? String
        if (!conceptUri.isNullOrBlank()) {
            conceptsByScheme.getOrPut(schemeUri) { mutableListOf() }.add(row)
        }
    }

    val schemes = schemeMap.map { (schemeUri, info) ->
        val concepts = conceptsByScheme[schemeUri] ?: emptyList()
        val children = buildCategoryHierarchy(concepts, null)

        CategoryScheme(
            id = info["id"] as String,
            label = info["label"] as String,
            title = info["title"] as? String,
            description = info["description"] as? String,
            logo = null,
            homepage = null,
            color = null,
            orderNumber = null,
            type = info["type"] as String,
            children = children.takeIf { it.isNotEmpty() },
        )
    }.sortedBy { it.orderNumber ?: Int.MAX_VALUE }

    return CategoryTreeResponse(schemes = schemes)
}

private fun buildCategoryHierarchy(
    concepts: List<Map<String, Any?>>,
    parentUri: String?,
): List<Category> {
    val atThisLevel = concepts.filter { row ->
        val data = row["data"] as? Map<*, *> ?: return@filter false
        val conceptUri = data["id"] as? String ?: return@filter false
        val broader = data["broader"] as? String ?: data["skos:broader"] as? String
        if (parentUri == null) {
            broader.isNullOrBlank()
        } else {
            broader == parentUri
        }
    }

    return atThisLevel.mapNotNull { row ->
        val data = row["data"] as? Map<*, *> ?: return@mapNotNull null
        val conceptUri = data["id"] as? String ?: return@mapNotNull null
        val children = buildCategoryHierarchy(concepts, conceptUri)

        Category(
            id = conceptUri,
            name = data["prefLabel"] as? String ?: data["label"] as? String ?: extractTypeName(conceptUri),
            description = data["definition"] as? String ?: data["description"] as? String,
            children = children.takeIf { it.isNotEmpty() },
        )
    }.distinctBy { it.id }
}

private fun buildSingleCategory(
    rootId: String,
    results: List<Map<String, Any?>>,
): Category? {
    val rootRow = results.find {
        val data = it["data"] as? Map<*, *>
        data?.get("id") == rootId
    } ?: return null

    val rootData = rootRow["data"] as? Map<*, *> ?: return null

    val children = buildCategoryHierarchy(results, rootId)

    return Category(
        id = rootId,
        name = rootData["prefLabel"] as? String ?: rootData["label"] as? String ?: extractTypeName(rootId),
        description = rootData["definition"] as? String ?: rootData["description"] as? String,
        children = children.takeIf { it.isNotEmpty() },
    )
}
