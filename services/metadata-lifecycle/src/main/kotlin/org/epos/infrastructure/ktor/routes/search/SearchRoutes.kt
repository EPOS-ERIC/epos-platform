package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.HttpStatusCode
import io.ktor.server.application.ApplicationCall
import io.ktor.server.application.log
import io.ktor.server.response.respond
import io.ktor.server.response.respondRedirect
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.infrastructure.ktor.routes.SearchItem
import org.epos.infrastructure.ktor.routes.SearchResponse
import org.epos.infrastructure.ktor.routes.extractTypeName
import org.epos.infrastructure.ktor.routes.requireActorHeaders
import org.epos.infrastructure.ktor.routes.resolvePublicBaseUrl
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.services.ShaclService

private const val SEARCH_QUERY_TIMING_ENV = "EPOS_SEARCH_QUERY_TIMING"

fun Route.searchRoutes(
    client: PropertyGraphClient,
    shaclService: ShaclService,
) {
    get("/api/v2/access/search") {
        executeSearch(call, client, shaclService)
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
        call.respondRedirect("/api/v2/access/search?$query", permanent = false)
    }
}

private suspend fun executeSearch(
    call: ApplicationCall,
    client: PropertyGraphClient,
    shaclService: ShaclService,
) {
    val params = call.request.queryParameters

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
    val includeConstraints = params["includeConstraints"]?.toBooleanStrictOrNull() ?: false
    val versioningStatuses =
        params.getAll("versioningStatus")?.map { it.lowercase() }
            ?: listOf("published")
    val resourceTypeInfos = shaclService.getResourceTypes()
    val shaclPrefixes =
        shaclService
            .getParser()
            .getCachedShapesGraph()
            ?.prefixes
            .orEmpty()
    val shapeByTargetClass = resourceTypeInfos.associate { it.targetClass to it.name }
    val shapeByNormalizedName = resourceTypeInfos.associateBy { normalizeTypeName(it.name) }

    val statusFilter = versioningStatuses

    val matchedResourceTypes =
        if (types.isNotEmpty()) {
            val normalizedTypes = types.map { normalizeTypeName(it) }
            val matchingTypes =
                resourceTypeInfos.filter { typeInfo ->
                    normalizeTypeName(typeInfo.name) in normalizedTypes
                }
            if (matchingTypes.isEmpty()) {
                call.respond(
                    SearchResponse(
                        total = 0,
                        totalItems = 0,
                        page = page,
                        totalPages = 0,
                        items = emptyList(),
                    ),
                )
                return
            }
            matchingTypes
        } else {
            null
        }

    val searchTerms = q?.split("\\s+".toRegex())?.filter { it.isNotBlank() } ?: emptyList()
    val typeValueFilter =
        matchedResourceTypes
            ?.flatMap { typeInfo ->
                val targetClass = typeInfo.targetClass
                val targetLocal = targetClass.substringAfterLast('/').substringAfterLast('#')
                listOf(typeInfo.name, targetClass, targetLocal)
            }?.filter { it.isNotBlank() }
            ?.toSet()
            .orEmpty()
    val queryParams =
        mapOf(
            "statuses" to statusFilter,
            "types" to typeValueFilter.map { it.lowercase() },
            "terms" to searchTerms.map { it.lowercase() },
            "offset" to offset,
            "limit" to limit,
        )

    try {
        val query =
            """
            MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version)
            WITH c, v, properties(v) AS vp
            WHERE toLower(v.status) IN ${'$'}statuses
              AND (
                size(${ '$' }types) = 0 OR any(t IN ${'$'}types WHERE
                  toLower(coalesce(c.type, '')) = t
                  OR toLower(coalesce(v.type, '')) = t
                  OR toLower(coalesce(vp['rdf:type'], '')) = t
                  OR toLower(last(split(coalesce(v.type, ''), '/'))) = t
                  OR toLower(last(split(coalesce(v.type, ''), '#'))) = t
                  OR toLower(last(split(coalesce(vp['rdf:type'], ''), '/'))) = t
                  OR toLower(last(split(coalesce(vp['rdf:type'], ''), '#'))) = t
                )
              )
              AND (
                size(${ '$' }terms) = 0 OR all(term IN ${'$'}terms WHERE
                    toLower(coalesce(v.title, '')) CONTAINS term
                    OR toLower(coalesce(v.description, '')) CONTAINS term
                    OR toLower(coalesce(v.label, '')) CONTAINS term
                )
              )
            RETURN properties(c) AS c, properties(v) AS v
            SKIP ${'$'}offset LIMIT ${'$'}limit
            """.trimIndent()

        val countQuery =
            """
            MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version)
            WITH c, v, properties(v) AS vp
            WHERE toLower(v.status) IN ${'$'}statuses
              AND (
                size(${ '$' }types) = 0 OR any(t IN ${'$'}types WHERE
                  toLower(coalesce(c.type, '')) = t
                  OR toLower(coalesce(v.type, '')) = t
                  OR toLower(coalesce(vp['rdf:type'], '')) = t
                  OR toLower(last(split(coalesce(v.type, ''), '/'))) = t
                  OR toLower(last(split(coalesce(v.type, ''), '#'))) = t
                  OR toLower(last(split(coalesce(vp['rdf:type'], ''), '/'))) = t
                  OR toLower(last(split(coalesce(vp['rdf:type'], ''), '#'))) = t
                )
              )
              AND (
                size(${ '$' }terms) = 0 OR all(term IN ${'$'}terms WHERE
                    toLower(coalesce(v.title, '')) CONTAINS term
                    OR toLower(coalesce(v.description, '')) CONTAINS term
                    OR toLower(coalesce(v.label, '')) CONTAINS term
                )
              )
            RETURN count(v) AS value
            """.trimIndent()

        val queryStartNanos = System.nanoTime()
        val rawResults = client.runQuery(query, queryParams)
        if (isSearchQueryTimingEnabled()) {
            val elapsedMs = (System.nanoTime() - queryStartNanos) / 1_000_000
            call.application.log.info(
                "search query timing: elapsedMs={}, rows={}, offset={}, limit={}, statuses={}, types={}, terms={}",
                elapsedMs,
                rawResults.size,
                offset,
                limit,
                statusFilter.joinToString(","),
                typeValueFilter.size,
                searchTerms.size,
            )
        }
        val results =
            rawResults
                .filter { row ->
                    val c = row["c"] as? Map<*, *> ?: return@filter false
                    val v = row["v"] as? Map<*, *> ?: return@filter false
                    val status = prop(v["status"])?.lowercase() ?: return@filter false
                    status in statusFilter
                }
        val totalCount =
            client
                .runQuery(countQuery, queryParams)
                .firstOrNull()
                ?.get("value")
                ?.toString()
                ?.toIntOrNull() ?: 0
        val totalPages = if (totalCount == 0) 0 else ((totalCount + limit - 1) / limit)
        val baseUrl = call.resolvePublicBaseUrl()
        val items: List<SearchItem> =
            results.mapNotNull { row ->
                val c = row["c"] as? Map<*, *> ?: return@mapNotNull null
                val v = row["v"] as? Map<*, *> ?: return@mapNotNull null

                val id = prop(c["id"]) ?: return@mapNotNull null
                val typeUri = prop(c["type"]) ?: ""
                val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")

                @Suppress("UNCHECKED_CAST")
                val data =
                    v
                        .filterKeys { it is String && it !in structuralKeys }
                        .mapKeys { (k, _) -> k as String }
                        .mapValues { (_, v2) -> propAny(v2) }

                val inferredFromId =
                    shapeByNormalizedName[normalizeTypeName(extractTypeName(id))]?.name
                        ?: extractTypeName(id).lowercase()
                val apiTypeRaw =
                    mapSearchTypeUriToApiType(typeUri)
                        ?: shapeByTargetClass[typeUri]
                        ?: shapeByNormalizedName[normalizeTypeName(typeUri)]?.name
                        ?: inferredFromId
                val apiType = canonicalApiType(apiTypeRaw)
                val accessHref = "$baseUrl/api/v2/access/$apiType/${java.net.URLEncoder.encode(id, "UTF-8")}"
                val fields: Map<String, Any?>? =
                    if (includeConstraints) {
                        buildSearchItemFields(
                            client = client,
                            resourceTypeInfos = resourceTypeInfos,
                            typeUri = typeUri,
                            apiType = apiType,
                            data = data,
                            publicBaseUrl = baseUrl,
                            shaclPrefixes = shaclPrefixes,
                        )
                    } else {
                        null
                    }

                SearchItem(
                    type = apiType,
                    id = id,
                    href = accessHref,
                    fields = fields,
                )
            }

        call.respond(
            SearchResponse(
                total = totalCount,
                totalItems = totalCount,
                page = page,
                totalPages = totalPages,
                items = items,
            ),
        )
    } catch (e: Exception) {
        call.application.log.error("search query failed: {}", e.message)
        call.respond(
            SearchResponse(
                total = 0,
                totalItems = 0,
                page = page,
                totalPages = 0,
                items = emptyList(),
            ),
        )
    }
}

private fun prop(value: Any?): String? =
    when (value) {
        is List<*> -> value.firstOrNull()?.toString()
        else -> value?.toString()
    }

private fun propAny(value: Any?): Any? =
    when (value) {
        is List<*> -> if (value.size == 1) value.firstOrNull() else value
        else -> value
    }

private fun List<Pair<String, String>>.toQueryString(): String =
    joinToString("&") { (key, value) ->
        "${java.net.URLEncoder.encode(key, "UTF-8")}=${java.net.URLEncoder.encode(value, "UTF-8")}"
    }

private fun normalizeTypeName(typeName: String): String = typeName.lowercase().replace("-", "").replace("_", "")

private fun mapSearchTypeUriToApiType(typeUri: String): String? =
    typeUri
        .substringAfterLast('/')
        .substringAfterLast('#')
        .takeIf { it.isNotBlank() }
        ?.lowercase()

private fun canonicalApiType(typeName: String): String = normalizeTypeName(typeName).ifBlank { "resource" }

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun buildSearchItemFields(
    client: PropertyGraphClient,
    resourceTypeInfos: List<org.epos.api.models.ResourceTypeInfo>,
    typeUri: String,
    apiType: String,
    data: Map<String, Any?>,
    publicBaseUrl: String,
    shaclPrefixes: Map<String, String>,
): Map<String, Any?> {
    val normalizedApiType = normalizeTypeName(apiType)
    val resourceType =
        resourceTypeInfos.firstOrNull {
            normalizeTypeName(it.name) == normalizedApiType || it.targetClass.equals(typeUri, ignoreCase = true)
        } ?: return emptyMap()

    val resourceTypeMap = resourceTypeInfos.associate { it.targetClass to it.name }
    val knownShapeNames = resourceTypeInfos.map { it.name }.toSet()

    val selected = linkedMapOf<String, Any?>()
    val selectedProps = resourceType.properties.filter { it.required || it.recommended }
    selectedProps.forEach { prop ->
        val field = extractFieldValue(data, prop, shaclPrefixes)
        if (field != null) {
            val fieldValue = field.value ?: return@forEach
            val key = selectFieldKey(prop, selected, field.sourceKey, shaclPrefixes)
            selected[key] =
                normalizeSearchFieldValue(
                    client = client,
                    value = fieldValue,
                    maxCount = prop.maxCount,
                    property = prop,
                    resourceTypeMap = resourceTypeMap,
                    knownShapeNames = knownShapeNames,
                    publicBaseUrl = publicBaseUrl,
                )
        }
    }
    return selected
}

private fun selectFieldKey(
    property: org.epos.api.shacl.PropertyInfo,
    current: Map<String, Any?>,
    sourceKey: String?,
    shaclPrefixes: Map<String, String>,
): String {
    if (!sourceKey.isNullOrBlank() && sourceKey.contains(':') && !current.containsKey(sourceKey)) {
        return sourceKey
    }
    val qualifiedBase =
        namespacedPropertyKey(property.path, shaclPrefixes)
            .ifBlank {
                property.name
                    .trim()
                    .lowercase()
                    .replace(" ", "")
            }
    if (!current.containsKey(qualifiedBase)) return qualifiedBase

    var idx = 2
    while (current.containsKey("${qualifiedBase}_$idx")) {
        idx++
    }
    return "${qualifiedBase}_$idx"
}

private fun normalizeSearchFieldValue(
    client: PropertyGraphClient,
    value: Any,
    maxCount: Int?,
    property: org.epos.api.shacl.PropertyInfo,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
    publicBaseUrl: String,
): Any {
    val relationShapeCandidates = relationShapeCandidates(property, resourceTypeMap, knownShapeNames)
    val relationValues = relationValues(value)
    if (relationShapeCandidates.isNotEmpty() && relationValues.isNotEmpty()) {
        val mapped =
            relationValues.map { objectId ->
                val link =
                    ResourceLinkResolver.resolveLink(
                        client = client,
                        objectId = objectId,
                        resourceTypeMap = resourceTypeMap,
                        normalizeShapeName = ::normalizeTypeName,
                        publicBaseUrl = publicBaseUrl,
                        preferredShape = null,
                    )
                val resolvedType = resolveRelationType(link?.resolvedShape, relationShapeCandidates)
                val href =
                    if (resolvedType != null) {
                        ResourceLinkResolver.buildNavigableAccessHref(publicBaseUrl, resolvedType, objectId)
                    } else {
                        link?.href
                    }
                linkedMapOf(
                    "id" to objectId,
                    "type" to resolvedType,
                    "href" to href,
                )
            }
        return if (maxCount == 1) mapped.firstOrNull() ?: emptyMap<String, Any?>() else mapped
    }

    return normalizeFieldValue(value, maxCount)
}

private fun normalizeFieldValue(
    value: Any,
    maxCount: Int?,
): Any =
    if (maxCount == 1) {
        value
    } else {
        when (value) {
            is List<*> -> value
            else -> listOf(value)
        }
    }

private fun extractFieldValue(
    data: Map<String, Any?>,
    property: org.epos.api.shacl.PropertyInfo,
    shaclPrefixes: Map<String, String>,
): MatchedFieldValue? {
    val localFromPath = property.path.substringAfterLast('/').substringAfterLast('#')
    val namespaced = namespacedPropertyKey(property.path, shaclPrefixes)
    val candidates = listOf(property.name, localFromPath, namespaced, property.path)
    val matchedKey = candidates.firstOrNull { key -> data.containsKey(key) }
    val raw = matchedKey?.let { data[it] } ?: return null
    if (raw == null) return null

    if (raw !is List<*>) return MatchedFieldValue(matchedKey, raw)

    val expectsRelation =
        property.classConstraint != null ||
            !property.classConstraints.isNullOrEmpty() ||
            property.nodeShapeRef != null ||
            !property.nodeShapeRefs.isNullOrEmpty()

    val relationLike = raw.filter { it is Map<*, *> }
    val literalLike = raw.filterNot { it is Map<*, *> }

    val normalized =
        if (expectsRelation) {
            when {
                relationLike.isEmpty() -> raw
                relationLike.size == 1 && property.maxCount == 1 -> relationLike.first()
                else -> relationLike
            }
        } else {
            when {
                literalLike.isEmpty() -> raw
                literalLike.size == 1 && property.maxCount == 1 -> literalLike.first()
                else -> literalLike
            }
        }
    return MatchedFieldValue(matchedKey, normalized)
}

private data class MatchedFieldValue(
    val sourceKey: String?,
    val value: Any?,
)

private fun namespacedPropertyKey(
    path: String,
    shaclPrefixes: Map<String, String>,
): String {
    val local = path.substringAfterLast('/').substringAfterLast('#').lowercase()
    val namespace = path.substringBeforeLast('#', path.substringBeforeLast('/') + "/")
    val prefix = preferredShaclPrefix(namespace, shaclPrefixes)
    return if (prefix != null) "$prefix:$local" else local
}

private fun preferredShaclPrefix(
    namespace: String,
    shaclPrefixes: Map<String, String>,
): String? {
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

private fun relationShapeCandidates(
    property: org.epos.api.shacl.PropertyInfo,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
): List<String> {
    val classCandidates =
        buildList {
            property.classConstraint?.let { add(it) }
            property.classConstraints?.let { addAll(it) }
        }

    val classShapes =
        classCandidates
            .mapNotNull { classIri ->
                resourceTypeMap[classIri]
                    ?: resourceTypeMap.entries
                        .firstOrNull {
                            normalizeTypeName(it.key.substringAfterLast('/').substringAfterLast('#')) ==
                                normalizeTypeName(classIri.substringAfterLast('/').substringAfterLast('#'))
                        }?.value
            }.filter { resolved -> knownShapeNames.any { normalizeTypeName(it) == normalizeTypeName(resolved) } }

    val nodeCandidates =
        buildList {
            property.nodeShapeRef?.let { add(it) }
            property.nodeShapeRefs?.let { addAll(it) }
        }

    val nodeShapes =
        nodeCandidates.mapNotNull { nodeRef ->
            val shapeCandidate =
                nodeRef
                    .substringAfterLast('/')
                    .substringAfterLast('#')
                    .removeSuffix("Shape")
                    .lowercase()
            knownShapeNames.firstOrNull { normalizeTypeName(it) == normalizeTypeName(shapeCandidate) }
        }

    return (classShapes + nodeShapes).distinctBy { normalizeTypeName(it) }
}

private fun resolveRelationType(
    resolvedFromGraph: String?,
    candidates: List<String>,
): String? {
    if (!resolvedFromGraph.isNullOrBlank()) {
        val matched = candidates.firstOrNull { normalizeTypeName(it) == normalizeTypeName(resolvedFromGraph) }
        if (matched != null) return matched
        return resolvedFromGraph
    }
    return if (candidates.size == 1) candidates.first() else candidates.firstOrNull()
}

private fun relationValues(value: Any): List<String> =
    when (value) {
        is List<*> -> value.mapNotNull { it?.toString()?.trim() }.filter { ResourceLinkResolver.isPotentialResourceIdentifier(it) }
        else -> listOf(value.toString().trim()).filter { ResourceLinkResolver.isPotentialResourceIdentifier(it) }
    }

private fun isSearchQueryTimingEnabled(): Boolean = System.getenv(SEARCH_QUERY_TIMING_ENV)?.toBooleanStrictOrNull() ?: false
