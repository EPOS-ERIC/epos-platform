package org.epos.infrastructure.ktor.routes

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.services.ShaclService
import java.net.URLDecoder
import java.net.URLEncoder

fun Route.accessRoutes(
    graphDBClient: GraphDBClient,
    shaclService: ShaclService,
) {
    get("/api/v2/access/{shape}/{id}") {
        val shape = call.parameters["shape"]?.trim().orEmpty()
        val idParam = call.parameters["id"]?.trim().orEmpty()
        if (shape.isBlank() || idParam.isBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing shape or id"))
            return@get
        }

        val resourceType = resolveResourceType(shape, shaclService)
        if (resourceType == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_TYPE_NOT_FOUND", "Unknown shape: $shape"))
            return@get
        }

        if (idParam.equals("all", ignoreCase = true)) {
            call.respond(buildAccessList(graphDBClient, resourceType.name, resourceType.targetClass, call.resolvePublicBaseUrl()))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(idParam, "UTF-8")
            } catch (_: Exception) {
                idParam
            }

        if (!entityExists(graphDBClient, decodedId, Config.GRAPH_PUBLISHED)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resource not found: $decodedId"))
            return@get
        }

        call.respond(
            buildAccessDetail(
                graphDBClient = graphDBClient,
                resourceTypeMap = shaclService.getResourceTypes().associate { it.targetClass to it.name },
                knownShapeNames = shaclService.getResourceTypes().map { it.name }.toSet(),
                shape = resourceType.name,
                entityId = decodedId,
                graphUri = Config.GRAPH_PUBLISHED,
                publicBaseUrl = call.resolvePublicBaseUrl(),
            ),
        )
    }
}

private fun resolveResourceType(shape: String, shaclService: ShaclService) =
    shaclService.getResourceTypes().find { normalizeShapeName(it.name) == normalizeShapeName(shape) }

private fun normalizeShapeName(typeName: String): String =
    typeName
        .lowercase()
        .replace("_", "")
        .replace("-", "")
        .replace(" ", "")

private fun buildAccessList(
    graphDBClient: GraphDBClient,
    shape: String,
    targetClass: String,
    publicBaseUrl: String,
): AccessListResponse {
    val query =
        """
        $DETAIL_PREFIXES
        SELECT DISTINCT ?s ?title
        FROM <${Config.GRAPH_PUBLISHED}>
        WHERE {
            ?s rdf:type <$targetClass> .
            OPTIONAL { ?s dcterms:title|rdfs:label|schema:name|foaf:name|skos:prefLabel ?title . }
        }
        ORDER BY ?s
        """.trimIndent()

    val items =
        graphDBClient.query(query).mapNotNull { row ->
            val id = row["s"] ?: return@mapNotNull null
            AccessListItem(
                id = id,
                type = shape,
                title = row["title"],
                href = "$publicBaseUrl/api/v2/access/$shape/${URLEncoder.encode(id, "UTF-8")}",
            )
        }

    return AccessListResponse(shape = shape, total = items.size, items = items)
}

private fun buildAccessDetail(
    graphDBClient: GraphDBClient,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
    shape: String,
    entityId: String,
    graphUri: String,
    publicBaseUrl: String,
): AccessDetailResponse {
    val escapedEntityId = sparqlStringLiteral(entityId)
    val query =
        """
        $DETAIL_PREFIXES
        SELECT ?predicate ?object ?objectType ?objectTitle
        FROM <$graphUri>
        WHERE {
            ?subject ?predicate ?object .
            FILTER(STR(?subject) = "$escapedEntityId")
            OPTIONAL { ?object rdf:type ?objectType . }
            OPTIONAL { ?object dcterms:title|rdfs:label|schema:name|foaf:name|skos:prefLabel ?objectTitle . }
        }
        ORDER BY ?predicate ?object
        """.trimIndent()

    val groupedProperties = linkedMapOf<String, MutableList<AccessPropertyValue>>()
    val existenceCache = mutableMapOf<String, Boolean>()

    graphDBClient.query(query).forEach { row ->
        val predicate = row["predicate"] ?: return@forEach
        val obj = row["object"] ?: return@forEach
        val propName = extractTypeName(predicate)
        val objectType =
            resolveObjectShape(
                objectTypeUri = row["objectType"],
                objectId = obj,
                propertyName = propName,
                resourceTypeMap = resourceTypeMap,
            )?.takeIf { knownShapeNames.contains(it) }

        val value =
            if (obj.startsWith("http://") || obj.startsWith("https://") || obj.startsWith("category:")) {
                val href =
                    if (isResolvableResource(graphDBClient, graphUri, obj, existenceCache)) {
                        buildNavigableAccessHref(publicBaseUrl, objectType, obj)
                    } else {
                        null
                    }
                if (href != null) {
                    AccessPropertyValue(
                        id = obj,
                        type = objectType,
                        title = row["objectTitle"],
                        href = href,
                    )
                } else {
                    AccessPropertyValue(value = obj)
                }
            } else {
                AccessPropertyValue(value = obj)
            }
        groupedProperties.getOrPut(propName) { mutableListOf() }.add(value)
    }

    val normalizedProperties = groupedProperties.mapValues { (_, values) -> normalizeAccessPropertyValues(values) }

    return AccessDetailResponse(
        id = entityId,
        shape = shape,
        href = "$publicBaseUrl/api/v2/access/$shape/${URLEncoder.encode(entityId, "UTF-8")}",
        properties = normalizedProperties,
    )
}

private fun normalizeAccessPropertyValues(values: List<AccessPropertyValue>): Any {
    if (values.isEmpty()) return emptyList<String>()

    val allPlainValues = values.all { it.value != null && it.id == null && it.type == null && it.title == null && it.href == null }
    if (allPlainValues) {
        val plainValues = values.mapNotNull { it.value }
        return if (plainValues.size == 1) plainValues.first() else plainValues
    }

    return if (values.size == 1) values.first() else values
}

private fun buildNavigableAccessHref(publicBaseUrl: String, objectType: String?, objectId: String): String? {
    if (objectType.isNullOrBlank()) return null
    return "$publicBaseUrl/api/v2/access/$objectType/${URLEncoder.encode(objectId, "UTF-8")}"
}

private fun isResolvableResource(
    graphDBClient: GraphDBClient,
    graphUri: String,
    objectId: String,
    existenceCache: MutableMap<String, Boolean>,
): Boolean {
    return existenceCache.getOrPut(objectId) {
        entityExists(graphDBClient, objectId, graphUri)
    }
}

private fun resolveObjectShape(
    objectTypeUri: String?,
    objectId: String,
    propertyName: String,
    resourceTypeMap: Map<String, String>,
): String? {
    if (!objectTypeUri.isNullOrBlank()) {
        resourceTypeMap[objectTypeUri]?.let { return it }
        mapTypeUriToApiType(objectTypeUri)?.let { return it }
    }

    val normalizedProperty = propertyName.lowercase()
    when (normalizedProperty) {
        "identifier" -> return "identifier"
        "publisher", "provider" -> return "organization"
        "spatial" -> return "location"
        "temporal" -> return "periodoftime"
        "contactpoint" -> return "contactpoint"
        "hasqualityannotation" -> return "qualityannotation"
    }

    val lowered = objectId.lowercase()
    return when {
        lowered.contains("/distribution/") -> "distribution"
        lowered.contains("/dataset/") -> "dataset"
        lowered.contains("softwareapplication") -> "softwareapplication"
        lowered.contains("softwaresourcecode") -> "softwaresourcecode"
        lowered.contains("bnode-identifier") || lowered.contains("/identifier/") -> "identifier"
        lowered.contains("bnode-location") || lowered.contains("/location/") -> "location"
        lowered.contains("bnode-periodoftime") || lowered.contains("periodoftime") -> "periodoftime"
        lowered.contains("annotation") -> "qualityannotation"
        else -> null
    }
}
