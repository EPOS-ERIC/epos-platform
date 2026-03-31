package org.epos.tests

import io.ktor.client.HttpClient
import org.epos.api.shacl.RelationshipGraph
import org.epos.api.shacl.ShaclParser
import org.epos.api.shacl.TargetSparql
import org.junit.jupiter.api.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class ShaclParserUnitTest {

    @Test
    fun `parser keeps OR constraints, qualified bnode and SPARQL target`() {
        val ttl = """
            @prefix sh: <http://www.w3.org/ns/shacl#> .
            @prefix ex: <http://example.org/> .

            ex:DatasetShape a sh:NodeShape ;
              sh:targetClass ex:Dataset ;
              sh:target [
                sh:select "SELECT ?this WHERE { ?this a ex:Dataset . }"
              ] ;
              sh:property [
                sh:path ex:publisher ;
                sh:or (
                  [ sh:class ex:Organization ]
                  [ sh:class ex:Person ]
                ) ;
                sh:qualifiedValueShape [ sh:class ex:Organization ] ;
                sh:qualifiedMinCount 1 ;
              ] .
        """.trimIndent()

        val parser = ShaclParser(HttpClient())
        val shapes = parser.parseShapesGraphFromString(ttl)

        val nodeShape = shapes.nodeShapes.first()
        val prop = nodeShape.properties.first()

        assertTrue(nodeShape.targets.any { it is TargetSparql })
        assertTrue(prop.nodeConstraints.classConstraint.contains("http://example.org/Organization"))
        assertTrue(prop.nodeConstraints.classConstraint.contains("http://example.org/Person"))
        assertNotNull(prop.qualifiedConstraints)
        assertTrue(prop.qualifiedConstraints.qualifiedValueShape.startsWith("_:"))

    }

    @Test
    fun `relationship graph supports inverse predicate paths from SHACL`() {
        val ttl = """
            @prefix sh: <http://www.w3.org/ns/shacl#> .
            @prefix ex: <http://example.org/> .

            ex:AShape a sh:NodeShape ;
              sh:targetClass ex:A ;
              sh:property [
                sh:path [ sh:inversePath ex:hasA ] ;
                sh:class ex:B ;
              ] .

            ex:BShape a sh:NodeShape ;
              sh:targetClass ex:B .
        """.trimIndent()

        val parser = ShaclParser(HttpClient())
        val shapes = parser.parseShapesGraphFromString(ttl)

        val graph = RelationshipGraph()
        graph.buildFromShacl(shapes)

        val path = graph.findPath("a", "b")
        assertNotNull(path)
        assertEquals("^<http://example.org/hasA>", path.sparqlPropertyPath)
    }

    @Test
    fun `shape definitions map resolves nodeShapeRef targets`() {
        val ttl = """
            @prefix sh: <http://www.w3.org/ns/shacl#> .
            @prefix ex: <http://example.org/> .

            ex:ContactPointType a sh:NodeShape ;
              sh:property [
                sh:path ex:kind ;
                sh:or (
                  [ sh:class ex:OrgContact ]
                  [ sh:class ex:PersonContact ]
                )
              ] .

            ex:CatalogShape a sh:NodeShape ;
              sh:targetClass ex:Catalog ;
              sh:property [
                sh:path ex:contactPoint ;
                sh:node ex:ContactPointType ;
              ] .
        """.trimIndent()

        val parser = ShaclParser(HttpClient())
        parser.parseShapesGraphFromString(ttl)
        val definitions = parser.getShapeDefinitions()

        val catalog = definitions["http://example.org/CatalogShape"]
        assertNotNull(catalog)

        val prop = catalog.properties.first { it.path == "http://example.org/contactPoint" }
        assertEquals("http://example.org/ContactPointType", prop.nodeShapeRef)

        val contactPointType = definitions["http://example.org/ContactPointType"]
        assertNotNull(contactPointType)
        val classes = contactPointType.properties.flatMap { it.classConstraints ?: listOfNotNull(it.classConstraint) }
        assertTrue(classes.contains("http://example.org/OrgContact"))
        assertTrue(classes.contains("http://example.org/PersonContact"))
    }

    @Test
    fun `transitive shape definitions are filtered and resolved refs are cycle-safe`() {
        val ttl = """
            @prefix sh: <http://www.w3.org/ns/shacl#> .
            @prefix ex: <http://example.org/> .

            ex:RootShape a sh:NodeShape ;
              sh:targetClass ex:Root ;
              sh:property [
                sh:path ex:child ;
                sh:node ex:BShape ;
              ] .

            ex:BShape a sh:NodeShape ;
              sh:property [
                sh:path ex:next ;
                sh:node ex:CShape ;
              ] .

            ex:CShape a sh:NodeShape ;
              sh:property [
                sh:path ex:back ;
                sh:node ex:BShape ;
              ] .

            ex:UnreachableShape a sh:NodeShape ;
              sh:property [
                sh:path ex:x ;
                sh:class ex:Something ;
              ] .
        """.trimIndent()

        val parser = ShaclParser(HttpClient())
        parser.parseShapesGraphFromString(ttl)

        val definitions = parser.getShapeDefinitions(
            onlyReachableFromResourceTypes = true,
            includeResolvedRefs = true
        )

        assertTrue(definitions.containsKey("http://example.org/BShape"))
        assertTrue(definitions.containsKey("http://example.org/CShape"))
        assertTrue(!definitions.containsKey("http://example.org/UnreachableShape"))

        val bResolved = definitions["http://example.org/BShape"]?.resolvedRefs.orEmpty()
        val cResolved = definitions["http://example.org/CShape"]?.resolvedRefs.orEmpty()

        assertTrue(bResolved.contains("http://example.org/CShape"))
        assertTrue(!bResolved.contains("http://example.org/BShape"))
        assertTrue(cResolved.contains("http://example.org/BShape"))
        assertTrue(!cResolved.contains("http://example.org/CShape"))
    }
}
