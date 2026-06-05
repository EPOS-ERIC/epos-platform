package org.epos.tests

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import com.fasterxml.jackson.module.kotlin.readValue
import io.kotest.matchers.shouldBe
import io.kotest.matchers.string.shouldContain
import io.ktor.client.request.get
import io.ktor.client.request.header
import io.ktor.client.request.post
import io.ktor.client.request.setBody
import io.ktor.client.statement.bodyAsText
import io.ktor.http.ContentType
import io.ktor.http.HttpStatusCode
import io.ktor.http.contentType
import io.ktor.server.testing.application
import io.ktor.server.testing.testApplication
import org.epos.module
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.TestInstance

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ApplicationIntegrationTest : BaseIntegrationTest() {
    private val objectMapper = jacksonObjectMapper()

    @BeforeEach
    fun setup() {
        clearAllGraphs()
    }

    @Test
    fun `health endpoint should return OK`() =
        testApplication {
            application {
                module()
            }

            val response = client.get("/health")

            response.status shouldBe HttpStatusCode.OK
            response.bodyAsText() shouldContain "ok"
        }

    @Test
    fun `should create draft entity`() =
        testApplication {
            application {
                module()
            }

            val response =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "mailto:test@example.org")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Test Dataset",
                                "http://purl.org/dc/terms/description": "A test dataset"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            response.status shouldBe HttpStatusCode.Created

            val json = objectMapper.readValue<Map<String, Any>>(response.bodyAsText())
            json["success"] shouldBe true

            @Suppress("UNCHECKED_CAST")
            val data = json["data"] as? Map<String, Any>
            data?.get("status") shouldBe "draft"
        }

    @Test
    fun `should get entity as JSON`() =
        testApplication {
            application {
                module()
            }

            // Insert a dataset directly into published graph for testing
            val datasetUri = "http://example.org/dataset/test-json-1"
            val vf =
                org.eclipse.rdf4j.model.util.Values
                    .getValueFactory()
            val model =
                org.eclipse.rdf4j.model.impl
                    .LinkedHashModel()
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
    fun `should get entity via CRUD endpoint as Turtle`() =
        testApplication {
            application {
                module()
            }

            // Create entity via CRUD endpoint
            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Test Dataset"
                            }
                        }
                        """.trimIndent(),
                    )
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
    fun `put should keep replace semantics and remove omitted relation properties`() =
        testApplication {
            application {
                module()
            }

            val datasetUri = "http://example.org/dataset/target-1"

            val createResponse =
                client.post("/api/v2/distribution") {
                    contentType(ContentType.Application.Json)
                    setBody(
                        """
                        {
                            "type": "distribution",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Distribution v2 draft",
                                "http://www.w3.org/ns/dcat#servesDataset": "$datasetUri"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val data = createJson["data"] as Map<String, Any>
            val distributionUri = data["uri"] as String

            val relationPresentBeforePut =
                graphDBClient.ask(
                    """
                    ASK {
                        GRAPH <${org.epos.graphdb.Config.GRAPH_DRAFT}> {
                            <$distributionUri> <http://www.w3.org/ns/dcat#servesDataset> <$datasetUri> .
                        }
                    }
                    """.trimIndent(),
                )
            relationPresentBeforePut shouldBe true

            val encodedDistributionUri = java.net.URLEncoder.encode(distributionUri, "UTF-8")
            val putResponse =
                client.put("/api/v2/distribution/$encodedDistributionUri") {
                    contentType(ContentType.Application.Json)
                    setBody(
                        """
                        {
                            "properties": {
                                "http://purl.org/dc/terms/title": "Distribution v2 draft updated"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            putResponse.status shouldBe HttpStatusCode.OK

            val relationPresentAfterPut =
                graphDBClient.ask(
                    """
                    ASK {
                        GRAPH <${org.epos.graphdb.Config.GRAPH_DRAFT}> {
                            <$distributionUri> <http://www.w3.org/ns/dcat#servesDataset> <$datasetUri> .
                        }
                    }
                    """.trimIndent(),
                )
            relationPresentAfterPut shouldBe false
        }

    @Test
    fun `patch should merge properties and keep existing relation triples`() =
        testApplication {
            application {
                module()
            }

            val datasetUri = "http://example.org/dataset/target-2"

            val createResponse =
                client.post("/api/v2/distribution") {
                    contentType(ContentType.Application.Json)
                    setBody(
                        """
                        {
                            "type": "distribution",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Distribution v2 draft",
                                "http://www.w3.org/ns/dcat#servesDataset": "$datasetUri"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val data = createJson["data"] as Map<String, Any>
            val distributionUri = data["uri"] as String

            val encodedDistributionUri = java.net.URLEncoder.encode(distributionUri, "UTF-8")
            val patchResponse =
                client.patch("/api/v2/distribution/$encodedDistributionUri") {
                    contentType(ContentType.Application.Json)
                    setBody(
                        """
                        {
                            "properties": {
                                "http://purl.org/dc/terms/description": "Draft description update"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            patchResponse.status shouldBe HttpStatusCode.OK

            val relationPresentAfterPatch =
                graphDBClient.ask(
                    """
                    ASK {
                        GRAPH <${org.epos.graphdb.Config.GRAPH_DRAFT}> {
                            <$distributionUri> <http://www.w3.org/ns/dcat#servesDataset> <$datasetUri> .
                        }
                    }
                    """.trimIndent(),
                )
            relationPresentAfterPatch shouldBe true
        }

    @Test
    fun `submit endpoint should accept encoded draft version id`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Submit encoded id"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String
            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")

            val submitResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }

            submitResponse.status shouldBe HttpStatusCode.OK
            val submitJson = objectMapper.readValue<Map<String, Any>>(submitResponse.bodyAsText())
            submitJson["success"] shouldBe true
            @Suppress("UNCHECKED_CAST")
            val submitData = submitJson["data"] as Map<String, Any>
            submitData["uri"] shouldBe draftUri
            submitData["status"] shouldBe "submitted"
        }

    @Test
    fun `patch should return conflict for published resource`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Published target"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String

            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")
            val submitResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }
            submitResponse.status shouldBe HttpStatusCode.OK

            val publishResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/publish") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }
            publishResponse.status shouldBe HttpStatusCode.OK

            val publishJson = objectMapper.readValue<Map<String, Any>>(publishResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val publishData = publishJson["data"] as Map<String, Any>
            val publishedUri = publishData["uri"] as String
            val encodedPublishedUri = java.net.URLEncoder.encode(publishedUri, "UTF-8")

            val patchResponse =
                client.patch("/api/v2/dataset/$encodedPublishedUri") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "properties": {
                                "http://purl.org/dc/terms/description": "Should fail on published"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            patchResponse.status shouldBe HttpStatusCode.Conflict
            patchResponse.bodyAsText() shouldContain "Create/use a draft before patching"
        }

    @Test
    fun `draft endpoint should accept published version id`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Draft from published id"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String

            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")
            client
                .post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }.status shouldBe HttpStatusCode.OK

            val publishResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/publish") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }
            publishResponse.status shouldBe HttpStatusCode.OK

            val publishJson = objectMapper.readValue<Map<String, Any>>(publishResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val publishData = publishJson["data"] as Map<String, Any>
            val publishedUri = publishData["uri"] as String
            val encodedPublishedUri = java.net.URLEncoder.encode(publishedUri, "UTF-8")

            val newDraftResponse =
                client.post("/api/v2/dataset/$encodedPublishedUri/draft?version=3.0") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }

            newDraftResponse.status shouldBe HttpStatusCode.Created
            val newDraftJson = objectMapper.readValue<Map<String, Any>>(newDraftResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val newDraftData = newDraftJson["data"] as Map<String, Any>
            (newDraftData["uri"] as String) shouldContain "/3.0-draft"
            newDraftData["status"] shouldBe "draft"
        }

    @Test
    fun `put should return conflict for submitted resource`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Submitted target"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String
            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")

            val submitResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }
            submitResponse.status shouldBe HttpStatusCode.OK

            val putResponse =
                client.put("/api/v2/dataset/$encodedDraftUri") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "properties": {
                                "http://purl.org/dc/terms/title": "Should fail on submitted"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            putResponse.status shouldBe HttpStatusCode.Conflict
            putResponse.bodyAsText() shouldContain "Create/use a draft before updating"
        }

    @Test
    fun `delete should return conflict for published resource`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Delete published target"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String
            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")

            client
                .post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }.status shouldBe HttpStatusCode.OK

            val publishResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/publish") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }
            publishResponse.status shouldBe HttpStatusCode.OK
            val publishJson = objectMapper.readValue<Map<String, Any>>(publishResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val publishData = publishJson["data"] as Map<String, Any>
            val publishedUri = publishData["uri"] as String
            val encodedPublishedUri = java.net.URLEncoder.encode(publishedUri, "UTF-8")

            val deleteResponse =
                client.delete("/api/v2/dataset/$encodedPublishedUri") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }

            deleteResponse.status shouldBe HttpStatusCode.Conflict
            deleteResponse.bodyAsText() shouldContain "Only drafts can be deleted"
        }

    @Test
    fun `discard endpoint should move submitted to discarded`() =
        testApplication {
            application {
                module()
            }

            val createResponse =
                client.post("/api/v2/dataset") {
                    contentType(ContentType.Application.Json)
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                    setBody(
                        """
                        {
                            "type": "dataset",
                            "properties": {
                                "http://purl.org/dc/terms/title": "Discard target"
                            }
                        }
                        """.trimIndent(),
                    )
                }

            createResponse.status shouldBe HttpStatusCode.Created
            val createJson = objectMapper.readValue<Map<String, Any>>(createResponse.bodyAsText())

            @Suppress("UNCHECKED_CAST")
            val createData = createJson["data"] as Map<String, Any>
            val draftUri = createData["uri"] as String
            val encodedDraftUri = java.net.URLEncoder.encode(draftUri, "UTF-8")

            client
                .post("/api/v2/dataset/$encodedDraftUri/submit") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }.status shouldBe HttpStatusCode.OK

            val discardResponse =
                client.post("/api/v2/dataset/$encodedDraftUri/discard") {
                    header("X-Agent", "https://example.org/agent/test")
                    header("X-Group", "https://example.org/group/test")
                }

            discardResponse.status shouldBe HttpStatusCode.OK
            val discardJson = objectMapper.readValue<Map<String, Any>>(discardResponse.bodyAsText())
            discardJson["success"] shouldBe true
            @Suppress("UNCHECKED_CAST")
            val discardData = discardJson["data"] as Map<String, Any>
            discardData["status"] shouldBe "discarded"
        }

    @Test
    fun `should return 404 for non-existent entity`() =
        testApplication {
            application {
                module()
            }

            val missingEntity = java.net.URLEncoder.encode("http://example.org/nonexistent", "UTF-8")
            val response = client.get("/api/v2/dataset/$missingEntity")

            response.status shouldBe HttpStatusCode.NotFound
        }
}
