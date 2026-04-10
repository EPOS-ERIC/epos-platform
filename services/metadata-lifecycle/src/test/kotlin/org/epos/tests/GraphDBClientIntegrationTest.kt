package org.epos.tests

import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe
import io.kotest.matchers.collections.shouldHaveSize
import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.model.util.Values
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.model.vocabulary.RDFS
import org.epos.graphdb.Config
import org.junit.jupiter.api.*
import org.epos.tests.BaseIntegrationTest

@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class GraphDBClientIntegrationTest : BaseIntegrationTest() {

    @BeforeEach
    fun setup() {
        clearAllGraphs()
    }

    @Test
    fun `should insert and query triples in named graph`() {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()

        val subject = vf.createIRI("http://example.org/resource/1")
        model.add(subject, RDF.TYPE, vf.createIRI("http://example.org/MyClass"))
        model.add(subject, RDFS.LABEL, vf.createLiteral("Test Resource"))

        // Insert
        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        // Query
        val results = graphDBClient.query("""
            SELECT ?label
            FROM <${Config.GRAPH_PUBLISHED}>
            WHERE {
                <http://example.org/resource/1> <${RDFS.LABEL}> ?label .
            }
        """.trimIndent())

        results shouldHaveSize 1
        results[0]["label"] shouldBe "Test Resource"
    }

    @Test
    fun `should execute ASK query correctly`() {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()
        val subject = vf.createIRI("http://example.org/resource/2")
        model.add(subject, RDF.TYPE, vf.createIRI("http://example.org/MyClass"))

        graphDBClient.insertIntoGraph(model, Config.GRAPH_DRAFT)

        val exists = graphDBClient.ask("""
            ASK {
                GRAPH <${Config.GRAPH_DRAFT}> {
                    <http://example.org/resource/2> ?p ?o .
                }
            }
        """.trimIndent())

        exists shouldBe true

        val notExists = graphDBClient.ask("""
            ASK {
                GRAPH <${Config.GRAPH_DRAFT}> {
                    <http://example.org/nonexistent> ?p ?o .
                }
            }
        """.trimIndent())

        notExists shouldBe false
    }

    @Test
    fun `should delete triples from graph`() {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()
        val subject = vf.createIRI("http://example.org/resource/3")
        model.add(subject, RDF.TYPE, vf.createIRI("http://example.org/MyClass"))
        model.add(subject, RDFS.LABEL, vf.createLiteral("To Delete"))

        graphDBClient.insertIntoGraph(model, Config.GRAPH_DRAFT)

        // Verify inserted
        var exists = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <http://example.org/resource/3> ?p ?o . } }
        """.trimIndent())
        exists shouldBe true

        // Delete
        graphDBClient.deleteFromGraph("http://example.org/resource/3", Config.GRAPH_DRAFT)

        // Verify deleted
        exists = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <http://example.org/resource/3> ?p ?o . } }
        """.trimIndent())
        exists shouldBe false
    }

    @Test
    fun `should move subject between graphs`() {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()
        val subject = vf.createIRI("http://example.org/resource/4")
        model.add(subject, RDF.TYPE, vf.createIRI("http://example.org/MyClass"))
        model.add(subject, RDFS.LABEL, vf.createLiteral("Moving Resource"))

        graphDBClient.insertIntoGraph(model, Config.GRAPH_DRAFT)

        // Move from DRAFT to SUBMITTED
        graphDBClient.moveSubject("http://example.org/resource/4", Config.GRAPH_DRAFT, Config.GRAPH_SUBMITTED)

        // Verify not in DRAFT
        val inDraft = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <http://example.org/resource/4> ?p ?o . } }
        """.trimIndent())
        inDraft shouldBe false

        // Verify in SUBMITTED
        val inSubmitted = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_SUBMITTED}> { <http://example.org/resource/4> ?p ?o . } }
        """.trimIndent())
        inSubmitted shouldBe true

        // Verify label preserved
        val results = graphDBClient.query("""
            SELECT ?label
            FROM <${Config.GRAPH_SUBMITTED}>
            WHERE {
                <http://example.org/resource/4> <${RDFS.LABEL}> ?label .
            }
        """.trimIndent())

        results[0]["label"] shouldBe "Moving Resource"
    }

    @Test
    fun `should get subject triples`() {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()
        val subject = vf.createIRI("http://example.org/resource/5")
        model.add(subject, RDF.TYPE, vf.createIRI("http://example.org/MyClass"))
        model.add(subject, RDFS.LABEL, vf.createLiteral("Test"))
        model.add(subject, RDFS.COMMENT, vf.createLiteral("Comment"))

        graphDBClient.insertIntoGraph(model, Config.GRAPH_PUBLISHED)

        val retrieved = graphDBClient.getSubjectTriples("http://example.org/resource/5", Config.GRAPH_PUBLISHED)

        retrieved.size shouldBe 3
    }
}