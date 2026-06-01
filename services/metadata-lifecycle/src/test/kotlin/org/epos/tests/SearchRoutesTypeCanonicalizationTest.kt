package org.epos.tests

import io.kotest.matchers.shouldBe
import org.epos.infrastructure.ktor.routes.search.canonicalApiType
import org.epos.infrastructure.ktor.routes.search.canonicalSearchType
import org.junit.jupiter.api.Test

class SearchRoutesTypeCanonicalizationTest {
    @Test
    fun `organization aliases should map to organization`() {
        canonicalSearchType("organization") shouldBe "organization"
        canonicalSearchType("publisher") shouldBe "organization"
        canonicalSearchType("provider") shouldBe "organization"
    }

    @Test
    fun `api type should normalize aliases and formatting`() {
        canonicalApiType("Publisher") shouldBe "organization"
        canonicalApiType("PROVIDER") shouldBe "organization"
        canonicalApiType("organization") shouldBe "organization"
    }
}
