package org.epos.graphdb.client

import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.request.forms.*
import io.ktor.client.statement.*
import io.ktor.http.*
import kotlinx.coroutines.runBlocking
import org.eclipse.rdf4j.model.Model
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.repository.Repository
import org.eclipse.rdf4j.repository.sparql.SPARQLRepository
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import org.epos.graphdb.Config
import java.io.StringWriter

class GraphDBClient(private val httpClient: HttpClient) {
    val repository: Repository = SPARQLRepository(Config.SPARQL_ENDPOINT).apply {
        init()
    }

    /**
     * Executes a SPARQL SELECT query and returns rows as maps
     */
    fun query(sparqlQuery: String): List<Map<String, String>> {
        return repository.connection.use { conn ->
            val query = conn.prepareTupleQuery(sparqlQuery)
            val results = mutableListOf<Map<String, String>>()

            query.evaluate().use { result ->
                while (result.hasNext()) {
                    val binding = result.next()
                    val row = binding.bindingNames.associateWith { name ->
                        binding.getValue(name)?.stringValue() ?: ""
                    }
                    results.add(row)
                }
            }
            results
        }
    }

    /**
     * Executes a SPARQL ASK query
     */
    fun ask(sparqlQuery: String): Boolean {
        return repository.connection.use { conn ->
            val query = conn.prepareBooleanQuery(sparqlQuery)
            query.evaluate()
        }
    }

    /**
     * Executes a SPARQL UPDATE through the GraphDB REST API
     */
    fun update(sparqlUpdate: String) {
        runBlocking {
            try {
                // GraphDB expects UPDATE statements in the "update" form parameter.
                val url = Config.SPARQL_ENDPOINT

                val response = httpClient.submitForm(
                    url = url,
                    formParameters = Parameters.build {
                        append("update", sparqlUpdate)
                    }
                )

                if (!response.status.isSuccess()) {
                    val errorBody = response.bodyAsText()
                    println("  ❌ SPARQL UPDATE Error: ${response.status}")
                    println("  Response: $errorBody")
                    if (sparqlUpdate.length > 500) {
                        println("  Query preview: ${sparqlUpdate.take(500)}...")
                    } else {
                        println("  Query: $sparqlUpdate")
                    }
                    throw Exception("SPARQL UPDATE failed: ${response.status}")
                }
            } catch (e: Exception) {
                println("  ❌ SPARQL UPDATE Error: ${e.message}")
                if (sparqlUpdate.length > 500) {
                    println("  Query preview: ${sparqlUpdate.take(500)}...")
                } else {
                    println("  Query: $sparqlUpdate")
                }
                throw e
            }
        }
    }

    /**
     * Inserts an RDF model into a specific named graph via GraphDB REST API
     */
    fun insertIntoGraph(model: Model, graphUri: String) {
        if (model.isEmpty()) {
            println("  ⚠️  Model is empty, skipping insert")
            return
        }

        println("  📦 Inserting ${model.size} triples into graph...")

        // Use GraphDB REST API for POST requests.
        runBlocking {
            try {
                // Serialize model as Turtle (more compact than N-Triples).
                val stringWriter = StringWriter()
                Rio.write(model, stringWriter, RDFFormat.TURTLE)
                val rdfData = stringWriter.toString()

                // POST to /statements with context parameter for named graph.
                // GraphDB expects N-Triples syntax for context values: <URI>
                val contextParam = "<$graphUri>"
                val url = "${Config.SPARQL_UPDATE_ENDPOINT}?context=${java.net.URLEncoder.encode(contextParam, "UTF-8")}"

                val response = httpClient.post(url) {
                    contentType(ContentType("text", "turtle"))
                    setBody(rdfData)
                }

                if (response.status.isSuccess()) {
                    println("  ✅ Successfully inserted ${model.size} triples")
                } else {
                    val errorBody = response.bodyAsText()
                    println("  ❌ Error inserting data: ${response.status}")
                    println("  Response: $errorBody")
                    throw Exception("Failed to insert data: ${response.status}")
                }
            } catch (e: Exception) {
                println("  ❌ Error during HTTP insert: ${e.message}")
                throw e
            }
        }
    }

    /**
     * Deprecated method retained for backward compatibility
     */
    @Deprecated("Use insertIntoGraph with HTTP client")
    private fun insertBatch(model: Model, graphUri: String) {
        // This method is no longer used.
    }

    /**
     * Deletes all triples for a subject from a named graph using HTTP DELETE
     */
    fun deleteFromGraph(subjectUri: String, graphUri: String) {
        runBlocking {
            try {
                // Use GraphDB REST API for DELETE requests.
                val contextParam = "<$graphUri>"
                val subjParam = "<$subjectUri>"
                val url = "${Config.SPARQL_UPDATE_ENDPOINT}?" +
                        "context=${java.net.URLEncoder.encode(contextParam, "UTF-8")}&" +
                        "subj=${java.net.URLEncoder.encode(subjParam, "UTF-8")}"

                val response = httpClient.delete(url)

                if (!response.status.isSuccess()) {
                    val errorBody = response.bodyAsText()
                    println("  ❌ DELETE Error: ${response.status}")
                    println("  Response: $errorBody")
                    throw Exception("DELETE failed: ${response.status}")
                }
            } catch (e: Exception) {
                println("  ❌ Error deleting from graph: ${e.message}")
                throw e
            }
        }
    }

    /**
     * Moves all triples of a subject from one graph to another
     */
    fun moveSubject(subjectUri: String, fromGraph: String, toGraph: String) {
        // 1. Read triples from source graph.
        val model = getSubjectTriples(subjectUri, fromGraph)

        if (model.isEmpty()) {
            println("  ⚠️  No triples found for $subjectUri in $fromGraph")
            return
        }

        // 2. Insert triples into target graph.
        insertIntoGraph(model, toGraph)

        // 3. Delete triples from source graph.
        deleteFromGraph(subjectUri, fromGraph)
    }

    /**
     * Reads all triples of a subject from a graph
     */
    fun getSubjectTriples(subjectUri: String, graphUri: String): Model {
        repository.connection.use { conn ->
            val subject = conn.valueFactory.createIRI(subjectUri)
            val graph = conn.valueFactory.createIRI(graphUri)
            return conn.getStatements(subject, null, null, graph).use { statements ->
                val model = LinkedHashModel()
                statements.forEach { model.add(it) }
                model
            }
        }
    }

    fun close() {
        repository.shutDown()
    }
}
