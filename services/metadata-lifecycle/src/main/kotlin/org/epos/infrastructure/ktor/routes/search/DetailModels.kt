package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude

// =============================================================================
// Temporal Extent
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class TemporalExtent(
    val start: String? = null,
    val end: String? = null,
)

// =============================================================================
// Dataset Detail
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class DatasetDetail(
    val id: String,
    val title: String,
    val description: String? = null,
    val keywords: List<String>? = null,
    val categories: List<String>? = null,
    val publisher: List<String>? = null,
    val spatial: String? = null,
    val temporal: TemporalExtent? = null,
    val status: String,
    val distributions: List<ResourceRef>? = null,
    val related: List<ResourceRef>? = null,
    val relatedBy: List<ResourceRef>? = null,
)

// =============================================================================
// Distribution Detail
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class DistributionDetail(
    val id: String,
    val title: String,
    val description: String? = null,
    val accessUrl: String? = null,
    val operation: String? = null,
    val related: List<ResourceRef>? = null,
    val relatedBy: List<ResourceRef>? = null,
)

// =============================================================================
// Software Application Detail
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SoftwareApplicationDetail(
    val id: String,
    val title: String,
    val description: String? = null,
    val keywords: List<String>? = null,
    val categories: List<String>? = null,
    val downloadUrl: String? = null,
    val related: List<ResourceRef>? = null,
    val relatedBy: List<ResourceRef>? = null,
)

// =============================================================================
// Software Source Code Detail
// =============================================================================

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SoftwareSourceCodeDetail(
    val id: String,
    val title: String,
    val description: String? = null,
    val programmingLanguage: String? = null,
    val keywords: List<String>? = null,
    val categories: List<String>? = null,
    val repository: String? = null,
    val related: List<ResourceRef>? = null,
    val relatedBy: List<ResourceRef>? = null,
)
