package org.epos.tests

import io.ktor.client.*
import io.ktor.client.engine.cio.*
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.junit.jupiter.api.AfterAll
import org.junit.jupiter.api.BeforeAll
import org.testcontainers.containers.GenericContainer
import org.testcontainers.containers.wait.strategy.Wait
import org.testcontainers.utility.DockerImageName
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder
import java.nio.charset.StandardCharsets
import java.time.Duration

abstract class BaseIntegrationTest {

    companion object {
        private const val GRAPHDB_PORT = 7200

        class GraphDBContainer : GenericContainer<GraphDBContainer>(
            DockerImageName.parse("ontotext/graphdb:10.8.1")
        ) {
            init {
                withExposedPorts(GRAPHDB_PORT)
                withEnv("GDB_HEAP_SIZE", "1g")
                withEnv("GDB_MIN_MEM", "512m")
                withEnv("GDB_MAX_MEM", "1g")
                waitingFor(
                    Wait.forHttp("/rest/repositories")
                        .forStatusCode(200)
                        .withStartupTimeout(Duration.ofMinutes(3))
                )
            }
        }

        val graphDBContainer = GraphDBContainer()

        lateinit var httpClient: HttpClient
        lateinit var graphDBClient: GraphDBClient
        lateinit var testEndpoint: String

        @JvmStatic
        @BeforeAll
        fun setupContainer() {
            println("Starting GraphDB container...")
            graphDBContainer.start()

            val host = graphDBContainer.host
            val port = graphDBContainer.getMappedPort(GRAPHDB_PORT)
            testEndpoint = "http://$host:$port"

            println("GraphDB available at: $testEndpoint")

            httpClient = HttpClient(CIO)

            createTestRepository()

            System.setProperty("GRAPHDB_ENDPOINT", testEndpoint)
            System.setProperty("REPOSITORY_ID", "test-repo")

            graphDBClient = GraphDBClient(httpClient)

            println("Test setup complete")
        }

        @JvmStatic
        @AfterAll
        fun teardownContainer() {
            try {
                if (::graphDBClient.isInitialized) {
                    graphDBClient.close()
                }
                if (::httpClient.isInitialized) {
                    httpClient.close()
                }
            } catch (e: Exception) {
                println("Error during cleanup: ${e.message}")
            } finally {
                graphDBContainer.stop()
            }
        }

        private fun createTestRepository() {
            try {
                Thread.sleep(3000)

                // NativeStore configuration - supports SPARQL UPDATE
                val config = """
@prefix rdfs: <http://www.w3.org/2000/01/rdf-schema#>.
@prefix rep: <http://www.openrdf.org/config/repository#>.
@prefix sr: <http://www.openrdf.org/config/repository/sail#>.
@prefix sail: <http://www.openrdf.org/config/sail#>.
@prefix ns: <http://www.openrdf.org/config/sail/native#>.

[] a rep:Repository ;
   rep:repositoryID "test-repo" ;
   rdfs:label "Test Repository for Integration Tests" ;
   rep:repositoryImpl [
      rep:repositoryType "openrdf:SailRepository" ;
      sr:sailImpl [
         sail:sailType "openrdf:NativeStore" ;
         ns:tripleIndexes "spoc,posc"
      ]
   ].
""".trimIndent()

                val boundary = "----Boundary${System.currentTimeMillis()}"
                val url = URL("$testEndpoint/rest/repositories")
                val connection = url.openConnection() as HttpURLConnection
                connection.requestMethod = "POST"
                connection.doOutput = true
                connection.setRequestProperty("Content-Type", "multipart/form-data; boundary=$boundary")

                connection.outputStream.use { os ->
                    val writer = os.bufferedWriter(StandardCharsets.UTF_8)

                    writer.write("--$boundary\r\n")
                    writer.write("Content-Disposition: form-data; name=\"config\"; filename=\"repo-config.ttl\"\r\n")
                    writer.write("Content-Type: text/turtle\r\n")
                    writer.write("\r\n")
                    writer.write(config)
                    writer.write("\r\n")
                    writer.write("--$boundary--\r\n")
                    writer.flush()
                }

                val responseCode = connection.responseCode
                val responseBody = if (responseCode in 200..299) {
                    connection.inputStream?.bufferedReader()?.use { it.readText() } ?: ""
                } else {
                    connection.errorStream?.bufferedReader()?.use { it.readText() } ?: ""
                }

                when {
                    responseCode in 200..299 -> {
                        println("✓ Test repository created successfully")
                    }
                    responseBody.contains("already exists") || responseBody.contains("A repository") -> {
                        println("✓ Test repository already exists")
                    }
                    else -> {
                        println("❌ Repository creation failed: $responseCode")
                        println("Response: $responseBody")
                        throw Exception("Failed to create repository: $responseCode - $responseBody")
                    }
                }

                connection.disconnect()
                Thread.sleep(5000)

                val verifyUrl = URL("$testEndpoint/rest/repositories/test-repo")
                val verifyConnection = verifyUrl.openConnection() as HttpURLConnection
                verifyConnection.requestMethod = "GET"
                verifyConnection.setRequestProperty("Accept", "application/json")
                val verifyCode = verifyConnection.responseCode
                verifyConnection.disconnect()

                if (verifyCode == 200) {
                    println("✓ Repository verified and ready")
                } else {
                    throw Exception("Repository verification failed: $verifyCode")
                }

            } catch (e: Exception) {
                println("❌ Error creating test repository: ${e.message}")
                e.printStackTrace()
                throw e
            }
        }

        fun clearAllGraphs() {
            val graphs = listOf(
                Config.GRAPH_CONCEPTUAL,
                Config.GRAPH_DRAFT,
                Config.GRAPH_SUBMITTED,
                Config.GRAPH_PUBLISHED,
                Config.GRAPH_ARCHIVED,
                Config.GRAPH_PROVENANCE
            )

            graphs.forEach { graph ->
                try {
                    val contextParam = URLEncoder.encode("<$graph>", "UTF-8")
                    val url = "$testEndpoint/repositories/test-repo/statements?context=$contextParam"
                    val connection = URL(url).openConnection() as HttpURLConnection
                    connection.requestMethod = "DELETE"
                    connection.responseCode // Execute
                    connection.disconnect()
                } catch (e: Exception) {
                    // Ignore
                }
            }
        }
    }
}
