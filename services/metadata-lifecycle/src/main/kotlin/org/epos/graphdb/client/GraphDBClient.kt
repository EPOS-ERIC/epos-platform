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
     * Esegue una query SPARQL SELECT e ritorna i risultati come lista di mappe
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
     * Esegue una query SPARQL ASK
     */
    fun ask(sparqlQuery: String): Boolean {
        return repository.connection.use { conn ->
            val query = conn.prepareBooleanQuery(sparqlQuery)
            query.evaluate()
        }
    }

    /**
     * Esegue un SPARQL UPDATE usando l'API REST diretta di GraphDB
     */
    fun update(sparqlUpdate: String) {
        runBlocking {
            try {
                // GraphDB usa il parametro "query" sia per SELECT che per UPDATE
                val url = Config.SPARQL_ENDPOINT

                val response = httpClient.submitForm(
                    url = url,
                    formParameters = Parameters.build {
                        append("query", sparqlUpdate)
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
     * Inserisce un Model RDF in un Named Graph specifico usando l'API REST di GraphDB
     */
    fun insertIntoGraph(model: Model, graphUri: String) {
        if (model.isEmpty()) {
            println("  ⚠️  Model is empty, skipping insert")
            return
        }

        println("  📦 Inserting ${model.size} triples into graph...")

        // Usa l'API REST diretta di GraphDB per il POST
        runBlocking {
            try {
                // Converti il model in Turtle (più compatto di N-Triples)
                val stringWriter = StringWriter()
                Rio.write(model, stringWriter, RDFFormat.TURTLE)
                val rdfData = stringWriter.toString()

                // POST all'endpoint /statements con il parametro context per il Named Graph
                // GraphDB richiede il formato N-Triples per il context: <URI>
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
     * Metodo deprecato - mantenuto per retrocompatibilità
     */
    @Deprecated("Use insertIntoGraph with HTTP client")
    private fun insertBatch(model: Model, graphUri: String) {
        // Questo metodo non è più usato
    }

    /**
     * Cancella tutte le triple di un soggetto da un Named Graph usando HTTP DELETE
     */
    fun deleteFromGraph(subjectUri: String, graphUri: String) {
        runBlocking {
            try {
                // GraphDB REST API per DELETE
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
     * Sposta tutte le triple di un soggetto da un grafo a un altro
     */
    fun moveSubject(subjectUri: String, fromGraph: String, toGraph: String) {
        // 1. Leggi le triple dal grafo di origine
        val model = getSubjectTriples(subjectUri, fromGraph)

        if (model.isEmpty()) {
            println("  ⚠️  No triples found for $subjectUri in $fromGraph")
            return
        }

        // 2. Inserisci nel grafo di destinazione
        insertIntoGraph(model, toGraph)

        // 3. Cancella dal grafo di origine
        deleteFromGraph(subjectUri, fromGraph)
    }

    /**
     * Legge tutte le triple di un soggetto da un grafo
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
