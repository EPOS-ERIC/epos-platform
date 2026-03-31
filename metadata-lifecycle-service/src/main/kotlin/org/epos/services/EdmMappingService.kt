package org.epos.services

import io.ktor.client.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import kotlinx.coroutines.runBlocking
import org.eclipse.rdf4j.model.IRI
import org.eclipse.rdf4j.model.Model
import org.eclipse.rdf4j.model.vocabulary.OWL
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.eclipse.rdf4j.model.vocabulary.RDFS
import org.eclipse.rdf4j.rio.RDFFormat
import org.eclipse.rdf4j.rio.Rio
import java.io.StringReader

/**
 * Service for EPOS Data Model (EDM) mappings.
 * 
 * Loads the EDM schema mapping from GitHub and provides:
 * - Property name mappings (RDF property -> EDM property name)
 * - Type mappings (RDF class -> EDM class)
 * - Relation detection (which properties link to other entities)
 * 
 * This enables the API to return responses structured according to the
 * EPOS Data Model Java beans used in the resources-service.
 */
class EdmMappingService(private val httpClient: HttpClient) {
    
    companion object {
        const val EDM_MAPPING_URL = "https://raw.githubusercontent.com/epos-eu/EPOS_Data_Model_Mapping/refs/heads/main/edm-schema-shapes.ttl"
        const val EDM_NAMESPACE = "http://www.epos-eu.org/epos-data-model#"
    }
    
    // Property mappings: RDF property URI -> EDM property name
    private val propertyMappings: MutableMap<String, String> = mutableMapOf()
    
    // Type mappings: RDF class URI -> EDM type info
    private val typeMappings: MutableMap<String, EdmTypeInfo> = mutableMapOf()
    
    // Relations: RDF properties that link to other entities (range is an EDM class)
    private val relationProperties: MutableSet<String> = mutableSetOf()
    
    // Reverse lookup: EDM property name -> RDF property URIs
    private val edmToRdfProperties: MutableMap<String, MutableList<String>> = mutableMapOf()
    
    // EDM class to target RDF classes
    private val edmClassToRdfClasses: MutableMap<String, MutableList<String>> = mutableMapOf()
    
    init {
        runBlocking { loadMappings() }
    }
    
    private suspend fun loadMappings() {
        println("Loading EDM mappings from $EDM_MAPPING_URL...")
        
        try {
            val ttlContent = httpClient.get(EDM_MAPPING_URL).bodyAsText()
            val model = Rio.parse(StringReader(ttlContent), EDM_NAMESPACE, RDFFormat.TURTLE)
            
            parseMappings(model)
            
            println("EDM mappings loaded:")
            println("  - ${propertyMappings.size} property mappings")
            println("  - ${typeMappings.size} type mappings")
            println("  - ${relationProperties.size} relation properties")
        } catch (e: Exception) {
            println("Warning: Failed to load EDM mappings: ${e.message}")
            // Initialize with default mappings
            initializeDefaultMappings()
        }
    }
    
    private fun parseMappings(model: Model) {
        val vf = org.eclipse.rdf4j.model.impl.SimpleValueFactory.getInstance()
        val equivalentProperty = OWL.EQUIVALENTPROPERTY
        val equivalentClass = OWL.EQUIVALENTCLASS
        val domain = RDFS.DOMAIN
        val range = RDFS.RANGE
        
        // Find all EDM properties and their equivalent RDF properties
        model.filter(null, equivalentProperty, null).forEach { stmt ->
            val edmProp = stmt.subject
            val rdfProp = stmt.`object`
            
            if (edmProp is IRI && edmProp.namespace == EDM_NAMESPACE && rdfProp is IRI) {
                val edmName = edmProp.localName
                val rdfUri = rdfProp.stringValue()
                
                propertyMappings[rdfUri] = edmName
                edmToRdfProperties.getOrPut(edmName) { mutableListOf() }.add(rdfUri)
            }
        }
        
        // Find all EDM classes and their equivalent RDF classes
        model.filter(null, equivalentClass, null).forEach { stmt ->
            val edmClass = stmt.subject
            val rdfClass = stmt.`object`
            
            if (edmClass is IRI && edmClass.namespace == EDM_NAMESPACE && rdfClass is IRI) {
                val edmName = edmClass.localName
                val rdfUri = rdfClass.stringValue()
                
                typeMappings[rdfUri] = EdmTypeInfo(edmName, rdfUri)
                edmClassToRdfClasses.getOrPut(edmName) { mutableListOf() }.add(rdfUri)
            }
        }
        
        // Find relation properties (those with EDM class range)
        model.filter(null, range, null).forEach { stmt ->
            val prop = stmt.subject
            val rangeValue = stmt.`object`
            
            if (prop is IRI && prop.namespace == EDM_NAMESPACE && rangeValue is IRI) {
                if (rangeValue.namespace == EDM_NAMESPACE) {
                    // This property links to another EDM type, it's a relation
                    val propName = prop.localName
                    edmToRdfProperties[propName]?.forEach { rdfUri ->
                        relationProperties.add(rdfUri)
                    }
                }
            }
        }
    }
    
    private fun initializeDefaultMappings() {
        // Core property mappings
        val defaultMappings = mapOf(
            // Title/Name
            "http://purl.org/dc/terms/title" to "title",
            "http://schema.org/name" to "name",
            "http://www.w3.org/2004/02/skos/core#prefLabel" to "name",
            "http://www.w3.org/2000/01/rdf-schema#label" to "label",
            
            // Description
            "http://purl.org/dc/terms/description" to "description",
            "http://schema.org/description" to "description",
            "http://www.w3.org/2004/02/skos/core#definition" to "description",
            
            // Identifier
            "http://purl.org/dc/terms/identifier" to "identifier",
            "http://schema.org/identifier" to "identifier",
            
            // Dates
            "http://purl.org/dc/terms/created" to "created",
            "http://purl.org/dc/terms/modified" to "modified",
            "http://purl.org/dc/terms/issued" to "issued",
            "http://www.w3.org/ns/dcat#startDate" to "startDate",
            "http://www.w3.org/ns/dcat#endDate" to "endDate",
            
            // Relations
            "http://purl.org/dc/terms/publisher" to "publisher",
            "http://schema.org/provider" to "provider",
            "http://www.w3.org/ns/dcat#theme" to "category",
            "http://www.w3.org/ns/dcat#contactPoint" to "contactPoint",
            "http://schema.org/contactPoint" to "contactPoint",
            "http://www.w3.org/ns/dcat#distribution" to "distribution",
            "http://purl.org/dc/terms/spatial" to "spatialExtent",
            "http://purl.org/dc/terms/temporal" to "temporalExtent",
            
            // Keywords
            "http://www.w3.org/ns/dcat#keyword" to "keywords",
            
            // Access
            "http://purl.org/dc/terms/accessRights" to "accessRight",
            "http://purl.org/dc/terms/license" to "licence",
            "http://www.w3.org/ns/dcat#downloadURL" to "downloadURL",
            "http://purl.org/dc/terms/format" to "format",
            
            // Organization
            "http://schema.org/legalName" to "legalName",
            "http://schema.org/email" to "email",
            "http://schema.org/telephone" to "telephone",
            "http://schema.org/address" to "address",
            
            // Location
            "http://www.w3.org/ns/locn#geometry" to "location",
            "http://www.opengis.net/ont/geosparql#asWKT" to "location",
            
            // WebService/Operation
            "http://www.w3.org/ns/hydra/core#supportedOperation" to "supportedOperation",
            "http://www.w3.org/ns/hydra/core#method" to "method",
            "http://www.w3.org/ns/hydra/core#returns" to "returns",
            "http://www.w3.org/ns/hydra/core#template" to "template",
            "http://www.w3.org/ns/hydra/core#mapping" to "mapping",
            
            // SKOS
            "http://www.w3.org/2004/02/skos/core#inScheme" to "inScheme",
            "http://www.w3.org/2004/02/skos/core#broader" to "broader",
            "http://www.w3.org/2004/02/skos/core#narrower" to "narrower",
            "http://www.w3.org/2004/02/skos/core#hasTopConcept" to "topConcepts"
        )
        
        propertyMappings.putAll(defaultMappings)
        
        // Default type mappings
        val defaultTypes = mapOf(
            "http://www.w3.org/ns/dcat#Dataset" to "DataProduct",
            "http://www.w3.org/ns/dcat#Distribution" to "Distribution",
            "https://www.epos-eu.org/epos-dcat-ap#WebService" to "WebService",
            "http://schema.org/Organization" to "Organization",
            "http://xmlns.com/foaf/0.1/Organization" to "Organization",
            "http://schema.org/Person" to "Person",
            "http://schema.org/ContactPoint" to "ContactPoint",
            "http://www.w3.org/2004/02/skos/core#Concept" to "Category",
            "http://www.w3.org/2004/02/skos/core#ConceptScheme" to "CategoryScheme",
            "https://www.epos-eu.org/epos-dcat-ap#Facility" to "Facility",
            "https://www.epos-eu.org/epos-dcat-ap#Equipment" to "Equipment",
            "http://www.w3.org/ns/hydra/core#Operation" to "Operation",
            "http://www.w3.org/ns/hydra/core#IriTemplate" to "IriTemplate",
            "http://www.w3.org/ns/hydra/core#IriTemplateMapping" to "Mapping",
            "http://purl.org/dc/terms/Location" to "Location",
            "http://purl.org/dc/terms/PeriodOfTime" to "PeriodOfTime",
            "http://schema.org/PostalAddress" to "Address",
            "http://www.w3.org/2006/vcard/ns#Address" to "Address",
            "http://schema.org/QuantitativeValue" to "QuantitativeValue",
            "http://www.w3.org/ns/adms#Identifier" to "Identifier",
            "http://schema.org/PropertyValue" to "Identifier",
            "http://www.w3.org/ns/hydra/core#ApiDocumentation" to "Documentation",
            "http://schema.org/Service" to "Service",
            "http://schema.org/SoftwareApplication" to "SoftwareApplication",
            "http://schema.org/SoftwareSourceCode" to "SoftwareSourceCode"
        )
        
        defaultTypes.forEach { (rdfUri, edmName) ->
            typeMappings[rdfUri] = EdmTypeInfo(edmName, rdfUri)
        }
        
        // Default relation properties
        relationProperties.addAll(listOf(
            "http://purl.org/dc/terms/publisher",
            "http://schema.org/provider",
            "http://www.w3.org/ns/dcat#theme",
            "http://www.w3.org/ns/dcat#contactPoint",
            "http://schema.org/contactPoint",
            "http://www.w3.org/ns/dcat#distribution",
            "http://purl.org/dc/terms/spatial",
            "http://purl.org/dc/terms/temporal",
            "http://www.w3.org/ns/hydra/core#supportedOperation",
            "http://purl.org/dc/terms/conformsTo",
            "http://www.w3.org/ns/dcat#accessURL",
            "http://schema.org/manufacturer",
            "http://purl.org/dc/terms/isPartOf",
            "http://www.w3.org/2004/02/skos/core#inScheme",
            "http://www.w3.org/2004/02/skos/core#broader",
            "http://www.w3.org/2004/02/skos/core#narrower",
            "http://www.w3.org/2004/02/skos/core#hasTopConcept",
            "http://schema.org/affiliation",
            "http://schema.org/address",
            "http://www.w3.org/ns/hydra/core#mapping",
            "http://www.w3.org/ns/adms#identifier",
            "http://prov.org/qualifiedAttribution"
        ))
    }
    
    /**
     * Get the EDM property name for an RDF property URI.
     */
    fun getEdmPropertyName(rdfPropertyUri: String): String {
        return propertyMappings[rdfPropertyUri] 
            ?: rdfPropertyUri.substringAfterLast("/").substringAfterLast("#")
    }
    
    /**
     * Get the EDM type name for an RDF class URI.
     */
    fun getEdmTypeName(rdfClassUri: String): String {
        return typeMappings[rdfClassUri]?.edmName 
            ?: rdfClassUri.substringAfterLast("/").substringAfterLast("#")
    }
    
    /**
     * Check if an RDF property is a relation (links to another entity).
     */
    fun isRelationProperty(rdfPropertyUri: String): Boolean {
        return relationProperties.contains(rdfPropertyUri)
    }
    
    /**
     * Get all RDF property URIs that map to an EDM property name.
     */
    fun getRdfPropertiesForEdm(edmPropertyName: String): List<String> {
        return edmToRdfProperties[edmPropertyName] ?: emptyList()
    }
    
    /**
     * Get all known relation properties.
     */
    fun getRelationProperties(): Set<String> = relationProperties.toSet()
    
    /**
     * Get type info for an RDF class.
     */
    fun getTypeInfo(rdfClassUri: String): EdmTypeInfo? = typeMappings[rdfClassUri]
    
    /**
     * Get all property mappings.
     */
    fun getAllPropertyMappings(): Map<String, String> = propertyMappings.toMap()
}

/**
 * Information about an EDM type.
 */
data class EdmTypeInfo(
    val edmName: String,
    val rdfClassUri: String
)
