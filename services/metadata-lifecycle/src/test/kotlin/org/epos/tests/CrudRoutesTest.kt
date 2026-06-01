package org.epos.tests
import io.kotest.matchers.shouldBe
import io.kotest.matchers.string.shouldContain
import io.ktor.client.request.get
import io.ktor.client.statement.bodyAsText
import io.ktor.http.HttpStatusCode
import io.ktor.server.testing.testApplication
import org.epos.module
import org.junit.jupiter.api.Test

class CrudRoutesTest : Neo4jTestContainerSupport() {
    @Test
    fun `search endpoint should return result envelope`() =
        testApplication {
            application {
                module()
            }

            val response = client.get("/api/v2/access/search")

            response.status shouldBe HttpStatusCode.OK
            val body = response.bodyAsText()
            body shouldContain "\"total\""
            body shouldContain "\"items\""
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
            response.bodyAsText() shouldContain "error"
        }
}
