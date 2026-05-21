package org.epos.tests

import com.fasterxml.jackson.module.kotlin.jacksonObjectMapper
import io.kotest.matchers.string.shouldContain
import org.epos.infrastructure.ktor.routes.AccessPropertyValue
import org.junit.jupiter.api.Test

class AccessModelsTest {
    @Test
    fun `access property value should serialize predicate iri when present`() {
        val mapper = jacksonObjectMapper()
        val payload =
            AccessPropertyValue(
                id = "https://example.org/distribution/1",
                type = "distribution",
                href = "https://example.org/api/v2/access/distribution/1",
                predicateIri = "http://www.w3.org/ns/dcat#distribution",
            )

        val json = mapper.writeValueAsString(payload)
        json shouldContain "\"predicateIri\":\"http://www.w3.org/ns/dcat#distribution\""
    }
}
