package org.epos.tests

import io.kotest.assertions.throwables.shouldThrow
import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.model.util.Values
import org.eclipse.rdf4j.model.vocabulary.DCTERMS
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.model.vocabulary.RDFS
import org.epos.graphdb.Config
import org.epos.graphdb.services.EntityLifecycleService
import org.epos.graphdb.services.ProvenanceService
import org.junit.jupiter.api.*
import org.epos.tests.BaseIntegrationTest

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class EntityLifecycleServiceIntegrationTest : BaseIntegrationTest() {

    private lateinit var provenanceService: ProvenanceService
    private lateinit var lifecycleService: EntityLifecycleService

    @BeforeEach
    fun setup() {
        clearAllGraphs()
        provenanceService = ProvenanceService(graphDBClient)
        lifecycleService = EntityLifecycleService(graphDBClient, provenanceService)
    }

    private fun createPublishedEntity(conceptualUri: String, version: String = "1.0"): String {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()
        val versionUri = "$conceptualUri/v$version"
        val entityIRI = vf.createIRI(versionUri)

        model.add(entityIRI, RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Dataset"))
        model.add(entityIRI, RDFS.LABEL, vf.createLiteral("Test Dataset"))
        model.add(entityIRI, vf.createIRI("http://purl.org/pav/version"), vf.createLiteral(version))
        model.add(entityIRI, DCTERMS.IS_VERSION_OF, vf.createIRI(conceptualUri))

        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        // Create conceptual entity
        val conceptualModel = LinkedHashModel()
        conceptualModel.add(vf.createIRI(conceptualUri), RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Resource"))
        conceptualModel.add(vf.createIRI(conceptualUri), vf.createIRI("http://purl.org/pav/hasCurrentVersion"), entityIRI)
        graphDBClient.insertIntoGraph(conceptualModel, Config.GRAPH_CONCEPTUAL)

        // Record provenance
        provenanceService.recordInitialCreation(versionUri, conceptualUri, "test-source")

        return versionUri
    }

    @Test
    fun `should create draft from published version`() {
        val conceptualUri = "http://example.org/dataset/123"
        val publishedUri = createPublishedEntity(conceptualUri, "1.0")

        val draftUri = lifecycleService.createDraft(
            conceptualUri = conceptualUri,
            newVersionNumber = "2.0",
            agent = "mailto:test@example.org"
        )

        draftUri shouldBe "$conceptualUri/2.0-draft"

        // Verify draft exists
        val inDraft = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <$draftUri> ?p ?o . } }
        """.trimIndent())
        inDraft shouldBe true

        // Verify label was copied
        val results = graphDBClient.query("""
            SELECT ?label
            FROM <${Config.GRAPH_DRAFT}>
            WHERE {
                <$draftUri> <${RDFS.LABEL}> ?label .
            }
        """.trimIndent())
        results[0]["label"] shouldBe "Test Dataset"
    }

    @Test
    fun `should submit draft`() {
        val conceptualUri = "http://example.org/dataset/456"
        createPublishedEntity(conceptualUri)
        val draftUri = lifecycleService.createDraft(conceptualUri, "2.0")

        lifecycleService.submitDraft(draftUri, "mailto:test@example.org")

        // Verify not in DRAFT
        val inDraft = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <$draftUri> ?p ?o . } }
        """.trimIndent())
        inDraft shouldBe false

        // Verify in SUBMITTED
        val inSubmitted = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_SUBMITTED}> { <$draftUri> ?p ?o . } }
        """.trimIndent())
        inSubmitted shouldBe true
    }

    @Test
    fun `should publish submitted entity and archive old version`() {
        val conceptualUri = "http://example.org/dataset/789"
        val v1Uri = createPublishedEntity(conceptualUri, "1.0")
        val draftUri = lifecycleService.createDraft(conceptualUri, "2.0")
        lifecycleService.submitDraft(draftUri)

        lifecycleService.publish(draftUri, "mailto:test@example.org")

        val finalUri = draftUri.removeSuffix("-draft")

        // Verify v1 is archived
        val v1Archived = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_ARCHIVED}> { <$v1Uri> ?p ?o . } }
        """.trimIndent())
        v1Archived shouldBe true

        // Verify v2 is published
        val v2Published = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_PUBLISHED}> { <$finalUri> ?p ?o . } }
        """.trimIndent())
        v2Published shouldBe true

        // Verify conceptual points to new version
        val results = graphDBClient.query("""
            PREFIX pav: <http://purl.org/pav/>
            SELECT ?current
            FROM <${Config.GRAPH_CONCEPTUAL}>
            WHERE {
                <$conceptualUri> pav:hasCurrentVersion ?current .
            }
        """.trimIndent())
        results[0]["current"] shouldBe finalUri
    }

    @Test
    fun `should reject submission and return to draft`() {
        val conceptualUri = "http://example.org/dataset/101"
        createPublishedEntity(conceptualUri)
        val draftUri = lifecycleService.createDraft(conceptualUri, "2.0")
        lifecycleService.submitDraft(draftUri)

        lifecycleService.rejectSubmission(draftUri, "mailto:test@example.org")

        // Verify back in DRAFT
        val inDraft = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <$draftUri> ?p ?o . } }
        """.trimIndent())
        inDraft shouldBe true
    }

    @Test
    fun `should delete draft`() {
        val conceptualUri = "http://example.org/dataset/202"
        createPublishedEntity(conceptualUri)
        val draftUri = lifecycleService.createDraft(conceptualUri, "2.0")

        lifecycleService.deleteDraft(draftUri)

        val exists = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <$draftUri> ?p ?o . } }
        """.trimIndent())
        exists shouldBe false
    }

    @Test
    fun `should throw exception when creating draft without published version`() {
        val conceptualUri = "http://example.org/dataset/999"

        shouldThrow<IllegalStateException> {
            lifecycleService.createDraft(conceptualUri, "1.0")
        }
    }

    @Test
    fun `should throw exception when submitting non-existent draft`() {
        shouldThrow<IllegalArgumentException> {
            lifecycleService.submitDraft("http://example.org/nonexistent", "agent")
        }
    }
}