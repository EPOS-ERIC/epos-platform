package org.epos.infrastructure.ktor.routes

import com.fasterxml.jackson.annotation.JsonInclude
import com.fasterxml.jackson.annotation.JsonProperty

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacySearchResponse(
    val results: LegacyNode? = null,
    val filters: List<LegacyNodeFilter>? = null,
    val errorMessage: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyNode(
    val children: List<LegacyNode>? = null,
    val distributions: List<LegacyDiscoveryItem>? = null,
    val ddss: String? = null,
    val id: String? = null,
    val code: String? = null,
    val linkUrl: String? = null,
    val imgUrl: String? = null,
    val color: String? = null,
    val name: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyNodeFilter(
    val children: List<LegacyNodeFilter>? = null,
    val id: String? = null,
    val name: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyDiscoveryItem(
    val href: String? = null,
    val hrefExtended: String? = null,
    val id: String? = null,
    val uid: String? = null,
    val metaId: String? = null,
    val title: String? = null,
    val description: String? = null,
    val versioningStatus: String? = null,
    val availableFormats: List<LegacyAvailableFormat>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyAvailableFormat(
    val method: String? = null,
    val inputFormat: String? = null,
    val pluginId: String? = null,
    val label: String? = null,
    val format: String? = null,
    val originalFormat: String? = null,
    val href: String? = null,
    val type: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyDistribution(
    val errorMessage: String? = null,
    val operationid: String? = null,
    val href: String? = null,
    val hrefExtended: String? = null,
    val categories: LegacyNode? = null,
    val id: String? = null,
    val uid: String? = null,
    val metaId: String? = null,
    val type: String? = null,
    val title: String? = null,
    val description: String? = null,
    val license: String? = null,
    val downloadURL: String? = null,
    val keywords: List<String>? = null,
    val dataProvider: List<LegacyDataServiceProvider>? = null,
    val frequencyUpdate: String? = null,
    val internalID: List<String>? = null,
    val DOI: List<String>? = null,
    val spatial: SpatialInfo? = null,
    val temporalCoverage: TemporalCoverage? = null,
    val scienceDomain: List<String>? = null,
    val qualityAssurance: String? = null,
    val accessRights: String? = null,
    val availableFormats: List<LegacyAvailableFormat>? = null,
    val availableContactPoints: List<LegacyAvailableContactPoint>? = null,
    val versioningStatus: String? = null,
    val editorId: String? = null,
    val serviceName: String? = null,
    val serviceDescription: String? = null,
    val serviceProvider: LegacyDataServiceProvider? = null,
    val serviceSpatial: SpatialInfo? = null,
    val serviceTemporalCoverage: TemporalCoverage? = null,
    val serviceEndpoint: String? = null,
    val serviceDocumentation: String? = null,
    val serviceType: List<String>? = null,
    val endpoint: String? = null,
    val serviceParameters: List<LegacyServiceParameter>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyDistributionExtended(
    val errorMessage: String? = null,
    val href: String? = null,
    val hrefExtended: String? = null,
    val categories: LegacyNode? = null,
    val id: String? = null,
    val uid: String? = null,
    val metaId: String? = null,
    val type: String? = null,
    val title: String? = null,
    val description: String? = null,
    val license: String? = null,
    val downloadURL: String? = null,
    val keywords: List<String>? = null,
    val frequencyUpdate: String? = null,
    val scienceDomain: List<String>? = null,
    val hasQualityAnnotation: String? = null,
    val availableFormats: List<LegacyAvailableFormat>? = null,
    val availableContactPoints: List<LegacyAvailableContactPoint>? = null,
    val relatedDataProducts: List<LegacyDataProduct>? = null,
    val relatedWebservice: List<LegacyWebservice>? = null,
    val versioningStatus: String? = null,
    val editorId: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyDataServiceProvider(
    val dataProviderLegalName: String? = null,
    val dataProviderUrl: String? = null,
    val country: String? = null,
    val uid: String? = null,
    val metaid: String? = null,
    val instanceid: String? = null,
    val relatedDataProvider: List<LegacyDataServiceProvider>? = null,
    val relatedDataServiceProvider: List<LegacyDataServiceProvider>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyDataProduct(
    val href: String? = null,
    val id: String? = null,
    val uid: String? = null,
    val type: String? = null,
    val title: String? = null,
    val description: String? = null,
    val license: String? = null,
    val downloadURL: String? = null,
    val keywords: List<String>? = null,
    val dataProvider: List<LegacyDataServiceProvider>? = null,
    val frequencyUpdate: String? = null,
    val identifiers: List<Map<String, String>>? = null,
    val scienceDomain: List<String>? = null,
    val hasQualityAnnotation: String? = null,
    val accessRights: String? = null,
    val provenance: List<Map<String, String>>? = null,
    val contactPoints: List<Map<String, Any>>? = null,
    val spatial: SpatialInfo? = null,
    val temporalCoverage: List<TemporalCoverage>? = null,
    val categories: LegacyNode? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyWebservice(
    val href: String? = null,
    val id: String? = null,
    val metaid: String? = null,
    val uid: String? = null,
    val name: String? = null,
    val description: String? = null,
    val provider: LegacyDataServiceProvider? = null,
    val spatial: SpatialInfo? = null,
    val documentation: String? = null,
    val temporalCoverage: List<TemporalCoverage>? = null,
    val type: List<String>? = null,
    val contactPoints: List<Map<String, Any>>? = null,
    val operations: List<OperationInfo>? = null,
    val categories: Map<String, List<String>>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyOrganizationBean(
    val id: String? = null,
    val logo: String? = null,
    val url: String? = null,
    val name: String? = null,
    val country: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyMonitoringBean(
    val identifier: String? = null,
    val name: String? = null,
    val originalURL: String? = null,
    val validationRules: List<LegacyValidationRule>? = null,
    val contactPoints: List<LegacyContact>? = null,
    val TCSGroup: String? = null,
    val id: String? = null,
    val uid: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyValidationRule(
    val type: String? = null,
    val encodingFormat: String? = null,
    val schemaVersion: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyContact(
    val name: String? = null,
    val role: String? = null,
    val email: List<String>? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyAvailableContactPoint(
    val href: String? = null,
    val type: String? = null
)

@JsonInclude(JsonInclude.Include.NON_NULL)
data class LegacyServiceParameter(
    @JsonProperty("Enum")
    val enum: List<String>? = null,
    val defaultValue: String? = null,
    val label: String? = null,
    val maxValue: String? = null,
    val minValue: String? = null,
    val multipleValue: String? = null,
    val name: String? = null,
    val property: String? = null,
    val readOnlyValue: String? = null,
    val required: Boolean? = null,
    val type: String? = null,
    val valuePattern: String? = null
)
