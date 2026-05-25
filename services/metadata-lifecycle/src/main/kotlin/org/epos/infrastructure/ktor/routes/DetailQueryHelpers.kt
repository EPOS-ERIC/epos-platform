package org.epos.infrastructure.ktor.routes

import org.epos.propertygraph.Config
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.infrastructure.ktor.routes.search.ResourceLinkResolver
import java.net.URLEncoder

internal fun entityExists(
    client: PropertyGraphClient,
    entityUri: String,
): Boolean {
    return client.exists("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).values('id').limit(1).toList()")
}

internal fun buildRelatedByResources(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): List<ResourceRef> {
    val results = client.runQuery("""
        g.V().hasLabel('Version').has('id', ${gStr(entityUri)})
          .inE().as('e').outV().as('s')
          .select('e','s').by(label).by(valueMap(true))
          .toList()
    """.trimIndent())

    if (results.isEmpty()) return emptyList()

    val bySource = linkedMapOf<String, ResourceRef>()

    results.forEach { row ->
        val edgeLabel = row["e"]?.toString() ?: return@forEach
        if (edgeLabel in setOf("HAS_VERSION", "CURRENT_VERSION", "REVISION_OF", "SUPERSEDED_BY", "AFFECTED")) return@forEach

        val source = (row["s"] as? Map<*, *>) ?: (row["value"] as? Map<*, *>) ?: row as? Map<*, *> ?: return@forEach
        val sourceId = prop(source["id"]) ?: return@forEach
        if (bySource.containsKey(sourceId)) return@forEach

        val sourceType = inferTypeFromId(sourceId)
        bySource[sourceId] =
            ResourceRef(
                id = sourceId,
                title = prop(source["title"]),
                href = buildResolvableHref(publicBaseUrl, sourceType, sourceId),
            )
    }

    return bySource.values.filter { it.href != null }
}

internal fun buildRelatedResources(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): List<ResourceRef> {
    val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()
    val v = (result?.get("value") as? Map<*, *>) ?: (result as? Map<*, *>) ?: return emptyList()
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
    val refs = mutableListOf<ResourceRef>()
    val seen = mutableSetOf<String>()

    data.forEach { (_, value) ->
        val uris = when (value) {
            is String -> listOf(value)
            is List<*> -> value.mapNotNull { it?.toString() }
            else -> emptyList()
        }

        uris.forEach { uri ->
            if (ResourceLinkResolver.isPotentialResourceIdentifier(uri)) {
                if (seen.add(uri)) {
                    val targetType = inferTypeFromId(uri)
                    val href = buildResolvableHref(publicBaseUrl, targetType, uri)
                    if (href != null) {
                        refs.add(ResourceRef(id = uri, title = null, href = href))
                    }
                }
            }
        }
    }

    val relationRows = client.runQuery(
        """
        g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).as('s')
          .outE().as('e').inV().as('t')
          .select('e','t').by(label)
          .by(project('identifier','id')
            .by(coalesce(values('identifier'), constant('')))
            .by(coalesce(values('id'), constant(''))))
          .toList()
        """.trimIndent(),
    )

    relationRows.forEach { row ->
        val edgeLabel = row["e"]?.toString() ?: return@forEach
        if (!edgeLabel.startsWith("rel_")) return@forEach

        val target = row["t"] as? Map<*, *> ?: return@forEach
        val targetId = preferredResourceId(target) ?: return@forEach
        if (seen.add(targetId)) {
            val targetType = inferTypeFromId(targetId)
            val href = buildResolvableHref(publicBaseUrl, targetType, targetId)
            if (href != null) {
                refs.add(ResourceRef(id = targetId, title = null, href = href))
            }
        }
    }

    return refs
}

internal fun inferTypeFromId(resourceId: String): String? =
    if (Config.RELATION_STRICT_MODE) {
        null
    } else
    when {
        resourceId.contains("/dataset/", ignoreCase = true) -> "dataset"
        resourceId.contains("/distribution/", ignoreCase = true) -> "distribution"
        resourceId.contains("SoftwareApplication", ignoreCase = true) -> "softwareapplication"
        resourceId.contains("SoftwareSourceCode", ignoreCase = true) -> "softwaresourcecode"
        resourceId.contains("/concept/", ignoreCase = true) || resourceId.contains("Concept", ignoreCase = true) -> "category"
        resourceId.contains("/facility/", ignoreCase = true) -> "facility"
        resourceId.contains("/equipment/", ignoreCase = true) -> "equipment"
        resourceId.startsWith(Config.BASE_URI) -> null
        else -> null
    }

private fun buildResolvableHref(
    publicBaseUrl: String,
    apiType: String?,
    resourceId: String,
): String? {
    if (apiType.isNullOrBlank()) return null
    if (!resourceId.startsWith(Config.BASE_URI) && !resourceId.startsWith("category:")) return null
    return "$publicBaseUrl/api/v2/$apiType/${URLEncoder.encode(resourceId, "UTF-8")}"
}

internal fun extractKeywords(data: Map<*, *>): List<String> {
    val keywords = data["keyword"] ?: data["dcat:keyword"] ?: return emptyList()
    return when (keywords) {
        is String -> listOf(keywords)
        is List<*> -> keywords.mapNotNull { it as? String }
        else -> emptyList()
    }
}

internal fun extractCategories(data: Map<*, *>): List<String> {
    val themes = data["theme"] ?: data["dcat:theme"] ?: return emptyList()
    return when (themes) {
        is String -> listOf(themes)
        is List<*> -> themes.mapNotNull { it as? String }
        else -> emptyList()
    }
}

internal fun extractPublishers(data: Map<*, *>): List<String> {
    val publisher = data["publisher"] ?: data["dcterms:publisher"] ?: data["provider"] ?: data["schema:provider"] ?: return emptyList()
    return when (publisher) {
        is String -> listOf(publisher)
        is List<*> -> publisher.mapNotNull { it as? String }
        is Map<*, *> -> listOfNotNull(publisher["name"] as? String, publisher["legalName"] as? String)
        else -> emptyList()
    }
}

internal fun extractSpatialWkt(data: Map<*, *>): String? {
    val spatial = data["spatial"] ?: data["dcterms:spatial"] ?: data["bbox"] ?: data["dcat:bbox"]
    return when (spatial) {
        is String -> spatial
        is Map<*, *> -> spatial["asWKT"] as? String ?: spatial["wkt"] as? String
        else -> null
    }
}

internal fun extractTemporalExtent(data: Map<*, *>): TemporalExtent? {
    val temporal = data["temporal"] ?: data["dcterms:temporal"]
    return when (temporal) {
        is Map<*, *> -> {
            val start = temporal["startDate"] as? String ?: temporal["dcat:startDate"] as? String
            val end = temporal["endDate"] as? String ?: temporal["dcat:endDate"] as? String
            if (start != null || end != null) TemporalExtent(start = start, end = end) else null
        }
        else -> null
    }
}

internal fun buildImmediateRelationGraph(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): List<RelationGroup> {
    val outboundResult = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()
    val v = (outboundResult?.get("value") as? Map<*, *>) ?: (outboundResult as? Map<*, *>) ?: emptyMap<String, Any>()
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val outboundData = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
    data class PredicateDescriptor(val iri: String, val name: String)
    val grouped = linkedMapOf<Pair<String, PredicateDescriptor>, LinkedHashMap<String, ResourceRef>>()

    outboundData.forEach { (key, value) ->
        val predicate = key.toString()
        val uris = when (value) {
            is String -> listOf(value)
            is List<*> -> value.mapNotNull { it?.toString() }
            else -> emptyList()
        }

        uris.forEach { uri ->
            if (ResourceLinkResolver.isPotentialResourceIdentifier(uri)) {
                val otherType = inferTypeFromId(uri)
                val keyPair = "outbound" to PredicateDescriptor(iri = predicate, name = extractTypeName(predicate))
                val bucket = grouped.getOrPut(keyPair) { linkedMapOf() }
                if (!bucket.containsKey(uri)) {
                    bucket[uri] = ResourceRef(
                        id = uri,
                        title = null,
                        href = buildResolvableHref(publicBaseUrl, otherType, uri),
                    )
                }
            }
        }
    }

    val relationRows = client.runQuery(
        """
        g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).as('s')
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

    relationRows.forEach { row ->
        val edge = row["e"] as? Map<*, *> ?: return@forEach
        val edgeLabel = prop(edge["label"]) ?: return@forEach
        if (!edgeLabel.startsWith("rel_")) return@forEach

        val predicateIri = prop(edge["predicateIri"]) ?: edgeLabel.removePrefix("rel_")
        val predicateName = prop(edge["predicateKey"]) ?: edgeLabel.removePrefix("rel_")
        val target = row["t"] as? Map<*, *> ?: return@forEach
        val targetId = preferredResourceId(target) ?: return@forEach
        val otherType = inferTypeFromId(targetId)
        val keyPair = "outbound" to PredicateDescriptor(iri = predicateIri, name = predicateName)
        val bucket = grouped.getOrPut(keyPair) { linkedMapOf() }
        if (!bucket.containsKey(targetId)) {
            bucket[targetId] =
                ResourceRef(
                    id = targetId,
                    title = null,
                    href = buildResolvableHref(publicBaseUrl, otherType, targetId),
                )
        }
    }

    val inboundResults = client.runQuery("""
        g.V().hasLabel('Version').has('id', ${gStr(entityUri)})
          .inE().as('e').outV().as('s')
          .select('e','s')
          .by(project('label','predicateIri','predicateKey')
            .by(label)
            .by(coalesce(values('predicateIri'), constant('')))
            .by(coalesce(values('predicateKey'), constant(''))))
          .by(valueMap(true))
          .toList()
    """.trimIndent())

    inboundResults.forEach { row ->
        val edge = row["e"] as? Map<*, *> ?: return@forEach
        val edgeLabel = prop(edge["label"]) ?: return@forEach
        if (edgeLabel in setOf("HAS_VERSION", "CURRENT_VERSION", "REVISION_OF", "SUPERSEDED_BY", "AFFECTED")) return@forEach
        val predicateIri = prop(edge["predicateIri"]) ?: if (edgeLabel.startsWith("rel_")) edgeLabel.removePrefix("rel_") else edgeLabel
        val predicateName = prop(edge["predicateKey"]) ?: if (edgeLabel.startsWith("rel_")) edgeLabel.removePrefix("rel_") else edgeLabel

        val otherNode = (row["s"] as? Map<*, *>) ?: (row["value"] as? Map<*, *>) ?: row as? Map<*, *> ?: return@forEach
        val otherId = preferredResourceId(otherNode) ?: return@forEach
        val otherData = otherNode.filterKeys { it is String && it !in structuralKeys }
            .mapKeys { (k, _) -> k as String }
        val otherType = inferTypeFromId(otherId)
        val keyPair = "inbound" to PredicateDescriptor(iri = predicateIri, name = predicateName)
        val bucket = grouped.getOrPut(keyPair) { linkedMapOf() }
        if (!bucket.containsKey(otherId)) {
            bucket[otherId] = ResourceRef(
                id = otherId,
                title = prop(otherData["title"]),
                href = buildResolvableHref(publicBaseUrl, otherType, otherId),
            )
        }
    }

    return grouped.entries.map { (key, resources) ->
        val (direction, predicate) = key
        RelationGroup(
            direction = direction,
            predicate = RelationPredicate(iri = predicate.iri, name = predicate.name),
            resources = resources.values.toList(),
        )
    }
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun prop(value: Any?): String? = when (value) {
    null -> null
    is List<*> -> value.firstOrNull()?.toString()
    else -> value.toString()
}

private fun preferredResourceId(node: Map<*, *>): String? {
    val identifier = prop(node["identifier"])
    if (!identifier.isNullOrBlank()) return identifier
    val id = prop(node["id"])
    if (!id.isNullOrBlank()) return id
    return null
}
