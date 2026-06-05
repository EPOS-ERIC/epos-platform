package org.epos.infrastructure.ktor.routes

import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.server.request.accept
import io.ktor.server.request.contentType
import io.ktor.server.request.receive
import io.ktor.server.request.receiveParameters
import io.ktor.server.request.receiveText
import io.ktor.server.response.respond
import io.ktor.server.response.respondText
import io.ktor.server.routing.Route
import io.ktor.server.routing.post
import io.ktor.server.routing.route
import org.epos.api.models.ErrorResponse
import org.epos.propertygraph.client.PropertyGraphClient

fun Route.cypherRoutes(client: PropertyGraphClient) {
    route("/api/v2/cypher") {
        post {
            val contentType = call.request.contentType()
            val acceptHeader = call.request.accept() ?: "application/json"

            val query: String? =
                when {
                    contentType.match(ContentType.Application.FormUrlEncoded) -> {
                        val params = call.receiveParameters()
                        params["query"]
                    }

                    contentType.match(ContentType("application", "cypher")) -> {
                        call.receiveText()
                    }

                    contentType.match(ContentType.Application.Json) -> {
                        try {
                            val body = call.receive<Map<String, String>>()
                            body["query"]
                        } catch (e: Exception) {
                            null
                        }
                    }

                    else -> {
                        call.receiveText().takeIf { it.isNotBlank() }
                    }
                }

            if (query.isNullOrBlank()) {
                call.respond(
                    HttpStatusCode.BadRequest,
                    ErrorResponse("BadRequest", "Missing or empty 'query' parameter"),
                )
                return@post
            }

            val queryUpper = query.uppercase().trim()
            val writeOperations = listOf("CREATE", "MERGE", "DELETE", "REMOVE", "SET", "DROP")
            if (writeOperations.any { queryUpper.startsWith(it) }) {
                call.respond(
                    HttpStatusCode.Forbidden,
                    ErrorResponse("Forbidden", "Write operations are not allowed on this endpoint"),
                )
                return@post
            }

            try {
                val results = client.runQuery(query)

                val responseFormat =
                    when {
                        acceptHeader.contains("text/csv") -> "csv"
                        else -> "json"
                    }

                if (responseFormat == "csv") {
                    val csvBuilder = StringBuilder()
                    if (results.isNotEmpty()) {
                        csvBuilder.append(results.first().keys.joinToString(",")).append("\n")
                        results.forEach { row ->
                            csvBuilder.append(row.values.joinToString(",") { it?.toString() ?: "" }).append("\n")
                        }
                    }
                    call.respondText(csvBuilder.toString(), ContentType.parse("text/csv"))
                } else {
                    call.respond(results)
                }
            } catch (e: Exception) {
                call.respond(
                    HttpStatusCode.InternalServerError,
                    ErrorResponse("QueryError", "Query evaluation failed: ${e.message}"),
                )
            }
        }
    }
}
