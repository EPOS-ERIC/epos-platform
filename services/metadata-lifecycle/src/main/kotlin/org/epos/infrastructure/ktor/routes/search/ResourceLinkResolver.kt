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
        val conceptualFromVersion =
            if (UriUtils.hasVersion(requestedId)) {
                UriUtils.extractConceptualUri(requestedId)
            } else {
                null
            }

        // Resolve using strict-to-loose fallbacks to keep links stable.
        val versionId =
            queryPreferredVersionId(
                client = client,
                query = "MATCH (v:Version {id: ${'$'}requestedId}) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                parameters = mapOf("requestedId" to requestedId),
            )
        if (!versionId.isNullOrBlank()) return versionId

        if (!conceptualFromVersion.isNullOrBlank()) {
            val versionFromIdentifierWithinConceptual =
                queryPreferredVersionId(
                    client = client,
                    query = "MATCH (:Conceptual {id: ${'$'}conceptualId})-[:HAS_VERSION]->(v:Version {identifier: ${'$'}requestedId}) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                    parameters = mapOf("conceptualId" to conceptualFromVersion, "requestedId" to requestedId),
                )
            if (!versionFromIdentifierWithinConceptual.isNullOrBlank()) return versionFromIdentifierWithinConceptual
        }

        val conceptualId = conceptualFromVersion ?: requestedId

        val conceptualCurrentVersion =
            queryPreferredVersionId(
                client = client,
                query = "MATCH (:Conceptual {id: ${'$'}conceptualId})-[:CURRENT_VERSION]->(v:Version) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                parameters = mapOf("conceptualId" to conceptualId),
            )
        if (!conceptualCurrentVersion.isNullOrBlank()) return conceptualCurrentVersion

        val conceptualAnyVersion =
            queryPreferredVersionId(
                client = client,
                query = "MATCH (:Conceptual {id: ${'$'}conceptualId})-[:HAS_VERSION]->(v:Version) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                parameters = mapOf("conceptualId" to conceptualId),
            )
        if (!conceptualAnyVersion.isNullOrBlank()) return conceptualAnyVersion

        if (conceptualFromVersion.isNullOrBlank()) {
            val conceptualCurrentVersionFromIdentifier =
                queryPreferredVersionId(
                    client = client,
                    query = "MATCH (:Conceptual {identifier: ${'$'}requestedId})-[:CURRENT_VERSION]->(v:Version) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                    parameters = mapOf("requestedId" to requestedId),
                )
            if (!conceptualCurrentVersionFromIdentifier.isNullOrBlank()) return conceptualCurrentVersionFromIdentifier
        }

        val versionIdFromIdentifier =
            runCatching {
                queryPreferredVersionId(
                    client = client,
                    query = "MATCH (v:Version {identifier: ${'$'}requestedId}) RETURN v.identifier AS identifier, v.id AS id LIMIT 1",
                    parameters = mapOf("requestedId" to requestedId),
                )
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

    private fun queryPreferredVersionId(
        client: PropertyGraphClient,
        query: String,
        parameters: Map<String, Any> = emptyMap(),
    ): String? = client.runQuery(query, parameters).firstOrNull().toPreferredId()

    fun resolveEntityShapeFromGraph(
        client: PropertyGraphClient,
        entityId: String,
        resourceTypeMap: Map<String, String>,
        normalizeShapeName: (String) -> String,
    ): String? {
        val versionId = resolveAccessEntityId(client, entityId) ?: return null
        val conceptualFromVersion =
            if (UriUtils.hasVersion(versionId)) {
                UriUtils.extractConceptualUri(versionId)
            } else {
                null
            }
        val conceptualType =
            client
                .runQuery(
                    "MATCH (c:Conceptual {id: ${'$'}entityId}) RETURN c.type AS value LIMIT 1",
                    mapOf("entityId" to entityId),
                ).firstOrNull()
                ?.get("value")
                ?.toString()
                ?: client
                    .runQuery(
                        "MATCH (c:Conceptual {id: ${'$'}versionId}) RETURN c.type AS value LIMIT 1",
                        mapOf("versionId" to versionId),
                    ).firstOrNull()
                    ?.get("value")
                    ?.toString()
                ?: conceptualFromVersion?.let { conceptualId ->
                    client
                        .runQuery(
                            "MATCH (c:Conceptual {id: ${'$'}conceptualId}) RETURN c.type AS value LIMIT 1",
                            mapOf("conceptualId" to conceptualId),
                        ).firstOrNull()
                        ?.get("value")
                        ?.toString()
                }
                ?: client
                    .runQuery(
                        "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {id: ${'$'}versionId}) RETURN c.type AS value LIMIT 1",
                        mapOf("versionId" to versionId),
                    ).firstOrNull()
                    ?.get("value")
                    ?.toString()
                ?: client
                    .runQuery(
                        "MATCH (c:Conceptual)-[:HAS_VERSION]->(v:Version {identifier: ${'$'}versionId}) RETURN c.type AS value LIMIT 1",
                        mapOf("versionId" to versionId),
                    ).firstOrNull()
                    ?.get("value")
                    ?.toString()
                ?: return null

        return resourceTypeMap.entries
            .firstOrNull { (typeUri, shapeName) ->
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

}
