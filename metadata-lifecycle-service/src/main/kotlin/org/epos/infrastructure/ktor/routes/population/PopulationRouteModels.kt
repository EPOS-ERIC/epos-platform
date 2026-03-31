package org.epos.infrastructure.ktor.routes

data class PopulateFromUrlRequest(
    val indexUrl: String? = null
)

data class PopulateFromFileRequest(
    val url: String? = null,
    val path: String? = null
)

data class PopulateFromContentRequest(
    val content: String? = null,
    val sourceName: String? = null
)
