package org.epos.graphdb.services

import org.epos.graphdb.Config
import org.epos.graphdb.client.GraphDBClient
import org.epos.graphdb.utils.*

/**
 * Service for managing entity lifecycle transitions.
 * Handles DRAFT -> SUBMITTED -> PUBLISHED -> ARCHIVED workflow.
 */
class EntityLifecycleService(
    private val graphDBClient: GraphDBClient,
    private val provenanceService: ProvenanceService
) {

    /**
     * Gets the conceptual URI from a versioned URI.
     */
    private fun getConceptualUri(versionUri: String): String? =
        graphDBClient.query(Sparql.selectConceptualUri(versionUri))
            .firstOrNull()?.get("conceptual")

    /**
     * Gets the current published version of a conceptual entity.
     */
    fun getCurrentPublishedVersion(conceptualUri: String): String? =
        graphDBClient.query(Sparql.selectCurrentVersion(conceptualUri))
            .firstOrNull()?.get("currentVersion")

    /**
     * Creates a new DRAFT based on the current published version.
     */
    fun createDraft(
        conceptualUri: String,
        newVersionNumber: String,
        agent: String = Config.AGENT_SYSTEM
    ): String {
        // Get current published version
        val currentVersion = getCurrentPublishedVersion(conceptualUri)
            ?: throw IllegalStateException("No published version found for $conceptualUri")

        // Create URI for new draft
        val newDraftUri = "$conceptualUri/$newVersionNumber-draft"

        // Copy triples from published version, replacing subject URI
        val currentModel = graphDBClient.getSubjectTriples(currentVersion, Config.GRAPH_PUBLISHED)
        var newModel = ModelUtils.replaceSubject(currentModel, currentVersion, newDraftUri)

        // Update version metadata
        newModel = newModel.apply {
            removeIf { it.predicate.stringValue().contains("version") }
        }

        // Add new version metadata using DSL
        val versionModel = model {
            subject(newDraftUri) {
                version(newVersionNumber)
                isVersionOf(conceptualUri)
            }
        }
        newModel.addAll(versionModel)

        // Insert into DRAFT graph
        graphDBClient.insertIntoGraph(newModel, Config.GRAPH_DRAFT)

        // Record provenance
        provenanceService.recordDraftCreation(
            newVersionUri = newDraftUri,
            basedOnVersionUri = currentVersion,
            conceptualUri = conceptualUri,
            agent = agent
        )

        println("✅ Created DRAFT: $newDraftUri (based on $currentVersion)")
        return newDraftUri
    }

    /**
     * Submits a DRAFT for review (DRAFT -> SUBMITTED).
     */
    fun submitDraft(draftUri: String, agent: String = Config.AGENT_SYSTEM) {
        requireExists(draftUri, Config.GRAPH_DRAFT, "Draft")

        graphDBClient.moveSubject(draftUri, Config.GRAPH_DRAFT, Config.GRAPH_SUBMITTED)
        provenanceService.recordStateTransition(draftUri, "DRAFT", "SUBMITTED", agent)

        println("✅ Submitted: $draftUri (DRAFT -> SUBMITTED)")
    }

    /**
     * Publishes a SUBMITTED version (SUBMITTED -> PUBLISHED).
     * Archives the previous published version if it exists.
     */
    fun publish(submittedUri: String, agent: String = Config.AGENT_SYSTEM) {
        requireExists(submittedUri, Config.GRAPH_SUBMITTED, "Submitted version")

        // Get conceptual URI
        val conceptualUri = getConceptualUri(submittedUri)
            ?: throw IllegalStateException("Cannot find conceptual URI for $submittedUri")

        // Archive current published version if exists
        getCurrentPublishedVersion(conceptualUri)?.let { currentPublished ->
            graphDBClient.moveSubject(currentPublished, Config.GRAPH_PUBLISHED, Config.GRAPH_ARCHIVED)
            provenanceService.recordStateTransition(currentPublished, "PUBLISHED", "ARCHIVED", agent)
            println("  📦 Archived old version: $currentPublished")
        }

        // Remove "-draft" suffix if present
        val finalUri = submittedUri.removeSuffix("-draft")

        if (finalUri != submittedUri) {
            // URI changed - need to update all triples
            val model = graphDBClient.getSubjectTriples(submittedUri, Config.GRAPH_SUBMITTED)
            val newModel = ModelUtils.replaceSubject(model, submittedUri, finalUri)

            graphDBClient.deleteFromGraph(submittedUri, Config.GRAPH_SUBMITTED)
            graphDBClient.insertIntoGraph(newModel, Config.GRAPH_PUBLISHED)
        } else {
            graphDBClient.moveSubject(submittedUri, Config.GRAPH_SUBMITTED, Config.GRAPH_PUBLISHED)
        }

        // Update conceptual graph
        updateConceptualPointer(conceptualUri, finalUri)

        provenanceService.recordStateTransition(finalUri, "SUBMITTED", "PUBLISHED", agent)

        println("✅ Published: $finalUri (SUBMITTED -> PUBLISHED)")
        println("  🎯 Updated conceptual pointer: $conceptualUri -> $finalUri")
    }

    /**
     * Rejects a SUBMITTED version (SUBMITTED -> DRAFT).
     */
    fun rejectSubmission(submittedUri: String, agent: String = Config.AGENT_SYSTEM) {
        graphDBClient.moveSubject(submittedUri, Config.GRAPH_SUBMITTED, Config.GRAPH_DRAFT)
        provenanceService.recordStateTransition(submittedUri, "SUBMITTED", "DRAFT", agent)
        println("✅ Rejected: $submittedUri (SUBMITTED -> DRAFT)")
    }

    /**
     * Deletes a DRAFT version.
     */
    fun deleteDraft(draftUri: String) {
        graphDBClient.deleteFromGraph(draftUri, Config.GRAPH_DRAFT)
        println("✅ Deleted DRAFT: $draftUri")
    }

    // ============== Private Helpers ==============

    private fun requireExists(uri: String, graphUri: String, entityType: String) {
        val exists = graphDBClient.ask(Sparql.askExists(uri, graphUri))
        if (!exists) {
            throw IllegalArgumentException("$entityType $uri not found in ${graphUri.substringAfterLast("/")}")
        }
    }

    private fun updateConceptualPointer(conceptualUri: String, versionUri: String) {
        // Delete old pointer
        graphDBClient.deleteFromGraph(conceptualUri, Config.GRAPH_CONCEPTUAL)

        // Create new pointer
        val conceptualModel = model {
            subject(conceptualUri) {
                type(NS.DCAT_RESOURCE)
                hasCurrentVersion(versionUri)
            }
        }
        graphDBClient.insertIntoGraph(conceptualModel, Config.GRAPH_CONCEPTUAL)
    }
}