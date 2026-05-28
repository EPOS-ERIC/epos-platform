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
import org.epos.infrastructure.ktor.routes.resolvePublicBaseUrl
import org.epos.infrastructure.ktor.routes.requireActorHeaders
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
        val shapeByTargetClass = resourceTypeInfos.associate { it.targetClass to it.name }
        val shapeByNormalizedName = resourceTypeInfos.associateBy { normalizeTypeName(it.name) }

        val statusFilter = versioningStatuses

        val matchedResourceTypes = if (types.isNotEmpty()) {
            val normalizedTypes = types.map { normalizeTypeName(it) }
            val matchingTypes = resourceTypeInfos.filter { typeInfo ->
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
        val statusGremlinList = statusFilter.joinToString(",") { gStr(it) }
        val typeValueFilter =
            matchedResourceTypes
                ?.flatMap { typeInfo ->
                    val targetClass = typeInfo.targetClass
                    val targetLocal = targetClass.substringAfterLast('/').substringAfterLast('#')
                    listOf(typeInfo.name, targetClass, targetLocal)
                }
                ?.filter { it.isNotBlank() }
                ?.toSet()
                .orEmpty()
        val typeGremlinList = typeValueFilter.joinToString(",") { gStr(it) }
        val typePredicate = if (typeGremlinList.isBlank()) "" else ".has('type', within($typeGremlinList))"

        val textPredicate =
            if (searchTerms.isEmpty()) {
                ""
            } else {
                val termPredicates =
                    searchTerms.flatMap { term ->
                        val escapedTerm = gStr(term.lowercase())
                        listOf(
                            "org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.values('title').map { it.get().toString().toLowerCase() }.is(org.apache.tinkerpop.gremlin.process.traversal.TextP.containing($escapedTerm))",
                            "org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.values('description').map { it.get().toString().toLowerCase() }.is(org.apache.tinkerpop.gremlin.process.traversal.TextP.containing($escapedTerm))",
                            "org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.values('label').map { it.get().toString().toLowerCase() }.is(org.apache.tinkerpop.gremlin.process.traversal.TextP.containing($escapedTerm))",
                        )
                    }
                ".where(org.apache.tinkerpop.gremlin.process.traversal.dsl.graph.__.or(${termPredicates.joinToString(",")}))"
            }

        try {
            val query =
                """
                g.V().hasLabel('Version').has('status', within($statusGremlinList)).as('v')
                  $textPredicate
                  .in('HAS_VERSION').hasLabel('Conceptual')$typePredicate.as('c')
                  .select('c','v')
                  .by(valueMap('id','type'))
                  .by(valueMap(true))
                  .range($offset, ${offset + limit})
                  .toList()
                """.trimIndent()

            val countQuery =
                """
                g.V().hasLabel('Version').has('status', within($statusGremlinList)).as('v')
                  $textPredicate
                  .in('HAS_VERSION').hasLabel('Conceptual')$typePredicate.as('c')
                  .select('c','v')
                  .count()
                  .next()
                """.trimIndent()

            val queryStartNanos = System.nanoTime()
            val rawResults = client.runQuery(query)
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
            val results = rawResults
                .filter { row ->
                    val c = row["c"] as? Map<*, *> ?: return@filter false
                    val v = row["v"] as? Map<*, *> ?: return@filter false
                    val status = prop(v["status"])?.lowercase() ?: return@filter false
                    status in statusFilter
                }
            val totalCount = client.runQuery(countQuery).firstOrNull()?.get("value")?.toString()?.toIntOrNull() ?: 0
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
                    val data = v.filterKeys { it is String && it !in structuralKeys }
                        .mapKeys { (k, _) -> k as String }
                        .mapValues { (_, v2) -> propAny(v2) }

                    val inferredFromId = shapeByNormalizedName[normalizeTypeName(extractTypeName(id))]?.name
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
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Search failed"),
            )
        }
}

private fun prop(value: Any?): String? = when (value) {
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

private fun normalizeTypeName(typeName: String): String {
    return typeName.lowercase().replace("-", "").replace("_", "")
}

private fun mapSearchTypeUriToApiType(typeUri: String): String? =
    typeUri
        .substringAfterLast('/')
        .substringAfterLast('#')
        .takeIf { it.isNotBlank() }
        ?.lowercase()

private fun canonicalApiType(typeName: String): String {
    return normalizeTypeName(typeName).ifBlank { "resource" }
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun buildSearchItemFields(
    client: PropertyGraphClient,
    resourceTypeInfos: List<org.epos.api.models.ResourceTypeInfo>,
    typeUri: String,
    apiType: String,
    data: Map<String, Any?>,
    publicBaseUrl: String,
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
        val value = extractFieldValue(data, prop.path, prop.name)
        if (value != null) {
            selected[prop.name.lowercase()] =
                normalizeSearchFieldValue(
                    client = client,
                    value = value,
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
        val mapped = relationValues.map { objectId ->
            val link = ResourceLinkResolver.resolveLink(
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

private fun normalizeFieldValue(value: Any, maxCount: Int?): Any =
    if (maxCount == 1) {
        value
    } else {
        when (value) {
            is List<*> -> value
            else -> listOf(value)
        }
    }

private fun extractFieldValue(data: Map<String, Any?>, path: String, name: String): Any? {
    val localFromPath = path.substringAfterLast('/').substringAfterLast('#')
    val candidates = listOf(name, localFromPath, path)
    return candidates.firstNotNullOfOrNull { key -> data[key] }
}

private fun relationShapeCandidates(
    property: org.epos.api.shacl.PropertyInfo,
    resourceTypeMap: Map<String, String>,
    knownShapeNames: Set<String>,
): List<String> {
    val classCandidates = buildList {
        property.classConstraint?.let { add(it) }
        property.classConstraints?.let { addAll(it) }
    }

    val classShapes =
        classCandidates.mapNotNull { classIri ->
        resourceTypeMap[classIri]
            ?: resourceTypeMap.entries.firstOrNull {
                normalizeTypeName(it.key.substringAfterLast('/').substringAfterLast('#')) ==
                    normalizeTypeName(classIri.substringAfterLast('/').substringAfterLast('#'))
            }?.value
        }.filter { resolved -> knownShapeNames.any { normalizeTypeName(it) == normalizeTypeName(resolved) } }

    val nodeCandidates = buildList {
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

private fun resolveRelationType(resolvedFromGraph: String?, candidates: List<String>): String? {
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

private fun isSearchQueryTimingEnabled(): Boolean =
    System.getenv(SEARCH_QUERY_TIMING_ENV)?.toBooleanStrictOrNull() ?: false
