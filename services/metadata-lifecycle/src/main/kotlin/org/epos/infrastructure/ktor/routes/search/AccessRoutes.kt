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
        val canonicalShape = canonicalAccessShape(shape)
        val idParam = call.parameters["id"]?.trim().orEmpty()
        if (canonicalShape.isBlank() || idParam.isBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing shape or id"))
            return@get
        }

        val resourceTypes = shaclService.getResourceTypes()
        val resourceType = resolveResourceType(canonicalShape, resourceTypes)
        val resolvedShapeName = resourceType?.name ?: canonicalShape

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
                shaclPrefixes = shaclService.getParser().getCachedShapesGraph()?.prefixes.orEmpty(),
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

private fun canonicalAccessShape(shape: String): String =
    when (normalizeShapeName(shape)) {
        "publisher", "provider", "organization" -> "organization"
        else -> shape
    }

private fun buildAccessList(
    client: PropertyGraphClient,
    shape: String,
    resourceTypes: List<ResourceTypeInfo>,
    publicBaseUrl: String,
): AccessListResponse {
    val candidateTypes = typeCandidatesForShape(shape, resourceTypes).ifEmpty { listOf(shape.lowercase()) }
    val results = client.runQuery(
        """
        MATCH (c:Conceptual)-[:CURRENT_VERSION]->(v:Version)
        WHERE toLower(c.type) IN ${'$'}typeList
        RETURN {id: c.id, type: c.type} AS c, {title: coalesce(v.title, '')} AS v
        """.trimIndent(),
        mapOf("typeList" to candidateTypes.map { it.lowercase() }),
    )

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
    shaclPrefixes: Map<String, String>,
    shape: String,
    entityId: String,
    requestedId: String,
    publicBaseUrl: String,
): AccessDetailResponse {
    // Load version payload with progressively looser id/identifier fallbacks.
    val result =
        client.runQuery(
            "MATCH (v:Version {id: ${'$'}entityId}) RETURN properties(v) AS value LIMIT 1",
            mapOf("entityId" to entityId),
        ).firstOrNull()
            ?: client.runQuery(
                "MATCH (v:Version {identifier: ${'$'}entityId}) RETURN properties(v) AS value LIMIT 1",
                mapOf("entityId" to entityId),
            ).firstOrNull()
            ?: client.runQuery(
                "MATCH (v:Version) WHERE v.id = ${'$'}entityId RETURN properties(v) AS value LIMIT 1",
                mapOf("entityId" to entityId),
            ).firstOrNull()

    val rawMap = (result?.get("value") as? Map<*, *>) ?: (result as? Map<*, *>) ?: emptyMap<String, Any?>()
    val v = rawMap
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
    val groupedProperties = linkedMapOf<String, MutableList<AccessPropertyValue>>()
    val propertyWasArray = mutableMapOf<String, Boolean>()
    val existenceCache = mutableMapOf<String, Boolean>()

    data.forEach { (key, value) ->
        val propName = canonicalPropertyKeyFromShacl(key, shapeProperties, shaclPrefixes)
        if (isStructuralProperty(propName)) return@forEach
        val relationByShacl = isRelationPropertyByShacl(propName, shapeProperties)
        val values = expandAccessValues(value)
        propertyWasArray[propName] = value is List<*> || shouldBeArrayByShacl(propName, shapeProperties)

        values.forEach { item ->
            val obj = item?.toString() ?: return@forEach
            if (!relationByShacl) {
                groupedProperties.getOrPut(propName) { mutableListOf() }.add(AccessPropertyValue(value = obj))
                return@forEach
            }
            val predicateIri = resolvePredicateIriFromShacl(propName, shapeProperties)
            val objectType = resolveKnownObjectShape(obj, propName, shapeProperties, resourceTypeMap, knownShapeNames)
            val accessValue = buildAccessPropertyValue(
                client = client,
                objectId = obj,
                requestedId = requestedId,
                predicateIri = predicateIri,
                objectType = objectType,
                resourceTypeMap = resourceTypeMap,
                publicBaseUrl = publicBaseUrl,
                existenceCache = existenceCache,
            )
            groupedProperties.getOrPut(propName) { mutableListOf() }.add(accessValue)
        }
    }

    val relationRows =
        client.runQuery(
            """
            MATCH (s:Version {id: ${'$'}entityId})-[e]->(t:Version)
            RETURN {
              label: type(e),
              predicateIri: coalesce(e.predicateIri, ''),
              predicateKey: coalesce(e.predicateKey, '')
            } AS e,
            {
              identifier: coalesce(t.identifier, ''),
              id: coalesce(t.id, '')
            } AS t
            """.trimIndent(),
            mapOf("entityId" to entityId),
        ).ifEmpty {
            client.runQuery(
                """
                MATCH (s:Version {identifier: ${'$'}entityId})-[e]->(t:Version)
                RETURN {
                  label: type(e),
                  predicateIri: coalesce(e.predicateIri, ''),
                  predicateKey: coalesce(e.predicateKey, '')
                } AS e,
                {
                  identifier: coalesce(t.identifier, ''),
                  id: coalesce(t.id, '')
                } AS t
                """.trimIndent(),
                mapOf("entityId" to entityId),
            )
        }

    relationRows.forEach { row ->
        val edge = row["e"] as? Map<*, *> ?: return@forEach
        val edgeLabel = prop(edge["label"]) ?: return@forEach
        if (!edgeLabel.startsWith("rel_")) return@forEach

        val rawPropertyName = prop(edge["predicateKey"]) ?: edgeLabel.removePrefix("rel_")
        val propertyName = canonicalPropertyKeyFromShacl(rawPropertyName, shapeProperties, shaclPrefixes)
        if (shouldBeArrayByShacl(propertyName, shapeProperties)) {
            propertyWasArray[propertyName] = true
        }
        val predicateIri = prop(edge["predicateIri"])
        val target = row["t"] as? Map<*, *> ?: return@forEach
        val targetId =
            (prop(target["id"])?.takeIf { it.isNotBlank() }
                ?: prop(target["identifier"])?.takeIf { it.isNotBlank() })
                ?: return@forEach

        if (targetId == requestedId) return@forEach

        val objectType = resolveKnownObjectShape(targetId, propertyName, shapeProperties, resourceTypeMap, knownShapeNames)

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
        val resolvedType = resolvedLink?.resolvedShape ?: objectType
        val resolvedId = resolvedLink?.resolvedVersionId ?: targetId

        val relationValue =
            AccessPropertyValue(
                id = resolvedId,
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

    groupedProperties.replaceAll { _, values ->
        deduplicateLiteralAndResolvedValues(values)
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

private fun deduplicateLiteralAndResolvedValues(values: MutableList<AccessPropertyValue>): MutableList<AccessPropertyValue> {
    if (values.isEmpty()) return values

    val dedupedResolved = deduplicateResolvedValues(values)

    val resolvedIds =
        dedupedResolved
            .mapNotNull { it.id }
            .map { canonicalRelationIdentifier(it) }
            .toSet()
    if (resolvedIds.isEmpty()) return dedupedResolved

    return dedupedResolved
        .filterNot { value ->
            value.id == null && value.value != null && canonicalRelationIdentifier(value.value) in resolvedIds
        }.toMutableList()
}

private fun deduplicateResolvedValues(values: List<AccessPropertyValue>): MutableList<AccessPropertyValue> {
    val selected = linkedMapOf<String, AccessPropertyValue>()
    values.forEach { value ->
        val resolvedId = value.id
        if (resolvedId.isNullOrBlank()) {
            selected["raw:${selected.size}"] = value
            return@forEach
        }

        val key =
            listOf(
                canonicalRelationIdentifier(resolvedId),
                value.type.orEmpty(),
                value.predicateIri.orEmpty(),
            ).joinToString("|")
        val existing = selected[key]
        selected[key] =
            if (existing == null || isPreferredResolvedValue(value, existing)) {
                value
            } else {
                existing
            }
    }
    return selected.values.toMutableList()
}

private fun isPreferredResolvedValue(candidate: AccessPropertyValue, current: AccessPropertyValue): Boolean {
    val candidateHasVersionSuffix = candidate.id?.let { hasVersionSuffix(it) } ?: false
    val currentHasVersionSuffix = current.id?.let { hasVersionSuffix(it) } ?: false
    if (candidateHasVersionSuffix != currentHasVersionSuffix) {
        return candidateHasVersionSuffix
    }
    return candidate.href?.length ?: Int.MAX_VALUE < current.href?.length ?: Int.MAX_VALUE
}

private fun hasVersionSuffix(value: String): Boolean = value.trim().matches(Regex(".*?/v\\d+(?:\\.\\d+)*$"))

private fun canonicalRelationIdentifier(value: String): String =
    value.trim().replace(Regex("/v\\d+(?:\\.\\d+)*$"), "")

private fun shouldBeArrayByShacl(
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
): Boolean {
    val normalizedCandidates = propertyLookupCandidates(propertyName).map { normalizeShapeName(localPropertyName(it)) }
    val property = shapeProperties.firstOrNull { prop ->
        val propName = normalizeShapeName(localPropertyName(prop.name))
        val propPath = normalizeShapeName(localPropertyName(prop.path))
        normalizedCandidates.any { it == propName || it == propPath }
    } ?: return false

    return property.maxCount != 1
}

private fun isRelationPropertyByShacl(
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
): Boolean {
    val normalizedCandidates = propertyLookupCandidates(propertyName).map { normalizeShapeName(localPropertyName(it)) }
    val property = shapeProperties.firstOrNull { prop ->
        val propName = normalizeShapeName(localPropertyName(prop.name))
        val propPath = normalizeShapeName(localPropertyName(prop.path))
        normalizedCandidates.any { it == propName || it == propPath }
    } ?: return false

    return !property.classConstraint.isNullOrBlank() ||
        !property.classConstraints.isNullOrEmpty() ||
        !property.nodeShapeRef.isNullOrBlank() ||
        !property.nodeShapeRefs.isNullOrEmpty()
}

private fun localPropertyName(value: String): String =
    value.substringAfterLast('/').substringAfterLast('#').substringAfter(':')

private fun isResolvableResource(
    client: PropertyGraphClient,
    objectId: String,
    existenceCache: MutableMap<String, Boolean>,
): Boolean =
    existenceCache.getOrPut(objectId) {
        ResourceLinkResolver.resolveAccessEntityId(client, objectId) != null
    }

private fun resolveKnownObjectShape(
    objectId: String,
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
): String? =
    resolveObjectShape(
        objectId = objectId,
        propertyName = propertyName,
        shapeProperties = shapeProperties,
        resourceTypeMap = resourceTypeMap,
    )?.takeIf { candidate -> knownShapeNames.any { normalizeShapeName(it) == normalizeShapeName(candidate) } }

private fun buildAccessPropertyValue(
    client: PropertyGraphClient,
    objectId: String,
    requestedId: String,
    predicateIri: String?,
    objectType: String?,
    resourceTypeMap: Map<String, String>,
    publicBaseUrl: String,
    existenceCache: MutableMap<String, Boolean>,
): AccessPropertyValue {
    if (!ResourceLinkResolver.isPotentialResourceIdentifier(objectId)) {
        return AccessPropertyValue(value = objectId)
    }
    if (objectId == requestedId) {
        return AccessPropertyValue(value = objectId)
    }

    val resolvedLink =
        if (isResolvableResource(client, objectId, existenceCache)) {
            ResourceLinkResolver.resolveLink(
                client = client,
                objectId = objectId,
                resourceTypeMap = resourceTypeMap,
                normalizeShapeName = ::normalizeShapeName,
                publicBaseUrl = publicBaseUrl,
                preferredShape = objectType,
            )
        } else {
            null
        }
    val href = resolvedLink?.href
    val resolvedType = resolvedLink?.resolvedShape ?: objectType
    val resolvedId = resolvedLink?.resolvedVersionId ?: objectId
    val fallbackHref =
        if (!resolvedType.isNullOrBlank()) {
            ResourceLinkResolver.buildNavigableAccessHref(publicBaseUrl, resolvedType, resolvedId)
        } else {
            null
        }

    return if (!resolvedType.isNullOrBlank() || href != null || fallbackHref != null) {
        AccessPropertyValue(
            id = resolvedId,
            type = resolvedType,
            title = null,
            href = href ?: fallbackHref,
            predicateIri = predicateIri,
        )
    } else {
        AccessPropertyValue(value = objectId, predicateIri = predicateIri)
    }
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
    val normalizedCandidates = propertyLookupCandidates(propertyName).map { normalizeShapeName(it) }
    val property = shapeProperties.firstOrNull { prop ->
        val propName = normalizeShapeName(prop.name)
        val propPath = normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#'))
        normalizedCandidates.any { it == propName || it == propPath }
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
    val normalizedCandidates = propertyLookupCandidates(propertyName).map { normalizeShapeName(it) }
    val property = shapeProperties.firstOrNull { prop ->
        val propName = normalizeShapeName(prop.name)
        val propPath = normalizeShapeName(prop.path.substringAfterLast('/').substringAfterLast('#'))
        normalizedCandidates.any { it == propName || it == propPath }
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
            MATCH (c:Conceptual {id: ${'$'}entityId})
            RETURN c.type AS value
            LIMIT 1
            """.trimIndent(),
            mapOf("entityId" to entityId),
        ).firstOrNull()?.get("value")?.toString()
            ?: client.runQuery(
                """
                MATCH (c:Conceptual {identifier: ${'$'}entityId})
                RETURN c.type AS value
                LIMIT 1
                """.trimIndent(),
                mapOf("entityId" to entityId),
            ).firstOrNull()?.get("value")?.toString()
            ?: client.runQuery(
            """
            MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {id: ${'$'}entityId})
            RETURN c.type AS value
            LIMIT 1
            """.trimIndent(),
            mapOf("entityId" to entityId),
        ).firstOrNull()?.get("value")?.toString()
            ?: client.runQuery(
                """
                MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {identifier: ${'$'}entityId})
                RETURN c.type AS value
                LIMIT 1
                """.trimIndent(),
                mapOf("entityId" to entityId),
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

private fun prop(value: Any?): String? = when (value) {
    is List<*> -> value.firstOrNull()?.toString()
    else -> value?.toString()
}

private fun propertyLookupCandidates(propertyName: String): List<String> {
    val values = linkedSetOf(propertyName)
    propertyName.substringAfter(':', propertyName).takeIf { it != propertyName }?.let { values.add(it) }
    propertyName.substringAfter('_', propertyName).takeIf { it != propertyName }?.let { values.add(it) }
    return values.toList()
}

private fun canonicalPropertyKeyFromShacl(
    propertyName: String,
    shapeProperties: List<PropertyInfo>,
    shaclPrefixes: Map<String, String>,
): String {
    if (propertyName.contains(':')) return propertyName
    val normalizedCandidates = propertyLookupCandidates(propertyName).map { normalizeShapeName(localPropertyName(it)) }
    val property = shapeProperties.firstOrNull { prop ->
        val propName = normalizeShapeName(localPropertyName(prop.name))
        val propPath = normalizeShapeName(localPropertyName(prop.path))
        normalizedCandidates.any { it == propName || it == propPath }
    } ?: return propertyName

    val matches = shapeProperties.filter { prop ->
        val propName = normalizeShapeName(localPropertyName(prop.name))
        val propPath = normalizeShapeName(localPropertyName(prop.path))
        normalizedCandidates.any { it == propName || it == propPath }
    }
    if (matches.size > 1) return propertyName
    return namespacedPropertyKey(property.path, shaclPrefixes)
}

private fun namespacedPropertyKey(path: String, shaclPrefixes: Map<String, String>): String {
    val local = path.substringAfterLast('/').substringAfterLast('#').lowercase()
    val namespace = path.substringBeforeLast('#', path.substringBeforeLast('/') + "/")
    val prefix = preferredShaclPrefix(namespace, shaclPrefixes)
    return if (prefix != null) "$prefix:$local" else local
}

private fun preferredShaclPrefix(namespace: String, shaclPrefixes: Map<String, String>): String? {
    val normalizedNamespace = normalizeNamespace(namespace)
    val candidates =
        shaclPrefixes
            .filter { (_, ns) -> normalizeNamespace(ns) == normalizedNamespace }
            .keys
            .filter { it.isNotBlank() }
    if (candidates.isEmpty()) return null
    return candidates.minWithOrNull(compareBy<String>({ isGenericPrefix(it) }, { it.length }, { it }))
}

private fun isGenericPrefix(prefix: String): Boolean = prefix == "ns" || prefix.matches(Regex("ns\\d*"))

private fun normalizeNamespace(namespace: String): String = namespace.trimEnd('#', '/')

private fun isStructuralProperty(key: String): Boolean {
    val normalized = normalizeShapeName(localPropertyName(key))
    return normalized in setOf("id", "status", "versionnumber", "createdat", "updatedat")
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
