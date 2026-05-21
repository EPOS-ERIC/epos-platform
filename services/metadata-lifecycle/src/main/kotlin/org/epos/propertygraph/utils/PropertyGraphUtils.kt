package org.epos.propertygraph.utils

import org.epos.propertygraph.Config
import java.security.MessageDigest
import java.time.Instant
import java.util.UUID

object UriUtils {
    private val VERSION_PATTERN = Regex(".*/(v\\d+\\.\\d+.*|\\d+\\.\\d+.*)$")
    private val ABSOLUTE_URI_PATTERN = Regex("^[a-zA-Z][a-zA-Z0-9+.-]*:.+")

    fun isAbsolute(uri: String): Boolean = ABSOLUTE_URI_PATTERN.matches(uri)
    fun hasVersion(uri: String): Boolean = VERSION_PATTERN.matches(uri)

    fun extractConceptualUri(versionedUri: String): String =
        if (hasVersion(versionedUri)) versionedUri.substringBeforeLast("/") else versionedUri

    fun ensureVersioned(uri: String, version: String = "v1.0"): String =
        if (hasVersion(uri)) uri else "$uri/$version"

    fun ensureAbsolute(value: String, context: String): String {
        if (isAbsolute(value)) return value
        println("  WARNING: '$value' is not an absolute URI ($context). Converting.")
        val sanitized = value.replace(Regex("[^a-zA-Z0-9_-]"), "_")
        return "${Config.BASE_URI}/$context/$sanitized"
    }

    fun generateActivityUri(): String = "${Config.BASE_URI}/activity/${UUID.randomUUID()}"

    fun generateHash(content: String, length: Int = 8): String =
        MessageDigest.getInstance("MD5")
            .digest(content.toByteArray())
            .joinToString("") { "%02x".format(it) }
            .take(length)

    fun extractVersionNumber(uri: String): String {
        val regex = Regex(".*/v?([0-9.]+)(-draft)?$")
        return regex.find(uri)?.groupValues?.get(1) ?: "1.0"
    }

    fun nextVersion(currentVersion: String): String {
        val segment = currentVersion.substringAfterLast("/").removeSuffix("-draft")
        val match = Regex("v?(\\d+)(?:\\.(\\d+))?.*").find(segment)
        if (match == null) return "2.0"
        val major = match.groupValues[1].toIntOrNull() ?: 1
        return "${major + 1}.0"
    }
}

data class VersionRecord(
    val id: String,
    val status: String,
    val versionNumber: String,
    val data: Map<String, Any>,
    val createdAt: String,
    val updatedAt: String
)

data class ConceptualRecord(
    val id: String,
    val type: String,
    val createdAt: String
)

data class ActivityRecord(
    val id: String,
    val type: String,
    val fromStatus: String?,
    val toStatus: String?,
    val agent: String,
    val group: String,
    val timestamp: String
)

fun Map<String, Any?>.toVersionRecord(): VersionRecord? {
    val v = this["v"] as? Map<*, *> ?: return null
    val structuralKeys = setOf("id", "status", "versionNumber", "createdAt", "updatedAt")
    val dataMap = v.filterKeys { it is String && it !in structuralKeys }
        .mapKeys { (k, _) -> k as String }
        .mapValues { (_, v2) -> v2 ?: "" }
    return VersionRecord(
        id = v["id"] as? String ?: return null,
        status = v["status"] as? String ?: return null,
        versionNumber = v["versionNumber"] as? String ?: return null,
        data = dataMap,
        createdAt = v["createdAt"] as? String ?: "",
        updatedAt = v["updatedAt"] as? String ?: ""
    )
}

fun Map<String, Any?>.toConceptualRecord(): ConceptualRecord? {
    val c = this["c"] as? Map<*, *> ?: return null
    return ConceptualRecord(
        id = c["id"] as? String ?: return null,
        type = c["type"] as? String ?: "",
        createdAt = c["createdAt"] as? String ?: ""
    )
}
