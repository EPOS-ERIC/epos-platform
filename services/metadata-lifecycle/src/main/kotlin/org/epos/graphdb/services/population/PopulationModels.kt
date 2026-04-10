package org.epos.graphdb.services

/**
 * Result of a population operation.
 */
data class PopulationResult(
    val success: Boolean,
    val sourceUrl: String,
    val subjectsProcessed: Int,
    val triplesInserted: Int,
    val message: String,
    val error: String? = null
)
