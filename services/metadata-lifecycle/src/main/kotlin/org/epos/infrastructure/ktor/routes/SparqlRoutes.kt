package org.epos.infrastructure.ktor.routes

import io.ktor.http.ContentType
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.contentType
import io.ktor.server.request.receive
import io.ktor.server.request.receiveParameters
import io.ktor.server.request.receiveText
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.post
import io.ktor.server.routing.route
import com.fasterxml.jackson.databind.ObjectMapper
import org.epos.api.models.ErrorResponse
import org.epos.propertygraph.client.PropertyGraphClient

fun Route.sparqlRoutes(client: PropertyGraphClient) {
    route("/api/v2/sparql") {
        post {
            val contentType = call.request.contentType()
            val accept = call.request.headers[HttpHeaders.Accept].orEmpty()

            val query: String? = when {
                contentType.match(ContentType.Application.FormUrlEncoded) -> call.receiveParameters()["query"]
                contentType.match(ContentType.Application.Json) -> runCatching { call.receive<Map<String, String>>() }.getOrNull()?.get("query")
                contentType.match(ContentType.parse("application/sparql-query")) -> call.receiveText()
                else -> call.receiveText().takeIf { it.isNotBlank() }
            }

            if (query.isNullOrBlank()) {
                call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Missing or empty 'query' parameter"))
                return@post
            }

            val normalized = normalizeSparqlQuery(query)
            if (isWriteSparqlQuery(normalized)) {
                call.respond(HttpStatusCode.Forbidden, ErrorResponse("Forbidden", "SPARQL UPDATE operations are not allowed on this endpoint"))
                return@post
            }

            if (!isSupportedReadQuery(normalized)) {
                call.respond(HttpStatusCode.BadRequest, ErrorResponse("BadRequest", "Only SELECT, ASK, CONSTRUCT, and DESCRIBE are supported"))
                return@post
            }

            val responseType = chooseSparqlResponseContentType(accept)
            val rows = runCatching { client.runQuery(normalized) }.getOrElse { error ->
                val translatedRows = runTranslatedSparqlQuery(client, normalized)
                if (translatedRows != null) {
                    val queryType = normalized.substringBefore(' ').uppercase()
                    val body = renderSparqlResponse(queryType, translatedRows, responseType)
                    call.respondText(body, responseType)
                    return@post
                }
                call.respond(HttpStatusCode.BadRequest, ErrorResponse("QueryError", "SPARQL query execution failed: ${error.message}"))
                return@post
            }

            val queryType = normalized.substringBefore(' ').uppercase()
            val body = renderSparqlResponse(queryType, rows, responseType)
            call.respondText(body, responseType)
        }
    }
}

private fun normalizeSparqlQuery(query: String): String =
    query
        .lineSequence()
        .map { it.trim() }
        .filterNot { it.startsWith("#") }
        .joinToString(" ")
        .trim()

private fun isWriteSparqlQuery(normalizedQuery: String): Boolean {
    val upper = normalizedQuery.uppercase()
    return listOf("INSERT", "DELETE", "LOAD", "CLEAR", "CREATE", "DROP", "COPY", "MOVE", "ADD", "WITH")
        .any { op -> upper.startsWith(op) }
}

private fun isSupportedReadQuery(normalizedQuery: String): Boolean {
    val upper = normalizedQuery.uppercase()
    return upper.startsWith("SELECT") ||
        upper.startsWith("ASK") ||
        upper.startsWith("CONSTRUCT") ||
        upper.startsWith("DESCRIBE")
}

private fun chooseSparqlResponseContentType(accept: String): ContentType {
    if (accept.contains("application/sparql-results+xml", ignoreCase = true)) {
        return ContentType.parse("application/sparql-results+xml")
    }
    if (accept.contains("text/csv", ignoreCase = true)) {
        return ContentType.Text.CSV
    }
    return ContentType.parse("application/sparql-results+json")
}

private fun renderSparqlResponse(
    queryType: String,
    rows: List<Map<String, Any?>>,
    responseType: ContentType,
): String {
    return when {
        queryType == "ASK" -> renderAskResponse(rows, responseType)
        responseType.match(ContentType.parse("application/sparql-results+xml")) -> renderXmlRows(rows)
        responseType.match(ContentType.Text.CSV) -> renderCsvRows(rows)
        else -> renderJsonRows(rows)
    }
}

private fun runTranslatedSparqlQuery(
    client: PropertyGraphClient,
    normalizedQuery: String,
): List<Map<String, Any?>>? {
    val askAnyTriple = Regex("""^ASK\s+WHERE\s*\{\s*\?s\s+\?p\s+\?o\s*\}\s*$""", RegexOption.IGNORE_CASE)
    if (askAnyTriple.matches(normalizedQuery)) {
        val count =
            client.runQuery("g.V().limit(1).count().next()")
                .firstOrNull()?.get("value")?.toString()?.toLongOrNull() ?: 0L
        return listOf(mapOf("boolean" to (count > 0)))
    }

    val selectAnyTriple =
        Regex(
            """^SELECT\s+\*\s+WHERE\s*\{\s*\?s\s+\?p\s+\?o\s*\}\s*(LIMIT\s+(\d+))?\s*$""",
            RegexOption.IGNORE_CASE,
        )
    val match = selectAnyTriple.matchEntire(normalizedQuery) ?: return null
    val limit = match.groupValues.getOrNull(2)?.toIntOrNull() ?: 10

    return client.runQuery(
        """
        g.V().as('s').outE().as('e').inV().as('o')
          .select('s','e','o')
          .by(coalesce(values('id'), id()))
          .by(label)
          .by(coalesce(values('id'), id()))
          .limit($limit)
          .toList()
        """.trimIndent(),
    ).map { row ->
        mapOf(
            "s" to row["s"],
            "p" to row["e"],
            "o" to row["o"],
        )
    }
}

private fun renderAskResponse(rows: List<Map<String, Any?>>, responseType: ContentType): String {
    val boolFromValue = rows.firstNotNullOfOrNull { row -> row["boolean"] ?: row["value"] }
    val value = when (boolFromValue) {
        is Boolean -> boolFromValue
        is Number -> boolFromValue.toInt() != 0
        is String -> boolFromValue.equals("true", ignoreCase = true) || boolFromValue == "1"
        else -> rows.isNotEmpty()
    }
    return when {
        responseType.match(ContentType.parse("application/sparql-results+xml")) ->
            """<?xml version="1.0" encoding="UTF-8"?><sparql xmlns="http://www.w3.org/2005/sparql-results#"><head/><boolean>$value</boolean></sparql>"""
        responseType.match(ContentType.Text.CSV) -> "boolean\n$value\n"
        else -> jsonMapper.writeValueAsString(mapOf("head" to emptyMap<String, Any>(), "boolean" to value))
    }
}

private fun renderJsonRows(rows: List<Map<String, Any?>>): String {
    val vars = linkedSetOf<String>()
    rows.forEach { vars.addAll(it.keys) }
    val bindings = rows.map { row ->
        val mapped = mutableMapOf<String, Map<String, String>>()
        row.forEach { (key, value) ->
            if (value != null) {
                mapped[key] = toSparqlBinding(value)
            }
        }
        mapped
    }
    val payload = mapOf(
        "head" to mapOf("vars" to vars.toList()),
        "results" to mapOf("bindings" to bindings),
    )
    return jsonMapper.writeValueAsString(payload)
}

private fun renderXmlRows(rows: List<Map<String, Any?>>): String {
    val vars = linkedSetOf<String>()
    rows.forEach { vars.addAll(it.keys) }

    val headXml = vars.joinToString(separator = "") { variable ->
        "<variable name=\"${xmlEscape(variable)}\"/>"
    }
    val resultsXml = rows.joinToString(separator = "") { row ->
        val bindingsXml = row.entries.joinToString(separator = "") { (key, value) ->
            if (value == null) return@joinToString ""
            val term = toSparqlTerm(value)
            val valueXml = when (term.type) {
                "uri" -> "<uri>${xmlEscape(term.value)}</uri>"
                "bnode" -> "<bnode>${xmlEscape(term.value.removePrefix("_:"))}</bnode>"
                else -> "<literal>${xmlEscape(term.value)}</literal>"
            }
            "<binding name=\"${xmlEscape(key)}\">$valueXml</binding>"
        }
        "<result>$bindingsXml</result>"
    }
    return """<?xml version="1.0" encoding="UTF-8"?><sparql xmlns="http://www.w3.org/2005/sparql-results#"><head>$headXml</head><results>$resultsXml</results></sparql>"""
}

private fun renderCsvRows(rows: List<Map<String, Any?>>): String {
    if (rows.isEmpty()) return ""
    val columns = linkedSetOf<String>()
    rows.forEach { columns.addAll(it.keys) }

    val header = columns.joinToString(",")
    val dataRows = rows.joinToString("\n") { row ->
        columns.joinToString(",") { column -> csvEscape(row[column]?.toString().orEmpty()) }
    }
    return "$header\n$dataRows\n"
}

private fun csvEscape(value: String): String {
    if (!value.contains(',') && !value.contains('"') && !value.contains('\n')) return value
    return "\"${value.replace("\"", "\"\"")}\""
}

private fun xmlEscape(value: String): String =
    value
        .replace("&", "&amp;")
        .replace("<", "&lt;")
        .replace(">", "&gt;")
        .replace("\"", "&quot;")
        .replace("'", "&apos;")

private data class SparqlTerm(
    val type: String,
    val value: String,
)

private fun toSparqlBinding(value: Any): Map<String, String> {
    val term = toSparqlTerm(value)
    return mapOf(
        "type" to term.type,
        "value" to term.value,
    )
}

private fun toSparqlTerm(value: Any): SparqlTerm {
    val raw = value.toString().trim()
    if (raw.startsWith("_:")) {
        return SparqlTerm(type = "bnode", value = raw)
    }
    if (raw.startsWith("http://") || raw.startsWith("https://") || raw.startsWith("urn:")) {
        return SparqlTerm(type = "uri", value = raw)
    }
    return SparqlTerm(type = "literal", value = raw)
}

private val jsonMapper = ObjectMapper()
