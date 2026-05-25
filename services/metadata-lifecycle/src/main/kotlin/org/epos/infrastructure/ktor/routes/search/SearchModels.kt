package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude
import com.fasterxml.jackson.annotation.JsonSubTypes
import com.fasterxml.jackson.annotation.JsonTypeInfo

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

@JsonTypeInfo(
    use = JsonTypeInfo.Id.NAME,
    include = JsonTypeInfo.As.EXISTING_PROPERTY,
    property = "type",
    visible = true,
)
@JsonSubTypes(
    JsonSubTypes.Type(value = DatasetSearchItem::class, name = "dataset"),
    JsonSubTypes.Type(value = GenericSearchItem::class, name = "distribution"),
    JsonSubTypes.Type(value = GenericSearchItem::class, name = "softwareapplication"),
    JsonSubTypes.Type(value = GenericSearchItem::class, name = "softwaresourcecode"),
    JsonSubTypes.Type(value = GenericSearchItem::class, name = "category"),
)
sealed interface SearchItem {
    val type: String
    val id: String
    val href: String
    val fields: Map<String, Any?>?
}

@JsonInclude(JsonInclude.Include.NON_NULL)
@com.fasterxml.jackson.annotation.JsonPropertyOrder("type", "id", "href", "fields")
data class DatasetSearchItem(
    override val id: String,
    override val href: String,
    override val fields: Map<String, Any?>? = null,
) : SearchItem {
    override val type: String = "dataset"
}

@JsonInclude(JsonInclude.Include.NON_NULL)
@com.fasterxml.jackson.annotation.JsonPropertyOrder("type", "id", "href", "fields")
data class GenericSearchItem(
    override val type: String,
    override val id: String,
    override val href: String,
    override val fields: Map<String, Any?>? = null,
) : SearchItem

// =============================================================================
// Resource Reference (for hrefs in detail responses)
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ResourceRef(
    val id: String,
    val title: String? = null,
    val href: String? = null,
)
