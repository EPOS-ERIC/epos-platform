package org.epos.infrastructure.ktor.routes.search

import org.epos.propertygraph.client.PropertyGraphClient

internal object ResourceLinkResolver {
    data class ResolvedLink(
        val resolvedVersionId: String,
        val resolvedShape: String?,
        val href: String?,
    )

    fun resolveAccessEntityId(
        client: PropertyGraphClient,
        requestedId: String,
    ): String? {
        val versionId =
            client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(requestedId)}).values('id').limit(1).toList()")
                .firstOrNull()?.get("value")?.toString()
        if (!versionId.isNullOrBlank()) return versionId

        return client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(requestedId)})
              .out('CURRENT_VERSION').hasLabel('Version')
              .values('id').limit(1).toList()
            """.trimIndent(),
        ).firstOrNull()?.get("value")?.toString()
    }

    fun resolveEntityShapeFromGraph(
        client: PropertyGraphClient,
        entityId: String,
        resourceTypeMap: Map<String, String>,
        normalizeShapeName: (String) -> String,
    ): String? {
        val versionId = resolveAccessEntityId(client, entityId) ?: return null
        val conceptualType = client.runQuery(
            """
            g.V().hasLabel('Version').has('id', ${gStr(versionId)}).in('HAS_VERSION').hasLabel('Conceptual')
              .values('type').limit(1).toList()
            """.trimIndent(),
        ).firstOrNull()?.get("value")?.toString() ?: return null

        return resourceTypeMap.entries.firstOrNull { (typeUri, shapeName) ->
            normalizeShapeName(shapeName) == normalizeShapeName(conceptualType) ||
                normalizeShapeName(typeUri.substringAfterLast('/').substringAfterLast('#')) == normalizeShapeName(conceptualType)
        }?.value
    }

    fun isPotentialResourceIdentifier(value: String): Boolean {
        if (value.startsWith("http://") || value.startsWith("https://")) return true
        if (value.startsWith("category:")) return true
        return value.contains(":") && !value.contains(" ")
    }

    fun buildNavigableAccessHref(
        publicBaseUrl: String,
        objectType: String?,
        objectId: String,
    ): String? {
        if (objectType.isNullOrBlank()) return null
        return "$publicBaseUrl/api/v2/access/$objectType/${java.net.URLEncoder.encode(objectId, "UTF-8")}"
    }

    fun resolveLink(
        client: PropertyGraphClient,
        objectId: String,
        resourceTypeMap: Map<String, String>,
        normalizeShapeName: (String) -> String,
        publicBaseUrl: String,
        preferredShape: String? = null,
    ): ResolvedLink? {
        val versionId = resolveAccessEntityId(client, objectId) ?: return null
        val shape = preferredShape ?: resolveEntityShapeFromGraph(client, objectId, resourceTypeMap, normalizeShapeName)
        val href = buildNavigableAccessHref(publicBaseUrl, shape, objectId)
        return ResolvedLink(
            resolvedVersionId = versionId,
            resolvedShape = shape,
            href = href,
        )
    }

    private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
}
