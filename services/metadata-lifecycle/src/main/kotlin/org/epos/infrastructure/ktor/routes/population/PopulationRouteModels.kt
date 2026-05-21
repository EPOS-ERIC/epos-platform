package org.epos.infrastructure.ktor.routes

data class PopulateFromFileRequest(
    val url: String? = null,
    val path: String? = null,
    val indexUrl: String? = null,
)

data class PopulateFromContentRequest(
    val content: String? = null,
    val sourceName: String? = null,
)

data class ImportRevisionFromFileRequest(
    val url: String? = null,
    val path: String? = null,
    val versionMode: String? = null,
    val version: String? = null,
    val replaceDraftIfExists: Boolean? = null,
    val dryRun: Boolean? = null,
)

data class ImportRevisionFromContentRequest(
    val content: String? = null,
    val sourceName: String? = null,
    val versionMode: String? = null,
    val version: String? = null,
    val replaceDraftIfExists: Boolean? = null,
    val dryRun: Boolean? = null,
)

data class ReconcileRelationsRequest(
    val dryRun: Boolean? = null,
    val resourceId: String? = null,
    val maxVersions: Int? = null,
    val batchSize: Int? = null,
)

data class AuditRelationsRequest(
    val resourceId: String? = null,
)
