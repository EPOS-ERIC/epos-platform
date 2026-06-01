package org.epos.tests

import io.kotest.matchers.shouldBe
import io.kotest.matchers.string.shouldContain
import io.ktor.client.request.get
import io.ktor.client.statement.bodyAsText
import io.ktor.http.HttpStatusCode
import io.ktor.server.testing.testApplication
import org.epos.module
import org.junit.jupiter.api.Test

class HealthRoutesTest : Neo4jTestContainerSupport() {
    @Test
    fun `health endpoint returns ok and neo4j config`() =
        testApplication {
            application {
                module()
            }

            val response = client.get("/health")

            response.status shouldBe HttpStatusCode.OK
            val body = response.bodyAsText()
            body shouldContain "\"status\" : \"ok\""
            body shouldContain "neo4jUri"
            body shouldContain "neo4jDatabase"
        }
}
