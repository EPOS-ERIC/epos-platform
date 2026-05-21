package org.epos.infrastructure.ktor.routes.search

import io.ktor.http.HttpStatusCode
import io.ktor.server.response.respond
import io.ktor.server.routing.Route
import io.ktor.server.routing.get
import org.epos.api.models.ErrorResponse
import org.epos.infrastructure.ktor.routes.SoftwareApplicationDetail
import org.epos.infrastructure.ktor.routes.SoftwareSourceCodeDetail
import org.epos.infrastructure.ktor.routes.buildImmediateRelationGraph
import org.epos.infrastructure.ktor.routes.buildRelatedByResources
import org.epos.infrastructure.ktor.routes.buildRelatedResources
import org.epos.infrastructure.ktor.routes.extractCategories
import org.epos.infrastructure.ktor.routes.extractKeywords
import org.epos.infrastructure.ktor.routes.extractTypeName
import org.epos.infrastructure.ktor.routes.resolvePublicBaseUrl
import org.epos.propertygraph.client.PropertyGraphClient
import java.net.URLDecoder

fun Route.softwareRoutes(client: PropertyGraphClient) {
    get("/api/v2/software-application/{id}") {
        val appId = call.parameters["id"]
        if (appId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing software application ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(appId, "UTF-8")
            } catch (_: Exception) {
                appId
            }

        val resolvedId = resolveSoftwareVersionId(client, decodedId)
        if (resolvedId == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Software application not found: $decodedId"))
            return@get
        }

        try {
            val application =
                buildSoftwareApplicationDetail(
                    client = client,
                    entityUri = resolvedId,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(application)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch software application"),
            )
        }
    }

    get("/api/v2/software-source-code/{id}") {
        val codeId = call.parameters["id"]
        if (codeId.isNullOrBlank()) {
            call.respond(HttpStatusCode.BadRequest, ErrorResponse("BAD_REQUEST", "Missing software source code ID"))
            return@get
        }

        val decodedId =
            try {
                URLDecoder.decode(codeId, "UTF-8")
            } catch (_: Exception) {
                codeId
            }

        val resolvedId = resolveSoftwareVersionId(client, decodedId)
        if (resolvedId == null) {
            call.respond(HttpStatusCode.NotFound, ErrorResponse("RESOURCE_NOT_FOUND", "Software source code not found: $decodedId"))
            return@get
        }

        try {
            val sourceCode =
                buildSoftwareSourceCodeDetail(
                    client = client,
                    entityUri = resolvedId,
                    publicBaseUrl = call.resolvePublicBaseUrl(),
                )
            call.respond(sourceCode)
        } catch (e: Exception) {
            call.respond(
                HttpStatusCode.InternalServerError,
                ErrorResponse("INTERNAL_SERVER_ERROR", e.message ?: "Failed to fetch software source code"),
            )
        }
    }
}

private fun buildSoftwareApplicationDetail(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): SoftwareApplicationDetail {
    val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()

    @Suppress("UNCHECKED_CAST")
    val rawMap = (result?.get("value") as? Map<String, Any?>) ?: (result as? Map<String, Any?>) ?: emptyMap()
    val v = rawMap.mapValues { (_, vv) -> prop(vv) }
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v.filterKeys { it !in structuralKeys }

    val title = data["title"] as? String ?: data["name"] as? String ?: extractTypeName(entityUri)
    val description = data["description"] as? String
    val downloadUrl = data["downloadUrl"] as? String ?: data["installUrl"] as? String

    val keywords = extractKeywords(data).takeIf { it.isNotEmpty() }
    val categories = extractCategories(data).takeIf { it.isNotEmpty() }

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

    return SoftwareApplicationDetail(
        id = entityUri,
        title = title,
        description = description,
        keywords = keywords,
        categories = categories,
        downloadUrl = downloadUrl,
        relations = relations,
        related = related,
        relatedBy = relatedBy,
    )
}

private fun buildSoftwareSourceCodeDetail(
    client: PropertyGraphClient,
    entityUri: String,
    publicBaseUrl: String,
): SoftwareSourceCodeDetail {
    val result = client.runQuery("g.V().hasLabel('Version').has('id', ${gStr(entityUri)}).valueMap(true).limit(1).toList()").firstOrNull()

    @Suppress("UNCHECKED_CAST")
    val rawMap = (result?.get("value") as? Map<String, Any?>) ?: (result as? Map<String, Any?>) ?: emptyMap()
    val v2 = rawMap.mapValues { (_, vv) -> prop(vv) }
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val data = v2.filterKeys { it !in structuralKeys }

    val title = data["title"] as? String ?: data["name"] as? String ?: extractTypeName(entityUri)
    val description = data["description"] as? String
    val programmingLanguage = data["programmingLanguage"] as? String
    val repository = data["codeRepository"] as? String

    val keywords = extractKeywords(data).takeIf { it.isNotEmpty() }
    val categories = extractCategories(data).takeIf { it.isNotEmpty() }

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

    return SoftwareSourceCodeDetail(
        id = entityUri,
        title = title,
        description = description,
        programmingLanguage = programmingLanguage,
        keywords = keywords,
        categories = categories,
        repository = repository,
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

private fun resolveSoftwareVersionId(
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
