package org.epos.tests

import io.kotest.matchers.shouldBe
import io.mockk.every
import io.mockk.mockk
import org.epos.infrastructure.ktor.routes.search.ResourceLinkResolver
import org.epos.propertygraph.client.PropertyGraphClient
import org.junit.jupiter.api.Test

class ResourceLinkResolverTest {
    @Test
    fun `should build href for PIC-like resolvable identifiers`() {
        val client = mockk<PropertyGraphClient>()
        val resourceTypeMap = mapOf("http://example.org/Organization" to "organization")

        every {
            client.runQuery(match { it.contains("hasLabel('Version')") && it.contains("PIC:999591306/v1.0") }, any())
        } returns listOf(mapOf("value" to "PIC:999591306/v1.0"))

        every {
            client.runQuery(match { it.contains("hasLabel('Conceptual')") && it.contains("PIC:999591306/v1.0") }, any())
        } returns listOf(mapOf("value" to "Organization"))

        val shape =
            ResourceLinkResolver.resolveEntityShapeFromGraph(
                client = client,
                entityId = "PIC:999591306/v1.0",
                resourceTypeMap = resourceTypeMap,
                normalizeShapeName = {
                    it
                        .lowercase()
                        .replace("-", "")
                        .replace("_", "")
                        .replace(" ", "")
                },
            )

        shape shouldBe "organization"

        val href = ResourceLinkResolver.buildNavigableAccessHref("http://localhost:8080", shape, "PIC:999591306/v1.0")
        href shouldBe "http://localhost:8080/api/v2/access/organization/PIC%3A999591306%2Fv1.0"
    }
}
