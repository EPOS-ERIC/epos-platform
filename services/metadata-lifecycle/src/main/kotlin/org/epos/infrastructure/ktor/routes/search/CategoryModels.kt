package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude

// =============================================================================
// Category
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class Category(
    val id: String,
    val name: String,
    val description: String? = null,
    val children: List<Category>? = null,
)

// =============================================================================
// Category Scheme
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class CategoryScheme(
    val id: String,
    val label: String,
    val title: String? = null,
    val description: String? = null,
    val logo: String? = null,
    val homepage: String? = null,
    val color: String? = null,
    val orderNumber: Int? = null,
    val type: String,
    val children: List<Category>? = null,
)

// =============================================================================
// Category Tree Response
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class CategoryTreeResponse(
    val schemes: List<CategoryScheme>,
)
