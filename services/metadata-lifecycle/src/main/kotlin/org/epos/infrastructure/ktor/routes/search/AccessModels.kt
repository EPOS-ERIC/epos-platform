package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AccessListResponse(
    val shape: String,
    val total: Int,
    val items: List<AccessListItem>,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AccessListItem(
    val id: String,
    val type: String,
    val title: String? = null,
    val href: String,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AccessDetailResponse(
    val id: String,
    val shape: String,
    val href: String,
    val properties: Map<String, Any>,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AccessPropertyValue(
    val value: String? = null,
    val id: String? = null,
    val type: String? = null,
    val title: String? = null,
    val href: String? = null,
)
