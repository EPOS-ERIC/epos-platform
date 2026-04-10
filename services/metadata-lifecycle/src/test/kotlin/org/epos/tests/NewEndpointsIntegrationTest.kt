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

/**
 * Integration tests for the new endpoints:
 * - Search API (/api/v2/search, /api/v2/details/{id})
 * - SPARQL endpoint (/api/v2/sparql)
 * - OAI-PMH endpoint (/oai-pmh)
 */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class NewEndpointsIntegrationTest : BaseIntegrationTest() {

    private val objectMapper = jacksonObjectMapper()

    @BeforeEach
    fun setup() {
        clearAllGraphs()
    }

    // =========================================================================
    // Search API Tests
    // =========================================================================

    @Test
    fun `search endpoint should return empty results when database is empty`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/search")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json["total"] shouldBe 0
        @Suppress("UNCHECKED_CAST")
        val results = json["results"] as List<*>
        results.size shouldBe 0
    }

    @Test
    fun `search endpoint should accept query parameters`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/search?q=test&type=dataset&page=1&pageSize=10")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json["page"] shouldBe 1
        json["pageSize"] shouldBe 10
    }

    @Test
    fun `search endpoint should return facets when requested`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/search?facets=true")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json.containsKey("facets") shouldBe true
    }

    @Test
    fun `details endpoint should return 404 for non-existent entity`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/details/http%3A%2F%2Fexample.org%2Fnonexistent")

        response.status shouldBe HttpStatusCode.NotFound
    }

    @Test
    fun `details endpoint should support different formats`() = testApplication {
        application {
            module()
        }

        // First create an entity
        val createResponse = client.post("/api/v2/dataset") {
            contentType(ContentType.Application.Json)
            header("X-Agent", "mailto:test@example.org")
            setBody("""
                {
                    "type": "dataset",
                    "properties": {
                        "http://purl.org/dc/terms/title": "Test Dataset for Details"
                    }
                }
            """.trimIndent())
        }

        val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())
        @Suppress("UNCHECKED_CAST")
        val data = createJson["data"] as? Map<String, Any>
        val entityUri = data?.get("uri") as? String

        if (entityUri != null) {
            // Test JSON format (default)
            val jsonResponse = client.get("/api/v2/details/${java.net.URLEncoder.encode(entityUri, "UTF-8")}?status=draft")
            jsonResponse.status shouldBe HttpStatusCode.OK

            // Test Turtle format
            val turtleResponse = client.get("/api/v2/details/${java.net.URLEncoder.encode(entityUri, "UTF-8")}?status=draft&format=turtle")
            turtleResponse.status shouldBe HttpStatusCode.OK
            turtleResponse.contentType() shouldBe ContentType.parse("text/turtle")
        }
    }

    // =========================================================================
    // SPARQL Endpoint Tests
    // =========================================================================

    @Test
    fun `sparql endpoint should accept SELECT query`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.FormUrlEncoded)
            setBody("query=SELECT * WHERE { ?s ?p ?o } LIMIT 10")
        }

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("application/sparql-results+json")
    }

    @Test
    fun `sparql endpoint should accept ASK query`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.FormUrlEncoded)
            setBody("query=ASK WHERE { ?s ?p ?o }")
        }

        response.status shouldBe HttpStatusCode.OK
    }

    @Test
    fun `sparql endpoint should reject UPDATE queries`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.FormUrlEncoded)
            setBody("query=INSERT DATA { <http://example.org/s> <http://example.org/p> <http://example.org/o> }")
        }

        response.status shouldBe HttpStatusCode.BadRequest
        response.bodyAsText() shouldContain "not allowed"
    }

    @Test
    fun `sparql endpoint should reject DELETE queries`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.FormUrlEncoded)
            setBody("query=DELETE WHERE { ?s ?p ?o }")
        }

        response.status shouldBe HttpStatusCode.BadRequest
    }

    @Test
    fun `sparql endpoint should accept JSON body`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.Json)
            setBody("""{"query": "SELECT * WHERE { ?s ?p ?o } LIMIT 5"}""")
        }

        response.status shouldBe HttpStatusCode.OK
    }

    @Test
    fun `sparql endpoint should accept sparql-query content type`() = testApplication {
        application {
            module()
        }

        val response = client.post("/api/v2/sparql") {
            contentType(ContentType.parse("application/sparql-query"))
            setBody("SELECT * WHERE { ?s ?p ?o } LIMIT 5")
        }

        response.status shouldBe HttpStatusCode.OK
    }

    @Test
    fun `sparql endpoint should respect Accept header for results format`() = testApplication {
        application {
            module()
        }

        // Request XML format
        val xmlResponse = client.post("/api/v2/sparql") {
            contentType(ContentType.Application.FormUrlEncoded)
            header(HttpHeaders.Accept, "application/sparql-results+xml")
            setBody("query=SELECT * WHERE { ?s ?p ?o } LIMIT 5")
        }

        xmlResponse.status shouldBe HttpStatusCode.OK
        xmlResponse.contentType()?.withoutParameters() shouldBe ContentType.parse("application/sparql-results+xml")
    }

    // =========================================================================
    // OAI-PMH Endpoint Tests
    // =========================================================================

    @Test
    fun `oai-pmh Identify verb should return repository info`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=Identify")

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.Application.Xml
        
        val body = response.bodyAsText()
        body shouldContain "OAI-PMH"
        body shouldContain "Identify"
        body shouldContain "repositoryName"
        body shouldContain "EPOS"
    }

    @Test
    fun `oai-pmh ListMetadataFormats should return supported formats`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=ListMetadataFormats")

        response.status shouldBe HttpStatusCode.OK
        
        val body = response.bodyAsText()
        body shouldContain "oai_dc"
        body shouldContain "dcat"
        body shouldContain "epos_dcat_ap"
    }

    @Test
    fun `oai-pmh ListSets should return resource type sets`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=ListSets")

        response.status shouldBe HttpStatusCode.OK
        
        val body = response.bodyAsText()
        body shouldContain "ListSets"
    }

    @Test
    fun `oai-pmh ListIdentifiers should work with metadataPrefix`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=ListIdentifiers&metadataPrefix=oai_dc")

        response.status shouldBe HttpStatusCode.OK
    }

    @Test
    fun `oai-pmh ListRecords should work with metadataPrefix`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=ListRecords&metadataPrefix=oai_dc")

        response.status shouldBe HttpStatusCode.OK
    }

    @Test
    fun `oai-pmh should return error for missing verb`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh")

        response.status shouldBe HttpStatusCode.OK // OAI-PMH returns errors in XML with 200 status
        response.bodyAsText() shouldContain "error"
    }

    @Test
    fun `oai-pmh should return error for invalid verb`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=InvalidVerb")

        response.status shouldBe HttpStatusCode.OK
        response.bodyAsText() shouldContain "error"
        response.bodyAsText() shouldContain "badVerb"
    }

    @Test
    fun `oai-pmh GetRecord should require identifier parameter`() = testApplication {
        application {
            module()
        }

        val response = client.get("/oai-pmh?verb=GetRecord&metadataPrefix=oai_dc")

        response.status shouldBe HttpStatusCode.OK
        response.bodyAsText() shouldContain "error"
        response.bodyAsText() shouldContain "badArgument"
    }

    @Test
    fun `oai-pmh should accept POST requests`() = testApplication {
        application {
            module()
        }

        val response = client.post("/oai-pmh") {
            contentType(ContentType.Application.FormUrlEncoded)
            setBody("verb=Identify")
        }

        response.status shouldBe HttpStatusCode.OK
        response.bodyAsText() shouldContain "Identify"
    }

    // =========================================================================
    // Resource Types Endpoint Test
    // =========================================================================

    @Test
    fun `resource-types endpoint should return SHACL types`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/resource-types")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        
        @Suppress("UNCHECKED_CAST")
        val types = json["types"] as? List<*>
        types?.isNotEmpty() shouldBe true
    }
}
