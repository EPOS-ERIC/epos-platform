package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.databind.ObjectMapper
import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.header
import io.ktor.server.request.receive
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.delete
import io.ktor.server.routing.get
import io.ktor.server.routing.patch
import io.ktor.server.routing.post
import io.ktor.server.routing.put
import io.ktor.server.routing.route
import org.epos.api.controllers.ResourceController
import org.epos.api.models.ApiResponse
import org.epos.api.models.CreateEntityRequest
import org.epos.api.models.EntityResponse
import org.epos.api.models.ErrorResponse
import org.epos.api.models.ResourceTypeInfo
import org.epos.api.models.ResourceTypesResponse
import org.epos.api.models.UpdateEntityRequest
import org.epos.services.ShaclService
import java.net.URLDecoder

fun Route.resourceTypeRoutes(
    shaclService: ShaclService,
    resourceController: ResourceController,
) {
    get("/api/v2/resource-types") {
        val resourceTypes = shaclService.getResourceTypes()
        resourceController.setResourceType(resourceTypes)

        val view = call.request.queryParameters["view"]?.lowercase() ?: "parsed"
        if (view == "shacl") {
            val model = shaclService.getParser().getCachedModel()
            if (model == null) {
                call.respond(
                    HttpStatusCode.ServiceUnavailable,
                    ErrorResponse("NotReady", "SHACL model is not initialized yet"),
                )
                return@get
            }

            val formatParam = call.request.queryParameters["format"]?.lowercase()
            val compact = call.request.queryParameters["compact"]?.toBoolean() ?: false
            val acceptHeader = call.request.header(HttpHeaders.Accept)?.lowercase()
            val rdfFormat = resolveRdfFormat(formatParam, acceptHeader)
            val contentType = resolveRdfContentType(rdfFormat)
            val body = serializeModel(model, rdfFormat, compact)
            call.respondText(body, contentType)
            return@get
        }

        val includeShapeDefinitions = call.request.queryParameters["includeShapeDefinitions"]?.toBoolean() ?: false
        val includeTransitiveShapeDefinitions = call.request.queryParameters["includeTransitiveShapeDefinitions"]?.toBoolean() ?: false
        val includeResolvedRefs = call.request.queryParameters["includeResolvedRefs"]?.toBoolean() ?: false

        if (!includeShapeDefinitions && !includeTransitiveShapeDefinitions) {
            call.respond(resourceTypes)
            return@get
        }

        val shapeDefinitions =
            shaclService.getParser().getShapeDefinitions(
                onlyReachableFromResourceTypes = includeTransitiveShapeDefinitions,
                includeResolvedRefs = includeResolvedRefs,
            )
        call.respond(ResourceTypesResponse(resourceTypes = resourceTypes, shapeDefinitions = shapeDefinitions))
    }
}

private fun serializeModel(
    model: org.eclipse.rdf4j.model.Model,
    format: org.eclipse.rdf4j.rio.RDFFormat,
    compact: Boolean = false,
): String {
    if (compact) {
        if (format == org.eclipse.rdf4j.rio.RDFFormat.JSONLD) {
            return serializeCompactJsonLd(model)
        }
        if (format == org.eclipse.rdf4j.rio.RDFFormat.TURTLE) {
            return serializeCompactTurtle(model)
        }
    }

    val writer = java.io.StringWriter()
    org.eclipse.rdf4j.rio.Rio
        .write(model, writer, format)
    return writer.toString()
}

private fun serializeCompactTurtle(model: org.eclipse.rdf4j.model.Model): String {
    val namespaces = linkedMapOf<String, String>()
    model.namespaces
        .filter { it.prefix.isNotBlank() }
        .sortedBy { it.prefix }
        .forEach { namespaces[it.prefix] = it.name }

    val sortedStatements = model.sortedBy { statementSortKey(it) }
    val statementsBySubject = sortedStatements.groupBy { it.subject }

    val builder = StringBuilder()
    namespaces.forEach { (prefix, namespace) ->
        builder
            .append("@prefix ")
            .append(prefix)
            .append(": <")
            .append(namespace)
            .append("> .\n")
    }

    if (namespaces.isNotEmpty() && statementsBySubject.isNotEmpty()) {
        builder.append("\n")
    }

    statementsBySubject.entries.sortedBy { it.key.stringValue() }.forEachIndexed { index, (subject, statements) ->
        val byPredicate = statements.groupBy { it.predicate }
        builder.append(formatTurtleValue(subject, namespaces)).append("\n")

        val predicateEntries = byPredicate.entries.sortedBy { it.key.stringValue() }
        predicateEntries.forEachIndexed { predIndex, (predicate, predicateStatements) ->
            builder
                .append("  ")
                .append(formatPredicate(predicate, namespaces))
                .append(" ")

            val objectValues =
                predicateStatements
                    .map { formatTurtleValue(it.`object`, namespaces) }
                    .distinct()
                    .sorted()

            builder.append(objectValues.joinToString(", "))
            if (predIndex < predicateEntries.lastIndex) {
                builder.append(" ;")
            } else {
                builder.append(" .")
            }
            builder.append("\n")
        }

        if (index < statementsBySubject.size - 1) {
            builder.append("\n")
        }
    }

    return builder.toString()
}

private fun serializeCompactJsonLd(model: org.eclipse.rdf4j.model.Model): String {
    val context = linkedMapOf<String, String>()
    model.namespaces
        .filter { it.prefix.isNotBlank() }
        .sortedBy { it.prefix }
        .forEach { namespace -> context[namespace.prefix] = namespace.name }

    val bySubject = linkedMapOf<String, MutableMap<String, Any>>()
    model.sortedBy { statementSortKey(it) }.forEach { statement ->
        val subject =
            when (val subjectValue = statement.subject) {
                is org.eclipse.rdf4j.model.IRI -> compactIri(subjectValue, context)
                is org.eclipse.rdf4j.model.BNode -> "_:${subjectValue.id}"
                else -> statement.subject.stringValue()
            }

        val node =
            bySubject.getOrPut(subject) {
                linkedMapOf<String, Any>("@id" to subject)
            }

        val predicate =
            if (statement.predicate.stringValue() == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type") {
                "@type"
            } else {
                compactIri(statement.predicate, context)
            }

        val value = toCompactJsonLdValue(statement, context, predicate)
        node[predicate] = mergeJsonValue(node[predicate], value)
    }

    val result =
        linkedMapOf<String, Any>(
            "@context" to context,
            "@graph" to bySubject.values.toList(),
        )

    return ObjectMapper().writerWithDefaultPrettyPrinter().writeValueAsString(result)
}

private fun statementSortKey(statement: org.eclipse.rdf4j.model.Statement): String =
    "${statement.subject.stringValue()}|${statement.predicate.stringValue()}|${statement.`object`.stringValue()}"

private fun compactIri(
    iri: org.eclipse.rdf4j.model.IRI,
    context: Map<String, String>,
): String {
    val prefix = context.entries.firstOrNull { (_, namespace) -> namespace == iri.namespace }?.key
    return if (prefix != null) {
        "$prefix:${iri.localName}"
    } else {
        iri.stringValue()
    }
}

private fun toCompactJsonLdValue(
    statement: org.eclipse.rdf4j.model.Statement,
    context: Map<String, String>,
    predicate: String,
): Any =
    when (val obj = statement.`object`) {
        is org.eclipse.rdf4j.model.IRI -> {
            val compacted = compactIri(obj, context)
            if (predicate == "@type") compacted else mapOf("@id" to compacted)
        }

        is org.eclipse.rdf4j.model.BNode -> {
            mapOf("@id" to "_:${obj.id}")
        }

        is org.eclipse.rdf4j.model.Literal -> {
            val language = obj.language.orElse(null)
            val datatype = obj.datatype?.stringValue()
            when {
                language != null -> mapOf("@value" to obj.label, "@language" to language)
                datatype == null || datatype == "http://www.w3.org/2001/XMLSchema#string" -> obj.label
                else -> mapOf("@value" to obj.label, "@type" to compactIri(obj.datatype, context))
            }
        }

        is org.eclipse.rdf4j.model.Value -> {
            obj.stringValue()
        }

        else -> {
            obj.stringValue()
        }
    }

private fun mergeJsonValue(
    current: Any?,
    next: Any,
): Any =
    when (current) {
        null -> next
        is List<*> -> current + next
        else -> listOf(current, next)
    }

private fun formatPredicate(
    predicate: org.eclipse.rdf4j.model.IRI,
    namespaces: Map<String, String>,
): String {
    if (predicate.stringValue() == "http://www.w3.org/1999/02/22-rdf-syntax-ns#type") {
        return "a"
    }
    return compactIri(predicate, namespaces)
}

private fun formatTurtleValue(
    value: org.eclipse.rdf4j.model.Value,
    namespaces: Map<String, String>,
): String =
    when (value) {
        is org.eclipse.rdf4j.model.IRI -> compactIri(value, namespaces)
        is org.eclipse.rdf4j.model.BNode -> "_:${value.id}"
        is org.eclipse.rdf4j.model.Literal -> formatTurtleLiteral(value, namespaces)
        else -> "<${value.stringValue()}>"
    }

private fun formatTurtleLiteral(
    literal: org.eclipse.rdf4j.model.Literal,
    namespaces: Map<String, String>,
): String {
    val escaped =
        literal.label
            .replace("\\", "\\\\")
            .replace("\"", "\\\"")
            .replace("\n", "\\n")
            .replace("\r", "\\r")
            .replace("\t", "\\t")

    val language = literal.language.orElse(null)
    if (language != null) {
        return "\"$escaped\"@$language"
    }

    val datatype = literal.datatype?.stringValue()
    if (datatype == null || datatype == "http://www.w3.org/2001/XMLSchema#string") {
        return "\"$escaped\""
    }

    return "\"$escaped\"^^${compactIri(literal.datatype, namespaces)}"
}

private fun resolveRdfFormat(
    formatParam: String?,
    acceptHeader: String?,
): org.eclipse.rdf4j.rio.RDFFormat =
    when {
        formatParam == "ttl" || formatParam == "turtle" -> org.eclipse.rdf4j.rio.RDFFormat.TURTLE

        formatParam == "jsonld" || formatParam == "json-ld" -> org.eclipse.rdf4j.rio.RDFFormat.JSONLD

        formatParam == "rdfxml" || formatParam == "rdf+xml" || formatParam == "xml" -> org.eclipse.rdf4j.rio.RDFFormat.RDFXML

        formatParam == "ntriples" || formatParam == "n-triples" || formatParam == "nt" -> org.eclipse.rdf4j.rio.RDFFormat.NTRIPLES

        acceptHeader?.contains("text/turtle") == true -> org.eclipse.rdf4j.rio.RDFFormat.TURTLE

        acceptHeader?.contains(
            "application/ld+json",
        ) == true || acceptHeader?.contains("application/json") == true -> org.eclipse.rdf4j.rio.RDFFormat.JSONLD

        acceptHeader?.contains("application/rdf+xml") == true -> org.eclipse.rdf4j.rio.RDFFormat.RDFXML

        acceptHeader?.contains("application/n-triples") == true -> org.eclipse.rdf4j.rio.RDFFormat.NTRIPLES

        else -> org.eclipse.rdf4j.rio.RDFFormat.TURTLE
    }

private fun resolveRdfContentType(format: org.eclipse.rdf4j.rio.RDFFormat): ContentType =
    when (format) {
        org.eclipse.rdf4j.rio.RDFFormat.TURTLE -> ContentType.parse("text/turtle")
        org.eclipse.rdf4j.rio.RDFFormat.JSONLD -> ContentType.parse("application/ld+json")
        org.eclipse.rdf4j.rio.RDFFormat.RDFXML -> ContentType.parse("application/rdf+xml")
        org.eclipse.rdf4j.rio.RDFFormat.NTRIPLES -> ContentType.parse("application/n-triples")
        else -> ContentType.Text.Plain
    }

fun Route.resourceCrudRoutes(
    resourceController: ResourceController,
    shaclService: ShaclService,
) {
    route("/api/v2/{resourceType}") {
        get {
            val resourceType = call.parameters["resourceType"] ?: ""
            if (resourceType.equals("search", ignoreCase = true)) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Endpoint not found"))
                return@get
            }
            if (resourceType.equals("entities", ignoreCase = true)) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Endpoint not found"))
                return@get
            }
            val status = call.request.queryParameters["status"] ?: "published"
            val page = call.request.queryParameters["page"]?.toIntOrNull() ?: 1
            val pageSize = call.request.queryParameters["pageSize"]?.toIntOrNull() ?: 20
            val agent =
                call.request
                    .header("X-Agent")
                    ?.trim()
                    ?.takeIf { it.isNotBlank() }

            val resourceTypes = shaclService.getResourceTypes()
            resourceController.setResourceType(resourceTypes)

            val response = resourceController.listEntities(resourceType, status, page, pageSize, agent)
            call.respond(response)
        }

        post {
            val resourceType = call.parameters["resourceType"] ?: ""
            if (resourceType.equals("entities", ignoreCase = true)) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Endpoint not found"))
                return@post
            }
            val request = call.receive<CreateEntityRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@post

            val response =
                resourceController.createEntity(
                    request.copy(type = resourceType),
                    actorHeaders.agent,
                    actorHeaders.group,
                )
            call.respond(HttpStatusCode.Created, response)
        }

        get("/{id}") {
            val entityUri = decodePathId(call.parameters["id"])
            val resourceType = call.parameters["resourceType"] ?: ""
            if (resourceType.equals("entities", ignoreCase = true)) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Endpoint not found"))
                return@get
            }
            val status = call.request.queryParameters["status"] ?: "published"

            val formatParam = call.request.queryParameters["format"]?.lowercase()
            val acceptHeader = call.request.header("Accept")?.lowercase() ?: "application/json"
            val requestedFormat = formatParam ?: acceptHeader

            val entity =
                resourceController.getEntity(entityUri, status, enrichReferences = true)
                    ?: resourceController.getEntityByConceptualId(entityUri, status, enrichReferences = true)

            if (entity == null) {
                call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                return@get
            }

            if (requestedFormat.contains("turtle") || requestedFormat == "ttl") {
                val ttl = convertEntityToTurtle(entity, shaclService.getResourceTypes())
                if (!ttl.isNullOrEmpty()) {
                    call.respondText(ttl, ContentType.parse("text/turtle"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }

            if (requestedFormat.contains("ld+json") || requestedFormat.contains("json-ld") || requestedFormat == "jsonld") {
                val jsonld = convertEntityToJsonLd(entity, shaclService.getResourceTypes())
                if (!jsonld.isNullOrEmpty()) {
                    call.respondText(jsonld, ContentType.parse("application/ld+json"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }

            if (requestedFormat.contains("rdf+xml") || requestedFormat == "rdfxml" || requestedFormat == "xml") {
                val rdfxml = convertEntityToRdfXml(entity, shaclService.getResourceTypes())
                if (!rdfxml.isNullOrEmpty()) {
                    call.respondText(rdfxml, ContentType.parse("application/rdf+xml"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }

            if (requestedFormat.contains("n-triples") || requestedFormat.contains("ntriples") || requestedFormat == "nt") {
                val ntriples = convertEntityToNTriples(entity, shaclService.getResourceTypes())
                if (!ntriples.isNullOrEmpty()) {
                    call.respondText(ntriples, ContentType.parse("application/n-triples"))
                } else {
                    call.respond(HttpStatusCode.NotFound, ErrorResponse("NotFound", "Entity not found"))
                }
                return@get
            }

            if (requestedFormat.contains("json") || requestedFormat == "*/*") {
                call.respond(entity)
                return@get
            }

            call.respond(
                HttpStatusCode.NotAcceptable,
                ErrorResponse(
                    "NotAcceptable",
                    "Unsupported format. Use ?format= with: json, turtle (or ttl), jsonld, rdfxml (or xml), ntriples (or nt)",
                ),
            )
        }

        put("/{id}") {
            val entityUri = decodePathId(call.parameters["id"])
            val request = call.receive<UpdateEntityRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@put

            val response = resourceController.updateEntity(entityUri, request)
            call.respondMutationResponse(response)
        }

        patch("/{id}") {
            val entityUri = decodePathId(call.parameters["id"])
            val request = call.receive<UpdateEntityRequest>()
            val actorHeaders = call.requireActorHeaders() ?: return@patch

            val response = resourceController.patchEntity(entityUri, request)
            call.respondMutationResponse(response)
        }

        delete("/{id}") {
            val entityUri = decodePathId(call.parameters["id"])
            call.requireActorHeaders() ?: return@delete
            val response = resourceController.deleteEntity(entityUri)
            call.respondMutationResponse(response)
        }
    }
}

private fun gStr(value: String): String = "'${value.replace("\\", "\\\\").replace("'", "\\'")}'"

private fun decodePathId(raw: String?): String {
    val value = raw?.trim().orEmpty()
    if (value.isBlank()) return ""
    return runCatching { URLDecoder.decode(value, "UTF-8") }.getOrDefault(value)
}

private suspend fun io.ktor.server.application.ApplicationCall.respondMutationResponse(response: ApiResponse) {
    if (response.success) {
        respond(response)
        return
    }

    val message = response.message.lowercase()
    val status =
        when {
            message.contains("not found") -> HttpStatusCode.NotFound
            message.contains("create/use a draft") || message.contains("only drafts") -> HttpStatusCode.Conflict
            else -> HttpStatusCode.BadRequest
        }
    respond(status, response)
}

internal fun convertEntityToTurtle(
    entity: EntityResponse,
    resourceTypes: List<ResourceTypeInfo>,
): String {
    val predicateMap = predicateMapForEntity(entity, resourceTypes)
    val builder = StringBuilder()
    builder.append("@prefix dcat: <http://www.w3.org/ns/dcat#> .\n")
    builder.append("@prefix dct: <http://purl.org/dc/terms/> .\n")
    builder.append("@prefix pav: <http://purl.org/pav/> .\n\n")

    builder.append("<${entity.uri}>\n")
    entity.properties.forEach { (key, value) ->
        val predicate = if (key == "type") "a" else predicateMap[key] ?: "dct:$key"
        val objectTokens = toRdfObjectTokens(value)
        if (objectTokens.isNotEmpty()) {
            builder.append("  $predicate ${objectTokens.joinToString(", ")} ;\n")
        }
    }
    builder.append("  dct:isVersionOf <${entity.conceptualUri}> .\n")
    return builder.toString()
}

internal fun convertEntityToTurtle(entity: EntityResponse): String = convertEntityToTurtle(entity, emptyList())

internal fun convertEntityToJsonLd(entity: EntityResponse): String {
    val mapper = ObjectMapper()
    val node =
        mutableMapOf<String, Any>(
            "@context" to
                mapOf(
                    "dcat" to "http://www.w3.org/ns/dcat#",
                    "dct" to "http://purl.org/dc/terms/",
                    "pav" to "http://purl.org/pav/",
                ),
            "@id" to entity.uri,
        )
    entity.properties.forEach { (key, value) ->
        node["dct:$key"] = value
    }
    node["dct:isVersionOf"] = mapOf("@id" to entity.conceptualUri)
    return mapper.writerWithDefaultPrettyPrinter().writeValueAsString(node)
}

internal fun convertEntityToJsonLd(
    entity: EntityResponse,
    resourceTypes: List<ResourceTypeInfo>,
): String {
    val predicateMap = predicateMapForEntity(entity, resourceTypes)
    val mapper = ObjectMapper()
    val node =
        mutableMapOf<String, Any>(
            "@context" to
                mapOf(
                    "dcat" to "http://www.w3.org/ns/dcat#",
                    "dct" to "http://purl.org/dc/terms/",
                    "pav" to "http://purl.org/pav/",
                ),
            "@id" to entity.uri,
        )
    entity.properties.forEach { (key, value) ->
        val predicate = predicateMap[key] ?: "dct:$key"
        node[predicate] = value
    }
    node[predicateMap["isVersionOf"] ?: "dct:isVersionOf"] = mapOf("@id" to entity.conceptualUri)
    return mapper.writerWithDefaultPrettyPrinter().writeValueAsString(node)
}

internal fun convertEntityToRdfXml(
    entity: EntityResponse,
    resourceTypes: List<ResourceTypeInfo>,
): String {
    val predicateMap = predicateMapForEntity(entity, resourceTypes)
    return "<rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n" +
        "  <rdf:Description rdf:about=\"${entity.uri}\">\n" +
        entity.properties
            .flatMap { (key, value) ->
                toRdfValues(value).map { item ->
                    val text =
                        item
                            .toString()
                            .replace("&", "&amp;")
                            .replace("<", "&lt;")
                            .replace(">", "&gt;")
                    "    <${predicateMap[key] ?: "dct:$key"}>$text</${predicateMap[key] ?: "dct:$key"}>"
                }
            }.joinToString("\n") +
        "\n  </rdf:Description>\n</rdf:RDF>"
}

internal fun convertEntityToRdfXml(entity: EntityResponse): String = convertEntityToRdfXml(entity, emptyList())

internal fun convertEntityToNTriples(
    entity: EntityResponse,
    resourceTypes: List<ResourceTypeInfo>,
): String {
    val predicateMap = predicateMapForEntity(entity, resourceTypes)
    return entity.properties
        .flatMap { (key, value) ->
            toRdfObjectTokens(value).map { obj ->
                val predicate = predicateMap[key] ?: "dct:$key"
                val predicateIri =
                    if (predicate.contains(":")) {
                        when (predicate.substringBefore(':')) {
                            "dct" -> "http://purl.org/dc/terms/${predicate.substringAfter(':')}"
                            "dcat" -> "http://www.w3.org/ns/dcat#${predicate.substringAfter(':')}"
                            "pav" -> "http://purl.org/pav/${predicate.substringAfter(':')}"
                            else -> predicate
                        }
                    } else {
                        predicate
                    }
                "<${entity.uri}> <$predicateIri> $obj ."
            }
        }.joinToString("\n") + "\n" +
        "<${entity.uri}> <http://purl.org/dc/terms/isVersionOf> <${entity.conceptualUri}> ."
}

internal fun convertEntityToNTriples(entity: EntityResponse): String = convertEntityToNTriples(entity, emptyList())

private fun predicateMapForEntity(
    entity: EntityResponse,
    resourceTypes: List<ResourceTypeInfo>,
): Map<String, String> {
    val normalizedType = entity.type.lowercase().replace("-", "")
    val typeInfo =
        resourceTypes.firstOrNull {
            it.name.lowercase().replace("-", "") == normalizedType
        } ?: return emptyMap()

    return typeInfo.properties.associate { property ->
        property.name to compactPath(property.path)
    }
}

private fun compactPath(path: String): String =
    when {
        path.startsWith("http://purl.org/dc/terms/") -> "dct:${path.substringAfterLast('/')}"
        path.startsWith("http://www.w3.org/ns/dcat#") -> "dcat:${path.substringAfter('#')}"
        path.startsWith("http://purl.org/pav/") -> "pav:${path.substringAfterLast('/')}"
        else -> "dct:${path.substringAfterLast('/').substringAfterLast('#')}"
    }

private fun toRdfValues(value: Any?): List<Any?> =
    when (value) {
        null -> emptyList()
        is List<*> -> value.flatMap { toRdfValues(it) }
        else -> listOf(value)
    }

private fun toRdfObjectTokens(value: Any?): List<String> =
    toRdfValues(value).map { item ->
        val text = item?.toString() ?: ""
        if (text.startsWith("http://") || text.startsWith("https://")) {
            "<$text>"
        } else {
            val escaped = text.replace("\\", "\\\\").replace("\"", "\\\"")
            "\"$escaped\""
        }
    }
