package org.epos.infrastructure.ktor.routes

import io.ktor.http.*
import io.ktor.server.request.*
import io.ktor.server.response.*
import io.ktor.server.routing.*
import org.epos.api.models.ErrorResponse
import org.epos.graphdb.client.GraphDBClient
import org.eclipse.rdf4j.query.QueryLanguage
import org.eclipse.rdf4j.query.resultio.sparqljson.SPARQLResultsJSONWriter
import org.eclipse.rdf4j.query.resultio.sparqlxml.SPARQLResultsXMLWriter
import org.eclipse.rdf4j.query.resultio.text.csv.SPARQLResultsCSVWriter
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import java.io.ByteArrayOutputStream

/**
 * SPARQL endpoint routes for direct query access.
 * Supports SELECT, ASK, CONSTRUCT, and DESCRIBE queries.
 */
fun Route.sparqlRoutes(graphDBClient: GraphDBClient) {
    route("/api/v2/sparql") {
        
        /**
         * POST /api/v2/sparql
         * Execute a SPARQL query.
         * 
         * Accepts:
         * - application/x-www-form-urlencoded with 'query' parameter
         * - application/sparql-query with query in body
         * - application/json with {"query": "..."} 
         * 
         * Returns results in format based on Accept header:
         * - application/sparql-results+json (default for SELECT/ASK)
         * - application/sparql-results+xml
         * - text/csv
         * - application/n-triples (for CONSTRUCT/DESCRIBE)
         * - text/turtle (for CONSTRUCT/DESCRIBE)
         * - application/rdf+xml (for CONSTRUCT/DESCRIBE)
         */
        post {
            val contentType = call.request.contentType()
            val acceptHeader = call.request.accept() ?: "application/sparql-results+json"
            
            // Extract query from different content types
            val query: String? = when {
                contentType.match(ContentType.Application.FormUrlEncoded) -> {
                    val params = call.receiveParameters()
                    params["query"]
                }
                contentType.match(ContentType("application", "sparql-query")) -> {
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
                    // Try to read as plain text
                    call.receiveText().takeIf { it.isNotBlank() }
                }
            }
            
            if (query.isNullOrBlank()) {
                call.respond(
                    HttpStatusCode.BadRequest,
                    ErrorResponse("BadRequest", "Missing or empty 'query' parameter")
                )
                return@post
            }
            
            // Validate query - prevent UPDATE/DELETE operations
            val queryUpper = query.uppercase().trim()
            if (queryUpper.startsWith("INSERT") || 
                queryUpper.startsWith("DELETE") || 
                queryUpper.startsWith("DROP") ||
                queryUpper.startsWith("CLEAR") ||
                queryUpper.startsWith("CREATE") ||
                queryUpper.startsWith("LOAD") ||
                queryUpper.startsWith("COPY") ||
                queryUpper.startsWith("MOVE") ||
                queryUpper.startsWith("ADD")) {
                call.respond(
                    HttpStatusCode.Forbidden,
                    ErrorResponse("Forbidden", "SPARQL UPDATE operations are not allowed on this endpoint")
                )
                return@post
            }
            
            try {
                graphDBClient.repository.connection.use { conn ->
                    // Determine query type
                    val parsedQuery = conn.prepareQuery(QueryLanguage.SPARQL, query)
                    
                    when (parsedQuery) {
                        is org.eclipse.rdf4j.query.TupleQuery -> {
                            // SELECT query
                            val outputStream = ByteArrayOutputStream()
                            val resultContentType = when {
                                acceptHeader.contains("application/sparql-results+xml") -> {
                                    parsedQuery.evaluate(SPARQLResultsXMLWriter(outputStream))
                                    "application/sparql-results+xml"
                                }
                                acceptHeader.contains("text/csv") -> {
                                    parsedQuery.evaluate(SPARQLResultsCSVWriter(outputStream))
                                    "text/csv"
                                }
                                else -> {
                                    parsedQuery.evaluate(SPARQLResultsJSONWriter(outputStream))
                                    "application/sparql-results+json"
                                }
                            }
                            call.respondBytes(
                                outputStream.toByteArray(),
                                ContentType.parse(resultContentType)
                            )
                        }
                        is org.eclipse.rdf4j.query.BooleanQuery -> {
                            // ASK query
                            val result = parsedQuery.evaluate()
                            val jsonResult = """{"head":{},"boolean":$result}"""
                            call.respondText(jsonResult, ContentType.parse("application/sparql-results+json"))
                        }
                        is org.eclipse.rdf4j.query.GraphQuery -> {
                            // CONSTRUCT or DESCRIBE query
                            val outputStream = ByteArrayOutputStream()
                            val (format, contentTypeStr) = when {
                                acceptHeader.contains("text/turtle") -> 
                                    RDFFormat.TURTLE to "text/turtle"
                                acceptHeader.contains("application/rdf+xml") -> 
                                    RDFFormat.RDFXML to "application/rdf+xml"
                                acceptHeader.contains("application/ld+json") -> 
                                    RDFFormat.JSONLD to "application/ld+json"
                                else -> 
                                    RDFFormat.NTRIPLES to "application/n-triples"
                            }
                            parsedQuery.evaluate().use { result ->
                                Rio.write(result, outputStream, format)
                            }
                            call.respondBytes(
                                outputStream.toByteArray(),
                                ContentType.parse(contentTypeStr)
                            )
                        }
                        else -> {
                            call.respond(
                                HttpStatusCode.BadRequest,
                                ErrorResponse("BadRequest", "Unsupported query type")
                            )
                        }
                    }
                }
            } catch (e: org.eclipse.rdf4j.query.MalformedQueryException) {
                call.respond(
                    HttpStatusCode.BadRequest,
                    ErrorResponse("BadRequest", "Malformed SPARQL query: ${e.message}")
                )
            } catch (e: org.eclipse.rdf4j.query.QueryEvaluationException) {
                call.respond(
                    HttpStatusCode.InternalServerError,
                    ErrorResponse("QueryError", "Query evaluation failed: ${e.message}")
                )
            } catch (e: Exception) {
                call.respond(
                    HttpStatusCode.InternalServerError,
                    ErrorResponse("InternalServerError", e.message ?: "Unknown error")
                )
            }
        }
    }
}
