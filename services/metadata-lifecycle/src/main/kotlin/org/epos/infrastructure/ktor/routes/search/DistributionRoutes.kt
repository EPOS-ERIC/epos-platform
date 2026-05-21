package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.infrastructure.ktor.routes.*
import org.epos.propertygraph.client.PropertyGraphClient
import java.net.URLDecoder

fun Route.distributionRoutes(client: PropertyGraphClient) {
    get("/api/v2/distribution/{id}") {
        val distributionId = call.parameters["id"]
        if (distributionId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing distribution ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(distributionId, "UTF-8")
            } catch (_: Exception) {
                distributionId
            }

        val resolvedId = resolveDistributionVersionId(client, decodedId)
        if (resolvedId == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Distribution not found: $decodedId"))
            return@get
        }

        try {
            val distribution =
                buildDistributionDetail(
                    client = client,
                    entityUri = resolvedId,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(distribution)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch distribution"),
            )
        }
    }
}

private fun buildDistributionDetail(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): DistributionDetail {
    val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()

    @Suppress("UNCHECKED_CAST")
    val rawMap = (result?.get("value") as? Map<String, Any?>) ?: (result as? Map<String, Any?>) ?: emptyMap()
    val v = rawMap.mapValues { (_, vv) -> prop(vv) }
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it !in structuralKeys }

    val title = data["title"] as? String ?: data["label"] as? String ?: extractTypeName(entityUri)
    val description = data["description"] as? String
    val accessUrl = data["accessURL"] as? String ?: data["downloadURL"] as? String
    val operation = data["operation"] as? String

    val related =
        buildRelatedResources(
            client = client,
            entityUri = entityUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    val relatedBy =
        buildRelatedByResources(
            client = client,
            entityUri = entityUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    val relations =
        buildImmediateRelationGraph(
            client = client,
            entityUri = entityUri,
            publicBaseUrl = publicBaseUrl,
        ).takeIf { it.isNotEmpty() }

    return DistributionDetail(
        id = entityUri,
        title = title,
        description = description,
        accessUrl = accessUrl,
        operation = operation,
        relations = relations,
        related = related,
        relatedBy = relatedBy,
    )
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"
private fun prop(value: Any?): Any? =
    when (value) {
        is List<*> -> if (value.size == 1) value.firstOrNull() else value
        else -> value
    }

private fun resolveDistributionVersionId(
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
