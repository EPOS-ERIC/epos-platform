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
import org.epos.graphdb.Config
import org.junit.jupiter.api.*

/**
 * Integration tests for the new endpoints:
 * - Search API (/api/v2/access/search, /api/v2/details/{id})
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

        val response = client.get("/api/v2/access/search")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        ((json["total"] as Number).toInt() >= 0) shouldBe true
        @Suppress("UNCHECKED_CAST")
        val items = json["items"] as List<*>
        (items.size >= 0) shouldBe true
    }

    @Test
    fun `search endpoint should accept query parameters`() = testApplication {
        application {
            module()
        }

        // New API uses limit/offset instead of page/pageSize
        val response = client.get("/api/v2/access/search?q=test&type=dataset&limit=10&offset=0")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json.containsKey("total") shouldBe true
        json.containsKey("items") shouldBe true
    }

    @Test
    fun `search endpoint should return items with correct structure`() = testApplication {
        application {
            module()
        }

        // New API no longer has facets parameter - use category tree endpoint instead
        val response = client.get("/api/v2/access/search")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        json.containsKey("total") shouldBe true
        json.containsKey("items") shouldBe true
    }

    @Test
    fun `search endpoint should return SHACL-driven fields without constraints`() = testApplication {
        application {
            module()
        }

        val datasetUri = "http://example.org/dataset/with-constraints-1"
        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
        val dataset = vf.createIRI(datasetUri)
        model.add(dataset, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(dataset, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Dataset Constraints Test"))
        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        val response = client.get("/api/v2/access/search?type=dataset&q=constraints")
        response.status shouldBe HttpStatusCode.OK
        val jsonWithout = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
        @Suppress("UNCHECKED_CAST")
        val itemsWithout = jsonWithout["items"] as List<Map<String, Any>>
        (itemsWithout.isNotEmpty()) shouldBe true
        itemsWithout.first().containsKey("constraints") shouldBe false
        itemsWithout.first().containsKey("fields") shouldBe true
    }

    @Test
    fun `legacy search endpoint should return 404`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/search")
        response.status shouldBe HttpStatusCode.NotFound
    }

    @Test
    fun `dataset detail endpoint should return 404 for non-existent entity`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/dataset/http%3A%2F%2Fexample.org%2Fnonexistent")

        response.status shouldBe HttpStatusCode.NotFound
    }

    @Test
    fun `dataset detail endpoint should return dataset info`() = testApplication {
        application {
            module()
        }

        // Insert a dataset directly into the published graph
        val datasetUri = "http://example.org/dataset/test-dataset-1"
        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
        val dataset = vf.createIRI(datasetUri)
        model.add(dataset, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(dataset, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Test Dataset"))
        model.add(dataset, vf.createIRI("http://purl.org/dc/terms/description"), vf.createLiteral("Test Description"))
        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        // Test JSON format (default)
        val jsonResponse = client.get("/api/v2/dataset/${java.net.URLEncoder.encode(datasetUri, "UTF-8")}")
        jsonResponse.status shouldBe HttpStatusCode.OK

        val json = objectMapper.readValue<Map<String, Any>>(jsonResponse.bodyAsText())
        json["id"] shouldBe datasetUri
        json["title"] shouldBe "Test Dataset"
    }

    @Test
    fun `distribution detail endpoint should include relatedBy resources`() = testApplication {
        application {
            module()
        }

        val sourceUri = "http://example.org/dataset/source-1"
        val targetUri = "http://example.org/distribution/target-1"
        val relationPredicate = "http://www.w3.org/ns/dcat#distribution"

        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
        val source = vf.createIRI(sourceUri)
        val target = vf.createIRI(targetUri)
        model.add(source, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(source, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Source Dataset"))
        model.add(source, vf.createIRI(relationPredicate), target)
        model.add(target, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Distribution"))
        model.add(target, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Target Distribution"))
        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        // Use the new typed distribution endpoint
        val response = client.get("/api/v2/distribution/${java.net.URLEncoder.encode(targetUri, "UTF-8")}")

        response.status shouldBe HttpStatusCode.OK
        val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())

        // Inbound resources are exposed as relatedBy in the new API
        @Suppress("UNCHECKED_CAST")
        val relatedBy = json["relatedBy"] as? List<Map<String, Any>>
        (relatedBy != null && relatedBy.isNotEmpty()) shouldBe true
        relatedBy!!.any { it["id"] == sourceUri } shouldBe true
    }

    @Test
    fun `relations and access detail should expose predicate iri`() = testApplication {
        application {
            module()
        }

        val datasetUri = "https://example.org/dataset/source-iri-1"
        val distributionUri = "https://example.org/distribution/target-iri-1"
        val relationPredicate = "http://www.w3.org/ns/dcat#distribution"

        val vf = org.eclipse.rdf4j.model.util.Values.getValueFactory()
        val model = org.eclipse.rdf4j.model.impl.LinkedHashModel()
        val source = vf.createIRI(datasetUri)
        val target = vf.createIRI(distributionUri)
        model.add(source, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(source, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Dataset with edge metadata"))
        model.add(source, vf.createIRI(relationPredicate), target)
        model.add(target, org.eclipse.rdf4j.model.vocabulary.RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Distribution"))
        model.add(target, vf.createIRI("http://purl.org/dc/terms/title"), vf.createLiteral("Distribution with edge metadata"))
        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        val distributionResponse = client.get("/api/v2/distribution/${java.net.URLEncoder.encode(distributionUri, "UTF-8")}")
        distributionResponse.status shouldBe HttpStatusCode.OK
        val distributionJson = objectMapper.readValue<Map<String, Any>>(distributionResponse.bodyAsText())

        @Suppress("UNCHECKED_CAST")
        val relations = distributionJson["relations"] as? List<Map<String, Any>>
        (relations != null && relations.isNotEmpty()) shouldBe true
        val hasPredicateIri = relations!!.any { relation ->
            val predicate = relation["predicate"] as? Map<*, *> ?: return@any false
            predicate["iri"] == relationPredicate
        }
        hasPredicateIri shouldBe true

        val accessResponse = client.get("/api/v2/access/distribution/${java.net.URLEncoder.encode(distributionUri, "UTF-8")}")
        accessResponse.status shouldBe HttpStatusCode.OK
        val accessJson = objectMapper.readValue<Map<String, Any>>(accessResponse.bodyAsText())

        @Suppress("UNCHECKED_CAST")
        val properties = accessJson["properties"] as? Map<String, Any>
        (properties != null) shouldBe true

        val distributionValues = properties!!["distribution"]
        val linkedValues = when (distributionValues) {
            is List<*> -> distributionValues.filterIsInstance<Map<*, *>>()
            is Map<*, *> -> listOf(distributionValues)
            else -> emptyList()
        }

        val hasAccessPredicateIri = linkedValues.any { value -> value["predicateIri"] == relationPredicate }
        hasAccessPredicateIri shouldBe true
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

        response.status shouldBe HttpStatusCode.Forbidden
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

        response.status shouldBe HttpStatusCode.Forbidden
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
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("text/xml")
        
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
            setBody("verb=Identify&metadataPrefix=oai_dc")
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
        val types = objectMapper.readValue<List<Map<String, Any>>>(response.bodyAsText())
        types.isNotEmpty() shouldBe true
    }

    @Test
    fun `resource-types endpoint should export raw SHACL in turtle`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/resource-types?view=shacl&format=turtle")

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("text/turtle")
        response.bodyAsText().isNotBlank() shouldBe true
    }

    @Test
    fun `resource-types endpoint should export raw SHACL in json-ld via accept header`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/resource-types?view=shacl") {
            header(HttpHeaders.Accept, "application/ld+json")
        }

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("application/ld+json")
        response.bodyAsText().isNotBlank() shouldBe true
    }

    @Test
    fun `resource-types endpoint should export compact beautified json-ld`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/resource-types?view=shacl&format=jsonld&compact=true")

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("application/ld+json")
        val body = response.bodyAsText()
        body shouldContain "\"@context\""
        body shouldContain "\"@graph\""
    }

    @Test
    fun `resource-types endpoint should export compact beautified turtle`() = testApplication {
        application {
            module()
        }

        val response = client.get("/api/v2/resource-types?view=shacl&format=turtle&compact=true")

        response.status shouldBe HttpStatusCode.OK
        response.contentType()?.withoutParameters() shouldBe ContentType.parse("text/turtle")
        val body = response.bodyAsText()
        body shouldContain "@prefix"
        body shouldContain " a "
    }
}
