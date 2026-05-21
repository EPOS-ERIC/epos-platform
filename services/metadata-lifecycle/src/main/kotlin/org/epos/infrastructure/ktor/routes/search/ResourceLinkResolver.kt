package org.epos.infrastructure.ktor.routes.search

import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils

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
        val conceptualFromVersion = if (UriUtils.hasVersion(requestedId)) UriUtils.extractConceptualUri(requestedId) else null

        val versionId =
            client.runQuery(
                """
                g.V().hasLabel('Version').has('id', ${gStr(requestedId)})
                  .project('identifier','id')
                  .by(coalesce(values('identifier'), constant('')))
                  .by(coalesce(values('id'), constant('')))
                  .limit(1).toList()
                """.trimIndent(),
        ).firstOrNull().toPreferredId()
        if (!versionId.isNullOrBlank()) return versionId

        if (!conceptualFromVersion.isNullOrBlank()) {
            val versionFromIdentifierWithinConceptual = client.runQuery(
                """
                g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualFromVersion)})
                  .out('HAS_VERSION').hasLabel('Version').has('identifier', ${gStr(requestedId)})
                  .project('identifier','id')
                  .by(coalesce(values('identifier'), constant('')))
                  .by(coalesce(values('id'), constant('')))
                  .limit(1).toList()
                """.trimIndent(),
            ).firstOrNull().toPreferredId()
            if (!versionFromIdentifierWithinConceptual.isNullOrBlank()) return versionFromIdentifierWithinConceptual
        }

        val conceptualCurrentVersion = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualFromVersion ?: requestedId)})
              .out('CURRENT_VERSION').hasLabel('Version')
              .project('identifier','id')
              .by(coalesce(values('identifier'), constant('')))
              .by(coalesce(values('id'), constant('')))
              .limit(1).toList()
            """.trimIndent(),
        ).firstOrNull().toPreferredId()
        if (!conceptualCurrentVersion.isNullOrBlank()) return conceptualCurrentVersion

        val conceptualAnyVersion = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualFromVersion ?: requestedId)})
              .out('HAS_VERSION').hasLabel('Version')
              .project('identifier','id')
              .by(coalesce(values('identifier'), constant('')))
              .by(coalesce(values('id'), constant('')))
              .limit(1).toList()
            """.trimIndent(),
        ).firstOrNull().toPreferredId()
        if (!conceptualAnyVersion.isNullOrBlank()) return conceptualAnyVersion

        if (conceptualFromVersion.isNullOrBlank()) {
            val conceptualCurrentVersionFromIdentifier = client.runQuery(
                """
                g.V().hasLabel('Conceptual').has('identifier', ${gStr(requestedId)})
                  .out('CURRENT_VERSION').hasLabel('Version')
                  .project('identifier','id')
                  .by(coalesce(values('identifier'), constant('')))
                  .by(coalesce(values('id'), constant('')))
                  .limit(1).toList()
                """.trimIndent(),
            ).firstOrNull().toPreferredId()
            if (!conceptualCurrentVersionFromIdentifier.isNullOrBlank()) return conceptualCurrentVersionFromIdentifier
        }

        val versionIdFromIdentifier = runCatching {
            client.runQuery(
                """
                g.V().hasLabel('Version').has('identifier', ${gStr(requestedId)})
                  .project('identifier','id')
                  .by(coalesce(values('identifier'), constant('')))
                  .by(coalesce(values('id'), constant('')))
                  .limit(1).toList()
                """.trimIndent(),
            ).firstOrNull().toPreferredId()
        }.getOrNull()

        return versionIdFromIdentifier
    }

    private fun Map<String, Any?>?.toPreferredId(): String? {
        if (this == null) return null
        val identifier = this["identifier"]?.toString()?.trim().orEmpty()
        if (identifier.isNotBlank()) return identifier
        val id = this["id"]?.toString()?.trim().orEmpty()
        if (id.isNotBlank()) return id
        val fallback = this["value"]?.toString()?.trim().orEmpty()
        return fallback.ifBlank { null }
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
