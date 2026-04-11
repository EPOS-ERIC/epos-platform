package org.epos.api.controllers

import org.eclipse.rdf4j.model.impl.LinkedHashModel
import org.eclipse.rdf4j.model.util.Values
import org.eclipse.rdf4j.model.vocabulary.RDF
import org.epos.api.models.*
import org.epos.api.shacl.ShaclParser
import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.services.EntityLifecycleService
import org.epos.graphdb.services.ProvenanceService
import java.util.UUID

class ResourceController(
    private val graphDBClient: GraphDBClient,
    private val lifecycleService: EntityLifecycleService,
    private val provenanceService: ProvenanceService
) {
    var resourceTypes: List<ResourceTypeInfo> = listOf()

    fun setResourceType(resourceTypes: List<ResourceTypeInfo>) {
        this.resourceTypes = resourceTypes
    }

    fun getResourceType(): List<ResourceTypeInfo> {
        return resourceTypes
    }

    /**
     * List all entities of a specific type
     */
    fun listEntities(
        resourceType: String,
        status: String = "published",
        page: Int = 1,
        pageSize: Int = 20
    ): EntityListResponse {
        val graphUri = getGraphUriForStatus(status)
        val typeUri = getTypeUri(resourceType)

        val offset = (page - 1) * pageSize

        // Query to get entities
        val query = """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            PREFIX dcterms: <http://purl.org/dc/terms/>
            
            SELECT ?entity ?conceptual
            FROM <$graphUri>
            FROM <${Config.GRAPH_PROVENANCE}>
            WHERE {
                ?entity rdf:type <$typeUri> .
                OPTIONAL {
                    ?entity dcterms:isVersionOf ?conceptual .
                }
            }
            ORDER BY ?entity
            LIMIT $pageSize
            OFFSET $offset
        """.trimIndent()

        val results = graphDBClient.query(query)

        val entities = results.map { row ->
            val entityUri = row["entity"] ?: ""
            val conceptualUri = row["conceptual"]

            EntityResponse(
                uri = entityUri,
                type = resourceType,
                status = status,
                properties = getEntityProperties(entityUri, graphUri),
                conceptualUri = conceptualUri
            )
        }

        // Get total count
        val countQuery = """
            PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
            SELECT (COUNT(?entity) as ?count)
            FROM <$graphUri>
            WHERE {
                ?entity rdf:type <$typeUri> .
            }
        """.trimIndent()

        val total = graphDBClient.query(countQuery)
            .firstOrNull()?.get("count")?.toIntOrNull() ?: 0

        return EntityListResponse(
            entities = entities,
            total = total,
            page = page,
            pageSize = pageSize
        )
    }

    /**
     * Checks whether a string value is a managed URI
     */
    private fun isManagedUri(value: String): Boolean {
        return value.startsWith("http://") || value.startsWith("https://")
    }

    /**
     * Extracts resource type from a URI
     */
    private fun extractResourceTypeFromUri(uri: String): String? {
        val query = """
        PREFIX rdf: <http://www.w3.org/1999/02/22-rdf-syntax-ns#>
        
        SELECT ?type
        WHERE {
            GRAPH ?g {
                <$uri> rdf:type ?type .
            }
        }
        LIMIT 1
    """.trimIndent()

        try {
            val results = graphDBClient.query(query)
            val rdfType = results.firstOrNull()?.get("type")

            if (rdfType != null) {
                resourceTypes.forEach { resourceType ->
                    if (resourceType.targetClass.equals(rdfType, ignoreCase = true)) {
                        return resourceType.name
                    }
                }
            }
        } catch (e: Exception) {
            println("⚠️ Warning: Could not retrieve type for URI $uri: ${e.message}")
        }

        return null
    }

    /**
     * Converts a value into EntityReference when it is a managed URI
     */
    private fun enrichValue(value: Any): Any {
        return when (value) {
            is String -> {
                if (isManagedUri(value)) {
                    val resourceType = extractResourceTypeFromUri(value)
                    if (resourceType != null) {
                        EntityReference(
                            uri = value,
                            link = "/api/$resourceType/$value",
                            resourceType = "$resourceType"
                        )
                    } else {
                        value
                    }
                } else {
                    value
                }
            }
            is List<*> -> value.map { if (it != null) enrichValue(it) else it }
            is Map<*, *> -> value.mapValues { if (it.value != null) enrichValue(it.value!!) else it.value }
            else -> value
        }
    }

    /**
     * Get a single entity by URI
     */
    fun getEntity(entityUri: String, status: String = "published", enrichReferences: Boolean = true): EntityResponse? {
        val graphUri = getGraphUriForStatus(status)

        // Check if entity exists
        val exists = graphDBClient.ask("""
            ASK {
                GRAPH <$graphUri> {
                    <$entityUri> ?p ?o .
                }
            }
        """.trimIndent())

        if (!exists) return null

        val properties = getEntityProperties(entityUri, graphUri)

        // Enrich references if requested
        val enrichedProperties = if (enrichReferences) {
            properties.mapValues { (_, value) -> enrichValue(value) }
        } else {
            properties
        }

        val typeUri = enrichedProperties["type"] as? String ?: ""
        val resourceType = extractResourceName(typeUri)

        // Get conceptual URI
        val conceptualQuery = """
            PREFIX dcterms: <http://purl.org/dc/terms/>
            SELECT ?conceptual
            FROM <${Config.GRAPH_PROVENANCE}>
            WHERE {
                <$entityUri> dcterms:isVersionOf ?conceptual .
            }
        """.trimIndent()

        val conceptualUri = graphDBClient.query(conceptualQuery)
            .firstOrNull()?.get("conceptual")

        return EntityResponse(
            uri = entityUri,
            type = resourceType,
            status = status,
            properties = enrichedProperties,
            conceptualUri = conceptualUri
        )
    }

    /**
     * Get entity as RDF in various formats (includes blank nodes)
     */
    fun getEntityAsRdf(entityUri: String, status: String = "published", format: org.eclipse.rdf4j.rio.RDFFormat): String {
        val graphUri = getGraphUriForStatus(status)

        val model = graphDBClient.getSubjectTriples(entityUri, graphUri)

        if (model.isEmpty()) {
            return ""
        }

        // Add triples for connected blank nodes.
        val blankNodes = model.filter {
            it.`object` is org.eclipse.rdf4j.model.BNode
        }.map { it.`object` as org.eclipse.rdf4j.model.BNode }

        blankNodes.forEach { bnode ->
            val bnodeTriples = graphDBClient.repository.connection.use { conn ->
                val bnModel = org.eclipse.rdf4j.model.impl.LinkedHashModel()
                conn.getStatements(bnode, null, null, conn.valueFactory.createIRI(graphUri)).forEach {
                    bnModel.add(it)
                }
                bnModel
            }
            model.addAll(bnodeTriples)
        }

        val stringWriter = java.io.StringWriter()
        org.eclipse.rdf4j.rio.Rio.write(model, stringWriter, format)
        return stringWriter.toString()
    }

    /**
     * Create a new draft entity
     */
    fun createEntity(request: CreateEntityRequest, agent: String): ApiResponse {
        val vf = Values.getValueFactory()
        val model = LinkedHashModel()

        // Generate URI
        val entityId = UUID.randomUUID().toString()
        val conceptualUri = "${Config.ID_BASE}/${request.type}/$entityId"
        val draftUri = "$conceptualUri/1.0-draft"

        val entityIRI = vf.createIRI(draftUri)
        val typeIRI = vf.createIRI(getTypeUri(request.type))

        // Add type
        model.add(entityIRI, RDF.TYPE, typeIRI)

        // Add properties
        request.properties.forEach { (key, value) ->
            val propertyIRI = vf.createIRI(key)
            when (value) {
                is String -> model.add(entityIRI, propertyIRI, vf.createLiteral(value))
                is Number -> model.add(entityIRI, propertyIRI, vf.createLiteral(value.toString()))
                is Boolean -> model.add(entityIRI, propertyIRI, vf.createLiteral(value))
                is Map<*, *> -> {
                    // Handle nested objects (create blank nodes)
                    val blankNode = vf.createBNode()
                    model.add(entityIRI, propertyIRI, blankNode)
                    @Suppress("UNCHECKED_CAST")
                    val nestedMap = value as Map<String, Any>
                    nestedMap.forEach { (nestedKey, nestedValue) ->
                        val nestedProp = vf.createIRI(nestedKey)
                        model.add(blankNode, nestedProp, vf.createLiteral(nestedValue.toString()))
                    }
                }
            }
        }

        // Add version info
        model.add(entityIRI, vf.createIRI("http://purl.org/pav/version"), vf.createLiteral("1.0"))
        model.add(entityIRI, org.eclipse.rdf4j.model.vocabulary.DCTERMS.IS_VERSION_OF, vf.createIRI(conceptualUri))

        // Insert into draft graph
        graphDBClient.insertIntoGraph(model, Config.GRAPH_DRAFT)

        // Record provenance
        provenanceService.recordDraftCreation(
            newVersionUri = draftUri,
            basedOnVersionUri = draftUri,
            conceptualUri = conceptualUri,
            agent = agent
        )

        // Create conceptual entity
        val conceptualModel = LinkedHashModel()
        conceptualModel.add(vf.createIRI(conceptualUri), RDF.TYPE, vf.createIRI("http://www.w3.org/ns/dcat#Resource"))
        graphDBClient.insertIntoGraph(conceptualModel, Config.GRAPH_CONCEPTUAL)

        return ApiResponse(
            success = true,
            message = "Entity created successfully",
            data = mapOf(
                "uri" to draftUri,
                "conceptualUri" to conceptualUri,
                "status" to "draft"
            )
        )
    }

    /**
     * Update an existing draft entity
     */
    fun updateEntity(entityUri: String, request: UpdateEntityRequest, agent: String): ApiResponse {
        // Check if entity exists in draft
        val exists = graphDBClient.ask("""
            ASK {
                GRAPH <${Config.GRAPH_DRAFT}> {
                    <$entityUri> ?p ?o .
                }
            }
        """.trimIndent())

        if (!exists) {
            return ApiResponse(success = false, message = "Entity not found in draft status")
        }

        // Get existing model
        val existingModel = graphDBClient.getSubjectTriples(entityUri, Config.GRAPH_DRAFT)

        // Remove old property values (keep type and version info)
        val vf = Values.getValueFactory()
        val entityIRI = vf.createIRI(entityUri)

        val propertiesToKeep = setOf(
            RDF.TYPE.stringValue(),
            "http://purl.org/pav/version",
            "http://purl.org/dc/terms/isVersionOf"
        )

        existingModel.removeIf { statement ->
            statement.subject == entityIRI && statement.predicate.stringValue() !in propertiesToKeep
        }

        // Add new properties
        request.properties.forEach { (key, value) ->
            val propertyIRI = vf.createIRI(key)
            when (value) {
                is String -> existingModel.add(entityIRI, propertyIRI, vf.createLiteral(value))
                is Number -> existingModel.add(entityIRI, propertyIRI, vf.createLiteral(value.toString()))
                is Boolean -> existingModel.add(entityIRI, propertyIRI, vf.createLiteral(value))
            }
        }

        // Delete old and insert new
        graphDBClient.deleteFromGraph(entityUri, Config.GRAPH_DRAFT)
        graphDBClient.insertIntoGraph(existingModel, Config.GRAPH_DRAFT)

        return ApiResponse(
            success = true,
            message = "Entity updated successfully",
            data = mapOf("uri" to entityUri)
        )
    }

    /**
     * Submit a draft for review
     */
    fun submitEntity(entityUri: String, agent: String): ApiResponse {
        lifecycleService.submitDraft(entityUri, agent)

        return ApiResponse(
            success = true,
            message = "Entity submitted for review",
            data = mapOf("uri" to entityUri, "status" to "submitted")
        )
    }

    /**
     * Publish a submitted entity
     */
    fun publishEntity(entityUri: String, agent: String): ApiResponse {
        lifecycleService.publish(entityUri, agent)

        val finalUri = entityUri.removeSuffix("-draft")
        return ApiResponse(
            success = true,
            message = "Entity published successfully",
            data = mapOf("uri" to finalUri, "status" to "published")
        )
    }

    /**
     * Create a new draft from published version
     */
    fun createNewDraft(conceptualUri: String, newVersion: String, agent: String): ApiResponse {
        try {
            val draftUri = lifecycleService.createDraft(
                conceptualUri = conceptualUri,
                newVersionNumber = newVersion,
                agent = agent
            )

            return ApiResponse(
                success = true,
                message = "New draft created successfully",
                data = mapOf(
                    "uri" to draftUri,
                    "conceptualUri" to conceptualUri,
                    "version" to newVersion,
                    "status" to "draft"
                )
            )
        } catch (e: Exception) {
            return ApiResponse(
                success = false,
                message = "Failed to create draft: ${e.message}"
            )
        }
    }

    /**
     * Delete a draft entity
     */
    fun deleteEntity(entityUri: String): ApiResponse {
        lifecycleService.deleteDraft(entityUri)

        return ApiResponse(
            success = true,
            message = "Entity deleted successfully"
        )
    }

    /**
     * Get version history
     */
    fun getVersionHistory(conceptualUri: String): VersionHistoryResponse {
        val history = provenanceService.getVersionHistory(conceptualUri)

        val versions = history.map { version ->
            val versionUri = version["version"] ?: ""
            VersionInfo(
                uri = versionUri,
                version = extractVersionNumber(versionUri),
                timestamp = version["timestamp"] ?: "",
                agent = version["agent"],
                status = determineStatusForVersion(versionUri)
            )
        }

        return VersionHistoryResponse(
            conceptualUri = conceptualUri,
            versions = versions
        )
    }

    /**
     * Determines version status by checking all graphs
     */
    private fun determineStatusForVersion(versionUri: String): String {
        val inDraft = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_DRAFT}> { <$versionUri> ?p ?o . } }
        """.trimIndent())
        if (inDraft) return "draft"

        val inSubmitted = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_SUBMITTED}> { <$versionUri> ?p ?o . } }
        """.trimIndent())
        if (inSubmitted) return "submitted"

        val inPublished = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_PUBLISHED}> { <$versionUri> ?p ?o . } }
        """.trimIndent())
        if (inPublished) return "published"

        val inArchived = graphDBClient.ask("""
            ASK { GRAPH <${Config.GRAPH_ARCHIVED}> { <$versionUri> ?p ?o . } }
        """.trimIndent())
        if (inArchived) return "archived"

        return "unknown"
    }

    /**
     * Submit multiple entities for review (batch operation)
     */
    fun submitEntityBatch(entityUris: List<String>, agent: String): BatchOperationResponse {
        val results = entityUris.map { uri ->
            try {
                submitEntity(uri, agent)
            } catch (e: Exception) {
                ApiResponse(
                    success = false,
                    message = "Failed to submit $uri: ${e.message}",
                    data = mapOf("uri" to uri)
                )
            }
        }

        val successCount = results.count { it.success }
        val failureCount = results.count { !it.success }

        return BatchOperationResponse(
            success = failureCount == 0,
            message = "Batch submit completed: $successCount succeeded, $failureCount failed",
            results = results
        )
    }

    /**
     * Publish multiple entities (batch operation with dependency ordering)
     */
    fun publishEntityBatch(entityUris: List<String>, agent: String): BatchOperationResponse {
        // Sort URIs by dependency order (referenced entities first)
        val sortedUris = sortByDependencies(entityUris)

        val results = sortedUris.map { uri ->
            try {
                publishEntity(uri, agent)
            } catch (e: Exception) {
                ApiResponse(
                    success = false,
                    message = "Failed to publish $uri: ${e.message}",
                    data = mapOf("uri" to uri)
                )
            }
        }

        val successCount = results.count { it.success }
        val failureCount = results.count { !it.success }

        return BatchOperationResponse(
            success = failureCount == 0,
            message = "Batch publish completed: $successCount succeeded, $failureCount failed",
            results = results,
            data = mapOf("executionOrder" to sortedUris)
        )
    }

    /**
     * Sort entities by dependencies - entities that are referenced by others come first
     */
    private fun sortByDependencies(entityUris: List<String>): List<String> {
        val dependencyGraph = mutableMapOf<String, MutableSet<String>>()

        // Build dependency graph: entity -> list of entities it depends on
        entityUris.forEach { uri ->
            dependencyGraph[uri] = mutableSetOf()

            val query = """
            SELECT DISTINCT ?dependency
            FROM <${Config.GRAPH_SUBMITTED}>
            WHERE {
                <$uri> ?p ?dependency .
                FILTER(isIRI(?dependency))
            }
        """.trimIndent()

            try {
                val deps = graphDBClient.query(query)
                    .mapNotNull { it["dependency"] }
                    .filter { it in entityUris }

                dependencyGraph[uri]?.addAll(deps)
            } catch (e: Exception) {
                println("Warning: Could not determine dependencies for $uri: ${e.message}")
            }
        }

        // Topological sort using Kahn's algorithm
        val sorted = mutableListOf<String>()
        val inDegree = entityUris.associateWith { uri ->
            dependencyGraph.values.count { it.contains(uri) }
        }.toMutableMap()

        val queue = ArrayDeque(inDegree.filter { it.value == 0 }.map { it.key })

        while (queue.isNotEmpty()) {
            val current = queue.removeFirst()
            sorted.add(current)

            dependencyGraph[current]?.forEach { dep ->
                inDegree[dep] = (inDegree[dep] ?: 0) - 1
                if (inDegree[dep] == 0) {
                    queue.add(dep)
                }
            }
        }

        // If there are remaining items (circular dependencies), add them at the end
        val remaining = entityUris.filter { it !in sorted }
        sorted.addAll(remaining)

        return sorted
    }

    // Helper methods

    private fun getGraphUriForStatus(status: String): String = when (status.lowercase()) {
        "draft" -> Config.GRAPH_DRAFT
        "submitted" -> Config.GRAPH_SUBMITTED
        "published" -> Config.GRAPH_PUBLISHED
        "archived" -> Config.GRAPH_ARCHIVED
        else -> Config.GRAPH_PUBLISHED
    }

    fun getTypeUri(resourceType: String): String {
        for (type in resourceTypes) {
            if (type.name.equals(resourceType, ignoreCase = true)) {
                return type.targetClass
            }
        }
        return ""
    }

    private fun getEntityProperties(entityUri: String, graphUri: String): Map<String, Any> {
        val query = """
            SELECT ?p ?o
            FROM <$graphUri>
            WHERE {
                <$entityUri> ?p ?o .
            }
        """.trimIndent()

        val results = graphDBClient.query(query)
        val properties = mutableMapOf<String, Any>()

        results.forEach { row ->
            val predicate = row["p"] ?: ""
            val value = row["o"] ?: ""

            val propertyName = extractPropertyName(predicate)

            // If the value is a blank node (genid-...), expand it.
            if (value.startsWith("genid-") || value.startsWith("_:")) {
                val blankNodeProperties = getBlankNodeProperties(value, graphUri)
                properties[propertyName] = blankNodeProperties
            } else {
                properties[propertyName] = value
            }
        }

        return properties
    }

    /**
     * Retrieves properties of a blank node
     */
    private fun getBlankNodeProperties(blankNodeId: String, graphUri: String): Map<String, Any> {
        val query = """
            SELECT ?p ?o
            FROM <$graphUri>
            WHERE {
                ?bn ?p ?o .
                FILTER(str(?bn) = "$blankNodeId")
            }
        """.trimIndent()

        val results = graphDBClient.query(query)
        val properties = mutableMapOf<String, Any>()

        results.forEach { row ->
            val predicate = row["p"] ?: ""
            val value = row["o"] ?: ""

            val propertyName = extractPropertyName(predicate)

            // Recurse into nested blank nodes.
            if (value.startsWith("genid-") || value.startsWith("_:")) {
                properties[propertyName] = getBlankNodeProperties(value, graphUri)
            } else {
                properties[propertyName] = value
            }
        }

        return properties
    }

    /**
     * Get enhanced label for a group by trying multiple sources
     */
    private fun getEnhancedGroupLabel(groupKey: String, groupResults: List<Map<String, String>>): String? {
        // First, try to get label from the query results
        val labelFromQuery = groupResults.firstOrNull()?.get("groupLabel")
        if (!labelFromQuery.isNullOrBlank()) {
            return labelFromQuery
        }

        // If not found, query across all graphs for the label
        val labelQuery = """
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>
        PREFIX foaf: <http://xmlns.com/foaf/0.1/>
        PREFIX dcterms: <http://purl.org/dc/terms/>
        PREFIX schema: <http://schema.org/>
        
        SELECT ?label
        WHERE {
            GRAPH ?g {
                <$groupKey> ?labelProp ?label .
                FILTER(?labelProp IN (rdfs:label, skos:prefLabel, foaf:name, dcterms:title, schema:name))
            }
        }
        LIMIT 1
    """.trimIndent()

        return try {
            graphDBClient.query(labelQuery).firstOrNull()?.get("label")
        } catch (e: Exception) {
            null
        }
    }

    /**
     * Resolve property name to full URI
     * Supports common property names and full URIs
     */
    private fun resolvePropertyUri(propertyName: String): String {
        // If already a full URI, return as-is
        if (propertyName.startsWith("http://") || propertyName.startsWith("https://")) {
            return propertyName
        }

        // Common property mappings
        val commonProperties = mapOf(
            // Organization-related
            "publisher" to "http://purl.org/dc/terms/publisher",
            "provider" to "http://purl.org/dc/terms/provider",
            "creator" to "http://purl.org/dc/terms/creator",

            // Classification
            "theme" to "http://www.w3.org/ns/dcat#theme",
            "concept" to "http://www.w3.org/ns/dcat#theme",
            "category" to "http://www.w3.org/ns/dcat#theme",
            "keyword" to "http://www.w3.org/ns/dcat#keyword",

            // SKOS relationships
            "broader" to "http://www.w3.org/2004/02/skos/core#broader",
            "narrower" to "http://www.w3.org/2004/02/skos/core#narrower",
            "related" to "http://www.w3.org/2004/02/skos/core#related",
            "inScheme" to "http://www.w3.org/2004/02/skos/core#inScheme",
            "topConceptOf" to "http://www.w3.org/2004/02/skos/core#topConceptOf",
            "conceptscheme" to "http://www.w3.org/2004/02/skos/core#inScheme",

            // Contact and relations
            "contactPoint" to "http://www.w3.org/ns/dcat#contactPoint",
            "qualifiedRelation" to "http://www.w3.org/ns/dcat#qualifiedRelation",

            // Spatial and temporal
            "spatial" to "http://purl.org/dc/terms/spatial",
            "spatialResolutionInMeters" to "http://www.w3.org/ns/dcat#spatialResolutionInMeters",
            "temporal" to "http://purl.org/dc/terms/temporal",
            "temporalResolution" to "http://www.w3.org/ns/dcat#temporalResolution",

            // Resource metadata
            "type" to "http://purl.org/dc/terms/type",
            "format" to "http://purl.org/dc/terms/format",
            "mediaType" to "http://www.w3.org/ns/dcat#mediaType",
            "license" to "http://purl.org/dc/terms/license",
            "rights" to "http://purl.org/dc/terms/rights",
            "accessRights" to "http://purl.org/dc/terms/accessRights",
            "language" to "http://purl.org/dc/terms/language",

            // Standards and conformance
            "conformsTo" to "http://purl.org/dc/terms/conformsTo",
            "standard" to "http://purl.org/dc/terms/conformsTo",

            // Relationships
            "isPartOf" to "http://purl.org/dc/terms/isPartOf",
            "hasPart" to "http://purl.org/dc/terms/hasPart",
            "relation" to "http://purl.org/dc/terms/relation",
            "isReferencedBy" to "http://purl.org/dc/terms/isReferencedBy",
            "references" to "http://purl.org/dc/terms/references",

            // DCAT-specific
            "distribution" to "http://www.w3.org/ns/dcat#distribution",
            "landingPage" to "http://www.w3.org/ns/dcat#landingPage",
            "endpointURL" to "http://www.w3.org/ns/dcat#endpointURL",
            "servesDataset" to "http://www.w3.org/ns/dcat#servesDataset",
            "accessService" to "http://www.w3.org/ns/dcat#accessService"
        )

        return commonProperties[propertyName.lowercase()]
            ?: "http://purl.org/dc/terms/$propertyName"
    }

    /**
     * Get conceptual URI for a resource
     */
    private fun getConceptualUriForResource(resourceUri: String): String? {
        val query = """
        PREFIX dcterms: <http://purl.org/dc/terms/>
        SELECT ?conceptual
        FROM <${Config.GRAPH_PROVENANCE}>
        WHERE {
            <$resourceUri> dcterms:isVersionOf ?conceptual .
        }
    """.trimIndent()

        return graphDBClient.query(query).firstOrNull()?.get("conceptual")
    }

    private fun extractResourceName(uri: String): String {
        val parts = uri.split("/", "#")
        return parts.lastOrNull()?.lowercase() ?: "resource"
    }

    private fun extractPropertyName(uri: String): String {
        val parts = uri.split("/", "#")
        return parts.lastOrNull() ?: "property"
    }

    private fun extractVersionNumber(uri: String): String {
        val regex = Regex(".*/v?([0-9.]+)(-draft)?$")
        return regex.find(uri)?.groupValues?.get(1) ?: "1.0"
    }
}
