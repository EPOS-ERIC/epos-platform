package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SearchResponse(
    val results: List<SearchResultItem>,
    val total: Int,
    val page: Int,
    val pageSize: Int,
    val totalPages: Int,
    val facets: SearchFacets? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class GroupedSearchResponse(
    val results: FacetNode,
    val filters: List<FilterNode>,
    val total: Int,
    val page: Int,
    val pageSize: Int,
    val totalPages: Int
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class FacetNode(
    val name: String,
    val ddss: String? = null,
    val id: String? = null,
    val code: String? = null,
    val linkUrl: String? = null,
    val imgUrl: String? = null,
    val color: String? = null,
    val children: List<FacetNode>? = null,
    val distributions: List<SearchResultItem>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class FilterNode(
    val name: String,
    val id: String? = null,
    val children: List<FilterNode>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SearchResultItem(
    val id: String,
    val type: String,
    val title: String?,
    val description: String?,
    val modified: String?,
    val href: String,
    val availableFormats: List<AvailableFormat>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SearchFacets(
    val types: List<FacetValue>,
    val keywords: List<FacetValue>,
    val conceptSchemes: List<HierarchicalFacet>? = null,
    val dataProviders: List<HierarchicalFacet>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class FacetValue(
    val value: String,
    val label: String,
    val count: Int
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class HierarchicalFacet(
    val id: String,
    val label: String,
    val description: String? = null,
    val count: Int,
    val children: List<FacetValue>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class EntityDetailsResponse(
    val id: String,
    val type: String,
    val typeUri: String,
    val status: String,
    val properties: Map<String, Any>
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class EdmEntityResponse(
    val id: String,
    val instanceId: String? = null,
    val metaId: String? = null,
    val type: String,
    val typeUri: String,
    val status: String,
    val title: String? = null,
    val name: String? = null,
    val description: String? = null,
    val identifier: Any? = null,
    val keywords: List<String>? = null,
    val created: String? = null,
    val modified: String? = null,
    val issued: String? = null,
    val properties: Map<String, Any>? = null,
    val relations: Map<String, Any>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class RelatedEntity(
    val id: String,
    val type: String? = null,
    val name: String? = null,
    val href: String
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AggregatedEntityResponse(
    val id: String,
    val uid: String? = null,
    val metaId: String? = null,
    val type: String,
    val status: String,
    val title: String? = null,
    val description: String? = null,
    val license: String? = null,
    val downloadURL: String? = null,
    val keywords: List<String>? = null,
    val frequencyUpdate: String? = null,
    val scienceDomain: List<String>? = null,
    val qualityAssurance: String? = null,
    val editorId: String? = null,
    val href: String,
    val hrefExtended: String,
    val spatial: SpatialInfo? = null,
    val temporalCoverage: TemporalCoverage? = null,
    val dataProvider: List<DataServiceProvider>? = null,
    val identifiers: List<IdentifierInfo>? = null,
    val contactPoints: List<ContactPointInfo>? = null,
    val availableFormats: List<AvailableFormat>? = null,
    val categories: FacetNode? = null,
    val relatedDataProducts: List<DataProductInfo>? = null,
    val relatedWebservices: List<WebserviceInfo>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class SpatialInfo(
    val wkid: Int? = 4326,
    val location: String? = null,
    val paths: List<Any>? = null,
    val x: Double? = null,
    val y: Double? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class TemporalCoverage(
    val startDate: String? = null,
    val endDate: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class DataServiceProvider(
    val dataProviderLegalName: String? = null,
    val dataProviderUrl: String? = null,
    val country: String? = null,
    val uid: String? = null,
    val metaid: String? = null,
    val instanceid: String? = null,
    val relatedDataProvider: List<DataServiceProvider>? = null,
    val relatedDataServiceProvider: List<DataServiceProvider>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class IdentifierInfo(
    val type: String,
    val value: String
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ContactPointInfo(
    val id: String? = null,
    val metaid: String? = null,
    val uid: String? = null,
    val email: String? = null,
    val telephone: String? = null,
    val role: String? = null,
    val person: PersonInfo? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class PersonInfo(
    val id: String? = null,
    val metaid: String? = null,
    val uid: String? = null,
    val name: String? = null,
    val email: String? = null,
    val affiliation: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class AvailableFormat(
    val label: String? = null,
    val format: String? = null,
    val originalFormat: String? = null,
    val href: String? = null,
    val type: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class DataProductInfo(
    val id: String,
    val uid: String? = null,
    val type: String? = null,
    val version: String? = null,
    val title: String? = null,
    val description: String? = null,
    val license: String? = null,
    val downloadURL: String? = null,
    val keywords: List<String>? = null,
    val frequencyUpdate: String? = null,
    val identifiers: List<IdentifierInfo>? = null,
    val spatial: SpatialInfo? = null,
    val temporalCoverage: List<TemporalCoverage>? = null,
    val dataProvider: List<DataServiceProvider>? = null,
    val scienceDomain: List<String>? = null,
    val qualityAssurance: String? = null,
    val accessRights: String? = null,
    val provenance: List<Map<String, String>>? = null,
    val contactPoints: List<ContactPointInfo>? = null,
    val categories: FacetNode? = null,
    val href: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class WebserviceInfo(
    val id: String,
    val metaid: String? = null,
    val uid: String? = null,
    val name: String? = null,
    val description: String? = null,
    val provider: DataServiceProvider? = null,
    val spatial: SpatialInfo? = null,
    val temporalCoverage: List<TemporalCoverage>? = null,
    val documentation: String? = null,
    val type: List<String>? = null,
    val contactPoints: List<ContactPointInfo>? = null,
    val categories: Map<String, List<String>>? = null,
    val operations: List<OperationInfo>? = null,
    val href: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class OperationInfo(
    val id: String,
    val uid: String? = null,
    val method: String? = null,
    val endpoint: String? = null,
    val parameters: List<ServiceParameter>? = null,
    val href: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class ServiceParameter(
    val name: String,
    val label: String? = null,
    val type: String? = null,
    val property: String? = null,
    val required: Boolean = false,
    val minValue: String? = null,
    val maxValue: String? = null,
    val defaultValue: String? = null,
    val enumValues: List<String>? = null,
    val valuePattern: String? = null,
    val multipleValues: Boolean = false,
    val readOnlyValue: String? = null
)
