package org.epos.api.models

import com.fasterxml.jackson.annotation.JsonInclude
import org.epos.api.shacl.PropertyInfo

@JsonInclude(JsonInclude.Include.NON_NULL)
data class EntityResponse(
    val uri: String,
    val type: String,
    val status: String,
    val properties: Map<String, Any>,
    val version: String? = null,
    val conceptualUri: String? = null,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class EntityReference(
    val uri: String,
    val link: String,
    val resourceType: String,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class EntityListResponse(
    val entities: List<EntityResponse>,
    val total: Int,
    val page: Int,
    val pageSize: Int,
)

data class CreateEntityRequest(
    val type: String,
    val properties: Map<String, Any>,
)

data class UpdateEntityRequest(
    val properties: Map<String, Any>,
)

data class ApiResponse(
    val success: Boolean,
    val message: String,
    val data: Any? = null,
)

data class ErrorResponse(
    val error: String,
    val message: String,
    val details: String? = null,
)

data class VersionHistoryResponse(
    val conceptualUri: String,
    val versions: List<VersionInfo>,
)

data class VersionInfo(
    val uri: String,
    val version: String,
    val timestamp: String,
    val agent: String?,
    val group: String?,
    val status: String,
)

data class ResourceTypeInfo(
    val name: String,
    val targetClass: String,
    val properties: List<PropertyInfo>,
    val closed: Boolean = false,
    val description: String? = null,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ShapeDefinitionInfo(
    val id: String,
    val targetClasses: List<String>,
    val properties: List<PropertyInfo>,
    val resolvedRefs: List<String>? = null,
    val name: String? = null,
    val datatype: String? = null,
    val datatypes: List<String>? = null,
    val nodeKind: String? = null,
    val nodeKinds: List<String>? = null,
    val classConstraint: String? = null,
    val classConstraints: List<String>? = null,
    val nodeShapeRef: String? = null,
    val nodeShapeRefs: List<String>? = null,
    val ontologyEquivalentClasses: List<String>? = null,
    val ontologySubClassesOf: List<String>? = null,
    val closed: Boolean = false,
    val description: String? = null,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ResourceTypesResponse(
    val resourceTypes: List<ResourceTypeInfo>,
    val shapeDefinitions: Map<String, ShapeDefinitionInfo>? = null,
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class BatchOperationResponse(
    val success: Boolean,
    val message: String,
    val results: List<ApiResponse>,
    val data: Map<String, Any>? = null,
)

data class BatchOperationRequest(
    val entityUris: List<String>,
)
