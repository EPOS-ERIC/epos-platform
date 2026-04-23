package org.epos.tests

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import com.fasterxml.jackson.module.kotlin.readValue
import io.kotest.matchers.shouldBe
import io.kotest.matchers.string.shouldContain
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.server.testing.*
import org.epos.module
import org.junit.jupiter.api.*

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ApplicationIntegrationTest : BaseIntegrationTest() {

    private val objectMapper = jacksonObjectMapper()

    @BeforeEach
    fun setup() {
        clearAllGraphs()
    }

    @Test
    fun `health endpoint should return OK`() = testApplication {
        application {
            module()
        }

        val response = client.get("/health")

        response.status shouldBe HttpStatusCode.OK
        response.bodyAsText() shouldContain "ok"
    }

    @Test
    fun `should create draft entity`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/dataset") {
            contentType(ContentType.Application.Json)
            header("X-Agent", "mailto:test@example.org")
            setBody("""
                {
                    "type": "dataset",
                    "properties": {
                        "http://purl.org/dc/terms/title": "Test Dataset",
                        "http://purl.org/dc/terms/description": "A test dataset"
                    }
                }
            """.trimIndent())
        }

        response.status shouldBe HttpStatusCode.Created

        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json["success"] shouldBe true

        @Suppress("UNCHECKED_CAST")
        val data = json["data"] as? Map<String, Any>
        data?.get("status") shouldBe "draft"
    }

    @Test
    fun `should get entity as JSON`() = testApplication {
        application {
            module()
        }

        // Insert a dataset directly into published graph for testing
        val datasetUri = "http://example.org/dataset/test-json-1"
        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
        val dataset = vf.createIRI(datasetUri)
        model.add(dataset, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(dataset, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Test Dataset"))
        graphDBClient.insertIntoGraph(model, org.epos.graphdb.Config.GRAPH_PUBLISHED)

        // Get the entity using the new typed endpoint
        val encodedEntityUri = java.net.URLEncoder.encode(datasetUri, "UTF-8")
        val response = client.get("/api/v2/dataset/$encodedEntityUri")

        response.status shouldBe HttpStatusCode.OK
        response.bodyAsText() shouldContain "Test Dataset"
    }

    @Test
    fun `should get entity via CRUD endpoint as Turtle`() = testApplication {
        application {
            module()
        }

        // Create entity via CRUD endpoint
        val createResponse = client.post("/api/v2/dataset") {
            contentType(ContentType.Application.Json)
            setBody("""
                {
                    "type": "dataset",
                    "properties": {
                        "http://purl.org/dc/terms/title": "Test Dataset"
                    }
                }
            """.trimIndent())
        }

        val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

        @Suppress("UNCHECKED_CAST")
        val data = createJson["data"] as? Map<String, Any>
        val entityUri = data?.get("uri") as? String

        // Get as Turtle - newly created entities are in DRAFT status
        val encodedEntityUri = java.net.URLEncoder.encode(entityUri, "UTF-8")
        val response = client.get("/api/v2/dataset/$encodedEntityUri?format=turtle&status=draft")

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("text/turtle")
        response.bodyAsText() shouldContain "http://purl.org/dc/terms/title"
    }

    @Test
    fun `should return 404 for non-existent entity`() = testApplication {
        application {
            module()
        }

        val missingEntity = java.net.URLEncoder.encode("http://example.org/nonexistent", "UTF-8")
        val response = client.get("/api/v2/dataset/$missingEntity")

        response.status shouldBe HttpStatusCode.NotFound
    }
}
