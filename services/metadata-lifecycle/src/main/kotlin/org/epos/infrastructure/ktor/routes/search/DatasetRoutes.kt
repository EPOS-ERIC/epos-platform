package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.header
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ErrorResponse
import org.epos.infrastructure.ktor.routes.*
import org.epos.propertygraph.client.PropertyGraphClient
import org.epos.propertygraph.utils.UriUtils
import java.net.URLDecoder
import java.net.URLEncoder

fun Route.datasetRoutes(
    client: PropertyGraphClient,
    resourceController: ResourceController,
) {
    get("/api/v2/dataset/{id}") {
        val datasetId = call.parameters["id"]
        if (datasetId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing dataset ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(datasetId, "UTF-8")
            } catch (_: Exception) {
                datasetId
            }

        val status = call.request.queryParameters["status"]?.lowercase() ?: "published"
        val resolvedId = resolveVersionId(client, decodedId, status)

        if (resolvedId == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Dataset not found: $decodedId"))
            return@get
        }

        val formatParam = call.request.queryParameters["format"]?.lowercase()
        val acceptHeader = call.request.header("Accept")?.lowercase() ?: "application/json"
        val requestedFormat = formatParam ?: acceptHeader

        if (requestedFormat.contains("turtle") || requestedFormat == "ttl") {
            val ttl = resourceController.getEntity(resolvedId, status)?.let { convertEntityToTurtle(it) }
            if (!ttl.isNullOrEmpty()) {
                call.respondText(ttl, ContentType.parse("text/turtle"))
            } else {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Dataset not found"))
            }
            return@get
        }

        if (requestedFormat.contains("ld+json") || requestedFormat.contains("json-ld") || requestedFormat == "jsonld") {
            val jsonld = resourceController.getEntity(resolvedId, status)?.let { convertEntityToJsonLd(it) }
            if (!jsonld.isNullOrEmpty()) {
                call.respondText(jsonld, ContentType.parse("application/ld+json"))
            } else {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Dataset not found"))
            }
            return@get
        }

        try {
            val dataset =
                buildDatasetDetail(
                    client = client,
                    entityUri = resolvedId,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(dataset)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch dataset"),
            )
        }
    }
}

private fun buildDatasetDetail(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): DatasetDetail {
    val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()

    @Suppress("UNCHECKED_CAST")
    val rawMap = (result?.get("value") as? Map<String, Any?>) ?: (result as? Map<String, Any?>) ?: emptyMap()
    val v = rawMap.mapValues { (_, vv) -> prop(vv) }
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it !in structuralKeys }
    val status = (v["status"] as? String)?.uppercase() ?: "PUBLISHED"

    val title = data["title"] as? String ?: extractTypeName(entityUri)
    val description = data["description"] as? String

    val keywords = extractKeywords(data).takeIf { it.isNotEmpty() }
    val categories = extractCategories(data).takeIf { it.isNotEmpty() }
    val publishers = extractPublishers(data).takeIf { it.isNotEmpty() }
    val spatial = extractSpatialWkt(data)
    val temporal = extractTemporalExtent(data)

    val distributions = getDatasetDistributions(data, publicBaseUrl).takeIf { it.isNotEmpty() }

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

    return DatasetDetail(
        id = entityUri,
        title = title,
        description = description,
        keywords = keywords,
        categories = categories,
        publisher = publishers,
        spatial = spatial,
        temporal = temporal,
        status = status,
        distributions = distributions,
        relations = relations,
        related = related,
        relatedBy = relatedBy,
    )
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun resolveVersionId(
    client: PropertyGraphClient,
    requestedId: String,
    status: String,
): String? {
    val versionId =
        client.runQuery(
            "g.V().hasLabel('Version').has('id', ${gStr(requestedId)}).has('status', ${gStr(status)}).values('id').limit(1).toList()",
        ).firstOrNull()?.get("value")?.toString()
    if (!versionId.isNullOrBlank()) return versionId

    val conceptualFromVersion = if (UriUtils.hasVersion(requestedId)) UriUtils.extractConceptualUri(requestedId) else null
    if (!conceptualFromVersion.isNullOrBlank()) {
        val byIdentifierWithinConceptual = client.runQuery(
            """
            g.V().hasLabel('Conceptual').has('id', ${gStr(conceptualFromVersion)})
              .out('HAS_VERSION').hasLabel('Version')
              .has('identifier', ${gStr(requestedId)})
              .has('status', ${gStr(status)})
              .values('id').limit(1).toList()
            """.trimIndent(),
        ).firstOrNull()?.get("value")?.toString()
        if (!byIdentifierWithinConceptual.isNullOrBlank()) return byIdentifierWithinConceptual
    }

    val currentVersion = client.runQuery(
        """
        g.V().hasLabel('Conceptual').has('id', ${gStr(requestedId)})
          .out('CURRENT_VERSION').hasLabel('Version').has('status', ${gStr(status)})
          .values('id').limit(1).toList()
        """.trimIndent(),
    ).firstOrNull()?.get("value")?.toString()
    if (!currentVersion.isNullOrBlank()) return currentVersion

    return client.runQuery(
        """
        g.V().hasLabel('Conceptual').has('id', ${gStr(requestedId)})
          .out('HAS_VERSION').hasLabel('Version').has('status', ${gStr(status)})
          .values('id').limit(1).toList()
        """.trimIndent(),
    ).firstOrNull()?.get("value")?.toString()
}
private fun prop(value: Any?): Any? =
    when (value) {
        is List<*> -> if (value.size == 1) value.firstOrNull() else value
        else -> value
    }

private fun getDatasetDistributions(data: Map<*, *>, publicBaseUrl: String): List<ResourceRef> {
    val distributions = data["distribution"] ?: data["dcat:distribution"] ?: return emptyList()
    val uris = when (distributions) {
        is String -> listOf(distributions)
        is List<*> -> distributions.mapNotNull { it as? String }
        else -> emptyList()
    }

    return uris.mapNotNull { distUri ->
        ResourceRef(
            id = distUri,
            title = null,
            href = "$publicBaseUrl/api/v2/distribution/${URLEncoder.encode(distUri, "UTF-8")}",
        )
    }
}
