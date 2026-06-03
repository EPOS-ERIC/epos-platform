package org.epos.tests

import io.kotest.matchers.shouldBe
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import org.epos.infrastructure.ktor.routes.search.ResourceLinkResolver
import org.epos.propertygraph.client.PropertyGraphClient
import org.junit.jupiter.api.Test

class ResourceLinkResolverTest {
    @Test
    fun `should build href for PIC-like resolvable identifiers`() {
        val client = mockk<PropertyGraphClient>()
        val resourceTypeMap = mapOf("http://example.org/Organization" to "organization")
        val querySlot = slot<String>()
        val paramsSlot = slot<Map<String, Any>>()

        every { client.runQuery(capture(querySlot), capture(paramsSlot)) } answers {
            val query = querySlot.captured
            val entityId = paramsSlot.captured["entityId"]?.toString()
            val requestedId = paramsSlot.captured["requestedId"]?.toString()

            when {
                query.contains("MATCH (v:Version {id:") && requestedId == "PIC:999591306/v1.0" -> {
                    listOf(mapOf("id" to "PIC:999591306/v1.0", "identifier" to "PIC:999591306/v1.0"))
                }

                query.contains("MATCH (c:Conceptual {id:") && entityId == "PIC:999591306/v1.0" -> {
                    listOf(mapOf("value" to "Organization"))
                }

                else -> {
                    emptyList()
                }
            }
        }

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

    @Test
    fun `resolved link should prefer graph type over caller preferred shape`() {
        val client = mockk<PropertyGraphClient>()
        val resourceTypeMap =
            mapOf(
                "http://schema.org/Organization" to "organization",
                "http://www.w3.org/ns/adms#Identifier" to "identifier",
            )
        val querySlot = slot<String>()
        val paramsSlot = slot<Map<String, Any>>()

        every { client.runQuery(capture(querySlot), capture(paramsSlot)) } answers {
            val query = querySlot.captured
            val requestedId = paramsSlot.captured["requestedId"]?.toString()
            val versionId = paramsSlot.captured["versionId"]?.toString()

            when {
                query.contains("MATCH (v:Version {id:") && requestedId == "https://example.org/bnode-identifier-1/v1.0" -> {
                    listOf(mapOf("id" to "https://example.org/bnode-identifier-1/v1.0"))
                }

                query.contains("MATCH (v:Version {id:") && versionId == "https://example.org/bnode-identifier-1/v1.0" -> {
                    listOf(mapOf("value" to "http://www.w3.org/ns/adms#Identifier"))
                }

                else -> {
                    emptyList()
                }
            }
        }

        val link =
            ResourceLinkResolver.resolveLink(
                client = client,
                objectId = "https://example.org/bnode-identifier-1/v1.0",
                resourceTypeMap = resourceTypeMap,
                normalizeShapeName = {
                    it
                        .lowercase()
                        .replace("-", "")
                        .replace("_", "")
                        .replace(" ", "")
                },
                publicBaseUrl = "http://localhost:8080",
                preferredShape = "organization",
            )

        link?.resolvedShape shouldBe "identifier"
        link?.href shouldBe "http://localhost:8080/api/v2/access/identifier/https%3A%2F%2Fexample.org%2Fbnode-identifier-1%2Fv1.0"
    }
}
