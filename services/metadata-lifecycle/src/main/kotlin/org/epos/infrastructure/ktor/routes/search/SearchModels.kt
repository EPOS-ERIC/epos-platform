package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude

// =============================================================================
// New Search Response (v2 API)
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SearchResponse(
    val total: Int,
    val totalItems: Int,
    val page: Int,
    val totalPages: Int,
    val items: List<SearchItem>,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
@com.fasterxml.jackson.annotation.JsonPropertyOrder("type", "id", "href", "fields")
data class SearchItem(
    val type: String,
    val id: String,
    val href: String,
    val fields: Map<String, Any?>? = null,
)

// =============================================================================
// Resource Reference (for hrefs in detail responses)
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ResourceRef(
    val id: String,
    val title: String? = null,
    val href: String? = null,
)
