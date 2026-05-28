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

@JsonInclude(JsonInclude.Include.NON_NULL)
data class RelationPredicate(
    val iri: String,
    val name: String,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class RelationGroup(
    val direction: String,
    val predicate: RelationPredicate,
    val resources: List<ResourceRef>,
)
