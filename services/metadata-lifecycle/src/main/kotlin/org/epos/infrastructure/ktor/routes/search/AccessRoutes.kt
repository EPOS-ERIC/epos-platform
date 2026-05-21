package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.shacl.PropertyInfo
import org.epos.propertygraph.Config
import org.epos.infrastructure.ktor.routes.*
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.services.ShaclService
import com.fasterxml.jackson.databind.ObjectMapper
import java.net.URLDecoder

fun Route.accessRoutes(
    client: PropertyGraphClient,
    shaclService: ShaclService,
) {
    get("/api/v2/access/{shape}/{id}") {
        val shape = call.parameters["shape"]?.trim().orEmpty()
        val idParam = call.parameters["id"]?.trim().orEmpty()
        if (shape.isBlank() || idParam.isBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing shape or id"))
            return@get
        }

        val resourceTypes = shaclService.getResourceTypes()
        val resourceType = resolveResourceType(shape, resourceTypes)
        val resolvedShapeName = resourceType?.name ?: shape

        if (idParam.equals("all", ignoreCase = true)) {
            call.respond(buildAccessList(client, resolvedShapeName, resourceTypes, call.resolvePublicBaseUrl()))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(idParam, "UTF-8")
            } catch (_: Exception) {
                idParam
            }

        val resolvedEntityId = ResourceLinkResolver.resolveAccessEntityId(client, decodedId)
        if (resolvedEntityId == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resource not found: $decodedId"))
            return@get
        }

        if (!shapeMatchesEntityType(client, resolvedShapeName, resolvedEntityId, resourceTypes)) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Resource not found: $decodedId"))
            return@get
        }

        call.respond(
            buildAccessDetail(
                client = client,
                resourceTypeMap = resourceTypes.associate { it.targetClass to it.name },
                knownShapeNames = resourceTypes.map { it.name }.toSet(),
                shapeProperties = resourceType?.properties.orEmpty(),
                shape = resolvedShapeName,
                entityId = resolvedEntityId,
                requestedId = decodedId,
                publicBaseUrl = call.resolvePublicBaseUrl(),
            ),
        )
    }
}

private fun resolveResourceType(
    shape: String,
    resourceTypes: List<ResourceTypeInfo>,
) = resourceTypes.find { normalizeShapeName(it.name) == normalizeShapeName(shape) }

private fun normalizeShapeName(typeName: String): String =
    typeName
        .lowercase()
        .replace("_", "")
        .replace("-", "")
        .replace(" ", "")

private fun buildAccessList(
    client: PropertyGraphClient,
    shape: String,
    resourceTypes: List<ResourceTypeInfo>,
    publicBaseUrl: String,
): AccessListResponse {
    val candidateTypes = typeCandidatesForShape(shape, resourceTypes).ifEmpty { listOf(shape.lowercase()) }
    val typeList = candidateTypes.joinToString(",") { gStr(it) }
    val results = client.runQuery("""
        g.V().hasLabel('Conceptual').has('type', within($typeList)).as('c').out('CURRENT_VERSION').hasLabel('Version').as('v')
          .select('c','v')
          .by(project('id','type')
            .by(coalesce(values('id'), constant('')))
            .by(coalesce(values('type'), constant(''))))
          .by(project('title')
            .by(coalesce(values('title'), constant(''))))
          .toList()
    """.trimIndent())

    val items = results.mapNotNull { row ->
        val c = row["c"] as? Map<*, *> ?: return@mapNotNull null
        val v = row["v"] as? Map<*, *> ?: return@mapNotNull null
        val id = prop(c["id"]) ?: return@mapNotNull null
        val conceptualType = prop(c["type"]) ?: ""
        val conceptualShape = resolveShapeNameFromConceptualType(conceptualType, resourceTypes)
        if (normalizeShapeName(conceptualShape) != normalizeShapeName(shape)) return@mapNotNull null
        AccessListItem(
            id = id,
            type = shape,
            title = prop(v["title"]),
            href = "$publicBaseUrl/api/v2/access/$shape/${java.net.URLEncoder.encode(id, "UTF-8")}",
        )
    }

    return AccessListResponse(shape = shape, total = items.size, items = items)
}

private fun buildAccessDetail(
    client: PropertyGraphClient,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
    shapeProperties: List<PropertyInfo>,
    shape: String,
    entityId: String,
    requestedId: String,
    publicBaseUrl: String,
): AccessDetailResponse {
    val result =
        client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityId)}).valueMap().limit(1).toList()").firstOrNull()
            ?: client.runQuery("g.V().hasLabel('Version').has('identifier', ${gStr(entityId)}).valueMap().limit(1).toList()").firstOrNull()
            ?: client.runQuery("g.V().has('id', ${gStr(entityId)}).valueMap().limit(1).toList()").firstOrNull()

    val rawMap = (result?.get("value") as? Map<*, *>) ?: (result as? Map<*, *>) ?: emptyMap<String, Any?>()
    val v = rawMap
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
    val groupedProperties = linkedMapOf<String, MutableList<AccessPropertyValue>>()
    val propertyWasArray = mutableMapOf<String, Boolean>()
    val existenceCache = mutableMapOf<String, Boolean>()

    data.forEach { (key, value) ->
        val propName = key
        val values = expandAccessValues(value)
        propertyWasArray[propName] = value is List<*>

        values.forEach { item ->
            val obj = item?.toString() ?: return@forEach
            val predicateIri = resolvePredicateIriFromShacl(propName, shapeProperties)
            val objectType =
                resolveObjectShape(
                    objectId = obj,
                    propertyName = propName,
                    shapeProperties = shapeProperties,
                    resourceTypeMap = resourceTypeMap,
                )?.takeIf { candidate -> knownShapeNames.any { normalizeShapeName(it) == normalizeShapeName(candidate) } }

            val accessValue =
                if (ResourceLinkResolver.isPotentialResourceIdentifier(obj)) {
                    if (obj == requestedId) {
                        val plainSelfValue = AccessPropertyValue(value = obj)
                        groupedProperties.getOrPut(propName) { mutableListOf() }.add(plainSelfValue)
                        return@forEach
                    }
                    val resolvedLink =
                        if (isResolvableResource(client, obj, existenceCache)) {
                            ResourceLinkResolver.resolveLink(
                                client = client,
                                objectId = obj,
                                resourceTypeMap = resourceTypeMap,
                                normalizeShapeName = ::normalizeShapeName,
                                publicBaseUrl = publicBaseUrl,
                                preferredShape = objectType,
                            )
                        } else {
                            null
                        }
                    val href = resolvedLink?.href
                    val resolvedType = objectType ?: resolvedLink?.resolvedShape

                    if (!resolvedType.isNullOrBlank() || href != null) {
                        AccessPropertyValue(
                            id = obj,
                            type = resolvedType,
                            title = null,
                            href = href,
                            predicateIri = predicateIri,
                        )
                    } else {
                        AccessPropertyValue(value = obj, predicateIri = predicateIri)
                    }
                } else {
                    AccessPropertyValue(value = obj)
                }
            groupedProperties.getOrPut(propName) { mutableListOf() }.add(accessValue)
        }
    }

    val relationRows =
        client.runQuery(
            """
            g.V().hasLabel('Version').has('id', ${gStr(entityId)}).as('s')
              .outE().as('e').inV().as('t')
              .select('e','t')
              .by(project('label','predicateIri','predicateKey')
                .by(label)
                .by(coalesce(values('predicateIri'), constant('')))
                .by(coalesce(values('predicateKey'), constant(''))))
              .by(project('identifier','id')
                .by(coalesce(values('identifier'), constant('')))
                .by(coalesce(values('id'), constant(''))))
              .toList()
            """.trimIndent(),
        ).ifEmpty {
            client.runQuery(
                """
                g.V().hasLabel('Version').has('identifier', ${gStr(entityId)}).as('s')
                  .outE().as('e').inV().as('t')
                  .select('e','t')
                  .by(project('label','predicateIri','predicateKey')
                    .by(label)
                    .by(coalesce(values('predicateIri'), constant('')))
                    .by(coalesce(values('predicateKey'), constant(''))))
                  .by(project('identifier','id')
                    .by(coalesce(values('identifier'), constant('')))
                    .by(coalesce(values('id'), constant(''))))
                  .toList()
                """.trimIndent(),
            )
        }

    relationRows.forEach { row ->
        val edge = row["e"] as? Map<*, *> ?: return@forEach
        val edgeLabel = prop(edge["label"]) ?: return@forEach
        if (!edgeLabel.startsWith("rel_")) return@forEach

        val propertyName = prop(edge["predicateKey"]) ?: edgeLabel.removePrefix("rel_")
        val predicateIri = prop(edge["predicateIri"])
        val target = row["t"] as? Map<*, *> ?: return@forEach
        val targetId =
            (prop(target["identifier"])?.takeIf { it.isNotBlank() }
                ?: prop(target["id"])?.takeIf { it.isNotBlank() })
                ?: return@forEach

        if (targetId == requestedId) return@forEach

        val objectType =
            resolveObjectShape(
                objectId = targetId,
                propertyName = propertyName,
                shapeProperties = shapeProperties,
                resourceTypeMap = resourceTypeMap,
            )?.takeIf { candidate -> knownShapeNames.any { normalizeShapeName(it) == normalizeShapeName(candidate) } }

        val resolvedLink =
            if (isResolvableResource(client, targetId, existenceCache)) {
                ResourceLinkResolver.resolveLink(
                    client = client,
                    objectId = targetId,
                    resourceTypeMap = resourceTypeMap,
                    normalizeShapeName = ::normalizeShapeName,
                    publicBaseUrl = publicBaseUrl,
                    preferredShape = objectType,
                )
            } else {
                null
            }

        val href = resolvedLink?.href
        val resolvedType = objectType ?: resolvedLink?.resolvedShape

        val relationValue =
            AccessPropertyValue(
                id = targetId,
                type = resolvedType,
                title = null,
                href = href,
                predicateIri = predicateIri,
            )

        val bucket = groupedProperties.getOrPut(propertyName) { mutableListOf() }
        val alreadyPresent = bucket.any { it.id == relationValue.id && it.type == relationValue.type && it.href == relationValue.href }
        if (!alreadyPresent) {
            bucket.add(relationValue)
        }
    }

    val normalizedProperties = groupedProperties.mapValues { (property, values) ->
        normalizeAccessPropertyValues(values, forceArray = propertyWasArray[property] == true)
    }

    return AccessDetailResponse(
        id = entityId,
        shape = shape,
        href = "$publicBaseUrl/api/v2/access/$shape/${java.net.URLEncoder.encode(entityId, "UTF-8")}",
        properties = normalizedProperties,
    )
}

private fun normalizeAccessPropertyValues(values: List<AccessPropertyValue>, forceArray: Boolean = false): Any {
    if (values.isEmpty()) return emptyList<String>()

    val allPlainValues = values.all { it.value != null && it.id == null && it.type == null && it.title == null && it.href == null }
    if (allPlainValues) {
        val plainValues = values.mapNotNull { it.value }
        if (forceArray) return plainValues
        return if (plainValues.size == 1) plainValues.first() else plainValues
    }

    if (forceArray) return values
    return if (values.size == 1) values.first() else values
}

private fun isResolvableResource(
    client: PropertyGraphClient,
    objectId: String,
    existenceCache: MutableMap<String, Boolean>,
): Boolean =
    existenceCache.getOrPut(objectId) {
        ResourceLinkResolver.resolveAccessEntityId(client, objectId) != null
    }

private fun resolveObjectShape(
    objectId: String,
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
    resourceTypeMap: Map<String, String>,
): String? {
    val fromShacl = resolveObjectShapeFromShacl(propertyName, shapeProperties, resourceTypeMap)
    if (!fromShacl.isNullOrBlank()) return fromShacl

    if (Config.RELATION_STRICT_MODE) return null

    val normalizedProperty = propertyName.lowercase()
    when (normalizedProperty) {
        "identifier" -> return "identifier"
        "publisher", "provider" -> return "organization"
        "theme" -> return "category"
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

private fun resolveObjectShapeFromShacl(
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
    resourceTypeMap: Map<String, String>,
): String? {
    val normalizedProperty = normalizeShapeName(propertyName)
    val property = shapeProperties.firstOrNull { prop ->
        normalizeShapeName(prop.name) == normalizedProperty ||
            normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#')) == normalizedProperty
    } ?: return null

    val candidates = mutableListOf<String>()
    property.classConstraint?.let { candidates.add(it) }
    property.classConstraints?.let { candidates.addAll(it) }

    return candidates
        .firstNotNullOfOrNull { classIri ->
            resourceTypeMap[classIri]
                ?: resourceTypeMap.entries.firstOrNull {
                    normalizeShapeName(it.key.substringAfterLast('/').substringAfterLast('#')) ==
                        normalizeShapeName(classIri.substringAfterLast('/').substringAfterLast('#'))
                }?.value
        }
}

private fun resolvePredicateIriFromShacl(
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
): String? {
    val normalizedProperty = normalizeShapeName(propertyName)
    val property = shapeProperties.firstOrNull { prop ->
        normalizeShapeName(prop.name) == normalizedProperty ||
            normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#')) == normalizedProperty
    } ?: return null

    return property.path.takeIf { it.startsWith("http://") || it.startsWith("https://") }
}

private fun typeCandidatesForShape(shape: String, resourceTypes: List<ResourceTypeInfo>): Set<String> {
    val normalizedShape = normalizeShapeName(shape)
    val resourceType = resourceTypes.firstOrNull { normalizeShapeName(it.name) == normalizedShape } ?: return emptySet()

    val candidates = linkedSetOf<String>()
    candidates.add(resourceType.name.lowercase())

    val targetClass = resourceType.targetClass
    val localTargetClass = targetClass.substringAfterLast('/').substringAfterLast('#').lowercase()
    if (localTargetClass.isNotBlank()) candidates.add(localTargetClass)
    if (targetClass.isNotBlank()) candidates.add(targetClass.lowercase())

    return candidates
}

private fun shapeMatchesEntityType(
    client: PropertyGraphClient,
    requestedShape: String,
    entityId: String,
    resourceTypes: List<ResourceTypeInfo>,
): Boolean {
    val conceptualType =
        client.runQuery(
            """
            g.V().hasLabel('Version').has('id', ${gStr(entityId)}).in('HAS_VERSION').hasLabel('Conceptual')
              .values('type').limit(1).toList()
            """.trimIndent(),
        ).firstOrNull()?.get("value")?.toString()
            ?: client.runQuery(
                """
                g.V().hasLabel('Version').has('identifier', ${gStr(entityId)}).in('HAS_VERSION').hasLabel('Conceptual')
                  .values('type').limit(1).toList()
                """.trimIndent(),
            ).firstOrNull()?.get("value")?.toString()
            ?: return false

    val resolvedShape = resolveShapeNameFromConceptualType(conceptualType, resourceTypes)
    return normalizeShapeName(resolvedShape) == normalizeShapeName(requestedShape)
}

private fun resolveShapeNameFromConceptualType(
    conceptualType: String,
    resourceTypes: List<ResourceTypeInfo>,
): String {
    val normalizedType = normalizeShapeName(conceptualType.substringAfterLast('/').substringAfterLast('#'))
    return resourceTypes
        .firstOrNull { typeInfo ->
            normalizeShapeName(typeInfo.name) == normalizedType ||
                normalizeShapeName(typeInfo.targetClass.substringAfterLast('/').substringAfterLast('#')) == normalizedType
        }
        ?.name
        ?: conceptualType
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
private fun prop(value: Any?): String? = when (value) {
    is List<*> -> value.firstOrNull()?.toString()
    else -> value?.toString()
}

private val accessJsonMapper = ObjectMapper()

private fun expandAccessValues(value: Any?): List<Any?> {
    val normalized = normalizeAccessValue(value)
    return when (normalized) {
        is List<*> -> normalized.flatMap { item -> expandAccessValues(item) }
        else -> listOf(normalized)
    }
}

private fun normalizeAccessValue(value: Any?): Any? {
    return when (value) {
        is String -> {
            val trimmed = value.trim()
            if ((trimmed.startsWith("[") && trimmed.endsWith("]")) || (trimmed.startsWith("{") && trimmed.endsWith("}"))) {
                runCatching { accessJsonMapper.readValue(trimmed, Any::class.java) }.getOrDefault(value)
            } else {
                value
            }
        }
        else -> value
    }
}
