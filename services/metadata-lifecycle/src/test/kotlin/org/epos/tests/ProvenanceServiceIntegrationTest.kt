package org.epos.tests

import io.kotest.matchers.collections.shouldContain
import io.kotest.matchers.collections.shouldNotBeEmpty
import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe
import io.kotest.matchers.string.shouldContain
import org.epos.graphdb.Config
import org.epos.graphdb.services.ProvenanceService
import org.epos.tests.BaseIntegrationTest
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.TestInstance
import java.util.UUID

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class ProvenanceServiceIntegrationTest : BaseIntegrationTest() {
    private lateinit var provenanceService: ProvenanceService

    @BeforeEach
    fun setup() {
        clearAllGraphs()
        provenanceService = ProvenanceService(graphDBClient)
    }

    @Test
    fun `should record initial creation with provenance`() {
        val versionUri = "http://example.org/dataset/123/v1.0"
        val conceptualUri = "http://example.org/dataset/123"
        val source = "http://example.org/source/file.ttl"

        provenanceService.recordInitialCreation(
            versionUri = versionUri,
            conceptualUri = conceptualUri,
            source = source,
            agent = "mailto:test@example.org",
        )

        // Verify provenance was recorded
        val hasProvenance =
            graphDBClient.ask(
                """
                PREFIX prov: <http://www.w3.org/ns/prov#>
                ASK {
                    GRAPH <${Config.GRAPH_PROVENANCE}> {
                        <$versionUri> prov:wasGeneratedBy ?activity .
                        <$versionUri> prov:generatedAtTime ?time .
                        <$versionUri> prov:wasAttributedTo ?agent .
                    }
                }
                """.trimIndent(),
            )

        hasProvenance shouldBe true
    }

    @Test
    fun `should record draft creation with revision link`() {
        val newVersionUri = "http://example.org/dataset/123/v2.0-draft"
        val basedOnUri = "http://example.org/dataset/123/v1.0"
        val conceptualUri = "http://example.org/dataset/123"

        provenanceService.recordDraftCreation(
            newVersionUri = newVersionUri,
            basedOnVersionUri = basedOnUri,
            conceptualUri = conceptualUri,
            agent = "mailto:test@example.org",
        )

        // Verify revision link
        val hasRevision =
            graphDBClient.ask(
                """
                PREFIX prov: <http://www.w3.org/ns/prov#>
                ASK {
                    GRAPH <${Config.GRAPH_PROVENANCE}> {
                        <$newVersionUri> prov:wasRevisionOf <$basedOnUri> .
                    }
                }
                """.trimIndent(),
            )

        hasRevision shouldBe true
    }

    @Test
    fun `should record state transition`() {
        val versionUri = "http://example.org/dataset/${UUID.randomUUID()}/v1.0"

        provenanceService.recordStateTransition(
            versionUri = versionUri,
            fromStatus = "DRAFT",
            toStatus = "SUBMITTED",
            agent = "mailto:test@example.org",
        )

        // Verify state transition activity was recorded
        val results =
            graphDBClient.query(
                """
                PREFIX prov: <http://www.w3.org/ns/prov#>
                PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
                SELECT ?label
                FROM <${Config.GRAPH_PROVENANCE}>
                WHERE {
                    ?activity prov:used <$versionUri> ;
                             rdfs:label ?label .
                }
                """.trimIndent(),
            )

        results.shouldNotBeEmpty()
        val labels = results.mapNotNull { it["label"] }
        labels.shouldNotBeEmpty()
        labels.any { it.contains("State transition") && it.contains("DRAFT") && it.contains("SUBMITTED") } shouldBe true
    }

    @Test
    fun `should get version history`() {
        val conceptualUri = "http://example.org/dataset/${UUID.randomUUID()}"
        val v1Uri = "$conceptualUri/v1.0"
        val v2Uri = "$conceptualUri/v2.0"

        // Record two versions
        provenanceService.recordInitialCreation(v1Uri, conceptualUri, "source1.ttl")
        Thread.sleep(100) // Ensure different timestamps
        provenanceService.recordDraftCreation(v2Uri, v1Uri, conceptualUri)

        val history = provenanceService.getVersionHistory(conceptualUri)

        val versions = history.mapNotNull { it["version"] }
        versions.shouldContain(v1Uri)
        versions.shouldContain(v2Uri)
    }

    @Test
    fun `should convert non-absolute URIs to absolute`() {
        val versionUri = "http://example.org/dataset/123/v1.0"
        val conceptualUri = "http://example.org/dataset/123"

        // Use relative URIs that will be converted
        provenanceService.recordInitialCreation(
            versionUri = versionUri,
            conceptualUri = conceptualUri,
            source = "relative-source", // Will be converted
            agent = "relative-agent", // Will be converted
        )

        // Verify it was recorded (conversion happened)
        val hasProvenance =
            graphDBClient.ask(
                """
                PREFIX prov: <http://www.w3.org/ns/prov#>
                ASK {
                    GRAPH <${Config.GRAPH_PROVENANCE}> {
                        <$versionUri> prov:wasGeneratedBy ?activity .
                    }
                }
                """.trimIndent(),
            )

        hasProvenance shouldBe true
    }
}
